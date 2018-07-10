package lexer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/scanner"
	"unicode/utf8"

	"golang.org/x/exp/ebnf"
)

type ebnfRange struct {
	start, end rune
}

type ebnfLexer struct {
	r      *bufio.Reader
	def    *ebnfLexerDefinition
	pos    Position
	ranges map[*ebnf.Range]ebnfRange
	buf    *bytes.Buffer
}

func (e *ebnfLexer) Next() Token {
	if e.peek() == EOF {
		return EOFToken(e.pos)
	}
	pos := e.pos
	e.buf.Reset()
	for name, production := range e.def.productions {
		if e.match(name, production.Expr, e.buf) {
			return Token{
				Type:  e.def.symbols[name],
				Pos:   pos,
				Value: e.buf.String(),
			}
		}
	}
	e.panic("no match found")
	return Token{}
}

func (e *ebnfLexer) match(name string, expr ebnf.Expression, out *bytes.Buffer) bool { // nolint: gocyclo
	panicf := func(format string, args ...interface{}) { e.panicf(name+": "+format, args...) }

	switch n := expr.(type) {
	case ebnf.Alternative:
		for _, an := range n {
			if e.match(name, an, out) {
				return true
			}
		}
		return false

	case *ebnf.Group:
		return e.match(name, n.Body, out)

	case *ebnf.Name:
		return e.match(name, e.def.grammar[n.String].Expr, out)

	case *ebnf.Option:
		e.match(name, n.Body, out)
		return true

	case *ebnf.Range:
		erange := e.ranges[n]
		rn := e.peek()
		if rn < erange.start || rn > erange.end {
			return false
		}
		e.read()
		out.WriteRune(rn)
		return true

	case *ebnf.Repetition:
		for e.match(name, n.Body, out) {
		}
		return true

	case ebnf.Sequence:
		for i, sn := range n {
			if e.match(name, sn, out) {
				continue
			}
			if i > 0 {
				panicf("unexpected input %q", e.peek())
			} else {
				return false
			}
		}
		return true

	case *ebnf.Token:
		// If first rune doesn't match, we didn't match.
		if rn, _ := utf8.DecodeRuneInString(n.String); rn != e.peek() {
			return false
		}
		for _, rn := range n.String {
			if rn != e.read() {
				panicf("unexpected input %q, expected %q", e.peek(), n.String)
			}
		}
		out.WriteString(n.String)
		return true

	case *characterSet:
		if strings.ContainsRune(n.Set, e.peek()) {
			out.WriteRune(e.read())
			return true
		}
		return false

	case nil:
		if e.peek() == EOF {
			return false
		}
		panicf("expected EOF")
	}
	panic(fmt.Sprintf("unsupported lexer expression type %T", expr))
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
	ranges      map[*ebnf.Range]ebnfRange
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
		ranges:      map[*ebnf.Range]ebnfRange{},
	}
	for _, production := range ast {
		production.Expr = def.optimize(production.Expr)
	}
	return def, nil
}

func (e *ebnfLexerDefinition) Lex(r io.Reader) Lexer {
	return &ebnfLexer{
		r:      bufio.NewReader(r),
		def:    e,
		ranges: e.ranges,
		buf:    bytes.NewBuffer(make([]byte, 0, 128)),
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

// TODO: Add a "repeatedCharacterSet" to represent the common case of { set }

// Apply some optimizations to the EBNF.
func (e *ebnfLexerDefinition) optimize(expr ebnf.Expression) ebnf.Expression {
	switch n := expr.(type) {
	case ebnf.Alternative:
		// Convert alternate characters into a character set (eg. "a" | "b" | "c" | "true" becomes
		// set("abc") | "true").
		out := make(ebnf.Alternative, 0, len(n))
		set := ""
		for _, expr := range n {
			if t, ok := expr.(*ebnf.Token); ok && utf8.RuneCountInString(t.String) == 1 {
				set += t.String
			} else {
				if set != "" {
					out = append(out, &characterSet{Set: set})
					set = ""
				}
				out = append(out, e.optimize(expr))
			}
		}
		if set != "" {
			out = append(out, &characterSet{Set: set})
		}
		return out

	case ebnf.Sequence:
		for i, expr := range n {
			n[i] = e.optimize(expr)
		}

	case *ebnf.Group:
		n.Body = e.optimize(n.Body)

	case *ebnf.Option:
		n.Body = e.optimize(n.Body)

	case *ebnf.Repetition:
		n.Body = e.optimize(n.Body)

	case *ebnf.Range:
		start, _ := utf8.DecodeRuneInString(n.Begin.String)
		end, _ := utf8.DecodeRuneInString(n.End.String)
		e.ranges[n] = ebnfRange{start: start, end: end}
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
	return Errorf(Position(expr.Pos()), "unknown EBNF expression %T", expr)
}
