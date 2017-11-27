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
	Parse(lex lexer.Lexer, parent reflect.Value) []reflect.Value
	String() string
}

func decorate(name string) {
	if msg := recover(); msg != nil {
		panic(name + ": " + msg.(string))
	}
}

// A node that proxies to an implementation that implements the Parseable interface.
type parseable struct {
	t reflect.Type
}

func (p *parseable) String() string {
	return p.t.String()
}

func (p *parseable) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	rv := reflect.New(p.t.Elem())
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

func (s *strct) String() string {
	return s.expr.String()
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

func (s *strct) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	sv := reflect.New(s.typ).Elem()
	s.maybeInjectPos(lex.Peek().Pos, sv)
	if s.expr.Parse(lex, sv) == nil {
		return nil
	}
	return []reflect.Value{sv}
}

// <expr> {"|" <expr>}
type disjunction []node

func (e disjunction) String() string {
	out := []string{}
	for _, n := range e {
		out = append(out, n.String())
	}
	return strings.Join(out, " | ")
}

func (e disjunction) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	for _, a := range e {
		if value := a.Parse(lex, parent); value != nil {
			return value
		}
	}
	return nil
}

// <node> ...
type sequence []node

func (a sequence) String() string {
	return a[0].String()
}

func (a sequence) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	for i, n := range a {
		// If first value doesn't match, we early exit, otherwise all values must match.
		child := n.Parse(lex, parent)
		if child == nil {
			if i == 0 {
				return nil
			}
			lexer.Panicf(lex.Peek().Pos, "expected ( %s ) not %q", n, lex.Peek())
		}
		if len(child) == 0 && out == nil {
			out = []reflect.Value{}
		} else {
			out = append(out, child...)
		}
	}
	return out
}

// @<expr>
type reference struct {
	field reflect.StructField
	node  node
}

func (r *reference) String() string {
	return r.field.Name + ":" + r.node.String()
}

func (r *reference) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	pos := lex.Peek().Pos
	v := r.node.Parse(lex, parent)
	if v == nil {
		return nil
	}
	setField(pos, parent, r.field, v)
	return []reflect.Value{parent}
}

type tokenReference struct {
	typ        rune
	identifier string
}

func (t *tokenReference) String() string {
	return t.identifier
}

func (t *tokenReference) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	token := lex.Peek()
	if token.Type != t.typ {
		return nil
	}
	lex.Next()
	return []reflect.Value{reflect.ValueOf(token.Value)}
}

// [ <expr> ]
type optional struct {
	node node
}

func (o *optional) String() string {
	return o.node.String()
}

func (o *optional) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
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

func (r *repetition) String() string {
	return r.node.String()
}

// Parse a repetition. Once a repetition is encountered it will always match, so grammars
// should ensure that branches are differentiated prior to the repetition.
func (r *repetition) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
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

// Match a token literal exactly "...".
type literal struct {
	s string
	t rune
}

func (s *literal) String() string {
	if s.t != -1 {
		return fmt.Sprintf("%q:%d", s.s, s.t)
	}
	return fmt.Sprintf("%q", s.s)
}

func (s *literal) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	token := lex.Peek()
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
	defer decorate(strct.Type().String() + "." + field.Name)

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

	fieldValue = conform(f.Type(), fieldValue)

	// Strings concatenate all captured tokens.
	if f.Kind() == reflect.String {
		for _, v := range fieldValue {
			f.Set(reflect.ValueOf(f.String() + v.String()))
		}
		return
	}

	// All other types are treated as scalar.
	if len(fieldValue) != 1 {
		values := []interface{}{}
		for _, v := range fieldValue {
			values = append(values, v.Interface())
		}
		panicf("a single value must be assigned to a field of type %s but have %#v", f.Type(), values)
	}

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
	panic(fmt.Sprintf(f, args...))
}
