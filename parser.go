// Package parser constructs parsers from definitions in struct tags and parses directly into those
// structs. The approach is philosophically similar to how other marshallers work in Go,
// "unmarshalling" an instance of a grammar into a struct.
//
// The annotation syntax supported is:
//
// - `@<expr>` Capture subexpression into the field.
// - `@@` Recursively capture using the fields own type.
// - `@Identifier` Match token of the given name and capture it.
// - `{ ... }` Match 0 or more times.
// - `( ... )` Group.
// - `[ ... ]` Optional.
// - `"..."` Match the literal.
// - `"."…"."` Match rune in range.
// - `.` Period matches any single character.
// - `<expr> | <expr>` Match one of the alternatives.
//
// Here's an example of an EBNF grammar.
//
// 		type Group struct {
// 			Expression *Expression `"(" @@ ")""`
// 		}
//
// 		type Option struct {
// 			Expression *Expression `"[" @@ "]""`
// 		}
//
// 		type Repetition struct {
// 			Expression *Expression `"{" @@ "}""`
// 		}
//
// 		type Literal struct {
// 			Start string `@String"` // Lexer token "String""
// 			End   string `[ "…" @String ]"`
// 		}
//
// 		type Term struct {
// 			Name       string      `@Ident |"`
// 			Literal    *Literal    `@@ |"`
// 			Group      *Group      `@@ |"`
// 			Option     *Option     `@@ |"`
// 			Repetition *Repetition `@@"`
// 		}
//
// 		type Sequence struct {
// 			Terms []*Term `@@ { @@ }"`
// 		}
//
// 		type Expression struct {
// 			Alternatives []*Sequence `@@ { "|" @@ }"`
// 		}
//
// 		type Expressions []*Expression
//
// 		type Production struct {
// 			Name        string      `@Ident "=""`
// 			Expressions Expressions `@@ { @@ } ".""`
// 		}
//
// 		type EBNF struct {
// 			Productions []*Production `{ @@ }"`
// 		}

package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

// A node in the grammar.
type node interface {
	// Parse from scanner into value.
	Parse(lexer Lexer, parent reflect.Value) reflect.Value
	String() string
}

type Parser struct {
	root  node
	lexer LexerDefinition
}

type generatorContext struct {
	LexerDefinition
	typeNodes map[reflect.Type]node
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

func (p *Parser) String() string {
	return p.root.String()
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
	if !pv.IsValid() {
		panic("invalid syntax")
	}
	rv.Elem().Set(reflect.Indirect(pv))
	return
}

func (p *Parser) ParseString(s string, v interface{}) error {
	return p.Parse(strings.NewReader(s), v)
}

func (p *Parser) ParseBytes(b []byte, v interface{}) error {
	return p.Parse(bytes.NewReader(b), v)
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
	return fmt.Sprintf("strct(type=%s, expr=%s)", s.typ, s.expr)
}

func (s *strct) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	sv := reflect.New(s.typ).Elem()
	v := s.expr.Parse(lexer, sv)
	if !v.IsValid() {
		return v
	}
	return sv
}

// <expr> {"|" <expr>}
type expression []node

func (e expression) String() string {
	out := []string{}
	for _, n := range e {
		out = append(out, n.String())
	}
	return strings.Join(out, "|")
}

func (e expression) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	for _, a := range e {
		if value := a.Parse(lexer, parent); value.IsValid() {
			return value
		}
	}
	return reflect.Value{}
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
	return out
}

// <node> ...
type alternative []node

func (a alternative) String() string {
	out := []string{}
	for _, n := range a {
		out = append(out, n.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(out, " "))
}

func (a alternative) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	var value reflect.Value
	for i, n := range a {
		// If first value doesn't match, we early exit, otherwise all values must match.
		value = n.Parse(lexer, parent)
		if !value.IsValid() {
			if i == 0 {
				return reflect.Value{}
			}
			panicf("expected %s", n)
		}
	}
	return value
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
	return elements
}

// .
type dot struct{}

func (d dot) String() string {
	return "."
}

func (d dot) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	r := lexer.Next()
	if r.EOF() {
		return reflect.Value{}
	}
	return reflect.ValueOf(r)
}

// @<expr>
type reference fieldReceiver

func (r *reference) String() string {
	return fmt.Sprintf("@(field=%s, node=%s)", r.field.Name, r.node)
}

func (r *reference) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	v := r.node.Parse(lexer, parent)
	if !v.IsValid() {
		return v
	}
	setField(parent, r.field, v)
	return v
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
		return parseRepitition(context, slexer)
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
	return fmt.Sprintf("token(%q)", t.identifier)
}

func (t *tokenReference) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	token := lexer.Peek()
	if token.Type != t.typ {
		return reflect.Value{}
	}
	lexer.Next()
	return reflect.ValueOf(token.Value)
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

type fieldReceiver struct {
	field reflect.StructField
	node  node
}

// [ <expr> ]
type optional struct {
	node node
}

func (o *optional) String() string {
	return fmt.Sprintf("[%s]", o.node)
}

func (o *optional) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	v := o.node.Parse(lexer, parent)
	if !v.IsValid() {
		// FIXME(alec): This is a bit of a hack. Without this, the optional itself is treated as
		// invalid and parsing backtracks, even though it should "match" nothing.
		// A fix is to return some kind of sentinel value that @ recognises and discards.
		return reflect.ValueOf("")
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
type repitition fieldReceiver

func (r *repitition) String() string {
	return fmt.Sprintf("{ %s }", r.node)
}

func (r *repitition) Parse(lexer Lexer, parent reflect.Value) (out reflect.Value) {
	switch r.field.Type.Kind() {
	case reflect.Slice:
		out = reflect.MakeSlice(r.field.Type, 0, 0)
	default:
		typ := indirectType(r.field.Type)
		if typ.Kind() == reflect.String {
			out = reflect.New(typ).Elem()
		} else {
			panicf("can't accumulate into %s", r.field.Type)
		}
	}
	for {
		v := r.node.Parse(lexer, parent)
		if !v.IsValid() {
			break
		}
		if out.Type().Kind() == reflect.Slice && out.Type().Elem().Kind() == reflect.Ptr {
			v = v.Addr()
		}
		switch r.field.Type.Kind() {
		case reflect.Slice:
			out = reflect.Append(out, v)
		case reflect.String:
			out = reflect.ValueOf(out.String() + v.String())
		}
	}
	return out
}

func parseRepitition(context *generatorContext, slexer *structLexer) node {
	field := slexer.Field()
	slexer.Next() // {
	n := &repitition{
		field: field,
		node:  parseExpression(context, slexer),
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

func (s srange) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	token := lexer.Peek()
	if token.Type < s.start || token.Type > s.end {
		return reflect.Value{}
	}
	lexer.Next()
	return reflect.ValueOf(token.Value)
}

// Match a string exactly "..."
type str string

func (s str) String() string {
	return fmt.Sprintf("%q", string(s))
}

func (s str) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	token := lexer.Peek()
	if token.Value != string(s) {
		return reflect.Value{}
	}
	return reflect.ValueOf(lexer.Next().Value)
}

// Set field.
//
// If field is a pointer the pointer will be set to the value. If field is a string, value will be
// appended. If field is a slice, value will be appended to slice.
func setField(strct reflect.Value, field reflect.StructField, fieldValue reflect.Value) {
	fieldValue = reflect.Indirect(fieldValue)
	f := strct.FieldByIndex(field.Index)
	switch f.Kind() {
	case reflect.Slice:
		if fieldValue.Kind() == reflect.Struct {
			fieldValue = fieldValue.Addr()
		}
		f.Set(reflect.Append(f, fieldValue))

	case reflect.Ptr:
		ptr := reflect.New(fieldValue.Type())
		ptr.Elem().Set(fieldValue)
		fieldValue = ptr
		fallthrough
	default:
		if f.Kind() == reflect.String {
			// For strings, we append.
			fieldValue = reflect.ValueOf(f.String() + fieldValue.String())
		}
		f.Set(fieldValue)
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
