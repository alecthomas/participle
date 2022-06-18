package participle

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

type unionDef struct {
	typ     reflect.Type
	members []reflect.Type
}

type customDef struct {
	typ     reflect.Type
	parseFn reflect.Value
}

// A Parser for a particular grammar and lexer.
type Parser struct {
	root            node
	lex             lexer.Definition
	typ             reflect.Type
	useLookahead    int
	caseInsensitive map[string]bool
	mappers         []mapperByToken
	unionDefs       []unionDef
	customDefs      []customDef
	elide           []string
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
// See documentation for details.
func Build(grammar interface{}, options ...Option) (parser *Parser, err error) {
	// Configure Parser struct with defaults + options.
	p := &Parser{
		lex:             lexer.TextScannerLexer,
		caseInsensitive: map[string]bool{},
		useLookahead:    1,
	}
	for _, option := range options {
		if err = option(p); err != nil {
			return nil, err
		}
	}

	symbols := p.lex.Symbols()
	if len(p.mappers) > 0 {
		mappers := map[lexer.TokenType][]Mapper{}
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
	if err := context.addCustomDefs(p.customDefs); err != nil {
		return nil, err
	}
	if err := context.addUnionDefs(p.unionDefs); err != nil {
		return nil, err
	}

	v := reflect.ValueOf(grammar)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	p.typ = v.Type()
	p.root, err = context.parseType(p.typ)
	if err != nil {
		return nil, err
	}
	if err := validate(p.root); err != nil {
		return nil, err
	}
	return p, nil
}

// Lexer returns the parser's builtin lexer.
func (p *Parser) Lexer() lexer.Definition {
	return p.lex
}

// Lex uses the parser's lexer to tokenise input.
// Parameter filename is used as an opaque prefix in error messages.
func (p *Parser) Lex(filename string, r io.Reader) ([]lexer.Token, error) {
	lex, err := p.lex.Lex(filename, r)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.ConsumeAll(lex)
	return tokens, err
}

// ParseFromLexer into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return a Error.
func (p *Parser) ParseFromLexer(lex *lexer.PeekingLexer, v interface{}, options ...ParseOption) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	rt := rv.Type()
	if rt != p.typ {
		return fmt.Errorf("must parse into value of type %s not %T", p.typ, v)
	}
	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct, not %s", rt)
	}
	caseInsensitive := map[lexer.TokenType]bool{}
	for sym, tt := range p.lex.Symbols() {
		if p.caseInsensitive[sym] {
			caseInsensitive[tt] = true
		}
	}
	ctx := newParseContext(lex, p.useLookahead, caseInsensitive)
	defer func() { *lex = *ctx.PeekingLexer }()
	for _, option := range options {
		option(ctx)
	}
	// If the grammar implements Parseable, use it.
	if parseable, ok := v.(Parseable); ok {
		return p.rootParseable(ctx, parseable)
	}
	return p.parseOne(ctx, rv)
}

func (p *Parser) parse(lex lexer.Lexer, v interface{}, options ...ParseOption) (err error) {
	peeker, err := lexer.Upgrade(lex, p.getElidedTypes()...)
	if err != nil {
		return err
	}
	return p.ParseFromLexer(peeker, v, options...)
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser) Parse(filename string, r io.Reader, v interface{}, options ...ParseOption) (err error) {
	if filename == "" {
		filename = lexer.NameOfReader(r)
	}
	lex, err := p.lex.Lex(filename, r)
	if err != nil {
		return err
	}
	return p.parse(lex, v, options...)
}

// ParseString from s into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser) ParseString(filename string, s string, v interface{}, options ...ParseOption) (err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.StringDefinition); ok {
		lex, err = sl.LexString(filename, s)
	} else {
		lex, err = p.lex.Lex(filename, strings.NewReader(s))
	}
	if err != nil {
		return err
	}
	return p.parse(lex, v, options...)
}

// ParseBytes from b into grammar v which must be of the same type as the grammar passed to
// Build(). Parameter filename is used as an opaque prefix in error messages.
//
// This may return an Error.
func (p *Parser) ParseBytes(filename string, b []byte, v interface{}, options ...ParseOption) (err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.BytesDefinition); ok {
		lex, err = sl.LexBytes(filename, b)
	} else {
		lex, err = p.lex.Lex(filename, bytes.NewReader(b))
	}
	if err != nil {
		return err
	}
	return p.parse(lex, v, options...)
}

func (p *Parser) parseOne(ctx *parseContext, rv reflect.Value) error {
	err := p.parseInto(ctx, rv)
	if err != nil {
		return err
	}
	token := ctx.Peek()
	if !token.EOF() && !ctx.allowTrailing {
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: token})
	}
	return nil
}

func (p *Parser) parseInto(ctx *parseContext, rv reflect.Value) error {
	if rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to a struct, but is a nil %s", rv.Type())
	}
	pv, err := p.root.Parse(ctx, rv.Elem())
	if len(pv) > 0 && pv[0].Type() == rv.Elem().Type() {
		rv.Elem().Set(reflect.Indirect(pv[0]))
	}
	if err != nil {
		return err
	}
	if pv == nil {
		token := ctx.Peek()
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: token})
	}
	return nil
}

func (p *Parser) rootParseable(ctx *parseContext, parseable Parseable) error {
	if err := parseable.Parse(ctx.PeekingLexer); err != nil {
		if err == NextMatch {
			err = UnexpectedTokenError{Unexpected: ctx.Peek()}
		} else {
			err = &ParseError{Msg: err.Error(), Pos: ctx.Peek().Pos}
		}
		return ctx.DeepestError(err)
	}
	peek := ctx.Peek()
	if !peek.EOF() && !ctx.allowTrailing {
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: peek})
	}
	return nil
}

func (p *Parser) getElidedTypes() []lexer.TokenType {
	symbols := p.lex.Symbols()
	elideTypes := make([]lexer.TokenType, 0, len(p.elide))
	for _, elide := range p.elide {
		rn, ok := symbols[elide]
		if !ok {
			panic(fmt.Errorf("Elide() uses unknown token %q", elide))
		}
		elideTypes = append(elideTypes, rn)
	}
	return elideTypes
}
