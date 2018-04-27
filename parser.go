package participle

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// A Parser for a particular grammar and lexer.
type Parser struct {
	root node
	lex  lexer.Definition
	typ  reflect.Type
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
	defer recoverToError(&err)
	if lex == nil {
		lex = lexer.TextScannerLexer
	}
	context := newGeneratorContext(lex)
	typ := reflect.TypeOf(grammar)
	root := context.parseType(typ)
	applyLookahead(root, map[node]bool{})
	return &Parser{root: root, lex: lex, typ: typ}, nil
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// participle.Build().
func (p *Parser) Parse(r io.Reader, v interface{}) (err error) {
	if reflect.TypeOf(v) != p.typ {
		return fmt.Errorf("must parse into value of type %s not %T", p.typ, v)
	}
	defer recoverToError(&err)
	lex := p.lex.Lex(r)
	// If the grammar implements Parseable, use it.
	if parseable, ok := v.(Parseable); ok {
		return p.rootParseable(lex, parseable)
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv := p.root.Parse(lex, rv.Elem())
	if !lex.Peek(0).EOF() {
		lexer.Panicf(lex.Peek(0).Pos, "expected %s but got %q", stringer(p.root, 1), lex.Peek(0))
	}
	if pv == nil {
		lexer.Panic(lex.Peek(0).Pos, "invalid syntax")
	}
	rv.Elem().Set(reflect.Indirect(pv[0]))
	return
}

func (p *Parser) rootParseable(lex lexer.Lexer, parseable Parseable) error {
	err := parseable.Parse(lex)
	peek := lex.Peek(0)
	if err == NextMatch {
		return lexer.Errorf(peek.Pos, "invalid syntax")
	}
	if err == nil && !peek.EOF() {
		return lexer.Errorf(peek.Pos, "unexpected token %q", peek)
	}
	return err
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
