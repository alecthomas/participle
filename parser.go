package participle

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Seperate struct so Option doesn't need to be generic.
type options struct {
	root            node
	trace           io.Writer
	lex             lexer.Definition
	typ             reflect.Type
	useLookahead    int
	caseInsensitive map[string]bool
	mappers         []mapperByToken
	elide           []string
}

// A Parser for a particular grammar and lexer.
type Parser[T any] struct{ options }

// MustBuild calls Build(grammar, options...) and panics if an error occurs.
func MustBuild[T any](options ...Option) *Parser[T] {
	parser, err := Build[T](options...)
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
func Build[T any](opts ...Option) (parser *Parser[T], err error) {
	// Configure Parser struct with defaults + options.
	p := &Parser[T]{options{
		lex:             lexer.TextScannerLexer,
		caseInsensitive: map[string]bool{},
		useLookahead:    1,
	}}
	for _, option := range opts {
		if err = option(&p.options); err != nil {
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
	var zero T
	v := reflect.ValueOf(&zero)
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
	if p.trace != nil {
		p.root = injectTrace(p.trace, 0, p.root)
	}
	return p, nil
}

// Lexer returns the parser's builtin lexer.
func (p *Parser[T]) Lexer() lexer.Definition {
	return p.lex
}

// Lex uses the parser's lexer to tokenise input.
func (p *Parser[T]) Lex(filename string, r io.Reader) ([]lexer.Token, error) {
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
func (p *Parser[T]) ParseFromLexer(lex *lexer.PeekingLexer, options ...ParseOption) (*T, error) {
	v := new(T)
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
		return nil, fmt.Errorf("must parse into value of type %s not %T", p.typ, v)
	}
	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("target must be a pointer to a struct, not %s", rt)
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
	var vi interface{} = v
	// If the grammar implements Parseable, use it.
	if parseable, ok := vi.(Parseable); ok {
		return v, p.rootParseable(ctx, parseable)
	}
	if stream.IsValid() {
		return v, p.parseStreaming(ctx, stream)
	}
	return v, p.parseOne(ctx, rv)
}

func (p *Parser[T]) parse(lex lexer.Lexer, options ...ParseOption) (v *T, err error) {
	peeker, err := lexer.Upgrade(lex, p.getElidedTypes()...)
	if err != nil {
		return nil, err
	}
	return p.ParseFromLexer(peeker, options...)
}

// Parse from r into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return an Error.
func (p *Parser[T]) Parse(filename string, r io.Reader, options ...ParseOption) (v *T, err error) {
	if filename == "" {
		filename = lexer.NameOfReader(r)
	}
	lex, err := p.lex.Lex(filename, r)
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

// ParseString from s into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return an Error.
func (p *Parser[T]) ParseString(filename string, s string, options ...ParseOption) (v *T, err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.StringDefinition); ok {
		lex, err = sl.LexString(filename, s)
	} else {
		lex, err = p.lex.Lex(filename, strings.NewReader(s))
	}
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

// ParseBytes from b into grammar v which must be of the same type as the grammar passed to
// Build().
//
// This may return an Error.
func (p *Parser[T]) ParseBytes(filename string, b []byte, options ...ParseOption) (v *T, err error) {
	var lex lexer.Lexer
	if sl, ok := p.lex.(lexer.BytesDefinition); ok {
		lex, err = sl.LexBytes(filename, b)
	} else {
		lex, err = p.lex.Lex(filename, bytes.NewReader(b))
	}
	if err != nil {
		return nil, err
	}
	return p.parse(lex, options...)
}

func (p *Parser[T]) parseStreaming(ctx *parseContext, rv reflect.Value) error {
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

func (p *Parser[T]) parseOne(ctx *parseContext, rv reflect.Value) error {
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

func (p *Parser[T]) parseInto(ctx *parseContext, rv reflect.Value) error {
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

func (p *Parser[T]) rootParseable(ctx *parseContext, parseable Parseable) error {
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

func (p *Parser[T]) getElidedTypes() []lexer.TokenType {
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
