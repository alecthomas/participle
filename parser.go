// Package participle constructs parsers from definitions in struct tags and parses directly into
// those structs. The approach is philosophically similar to how other marshallers work in Go,
// "unmarshalling" an instance of a grammar into a struct.
//
// The supported annotation syntax is:
//
//     - `@<expr>` Capture subexpression into the field.
//     - `@@` Recursively capture using the fields own type.
//     - `@Identifier` Match token of the given name and capture it.
//     - `{ ... }` Match 0 or more times.
//     - `( ... )` Group.
//     - `[ ... ]` Optional.
//     - `"..."` Match the literal.
//     - `"."…"."` Match rune in range.
//     - `.` Period matches any single character.
//     - `<expr> | <expr>` Match one of the alternatives.
//
// Here's an example of an EBNF grammar.
//
//     type Group struct {
//         Expression *Expression `"(" @@ ")"`
//     }
//
//     type Option struct {
//         Expression *Expression `"[" @@ "]"`
//     }
//
//     type Repetition struct {
//         Expression *Expression `"{" @@ "}"`
//     }
//
//     type Literal struct {
//         Start string `@String` // Lexer token "String"
//         End   string `[ "…" @String ]`
//     }
//
//     type Term struct {
//         Name       string      `@Ident |`
//         Literal    *Literal    `@@ |`
//         Group      *Group      `@@ |`
//         Option     *Option     `@@ |`
//         Repetition *Repetition `@@`
//     }
//
//     type Sequence struct {
//         Terms []*Term `@@ { @@ }`
//     }
//
//     type Expression struct {
//         Alternatives []*Sequence `@@ { "|" @@ }`
//     }
//
//     type Expressions []*Expression
//
//     type Production struct {
//         Name        string      `@Ident "="`
//         Expressions Expressions `@@ { @@ } "."`
//     }
//
//     type EBNF struct {
//         Productions []*Production `{ @@ }`
//     }
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
	"unicode/utf8"
)

var (
	positionType = reflect.TypeOf(Position{})
)

// A node in the grammar.
type node interface {
	// Parse from scanner into value.
	Parse(lexer Lexer, parent reflect.Value) []reflect.Value
	String() string
}

// Parseable will be called if a receiver in the grammar implements it.
type Parseable interface {
	Parse(values []string) error
}

type Parser struct {
	root  node
	lexer LexerDefinition
}

type generatorContext struct {
	LexerDefinition
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

	case dot:
		return "."

	case *reference:
		return fmt.Sprintf("@(field=%s, node=%s)", n.field.Name, nodePrinter(seen, n.node))

	case *tokenReference:
		return fmt.Sprintf("token(%q)", n.identifier)

	case *optional:
		return fmt.Sprintf("[%s]", nodePrinter(seen, n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", nodePrinter(seen, n.node))

	case srange:
		return fmt.Sprintf("%q … %q", n.start, n.end)

	case str:
		return fmt.Sprintf("%q", string(n))

	}
	return "?"
}

func MustParse(grammar interface{}, lexer LexerDefinition) *Parser {
	parser, err := Parse(grammar, lexer)
	if err != nil {
		panic(err)
	}
	return parser
}

// Generate a parser for the given grammar.
func Parse(grammar interface{}, lexer LexerDefinition) (parser *Parser, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(msg.(string))
		}
	}()
	if lexer == nil {
		lexer = DefaultLexerDefinition
	}
	context := &generatorContext{
		LexerDefinition: lexer,
		typeNodes:       map[reflect.Type]node{},
	}
	root := parseType(context, reflect.TypeOf(grammar))
	return &Parser{root: root, lexer: lexer}, nil
}

// Parse from Lexer l into grammar v.
func (p *Parser) Parse(r io.Reader, v interface{}) (err error) {
	lexer := p.lexer.Lex(r)
	defer func() {
		if msg := recover(); msg != nil {
			pos := lexer.Position()
			err = fmt.Errorf("%s:%d:%d: %s (near %q)",
				pos.Filename, pos.Line, pos.Column, msg, lexer.Peek())
		}
	}()
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv := p.root.Parse(lexer, rv.Elem())
	if !lexer.Peek().EOF() {
		panic("unexpected token")
	}
	if pv == nil {
		panic("invalid syntax")
	}
	rv.Elem().Set(reflect.Indirect(pv[0]))
	return
}

func (p *Parser) ParseString(s string, v interface{}) error {
	return p.Parse(strings.NewReader(s), v)
}

func (p *Parser) ParseBytes(b []byte, v interface{}) error {
	return p.Parse(bytes.NewReader(b), v)
}

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
	defer decorate(indirectType(t).Name())
	if n, ok := context.typeNodes[t]; ok {
		return n
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Ptr:
		t = indirectType(t.Elem())
		fallthrough

	case reflect.Struct:
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
			panic("unexpected input " + string(slexer.Peek().Value))
		}
		out.expr = e
		return out
	}
	panic("expected struct type but got " + t.String())
}

type strct struct {
	typ  reflect.Type
	expr node
}

func (s *strct) String() string {
	return s.expr.String()
}

func (s *strct) maybeInjectPos(lexer Lexer, v reflect.Value) {
	// Fast path
	if f := v.FieldByName("Pos"); f.IsValid() {
		f.Set(reflect.ValueOf(lexer.Position()))
		return
	}

	// Iterate over fields.
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() == positionType {
			f.Set(reflect.ValueOf(lexer.Position()))
			break
		}
	}
}

func (s *strct) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	sv := reflect.New(s.typ).Elem()
	s.maybeInjectPos(lexer, sv)
	if s.expr.Parse(lexer, sv) == nil {
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

func (e expression) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	for _, a := range e {
		if value := a.Parse(lexer, parent); value != nil {
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

func (a alternative) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	for i, n := range a {
		// If first value doesn't match, we early exit, otherwise all values must match.
		child := n.Parse(lexer, parent)
		if child == nil {
			if i == 0 {
				return nil
			}
			panicf("expected %s", n)
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
		case EOF:
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

// .
type dot struct{}

func (d dot) String() string {
	return "."
}

func (d dot) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	r := lexer.Next()
	if r.EOF() {
		return nil
	}
	return []reflect.Value{reflect.ValueOf(r)}
}

// @<expr>
type reference struct {
	field reflect.StructField
	node  node
}

func (r *reference) String() string {
	return r.field.Name + ":" + r.node.String()
}

func (r *reference) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	v := r.node.Parse(lexer, parent)
	if v == nil {
		return nil
	}
	setField(parent, r.field, v)
	return []reflect.Value{parent}
}

func parseTerm(context *generatorContext, slexer *structLexer) node {
	r := slexer.Peek()
	switch r.Type {
	case '.':
		slexer.Next()
		return dot{}
	case '@':
		slexer.Next()
		token := slexer.Peek()
		field := slexer.Field()
		if token.Type == '@' {
			slexer.Next()
			defer decorate(field.Name)
			return &reference{field, parseType(context, indirectType(field.Type))}
		}
		if indirectType(field.Type).Kind() == reflect.Struct {
			panic("structs can only be parsed with @@")
		}
		return &reference{field, parseTerm(context, slexer)}
	case scanner.String, scanner.RawString, scanner.Char:
		return parseQuotedStringOrRange(slexer)
	case '[':
		return parseOptional(context, slexer)
	case '{':
		return parseRepetition(context, slexer)
	case '(':
		return parseGroup(context, slexer)
	case scanner.Ident:
		return parseTokenReference(context, slexer)
	case EOF:
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
	return fmt.Sprintf("%s", t.identifier)
}

func (t *tokenReference) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	token := lexer.Peek()
	if token.Type != t.typ {
		return nil
	}
	lexer.Next()
	return []reflect.Value{reflect.ValueOf(token.Value)}
}

// A reference in the form <identifier> refers to an existing production,
// typically from the lexer struct provided to Parse().
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

func (o *optional) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	v := o.node.Parse(lexer, parent)
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
func (r *repetition) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	out = []reflect.Value{}
	for {
		v := r.node.Parse(lexer, parent)
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

func parseQuotedStringOrRange(lexer *structLexer) node {
	start := parseQuotedString(lexer)
	if lexer.Peek().Type != '…' {
		return str(start)
	}
	if len(start) != 1 {
		panic("start of range must be 1 character long")
	}
	lexer.Next() // …
	end := parseQuotedString(lexer)
	if len(end) != 1 {
		panic("end of range must be 1 character long")
	}
	startch, _ := utf8.DecodeRuneInString(start)
	endch, _ := utf8.DecodeRuneInString(end)
	return srange{startch, endch}
}

func parseQuotedString(lexer *structLexer) string {
	token := lexer.Next()
	if token.Type != scanner.String && token.Type != scanner.RawString && token.Type != scanner.Char {
		panic("expected quoted string but got " + token.String())
	}
	return token.Value
}

// "a" … "b"
type srange struct {
	start rune
	end   rune
}

func (s srange) String() string {
	return fmt.Sprintf("%q … %q", s.start, s.end)
}

func (s srange) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	token := lexer.Peek()
	if token.Type < s.start || token.Type > s.end {
		return nil
	}
	lexer.Next()
	return []reflect.Value{reflect.ValueOf(token.Value)}
}

// Match a string exactly "..."
type str string

func (s str) String() string {
	return fmt.Sprintf("%q", string(s))
}

func (s str) Parse(lexer Lexer, parent reflect.Value) (out []reflect.Value) {
	token := lexer.Peek()
	if token.Value != string(s) {
		return nil
	}
	return []reflect.Value{reflect.ValueOf(lexer.Next().Value)}
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
func setField(strct reflect.Value, field reflect.StructField, fieldValue []reflect.Value) {
	f := strct.FieldByIndex(field.Index)
	switch f.Kind() {
	case reflect.Slice:
		fieldValue = conform(f.Type().Elem(), fieldValue)
		f.Set(reflect.Append(f, fieldValue...))

	case reflect.Ptr:
		fv := reflect.New(f.Type().Elem()).Elem()
		f.Set(fv.Addr())
		f = fv
		fallthrough

	default:
		if f.CanAddr() {
			if p, ok := f.Addr().Interface().(Parseable); ok {
				ifv := []string{}
				for _, v := range fieldValue {
					ifv = append(ifv, v.Interface().(string))
				}
				err := p.Parse(ifv)
				if err != nil {
					panic(err.Error())
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
				panicf("expected integer but got %q", fieldValue[0].String())
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
