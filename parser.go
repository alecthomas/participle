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
	root            node
	lex             lexer.Definition
	typ             reflect.Type
	useLookahead    bool
	caseInsensitive map[string]bool
	mappers         []mapperByToken
}

// MustBuild calls Build(grammar, options...) and panics if an error occurs.
func MustBuild(grammar interface{}, options ...Option) *Parser {
	parser, err := Build(grammar, options...)
	if err != nil {
		panic(err)
	}
	return parser
}

// Build constructs a parser for the given grammar.
//
// If "Lexer()" is not provided as an option, a default lexer based on text/scanner will be used. This scans typical Go-
// like tokens.
//
// See documentation for details
func Build(grammar interface{}, options ...Option) (parser *Parser, err error) {
	gv := reflect.ValueOf(grammar)
	if gv.Kind() != reflect.Ptr || gv.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected the grammar to be a pointer to a struct but got a %T", grammar)
	}
	// Configure Parser struct with defaults + options.
	p := &Parser{
		lex:             lexer.TextScannerLexer,
		caseInsensitive: map[string]bool{},
	}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("nil Option passed, signature has changed; " +
				"if you intended to provide a custom Lexer, try participle.Build(grammar, participle.Lexer(lexer))")
		}
		if err = option(p); err != nil {
			return nil, err
		}
	}

	if len(p.mappers) > 0 {
		mappers := map[rune][]Mapper{}
		symbols := p.lex.Symbols()
		for _, mapper := range p.mappers {
			if len(mapper.symbols) == 0 {
				mappers[lexer.EOF] = append(mappers[lexer.EOF], mapper.mapper)
			} else {
				for _, symbol := range mapper.symbols {
					if rn, ok := symbols[symbol]; !ok {
						return nil, fmt.Errorf("mapper %#v uses unknown token %q", mapper, symbol)
					} else { // nolint: golint
						mappers[rn] = append(mappers[rn], mapper.mapper)
					}
				}
			}
		}
		p.lex = &mappingLexerDef{p.lex, func(t lexer.Token) (lexer.Token, error) {
			combined := make([]Mapper, 0, len(mappers[t.Type])+len(mappers[lexer.EOF]))
			combined = append(combined, mappers[lexer.EOF]...)
			combined = append(combined, mappers[t.Type]...)

			var err error
			for _, m := range combined {
				t, err = m(t)
				if err != nil {
					return t, err
				}
			}
			return t, nil
		}}
	}

	context := newGeneratorContext(p.lex)
	p.typ = reflect.TypeOf(grammar)
	p.root, err = context.parseType(p.typ)
	if err != nil {
		return nil, err
	}
	p.root.(*strct).root = true
	// TODO: Fix lookahead - see SQL example.
	if p.useLookahead {
		return p, applyLookahead(p.root, map[node]bool{})
	}
	return p, nil
}

// Lex uses the parser's lexer to tokenise input.
func (p *Parser) Lex(r io.Reader) ([]lexer.Token, error) {
	lex, err := p.lex.Lex(r)
	if err != nil {
		return nil, err
	}
	return lexer.ConsumeAll(lex)
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// participle.Build().
func (p *Parser) Parse(r io.Reader, v interface{}) (err error) {
	if reflect.TypeOf(v) != p.typ {
		return fmt.Errorf("must parse into value of type %s not %T", p.typ, v)
	}
	baseLexer, err := p.lex.Lex(r)
	if err != nil {
		return err
	}
	lex := lexer.Upgrade(baseLexer)
	caseInsensitive := map[rune]bool{}
	for sym, rn := range p.lex.Symbols() {
		if p.caseInsensitive[sym] {
			caseInsensitive[rn] = true
		}
	}
	ctx := parseContext{PeekingLexer: lex, caseInsensitive: caseInsensitive}
	// If the grammar implements Parseable, use it.
	if parseable, ok := v.(Parseable); ok {
		return p.rootParseable(lex, parseable)
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}
	pv, err := p.root.Parse(ctx, rv.Elem())
	if len(pv) > 0 && pv[0].Type() == rv.Elem().Type() {
		rv.Elem().Set(reflect.Indirect(pv[0]))
	}
	if err != nil {
		return err
	}
	token, err := lex.Peek(0)
	if err != nil {
		return err
	} else if !token.EOF() {
		return lexer.Errorf(token.Pos, "expected %s but got %q", p.root, token)
	}
	if pv == nil {
		return lexer.Errorf(token.Pos, "invalid syntax")
	}
	return nil
}

func (p *Parser) rootParseable(lex lexer.PeekingLexer, parseable Parseable) error {
	peek, err := lex.Peek(0)
	if err != nil {
		return err
	}
	err = parseable.Parse(lex)
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
