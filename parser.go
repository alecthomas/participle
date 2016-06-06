// Package parser constructs parsers from definitions in struct tags and parses directly into those
// structs. The approach is philosophically similar to how other marshallers work in Go,
// "unmarshalling" an instance of a grammar into a struct.
//
// Here's an example of an EBNF parser. First, we define some convenience lexer tokens:
//
//     type Lexer struct {
//       Identifier string      `("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`
//       String     string      `"\"" {"\\" . | .} "\""`
//       Whitespace lexer.Skip  `" " | "\t" | "\n" | "\r"`
//     }
//
// Then the grammar itself:
//
//     type EBNF struct {
//       Productions []*Production
//     }
//
//     type Production struct {
//       Name       string      `@Identifier "="`
//       Expression *Expression `[ @@ ] "."`
//     }
//
//     type Expression struct {
//       Alternatives []*Term `@@ { "|" @@ }`
//     }
//
//     type Term struct {
//       Name       *string       `@Identifier |`
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
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"unicode"
)

// A node in the grammar.
type node interface {
	// Parse from scanner into value.
	Parse(scan Scanner) reflect.Value
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

// Parse from Scanner s into grammar v.
func (p *Parser) Parse(s Scanner, v interface{}) (err error) {
	// defer func() {
	// 	if msg := recover(); msg != nil {
	// 		err = errors.New(msg.(string))
	// 	}
	// }()
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv := p.root.Parse(s)
	if !pv.IsValid() {
		return errors.New("invalid syntax")
	}
	rv.Elem().Set(reflect.Indirect(pv))
	return
}

func (p *Parser) ParseReader(r io.Reader, v interface{}) error {
	return p.Parse(ReaderScanner(r), v)
}

func (p *Parser) ParseString(s string, v interface{}) error {
	return p.Parse(StringScanner(s), v)
}

func (p *Parser) ParseBytes(b []byte, v interface{}) error {
	return p.Parse(ByteScanner(b), v)
}

func decorate(name string) {
	if msg := recover(); msg != nil {
		panic(name + ": " + msg.(string))
	}
}

// Takes a type and builds a tree of nodes out of it.
func parseType(productions map[string]node, t reflect.Type) *strct {
	defer decorate(t.Name())
	switch t.Kind() {
	case reflect.Struct:
		scan := newStructScanner(t)
		e := parseExpression(productions, scan)
		if peekNextNonSpace(scan) != EOF {
			panic("unexpected input: " + string(scan.Peek()))
		}
		return &strct{typ: t, expr: e}

	case reflect.Ptr:
		return parseType(productions, t.Elem())
	}
	panic("expected struct type but got " + t.String())
}

type strct struct {
	typ  reflect.Type
	expr expression
}

func (s *strct) Parse(scan Scanner) reflect.Value {
	return s.expr.Parse(scan)
}

// <expr> {"|" <expr>}
type expression []alternative

func (e expression) Parse(scan Scanner) reflect.Value {
	for _, a := range e {
		if value := a.Parse(scan); value.IsValid() {
			return value
		}
	}
	return reflect.Value{}
}

func parseExpression(productions map[string]node, scan *structScanner) expression {
	out := expression{}
	for {
		out = append(out, parseAlternative(productions, scan))
		if peekNextNonSpace(scan) != '|' {
			break
		}
		scan.Next()
	}
	return out
}

// <node> {<node>}
type alternative []node

func (a alternative) Parse(scan Scanner) reflect.Value {
	var value reflect.Value
	for i, n := range a {
		// If first value doesn't match, we early exit, otherwise all values must match.
		value = n.Parse(scan)
		if !value.IsValid() {
			if i == 0 {
				return reflect.Value{}
			}
			panic("expression did not match")
		}
	}
	return value
}

func parseAlternative(productions map[string]node, scan *structScanner) alternative {
	elements := []node{}
loop:
	for {
		switch scan.Peek() {
		case EOF:
			break loop
		case ' ', '\t':
			scan.Next()
			continue loop
		default:
			term := parseTerm(productions, scan)
			if term == nil {
				break loop
			}
			elements = append(elements, term)
		}
	}
	return elements
}

type dot struct{}

func (d dot) Parse(scan Scanner) reflect.Value {
	r := scan.Next()
	if r == EOF {
		return reflect.Value{}
	}
	return reflect.ValueOf(r)
}

// @@
type self struct {
	field reflect.StructField
	strct node
}

func (s *self) Parse(scan Scanner) reflect.Value {
	return s.strct.Parse(scan)
}

// @<expr>
type reference struct {
	field reflect.StructField
	expr  node
}

func (r *reference) Parse(scan Scanner) reflect.Value {
	return r.expr.Parse(scan)
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func parseTerm(productions map[string]node, scan *structScanner) node {
	r := scan.Peek()
	switch r {
	case '.':
		scan.Next()
		return dot{}
	case '@':
		scan.Next()
		r := scan.Peek()
		f := scan.Field()
		if r == '@' {
			scan.Next()
			defer decorate(f.Name)
			return &self{f, parseType(productions, f.Type)}
		}
		return &reference{f, parseTerm(productions, scan)}
	case '"':
		return parseQuotedStringOrRange(scan)
	case '[':
		return parseOptional(productions, scan)
	case '{':
		return parseRepitition(productions, scan)
	case '(':
		return parseGroup(productions, scan)
	case EOF:
		return nil
	default:
		if isIdentifierStart(r) {
			return parseProductionReference(productions, scan)
		}
		return nil
	}
}

// A reference in the form @<identifier> refers to an existing production,
// typically from the lexer struct provided to Parse().
func parseProductionReference(productions map[string]node, scan *structScanner) node {
	out := bytes.Buffer{}
	out.WriteRune(scan.Next())
	for {
		r := scan.Peek()
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out.WriteRune(scan.Next())
		} else {
			break
		}
	}
	alias, ok := productions[out.String()]
	if !ok {
		panic(fmt.Sprintf("unknown production %q", out.String()))
	}
	return alias
}

type fieldReceiver struct {
	field reflect.StructField
	node  node
}

type optional expression

func (o optional) Parse(scan Scanner) reflect.Value {
	panic("not implemented")
}

func parseOptional(productions map[string]node, scan *structScanner) optional {
	scan.Next() // [
	optional := optional(parseExpression(productions, scan))
	next := scan.Next()
	if next != ']' {
		panic("expected ] but got " + string(next))
	}
	return optional
}

// { <expr> }
type repitition struct {
	fieldReceiver
	elementType reflect.Type
}

func (r *repitition) Parse(scan Scanner) reflect.Value {
	slice := reflect.New(reflect.SliceOf(r.elementType))
	for {
		if element := r.node.Parse(scan); element.IsValid() {
			slice = reflect.AppendSlice(slice, element)
		} else {
			break
		}
	}
	return slice
}

func parseRepitition(productions map[string]node, scan *structScanner) *repitition {
	scan.Next() // {
	n := &repitition{
		fieldReceiver: fieldReceiver{
			field: scan.Field(),
			node:  parseExpression(productions, scan),
		},
		elementType: scan.Field().Type.Elem(),
	}
	next := scan.Next()
	if next != '}' {
		panic("expected } but got " + string(next))
	}
	return n
}

type group expression

func (g group) Parse(scan Scanner) reflect.Value {
	return ((expression)(g)).Parse(scan)
}

func parseGroup(productions map[string]node, scan *structScanner) group {
	scan.Next() // (
	n := group(parseExpression(productions, scan))
	next := scan.Next()
	if next != ')' {
		panic("expected ) but got " + string(next))
	}
	return n
}

func parseQuotedStringOrRange(scan *structScanner) node {
	n := parseQuotedString(scan)
	if peekNextNonSpace(scan) != '…' {
		return n
	}
	scan.Next()
	if peekNextNonSpace(scan) != '"' {
		panic("expected ending quoted string for range but got " + string(scan.Peek()))
	}
	return srange{n, parseQuotedString(scan)}
}

// "a" … "b"
type srange struct {
	start str
	end   str
}

func (s srange) Parse(scan Scanner) reflect.Value {
	panic("not implemented")
}

// Match a string exactly "..."
type str string

func (s str) Parse(scan Scanner) reflect.Value {
	out := ""
	for i, r := range s {
		if scan.Peek() != r {
			if i == 0 {
				return reflect.Value{}
			}
			panic("expected '" + s + "'")
		}
		out += string(scan.Next())
	}
	return reflect.ValueOf(out)
}

func parseQuotedString(scan Scanner) str {
	scan.Next() // "
	out := bytes.Buffer{}
loop:
	for {
		switch scan.Peek() {
		case '\\':
			// TODO: Support octal escape codes
			scan.Next()
			out.WriteRune(scan.Next())
		case '"':
			scan.Next()
			break loop
		case EOF:
			panic("unexpected EOF")
		default:
			out.WriteRune(scan.Next())
		}
	}
	return str(out.String())
}

func peekNextNonSpace(scan Scanner) rune {
	for unicode.IsSpace(scan.Peek()) {
		scan.Next()
	}
	return scan.Peek()
}
