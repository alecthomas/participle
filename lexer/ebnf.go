package lexer

import (
	"bufio"
	"io"
	"reflect"
	"strings"
	"text/scanner"
	"unicode/utf8"

	"golang.org/x/exp/ebnf"
)

type ebnfLexer struct {
	r      *bufio.Reader
	def    *ebnfLexerDefinition
	pos    Position
	peeked *Token
}

func (e *ebnfLexer) Peek() Token {
	if e.peeked != nil {
		return *e.peeked
	}
	token := e.readToken()
	e.peeked = &token
	return token
}

func (e *ebnfLexer) Next() Token {
	t := e.Peek()
	e.peeked = nil
	return t
}

func (e *ebnfLexer) readToken() Token {
	if e.peek() == EOF {
		return EOFToken
	}
	pos := e.pos
	for name, production := range e.def.productions {
		if match := e.match(production.Expr); match != nil {
			return Token{
				Type:  e.def.symbols[name],
				Pos:   pos,
				Value: strings.Join(match, ""),
			}
		}
	}
	e.panic("no match found")
	return Token{}
}

func (e *ebnfLexer) match(expr ebnf.Expression) []string { // nolint: gocyclo
	switch n := expr.(type) {
	case ebnf.Alternative:
		for _, an := range n {
			if match := e.match(an); match != nil {
				return match
			}
		}
		return nil

	case *ebnf.Group:
		return e.match(n.Body)

	case *ebnf.Name:
		return e.match(e.def.grammar[n.String].Expr)

	case *ebnf.Option:
		match := e.match(n.Body)
		if match == nil {
			match = []string{}
		}
		return match

	case *ebnf.Range:
		// TODO: Ideally this would be cached somewhere.
		// Doing an optimisation pass would probably be smart.
		start, _ := utf8.DecodeRuneInString(n.Begin.String)
		end, _ := utf8.DecodeRuneInString(n.End.String)
		rn := e.peek()
		if rn < start || rn > end {
			return nil
		}
		e.read()
		return []string{string(rn)}

	case *ebnf.Repetition:
		var out []string
		for {
			match := e.match(n.Body)
			if match != nil {
				out = append(out, match...)
			} else {
				break
			}
		}
		return out

	case ebnf.Sequence:
		var out []string
		for i, sn := range n {
			match := e.match(sn)
			if match != nil {
				out = append(out, match...)
				continue
			}
			if i > 0 {
				e.panicf("expected %q", sn)
			} else {
				break
			}
		}
		return out

	case *ebnf.Token:
		// If first rune doesn't match, we didn't match.
		if rn, _ := utf8.DecodeRuneInString(n.String); rn != e.peek() {
			return nil
		}
		for _, rn := range n.String {
			if rn != e.read() {
				e.panicf("expected %q", n.String)
			}
		}
		return []string{n.String}

	case *characterSet:
		if strings.ContainsRune(n.Set, e.peek()) {
			return []string{string(e.read())}
		}
		return nil

	case nil:
		if e.peek() == EOF {
			return nil
		}
		e.panic("expected EOF")
	}
	panic("unsupported lexer expression type " + reflect.TypeOf(expr).String())
}

func (e *ebnfLexer) peek() rune {
	// This is a bit more involved than I would like.
	rn, _, err := e.r.ReadRune()
	if err == io.EOF {
		return EOF
	}
	if err != nil {
		e.panicf("failed to read rune: %s", err)
	}
	if err = e.r.UnreadRune(); err != nil {
		e.panicf("failed to unread rune: %s", err)
	}
	return rn
}

func (e *ebnfLexer) read() rune {
	rn, n, err := e.r.ReadRune()
	if err == io.EOF {
		return EOF
	}
	if err != nil {
		e.panicf("failed to read rune: %s", err)
	}
	e.pos.Offset += n
	if rn == '\n' {
		e.pos.Line++
		e.pos.Column = 1
	} else {
		e.pos.Column++
	}
	return rn
}

func (e *ebnfLexer) panic(msg string) {
	Panic(e.pos, msg)
}

func (e *ebnfLexer) panicf(msg string, args ...interface{}) {
	Panicf(e.pos, msg, args...)
}

type ebnfLexerDefinition struct {
	grammar     ebnf.Grammar
	symbols     map[string]rune
	productions ebnf.Grammar
}

// EBNF creates a Lexer from an EBNF grammar.
//
// The EBNF grammar syntax is as defined by "golang.org/x/exp/ebnf". Upper-case productions are
// exported as symbols. All productions are lexical.
//
// Here's an example grammar for parsing whitespace and identifiers:
//
// 		Identifier = alpha { alpha | number } .
//		Whitespace = "\n" | "\r" | "\t" | " " .
//		alpha = "a"…"z" | "A"…"Z" | "_" .
//		number = "0"…"9" .
func EBNF(grammar string) (Definition, error) {
	// Parse grammar.
	r := strings.NewReader(grammar)
	ast, err := ebnf.Parse("<grammar>", r)
	if err != nil {
		return nil, err
	}

	// Validate grammar.
	for _, production := range ast {
		if err = validate(ast, production); err != nil {
			return nil, err
		}
	}
	// Assign constants for roots.
	rn := EOF - 1
	symbols := map[string]rune{
		"EOF": EOF,
	}

	// Optimize and export public productions.
	productions := ebnf.Grammar{}
	for symbol, production := range ast {
		ch := symbol[0:1]
		production.Expr = optimize(production.Expr)
		if strings.ToUpper(ch) == ch {
			symbols[symbol] = rn
			productions[symbol] = production
			rn--
		}
	}
	def := &ebnfLexerDefinition{
		grammar:     ast,
		symbols:     symbols,
		productions: productions,
	}
	return def, nil
}

func (e *ebnfLexerDefinition) Lex(r io.Reader) Lexer {
	return &ebnfLexer{
		r:   bufio.NewReader(r),
		def: e,
		pos: Position{
			Filename: NameOfReader(r),
			Line:     1,
			Column:   1,
		},
	}
}

func (e *ebnfLexerDefinition) Symbols() map[string]rune {
	return e.symbols
}

type characterSet struct {
	pos scanner.Position
	Set string
}

func (c *characterSet) Pos() scanner.Position {
	return c.pos
}

// Apply some optimizations to the EBNF.
//
// Convert alternate characters into a character set (eg. "a" | "b" | "c" | "true" becomes
// set("abc") | "true").
func optimize(expr ebnf.Expression) ebnf.Expression {
	switch n := expr.(type) {
	case ebnf.Alternative:
		out := make(ebnf.Alternative, 0, len(n))
		set := ""
		for _, e := range n {
			if t, ok := e.(*ebnf.Token); ok && utf8.RuneCountInString(t.String) == 1 {
				set += t.String
			} else {
				out = append(out, optimize(e))
			}
		}
		if set != "" {
			out = append(out, &characterSet{Set: set})
		}
		return out

	case ebnf.Sequence:
		for i, e := range n {
			n[i] = optimize(e)
		}

	case *ebnf.Group:
		n.Body = optimize(n.Body)

	case *ebnf.Option:
		n.Body = optimize(n.Body)

	case *ebnf.Repetition:
		n.Body = optimize(n.Body)
	}
	return expr
}

// Validate the grammar against the lexer rules.
func validate(grammar ebnf.Grammar, expr ebnf.Expression) error { // nolint: gocyclo
	switch n := expr.(type) {
	case *ebnf.Production:
		return validate(grammar, n.Expr)

	case ebnf.Alternative:
		for _, e := range n {
			if err := validate(grammar, e); err != nil {
				return err
			}
		}
		return nil

	case *ebnf.Group:
		return validate(grammar, n.Body)

	case *ebnf.Name:
		if grammar[n.String] == nil {
			return Errorf(Position(n.Pos()), "unknown production %q", n.String)
		}
		return nil

	case *ebnf.Option:
		return validate(grammar, n.Body)

	case *ebnf.Range:
		if utf8.RuneCountInString(n.Begin.String) != 1 {
			return Errorf(Position(n.Pos()), "start of range must be a single rune")
		}
		if utf8.RuneCountInString(n.End.String) != 1 {
			return Errorf(Position(n.Pos()), "end of range must be a single rune")
		}
		return nil

	case *ebnf.Repetition:
		return validate(grammar, n.Body)

	case ebnf.Sequence:
		for _, e := range n {
			if err := validate(grammar, e); err != nil {
				return err
			}
		}
		return nil

	case *ebnf.Token:
		return nil

	case nil:
		return nil
	}
	return Errorf(Position(expr.Pos()), "unknown EBNF expression "+reflect.TypeOf(expr).String())
}
