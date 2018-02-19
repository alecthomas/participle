package participle

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// A Parser for a particular grammar and lexer.
type Parser struct {
	root node
	lex  lexer.Definition
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
	context := newGeneratorContext(lex)
	root := context.parseType(reflect.TypeOf(grammar))
	return &Parser{root: root, lex: lex}, nil
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// participle.Build().
func (p *Parser) Parse(r io.Reader, v interface{}) (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(msg.(string))
		}
	}()
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

// Ebnf representation of the grammar.
func (p *Parser) Ebnf() string {
	return dumpEbnfNode(p.root)
}
