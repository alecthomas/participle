package participle

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// A Parser for a particular grammar and lexer.
type Parser struct {
	root            node
	trace           io.Writer
	lex             lexer.Definition
	typ             reflect.Type
	useLookahead    int
	caseInsensitive map[string]bool
	mappers         []mapperByToken
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
// See documentation for details
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
		mappers := map[rune][]Mapper{}
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
	v := reflect.ValueOf(grammar)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	p.typ = v.Type()
	p.root, err = context.parseType(p.typ)
	if err != nil {
		return nil, err
	}
	if p.trace != nil {
		p.root = injectTrace(p.trace, 0, p.root)
	}
	return p, nil
}

// Lexer returns the parser's builtin lexer.
func (p *Parser) Lexer() lexer.Definition {
	return p.lex
}

// Lex uses the parser's lexer to tokenise input.
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
	var stream reflect.Value
	if rv.Kind() == reflect.Chan {
		stream = rv
		rt := rv.Type().Elem()
		rv = reflect.New(rt).Elem()
	}
	rt := rv.Type()
	if rt != p.typ {
		return fmt.Errorf("must parse into value of type %s not %T", p.typ, v)
	}
	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct, not %s", rt)
	}
	caseInsensitive := map[rune]bool{}
	for sym, rn := range p.lex.Symbols() {
		if p.caseInsensitive[sym] {
			caseInsensitive[rn] = true
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
	if stream.IsValid() {
		return p.parseStreaming(ctx, stream)
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
// Build().
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
// Build().
//
// This may return an Error.
func (p *Parser) ParseString(filename string, s string, v interface{}, options ...ParseOption) (err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.StringDefinition); ok {
		lex, err = sl.LexString(filename, s)
	} else {
		lex, err = p.lex.Lex(filename, strings.NewReader(s))
	}
	return p.parse(lex, v, options...)
}

// ParseBytes from b into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return an Error.
func (p *Parser) ParseBytes(filename string, b []byte, v interface{}, options ...ParseOption) (err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.BytesDefinition); ok {
		lex, err = sl.LexBytes(filename, b)
	} else {
		lex, err = p.lex.Lex(filename, bytes.NewReader(b))
	}
	return p.parse(lex, v, options...)
}

func (p *Parser) parseStreaming(ctx *parseContext, rv reflect.Value) error {
	t := rv.Type().Elem().Elem()
	for {
		if token, _ := ctx.Peek(0); token.EOF() {
			rv.Close()
			return nil
		}
		v := reflect.New(t)
		if err := p.parseInto(ctx, v); err != nil {
			return err
		}
		rv.Send(v)
	}
}

func (p *Parser) parseOne(ctx *parseContext, rv reflect.Value) error {
	err := p.parseInto(ctx, rv)
	if err != nil {
		return err
	}
	token, err := ctx.Peek(0)
	if err != nil {
		return err
	} else if !token.EOF() && !ctx.allowTrailing {
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
		token, _ := ctx.Peek(0)
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: token})
	}
	return nil
}

func (p *Parser) rootParseable(ctx *parseContext, parseable Parseable) error {
	peek, err := ctx.Peek(0)
	if err != nil {
		return err
	}
	err = parseable.Parse(ctx.PeekingLexer)
	if err == NextMatch {
		token, _ := ctx.Peek(0)
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: token})
	}
	peek, err = ctx.Peek(0)
	if err != nil {
		return err
	}
	if !peek.EOF() && !ctx.allowTrailing {
		return ctx.DeepestError(UnexpectedTokenError{Unexpected: peek})
	}
	return nil
}

func (p *Parser) getElidedTypes() []rune {
	symbols := p.lex.Symbols()
	elideTypes := make([]rune, 0, len(p.elide))
	for _, elide := range p.elide {
		rn, ok := symbols[elide]
		if !ok {
			panic(fmt.Errorf("Elide() uses unknown token %q", elide))
		}
		elideTypes = append(elideTypes, rn)
	}
	return elideTypes
}
