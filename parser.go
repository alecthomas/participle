// Package parser constructs parsers from definitions in struct tags and parses directly into those
// structs. The approach is philosophically similar to how other marshallers work in Go,
// "unmarshalling" an instance of a grammar into a struct.
//
// The annotation syntax supported is:
//
// - `@<term>` Capture term into the field.
//   - `@@` Recursively capture using the fields own type.
//   - `@Identifier` Map token of the given name onto the field.
// - `{ ... }` Match 0 or more times.
// - `( ... )` Group.
// - `[ ... ]` Optional.
// - `"..."` Match the literal.
// - `"."…"."` Match rune in range.
// - `.` Period matches any single character.
// - `... | ...` Match one of the alternatives.
//
// Here's an example of an EBNF grammar.
//
//     type EBNF struct {
//       Productions []*Production
//     }
//
//     type Production struct {
//       Name       string      `@Ident "="`
//       Expression *Expression `[ @@ ] "."`
//     }
//
//     type Expression struct {
//       Alternatives []*Term `@@ { "|" @@ }`
//     }
//
//     type Term struct {
//       Name       *string       `@Ident |`
//       TokenRange *TokenRange   `@@ |`
//       Group      *Group        `@@ |`
//       Option     *Option       `@@ |`
//       Repetition *Repetition   `@@`
//     }
//
//     type Group struct {
//       Expression *Expression `"(" @@ ")"`
//     }
//
//     type Option struct {
//       Expression *Expression `"[" @@ "]"`
//     }
//
//     type Repetition struct {
//       Expression *Expression `"{" @@ "}"`
//     }
//
//     type TokenRange struct {
//       Start string  `@String` // Lexer token "String"
//       End   *string ` [ "…" @String ]`
//     }
package parser

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"text/scanner"
	"unicode"
)

// A node in the grammar.
type node interface {
	// Parse from scanner into value.
	Parse(lexer Lexer, parent reflect.Value) reflect.Value
}

type Parser struct {
	root node
}

// Generate a parser for the given grammar.
func Parse(grammar interface{}, aliases interface{}) (parser *Parser, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(msg.(string))
		}
	}()
	productions := map[string]node{}
	root := parseType(productions, reflect.TypeOf(grammar))
	return &Parser{root: root}, nil
}

// Parse from Lexer l into grammar v.
func (p *Parser) Parse(l Lexer, v interface{}) (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(msg.(string))
		}
	}()
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv := p.root.Parse(l, rv.Elem())
	if !pv.IsValid() {
		return errors.New("did not match")
	}
	rv.Elem().Set(reflect.Indirect(pv))
	return
}

func (p *Parser) ParseReader(r io.Reader, v interface{}) error {
	return p.Parse(Lex(r), v)
}

func (p *Parser) ParseString(s string, v interface{}) error {
	return p.Parse(LexString(s), v)
}

func (p *Parser) ParseBytes(b []byte, v interface{}) error {
	return p.Parse(LexBytes(b), v)
}

func decorate(name string) {
	// if msg := recover(); msg != nil {
	// 	panic(name + ": " + msg.(string))
	// }
}

// Takes a type and builds a tree of nodes out of it.
func parseType(productions map[string]node, t reflect.Type) node {
	defer decorate(t.Name())
	switch t.Kind() {
	case reflect.Slice:
		elem := indirectType(t.Elem())
		lexer := newStructLexer(elem)
		e := parseExpression(productions, lexer)
		if !lexer.Peek().EOF() {
			panic("unexpected input " + string(lexer.Peek().Value))
		}
		return &strct{typ: t, expr: e}
	case reflect.Struct:
		lexer := newStructLexer(t)
		e := parseExpression(productions, lexer)
		if !lexer.Peek().EOF() {
			panic("unexpected input " + string(lexer.Peek().Value))
		}
		return &strct{typ: t, expr: e}

	case reflect.Ptr:
		return parseType(productions, t.Elem())
	}
	panic("expected struct type but got " + t.String())
}

type strct struct {
	typ  reflect.Type
	expr node
}

func (s *strct) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	sv := reflect.New(s.typ).Elem()
	return s.expr.Parse(lexer, sv)
}

// <expr> {"|" <expr>}
type expression []node

func (e expression) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	for _, a := range e {
		if value := a.Parse(lexer, parent); value.IsValid() {
			return value
		}
	}
	return reflect.Value{}
}

func parseExpression(productions map[string]node, lexer *structLexer) node {
	out := expression{}
	for {
		out = append(out, parseAlternative(productions, lexer))
		if lexer.Peek().Type != '|' {
			break
		}
		lexer.Next() // |
	}
	return out
}

// <node> {<node>}
type alternative []node

func (a alternative) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	var value reflect.Value
	for i, n := range a {
		// If first value doesn't match, we early exit, otherwise all values must match.
		value = n.Parse(lexer, parent)
		if !value.IsValid() {
			if i == 0 {
				return reflect.Value{}
			}
			panic("expression did not match")
		}
	}
	return value
}

func parseAlternative(productions map[string]node, lexer *structLexer) node {
	elements := alternative{}
loop:
	for {
		switch lexer.Peek().Type {
		case EOF:
			break loop
		default:
			term := parseTerm(productions, lexer)
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

func (d dot) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	r := lexer.Next()
	if r.EOF() {
		return reflect.Value{}
	}
	return reflect.ValueOf(r)
}

// @@
type self fieldReceiver

func (s *self) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	v := s.node.Parse(lexer, parent)
	if v.IsValid() {
		setField(parent, s.field, v)
		return parent
	}
	return reflect.Value{}
}

// @<expr>
type reference fieldReceiver

func (r *reference) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	v := r.node.Parse(lexer, parent)
	if v.IsValid() {
		setField(parent, r.field, v)
		return parent
	}
	return reflect.Value{}
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func parseTerm(productions map[string]node, lexer *structLexer) node {
	r := lexer.Peek()
	switch r.Type {
	case '.':
		lexer.Next()
		return dot{}
	case '@':
		lexer.Next()
		r := lexer.Peek()
		f := lexer.Field()
		if r.Type == '@' {
			lexer.Next()
			defer decorate(f.Name)
			return &self{f, parseType(productions, indirectType(f.Type))}
		}
		if indirectType(f.Type).Kind() == reflect.Struct {
			panic("structs can only be parsed with @@")
		}
		return &reference{f, parseTerm(productions, lexer)}
	case scanner.String, scanner.RawString:
		return parseQuotedStringOrRange(lexer)
	case '[':
		return parseOptional(productions, lexer)
	case '{':
		return parseRepitition(productions, lexer)
	case '(':
		return parseGroup(productions, lexer)
	case EOF:
		return nil
	case scanner.Ident:
		return parseProductionReference(productions, lexer)
	default:
		return nil
	}
}

// A reference in the form @<identifier> refers to an existing production,
// typically from the lexer struct provided to Parse().
func parseProductionReference(productions map[string]node, lexer *structLexer) node {
	token := lexer.Next()
	if token.Type != scanner.Ident {
		panic("expected identifier")
	}
	alias, ok := productions[token.Value]
	if !ok {
		panic(fmt.Sprintf("unknown production %q", token.String()))
	}
	return alias
}

type fieldReceiver struct {
	field reflect.StructField
	node  node
}

// [ <expr> ]
type optional struct {
	node node
}

func (o optional) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	panic("not implemented")
}

func parseOptional(productions map[string]node, lexer *structLexer) node {
	lexer.Next() // [
	optional := optional{parseExpression(productions, lexer)}
	next := lexer.Peek()
	if next.Type != ']' {
		panic("expected ] but got " + next.String())
	}
	lexer.Next()
	return optional
}

// { <expr> }
type repitition struct {
	fieldReceiver
	elementType reflect.Type
}

func (r *repitition) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	for r.node.Parse(lexer, parent).IsValid() {
	}
	return parent
}

func parseRepitition(productions map[string]node, lexer *structLexer) node {
	lexer.Next() // {
	n := &repitition{
		fieldReceiver: fieldReceiver{
			field: lexer.Field(),
			node:  parseExpression(productions, lexer),
		},
		elementType: indirectType(lexer.Field().Type),
	}
	next := lexer.Next()
	if next.Type != '}' {
		panic("expected } but got " + next.String())
	}
	return n
}

func parseGroup(productions map[string]node, lexer *structLexer) node {
	lexer.Next() // (
	n := parseExpression(productions, lexer)
	next := lexer.Peek() // )
	if next.Type != ')' {
		panic("expected ) but got " + next.Value)
	}
	lexer.Next() // )
	return n
}

func parseQuotedStringOrRange(lexer *structLexer) node {
	start := parseQuotedString(lexer)
	if lexer.Peek().Type != '…' {
		return start
	}
	if len(start) != 1 {
		panic("start of range must be 1 character long")
	}
	lexer.Next() // …
	end := parseQuotedString(lexer)
	if len(end) != 1 {
		panic("end of range must be 1 character long")
	}
	return srange{start, end}
}

func parseQuotedString(lexer *structLexer) str {
	token := lexer.Next()
	if token.Type != scanner.String && token.Type != scanner.RawString {
		panic("expected quoted string but got " + token.String())
	}
	s, err := strconv.Unquote(token.Value)
	if err != nil {
		panic("invalid quoted string " + token.String())
	}
	return str(s)
}

// "a" … "b"
type srange struct {
	start str
	end   str
}

func (s srange) Parse(lexer Lexer, parent reflect.Value) reflect.Value {
	panic("not implemented")
}

// Match a string exactly "..."
type str string

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
// appended.
func setField(s reflect.Value, field reflect.StructField, fieldValue reflect.Value) {
	fieldValue = reflect.Indirect(fieldValue)
	f := s.FieldByIndex(field.Index)
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
