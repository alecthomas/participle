package participle

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

var (
	positionType  = reflect.TypeOf(lexer.Position{})
	captureType   = reflect.TypeOf((*Capture)(nil)).Elem()
	parseableType = reflect.TypeOf((*Parseable)(nil)).Elem()

	// NextMatch should be returned by Parseable.Parse() method implementations to indicate
	// that the node did not match and that other matches should be attempted, if appropriate.
	NextMatch = errors.New("no match") // nolint: golint
)

// A node in the grammar.
type node interface {
	// Parse from scanner into value.
	// Nodes should panic if parsing fails.
	Parse(lex lexer.PeekingLexer, parent reflect.Value) []reflect.Value
}

func decorate(name func() string) {
	if msg := recover(); msg != nil {
		switch msg := msg.(type) {
		case Error:
			panicf("%s: %s", name(), msg)
		case *lexer.Error:
			panic(&lexer.Error{Message: name() + ": " + msg.Message, Pos: msg.Pos})
		default:
			panic(msg)
		}
	}
}

func recoverToError(err *error) {
	if msg := recover(); msg != nil {
		switch msg := msg.(type) {
		case Error:
			*err = msg
		case *lexer.Error:
			*err = msg
		default:
			panic(msg)
		}
	}
}

// A node that proxies to an implementation that implements the Parseable interface.
type parseable struct {
	t reflect.Type
}

func (p *parseable) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	rv := reflect.New(p.t)
	v := rv.Interface().(Parseable)
	err := v.Parse(lex)
	if err != nil {
		if err == NextMatch {
			return nil
		}
		panic(err)
	}
	return []reflect.Value{rv.Elem()}
}

type strct struct {
	typ  reflect.Type
	expr node
}

func (s *strct) maybeInjectPos(pos lexer.Position, v reflect.Value) {
	// Fast path
	if f := v.FieldByName("Pos"); f.IsValid() && f.Type() == positionType {
		f.Set(reflect.ValueOf(pos))
		return
	}

	// Iterate over fields.
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() == positionType {
			f.Set(reflect.ValueOf(pos))
			break
		}
	}
}

func (s *strct) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	sv := reflect.New(s.typ).Elem()
	s.maybeInjectPos(lex.Peek(0).Pos, sv)
	if s.expr.Parse(lex, sv) == nil {
		return nil
	}
	return []reflect.Value{sv}
}

// <expr> {"|" <expr>}
type disjunction struct {
	nodes     []node
	lookahead []lookahead
}

func (e *disjunction) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	// for _, look := range e.lookahead {
	// 	lt := look.token

	// 	// This will occur with Parseable interfaces.
	// 	if look.depth == 0 && lt.EOF() {
	// 		if out = e.nodes[look.root].Parse(lex, parent); out != nil {
	// 			return out
	// 		}
	// 		continue
	// 	}

	// 	t := lex.Peek(look.depth)
	// 	if (lt.Value == "" || lt.Value == t.Value) && (lt.Type == lexer.EOF || lt.Type == t.Type) {
	// 		// repr.Println(lt, t)
	// 		// fmt.Println(stringer(e.nodes[look.root], 1))
	// 		return e.nodes[look.root].Parse(lex, parent)
	// 	}
	// }

	// Same logic without lookahead.
	for _, a := range e.nodes {
		if value := a.Parse(lex, parent); value != nil {
			return value
		}
	}
	return nil
}

// <node> ...
type sequence struct {
	head bool
	node node
	next *sequence
}

func (a *sequence) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	for n := a; n != nil; n = n.next {
		child := n.node.Parse(lex, parent)
		if child == nil {
			// Early exit if first value doesn't match, otherwise all values must match.
			if n == a {
				return nil
			}
			lexer.Panicf(lex.Peek(0).Pos, "unexpected %q (expected %s)", lex.Peek(0), stringer(n, 1))
		}
		out = append(out, child...)
	}
	return out
}

// @<expr>
type capture struct {
	field reflect.StructField
	node  node
}

func (r *capture) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	pos := lex.Peek(0).Pos
	v := r.node.Parse(lex, parent)
	if v == nil {
		return nil
	}
	setField(pos, parent, r.field, v)
	return []reflect.Value{parent}
}

// <identifier> - named lexer token reference
type reference struct {
	typ        rune
	identifier string // Used for informational purposes.
}

func (t *reference) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	token := lex.Peek(0)
	if token.Type != t.typ {
		return nil
	}
	return []reflect.Value{reflect.ValueOf(lex.Next().Value)}
}

// [ <expr> ]
type optional struct {
	node node
}

func (o *optional) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	v := o.node.Parse(lex, parent)
	if v == nil {
		return []reflect.Value{}
	}
	return v
}

// { <expr> }
type repetition struct {
	node node
}

// Parse a repetition. Once a repetition is encountered it will always match, so grammars
// should ensure that branches are differentiated prior to the repetition.
func (r *repetition) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	out = []reflect.Value{}
	for {
		v := r.node.Parse(lex, parent)
		if v == nil {
			break
		}
		out = append(out, v...)
	}
	return out
}

// Match a token literal exactly "..."[:<type>].
type literal struct {
	s  string
	t  rune
	tt string // Used for display purposes - symbolic name of t.
}

func (s *literal) Parse(lex lexer.PeekingLexer, parent reflect.Value) (out []reflect.Value) {
	token := lex.Peek(0)
	if token.Value == s.s && (s.t == -1 || s.t == token.Type) {
		return []reflect.Value{reflect.ValueOf(lex.Next().Value)}
	}
	return nil
}

// Attempt to transform values to given type.
//
// This will dereference pointers, and attempt to parse strings into integer values, floats, etc.
func conform(t reflect.Type, values []reflect.Value) (out []reflect.Value) {
	for _, v := range values {
		for t != v.Type() && t.Kind() == reflect.Ptr && v.Kind() != reflect.Ptr {
			v = v.Addr()
		}

		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(v.String(), 0, 64)
			if err == nil {
				v = reflect.New(t).Elem()
				v.SetInt(n)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(v.String(), 0, 64)
			if err == nil {
				v = reflect.New(t).Elem()
				v.SetUint(n)
			}

		case reflect.Bool:
			v = reflect.ValueOf(true)

		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(v.String(), 64)
			if err == nil {
				v = reflect.New(t).Elem()
				v.SetFloat(n)
			}
		}

		out = append(out, v)
	}
	return out
}

// Set field.
//
// If field is a pointer the pointer will be set to the value. If field is a string, value will be
// appended. If field is a slice, value will be appended to slice.
//
// For all other types, an attempt will be made to convert the string to the corresponding
// type (int, float32, etc.).
func setField(pos lexer.Position, strct reflect.Value, field reflect.StructField, fieldValue []reflect.Value) { // nolint: gocyclo
	defer decorate(func() string { return strct.Type().String() + "." + field.Name })

	f := strct.FieldByIndex(field.Index)
	switch f.Kind() {
	case reflect.Slice:
		fieldValue = conform(f.Type().Elem(), fieldValue)
		f.Set(reflect.Append(f, fieldValue...))
		return

	case reflect.Ptr:
		if f.IsNil() {
			fv := reflect.New(f.Type().Elem()).Elem()
			f.Set(fv.Addr())
			f = fv
		} else {
			f = f.Elem()
		}
	}

	if f.CanAddr() {
		if d, ok := f.Addr().Interface().(Capture); ok {
			ifv := []string{}
			for _, v := range fieldValue {
				ifv = append(ifv, v.Interface().(string))
			}
			err := d.Capture(ifv)
			if err != nil {
				lexer.Panic(pos, err.Error())
			}
			return
		}
	}

	// Strings concatenate all captured tokens.
	if f.Kind() == reflect.String {
		fieldValue = conform(f.Type(), fieldValue)
		for _, v := range fieldValue {
			f.Set(reflect.ValueOf(f.String() + v.String()).Convert(f.Type()))
		}
		return
	}

	// Coalesce multiple tokens into one. This allow eg. ["-", "10"] to be captured as separate tokens but
	// parsed as a single string "-10".
	if len(fieldValue) > 1 {
		out := []string{}
		for _, v := range fieldValue {
			out = append(out, v.String())
		}
		fieldValue = []reflect.Value{reflect.ValueOf(strings.Join(out, ""))}
	}

	fieldValue = conform(f.Type(), fieldValue)

	fv := fieldValue[0]

	switch f.Kind() {
	// Numeric types will increment if the token can not be coerced.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fv.Type() != f.Type() {
			f.SetInt(f.Int() + 1)
		} else {
			f.Set(fv)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if fv.Type() != f.Type() {
			f.SetUint(f.Uint() + 1)
		} else {
			f.Set(fv)
		}

	case reflect.Float32, reflect.Float64:
		if fv.Type() != f.Type() {
			f.SetFloat(f.Float() + 1)
		} else {
			f.Set(fv)
		}

	case reflect.Bool, reflect.Struct:
		if fv.Type() != f.Type() {
			panicf("value %q is not correct type %s", fv, f.Type())
		}
		f.Set(fv)

	default:
		panicf("unsupported field type %s for field %s", f.Type(), field.Name)
	}
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		return indirectType(t.Elem())
	}
	return t
}

func panicf(f string, args ...interface{}) {
	panic(Error(fmt.Sprintf(f, args...)))
}

type Error string

func (e Error) Error() string { return string(e) }
