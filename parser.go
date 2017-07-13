package participle

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"

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

// Capture can be implemented by fields in order to transform captured tokens into field values.
type Capture interface {
	Capture(values []string) error
}

// The Parseable interface can be implemented by any element in the grammar to provide custom parsing.
type Parseable interface {
	// Parse into the receiver.
	//
	// Should return NextMatch if no tokens matched and parsing should continue.
	// Nil should be returned if parsing was successful.
	Parse(lex lexer.Lexer) error
}

// A Parser for a particular grammar and lexer.
type Parser struct {
	root node
	lex  lexer.Definition
}

type generatorContext struct {
	lexer.Definition
	typeNodes map[reflect.Type]node
}

func dumpNode(v node) string {
	seen := map[reflect.Value]bool{}
	return nodePrinter(seen, v)
}

func nodePrinter(seen map[reflect.Value]bool, v node) string {
	if seen[reflect.ValueOf(v)] {
		return "<>"
	}
	seen[reflect.ValueOf(v)] = true
	switch n := v.(type) {
	case expression:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return strings.Join(out, "|")

	case *strct:
		return fmt.Sprintf("strct(type=%s, expr=%s)", n.typ, nodePrinter(seen, n.expr))

	case alternative:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return fmt.Sprintf("(%s)", strings.Join(out, " "))

	case *reference:
		return fmt.Sprintf("@(field=%s, node=%s)", n.field.Name, nodePrinter(seen, n.node))

	case *tokenReference:
		return fmt.Sprintf("token(%q)", n.identifier)

	case *optional:
		return fmt.Sprintf("[%s]", nodePrinter(seen, n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", nodePrinter(seen, n.node))

	case *literal:
		return n.String()

	}
	return "?"
}

// MustBuild calls Build(grammar, lex) and panics if an error occurs.
func MustBuild(grammar interface{}, lex lexer.Definition) *Parser {
	parser, err := Build(grammar, lex)
	if err != nil {
		panic(err)
	}
	return parser
}

// Build constructs a parser for the given grammar.
//
// If "lex" is nil, the default lexer based on text/scanner will be used. This scans typical Go-
// like tokens.
//
// See documentation for details
func Build(grammar interface{}, lex lexer.Definition) (parser *Parser, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			if s, ok := msg.(string); ok {
				err = errors.New(s)
			} else if e, ok := msg.(error); ok {
				err = e
			} else {
				panic("unsupported panic type, can not recover")
			}
		}
	}()
	if lex == nil {
		lex = lexer.TextScannerLexer
	}
	context := &generatorContext{
		Definition: lex,
		typeNodes:  map[reflect.Type]node{},
	}
	root := parseType(context, reflect.TypeOf(grammar))
	return &Parser{root: root, lex: lex}, nil
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// participle.Build().
func (p *Parser) Parse(r io.Reader, v interface{}) (err error) {
	lex := p.lex.Lex(r)
	// If the grammar implements Parseable, use it.
	if parseable, ok := v.(Parseable); ok {
		err = parseable.Parse(lex)
		peek := lex.Peek()
		if err == NextMatch {
			return lexer.Errorf(peek.Pos, "invalid syntax")
		}
		if err == nil && !peek.EOF() {
			return lexer.Errorf(peek.Pos, "unexpected token %q", peek)
		}
		return err
	}

	defer func() {
		if msg := recover(); msg != nil {
			if perr, ok := msg.(*lexer.Error); ok {
				err = perr
			} else {
				panicf("unexpected error %s", msg)
			}
		}
	}()
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv := p.root.Parse(lex, rv.Elem())
	if !lex.Peek().EOF() {
		lexer.Panicf(lex.Peek().Pos, "unexpected token %q", lex.Peek())
	}
	if pv == nil {
		lexer.Panic(lex.Peek().Pos, "invalid syntax")
	}
	rv.Elem().Set(reflect.Indirect(pv[0]))
	return
}

// ParseString is a convenience around Parse().
func (p *Parser) ParseString(s string, v interface{}) error {
	return p.Parse(strings.NewReader(s), v)
}

// ParseBytes is a convenience around Parse().
func (p *Parser) ParseBytes(b []byte, v interface{}) error {
	return p.Parse(bytes.NewReader(b), v)
}

// String representation of the grammar.
func (p *Parser) String() string {
	return dumpNode(p.root)
}

func decorate(name string) {
	if msg := recover(); msg != nil {
		panic(name + ": " + msg.(string))
	}
}

// Takes a type and builds a tree of nodes out of it.
func parseType(context *generatorContext, t reflect.Type) node {
	rt := t
	t = indirectType(t)
	defer decorate(t.Name())
	if n, ok := context.typeNodes[t]; ok {
		return n
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Ptr:
		t = indirectType(t.Elem())
		fallthrough

	case reflect.Struct:
		if rt.Implements(parseableType) {
			return &parseable{rt}
		}
		out := &strct{typ: t}
		context.typeNodes[t] = out
		slexer := lexStruct(t)
		defer func() {
			if msg := recover(); msg != nil {
				panic(slexer.Field().Name + ": " + msg.(string))
			}
		}()
		e := parseExpression(context, slexer)
		if !slexer.Peek().EOF() {
			panic("unexpected input " + slexer.Peek().Value)
		}
		out.expr = e
		return out
	}
	panic("expected struct type but got " + t.String())
}

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
	if f := v.FieldByName("Pos"); f.IsValid() {
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
type expression []node

func (e expression) String() string {
	out := []string{}
	for _, n := range e {
		out = append(out, n.String())
	}
	return strings.Join(out, " | ")
}

func (e expression) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
	for _, a := range e {
		if value := a.Parse(lex, parent); value != nil {
			return value
		}
	}
	return nil
}

func parseExpression(context *generatorContext, slexer *structLexer) node {
	out := expression{}
	for {
		out = append(out, parseAlternative(context, slexer))
		if slexer.Peek().Type != '|' {
			break
		}
		slexer.Next() // |
	}
	if len(out) == 1 {
		return out[0]
	}
	return out
}

// <node> ...
type alternative []node

func (a alternative) String() string {
	return a[0].String()
}

func (a alternative) Parse(lex lexer.Lexer, parent reflect.Value) (out []reflect.Value) {
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

func parseAlternative(context *generatorContext, slexer *structLexer) node {
	elements := alternative{}
loop:
	for {
		switch slexer.Peek().Type {
		case lexer.EOF:
			break loop
		default:
			term := parseTerm(context, slexer)
			if term == nil {
				break loop
			}
			elements = append(elements, term)
		}
	}
	if len(elements) == 1 {
		return elements[0]
	}
	return elements
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

func parseTerm(context *generatorContext, slexer *structLexer) node {
	r := slexer.Peek()
	switch r.Type {
	case '@':
		slexer.Next()
		token := slexer.Peek()
		field := slexer.Field()
		if token.Type == '@' {
			slexer.Next()
			return &reference{field, parseType(context, field.Type)}
		}
		if indirectType(field.Type).Kind() == reflect.Struct && !field.Type.Implements(captureType) {
			panic("structs can only be parsed with @@ or by implementing the Capture interface")
		}
		return &reference{field, parseTerm(context, slexer)}
	case scanner.String, scanner.RawString, scanner.Char:
		return parseLiteral(context, slexer)
	case '[':
		return parseOptional(context, slexer)
	case '{':
		return parseRepetition(context, slexer)
	case '(':
		return parseGroup(context, slexer)
	case scanner.Ident:
		return parseTokenReference(context, slexer)
	case lexer.EOF:
		slexer.Next()
		return nil
	default:
		return nil
	}
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

// A reference in the form <identifier> refers to an existing production,
// typically from the lex struct provided to Parse().
func parseTokenReference(context *generatorContext, slexer *structLexer) node {
	token := slexer.Next()
	if token.Type != scanner.Ident {
		panic("expected identifier")
	}
	typ, ok := context.Symbols()[token.Value]
	if !ok {
		panicf("unknown token type %q", token.String())
	}
	return &tokenReference{typ, token.Value}
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

func parseOptional(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // [
	optional := &optional{parseExpression(context, slexer)}
	next := slexer.Peek()
	if next.Type != ']' {
		panic("expected ] but got " + next.String())
	}
	slexer.Next()
	return optional
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

func parseRepetition(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // {
	n := &repetition{
		node: parseExpression(context, slexer),
	}
	next := slexer.Next()
	if next.Type != '}' {
		panic("expected } but got " + next.String())
	}
	return n
}

func parseGroup(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // (
	n := parseExpression(context, slexer)
	next := slexer.Peek() // )
	if next.Type != ')' {
		panic("expected ) but got " + next.Value)
	}
	slexer.Next() // )
	return n
}

func parseLiteral(context *generatorContext, lex *structLexer) node { // nolint: interfacer
	token := lex.Next()
	if token.Type != scanner.String && token.Type != scanner.RawString && token.Type != scanner.Char {
		panic("expected quoted string but got " + token.String())
	}
	s := token.Value
	t := rune(-1)
	token = lex.Peek()
	if token.Value == ":" {
		lex.Next()
		token = lex.Next()
		if token.Type != scanner.Ident {
			panic("expected identifier for literal type constraint but got " + token.String())
		}
		var ok bool
		t, ok = context.Symbols()[token.Value]
		if !ok {
			panic("unknown token type " + token.String() + " in literal type constraint")
		}
	}
	return &literal{s: s, t: t}
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

func conform(t reflect.Type, values []reflect.Value) (out []reflect.Value) {
	var last reflect.Value
	for _, v := range values {
		if last.IsValid() && last != v {
			panicf("inconsistent types %s and %s", v.Type(), last.Type())
		}
		last = v

		for t != v.Type() && t.Kind() == reflect.Ptr && v.Kind() != reflect.Ptr {
			v = v.Addr()
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
	f := strct.FieldByIndex(field.Index)
	switch f.Kind() {
	case reflect.Slice:
		fieldValue = conform(f.Type().Elem(), fieldValue)
		f.Set(reflect.Append(f, fieldValue...))

	case reflect.Ptr:
		if f.IsNil() {
			fv := reflect.New(f.Type().Elem()).Elem()
			f.Set(fv.Addr())
			f = fv
		} else {
			f = f.Elem()
		}
		fallthrough

	default:
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

		switch f.Kind() {
		case reflect.String:
			for _, v := range fieldValue {
				f.Set(reflect.ValueOf(f.String() + v.String()))
			}

		case reflect.Struct:
			if len(fieldValue) != 1 {
				values := []interface{}{}
				for _, v := range fieldValue {
					values = append(values, v.Interface())
				}
				panicf("a single value must be assigned to struct field but have %#v", values)
			}
			f.Set(fieldValue[0])

		case reflect.Bool:
			f.Set(reflect.ValueOf(true))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if len(fieldValue) != 1 {
				panicf("a single value must be assigned to an integer field but have %#v", fieldValue)
			}
			n, err := strconv.ParseInt(fieldValue[0].String(), 10, 64)
			if err != nil {
				panicf("expected integer but got %q", fieldValue[0].String())
			}
			f.SetInt(n)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if len(fieldValue) != 1 {
				panicf("a single value must be assigned to an unsigned integer field but have %#v", fieldValue)
			}
			n, err := strconv.ParseUint(fieldValue[0].String(), 10, 64)
			if err != nil {
				panicf("expected unsigned integer but got %q", fieldValue[0].String())
			}
			f.SetUint(n)

		case reflect.Float32, reflect.Float64:
			if len(fieldValue) != 1 {
				panicf("a single value must be assigned to a float field but have %#v", fieldValue)
			}
			n, err := strconv.ParseFloat(fieldValue[0].String(), 10)
			if err != nil {
				panicf("expected float but got %q", fieldValue[0].String())
			}
			f.SetFloat(n)

		default:
			panicf("unsupported field type %s for field %s", f.Type(), field.Name)
		}
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
