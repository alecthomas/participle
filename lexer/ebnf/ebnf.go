// Package ebnf is an EBNF lexer for Participle.
//
// The EBNF grammar syntax is as defined by "golang.org/x/exp/ebnf" with one extension:
// ranges also support exclusions, eg. "a"…"z"-"f" and "a"…"z"-"f"…"g".
package ebnf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf/internal"
)

// New creates a Lexer from an EBNF grammar.
//
// The EBNF grammar syntax is as defined by "golang.org/x/exp/ebnf" with one extension:
// ranges also support exclusions, eg. "a"…"z"-"f" and "a"…"z"-"f"…"g".
// Exclusions can be chained.
//
// Upper-case productions are exported as terminals. Lower-case productions are non-terminals.
// All productions are lexical.
//
// Here's an example grammar for parsing whitespace and identifiers:
//
// 		Identifier = alpha { alpha | number } .
//		Whitespace = "\n" | "\r" | "\t" | " " .
//		alpha = "a"…"z" | "A"…"Z" | "_" .
//		number = "0"…"9" .
func New(grammar string, options ...Option) (lexer.Definition, error) {
	// Parse grammar.
	r := strings.NewReader(grammar)
	ast, err := internal.Parse("<grammar>", r)
	if err != nil {
		return nil, err
	}

	// Validate grammar.
	for _, production := range ast.Index {
		if err = validate(ast, production); err != nil {
			return nil, err
		}
	}
	// Assign constants for roots.
	rn := lexer.EOF - 1
	symbols := map[string]rune{
		"EOF": lexer.EOF,
	}

	// Optimize and export public productions.
	productions := internal.Grammar{Index: map[string]*internal.Production{}}
	for _, namedProduction := range ast.Productions {
		symbol := namedProduction.Name
		production := namedProduction.Production
		ch := symbol[0:1]
		if strings.ToUpper(ch) == ch {
			symbols[symbol] = rn
			productions.Index[symbol] = production
			productions.Productions = append(productions.Productions, namedProduction)
			rn--
		}
	}
	def := &ebnfLexerDefinition{
		grammar:     ast,
		symbols:     symbols,
		productions: productions,
		elide:       map[string]bool{},
	}
	for _, production := range ast.Index {
		production.Expr = def.optimize(production.Expr)
	}
	for _, option := range options {
		option(def)
	}
	return def, nil
}

type ebnfLexer struct {
	r   *tokenReader
	def *ebnfLexerDefinition
	buf *bytes.Buffer
}

func (e *ebnfLexer) Next() (lexer.Token, error) {
nextToken:
	for {
		e.r.Begin()
		rn, err := e.peek()
		if err != nil {
			return lexer.Token{}, err
		} else if rn == lexer.EOF {
			return lexer.EOFToken(e.r.Pos()), nil
		}
		pos := e.r.Pos()
		for _, namedProduction := range e.def.productions.Productions {
			e.r.Rewind()
			e.buf.Reset()
			name := namedProduction.Name
			production := namedProduction.Production
			if ok, err := e.match(name, production.Expr, e.buf); err != nil {
				return lexer.Token{}, err
			} else if ok {
				if e.def.elide[name] {
					continue nextToken
				}
				return lexer.Token{
					Type:  e.def.symbols[name],
					Pos:   pos,
					Value: e.buf.String(),
				}, nil
			}
		}
		token := lexer.Token{Pos: pos, Value: string(rn)}
		return token, lexer.ErrorWithTokenf(token, "no match found for %c", rn)
	}
}

func (e *ebnfLexer) match(name string, expr internal.Expression, out *bytes.Buffer) (bool, error) { // nolint: gocyclo, unparam
	switch n := expr.(type) {
	case internal.Alternative:
		for _, an := range n {
			if ok, err := e.match(name, an, out); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
		return false, nil

	case *internal.Group:
		return e.match(name, n.Body, out)

	case *internal.Name:
		return e.match(name, e.def.grammar.Index[n.String].Expr, out)

	case *internal.Option:
		_, err := e.match(name, n.Body, out)
		if err != nil {
			return false, err
		}
		return true, nil

	case *internal.Range:
		return false, fmt.Errorf("internal.Range should not occur here")

	case *internal.Repetition:
		for {
			ok, err := e.match(name, n.Body, out)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}
		}

	case internal.Sequence:
		for i, sn := range n {
			if ok, err := e.match(name, sn, out); err != nil {
				return false, err
			} else if ok {
				continue
			}
			if i > 0 {
				return false, nil
			}
			return false, nil
		}
		return true, nil

	case *internal.Token:
		return true, lexer.Errorf(e.r.Pos(), "internal.Token should not occur")

	case *ebnfToken:
		// If first rune doesn't match, we didn't match.
		if rn, err := e.peek(); err != nil {
			return false, err
		} else if n.runes[0] != rn {
			return false, nil
		}
		for _, rn := range n.runes {
			if r, err := e.read(); err != nil {
				return false, err
			} else if r != rn {
				return false, nil
			}
			out.WriteRune(rn)
		}
		return true, nil

	case *characterSet:
		rn, err := e.peek()
		if err != nil {
			return false, err
		}
		if n.Has(rn) {
			_, err = e.read()
			out.WriteRune(rn)
			return true, err
		}
		return false, nil

	case *rangeSet:
		rn, err := e.peek()
		if err != nil {
			return false, err
		}
		if n.Has(rn) {
			_, err = e.read()
			out.WriteRune(rn)
			return true, err
		}
		return false, nil

	case *asciiSet:
		rn, err := e.peek()
		if err != nil {
			return false, err
		}
		if n.Has(rn) {
			_, err = e.read()
			out.WriteRune(rn)
			return true, err
		}
		return false, nil

	case nil:
		if rn, err := e.peek(); err != nil {
			return false, err
		} else if rn == lexer.EOF {
			return false, nil
		}
		return false, fmt.Errorf("expected lexer.EOF")
	}
	return false, fmt.Errorf("unsupported lexer expression type %T", expr)
}

func (e *ebnfLexer) peek() (rune, error) {
	return e.fixRuneRead(e.r.Peek())
}

func (e *ebnfLexer) read() (rune, error) {
	return e.fixRuneRead(e.r.Read())
}

func (e *ebnfLexer) fixRuneRead(rn rune, err error) (rune, error) {
	if err == io.EOF {
		return lexer.EOF, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read rune: %s", err)
	}
	return rn, nil
}

type ebnfLexerDefinition struct {
	grammar     internal.Grammar
	symbols     map[string]rune
	elide       map[string]bool
	productions internal.Grammar
}

func (e *ebnfLexerDefinition) Lex(r io.Reader) (lexer.Lexer, error) {
	return &ebnfLexer{
		r: newTokenReader(bufio.NewReader(r), lexer.Position{
			Filename: lexer.NameOfReader(r),
			Line:     1,
			Column:   1,
		}),
		def: e,
		buf: bytes.NewBuffer(make([]byte, 0, 128)),
	}, nil
}

func (e *ebnfLexerDefinition) Symbols() map[string]rune {
	return e.symbols
}

// Apply some optimizations to the EBNF.
func (e *ebnfLexerDefinition) optimize(expr internal.Expression) internal.Expression {
	switch n := expr.(type) {
	case internal.Alternative:
		// Convert alternate characters into a character set (eg. "a" | "b" | "c" | "true" becomes
		// set("abc") | "true").
		out := make(internal.Alternative, 0, len(n))
		set := ""
		for _, expr := range n {
			if t, ok := expr.(*internal.Token); ok && utf8.RuneCountInString(t.String) == 1 {
				set += t.String
				continue
			}
			// Hit a node that is not a single-character Token. Flush set?
			if set != "" {
				out = append(out, &characterSet{pos: n.Pos(), Set: set})
				set = ""
			}
			out = append(out, e.optimize(expr))
		}
		if set != "" {
			out = append(out, &characterSet{pos: n.Pos(), Set: set})
		}
		return out

	case internal.Sequence:
		for i, expr := range n {
			n[i] = e.optimize(expr)
		}

	case *internal.Group:
		n.Body = e.optimize(n.Body)

	case *internal.Option:
		n.Body = e.optimize(n.Body)

	case *internal.Repetition:
		n.Body = e.optimize(n.Body)

	case *internal.Range:
		// Convert range into a set.
		begin, end := beginEnd(n)
		set := &rangeSet{
			pos:     n.Pos(),
			include: [2]rune{begin, end},
		}

		for next := n.Exclude; next != nil; {
			switch n := next.(type) {
			case *internal.Range:
				begin, end := beginEnd(n)
				set.exclude = append(set.exclude, [2]rune{begin, end})
				next = n.Exclude
			case *internal.Token:
				rn, _ := utf8.DecodeRuneInString(n.String)
				set.exclude = append(set.exclude, [2]rune{rn, rn})
				next = nil
			default:
				panic(fmt.Sprintf("should not have encountered %T", n))
			}
		}
		// Use an asciiSet if the characters are in ASCII range.
		return makeSet(n.Pos(), set)

	case *internal.Token:
		return &ebnfToken{pos: n.Pos(), runes: []rune(n.String)}
	}
	return expr
}

func beginEnd(n *internal.Range) (rune, rune) {
	begin, _ := utf8.DecodeRuneInString(n.Begin.String)
	end := begin
	if n.End != nil {
		end, _ = utf8.DecodeRuneInString(n.End.String)
	}
	if begin > end {
		begin, end = end, begin
	}
	return begin, end
}

// Validate the grammar against the lexer rules.
func validate(grammar internal.Grammar, expr internal.Expression) error { // nolint: gocyclo
	switch n := expr.(type) {
	case *internal.Production:
		return validate(grammar, n.Expr)

	case internal.Alternative:
		for _, e := range n {
			if err := validate(grammar, e); err != nil {
				return err
			}
		}
		return nil

	case *internal.Group:
		return validate(grammar, n.Body)

	case *internal.Name:
		if grammar.Index[n.String] == nil {
			token := lexer.Token{Pos: lexer.Position(n.Pos()), Value: n.String}
			return lexer.ErrorWithTokenf(token, "unknown production %q", n.String)
		}
		return nil

	case *internal.Option:
		return validate(grammar, n.Body)

	case *internal.Range:
		if utf8.RuneCountInString(n.Begin.String) != 1 {
			token := lexer.Token{Pos: lexer.Position(n.Pos()), Value: n.Begin.String}
			return lexer.ErrorWithTokenf(token, "start of range must be a single rune")
		}
		if utf8.RuneCountInString(n.End.String) != 1 {
			token := lexer.Token{Pos: lexer.Position(n.Pos()), Value: n.End.String}
			return lexer.ErrorWithTokenf(token, "end of range must be a single rune")
		}
		return nil

	case *internal.Repetition:
		return validate(grammar, n.Body)

	case internal.Sequence:
		for _, e := range n {
			if err := validate(grammar, e); err != nil {
				return err
			}
		}
		return nil

	case *internal.Token:
		return nil

	case nil:
		return nil
	}
	return lexer.Errorf(lexer.Position(expr.Pos()), "unknown EBNF expression %T", expr)
}
