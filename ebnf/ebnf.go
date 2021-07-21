// Package ebnf contains the AST and parser for parsing the form of EBNF produced by Participle.
//
// The self-referential EBNF is:
//
//      EBNF = Production* .
//      Production = <ident> "=" Expression "." .
//      Expression = Sequence ("|" Sequence)* .
//      SubExpression = "(" ("?!" | "?=")? Expression ")" .
//      Sequence = Term+ .
//      Term = "~"? (<ident> | <string> | ("<" <ident> ">") | SubExpression) ("*" | "+" | "?" | "!")? .
package ebnf

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
)

var parser = participle.MustBuild(&EBNF{})

// Term in the EBNF grammar.
type Term struct {
	Negation bool `@("~")?`

	Name    string         `(   @Ident`
	Literal string         `  | @String`
	Token   string         `  | "<" @Ident ">"`
	Group   *SubExpression `  | @@ )`

	Repetition string `@("*" | "+" | "?" | "!")?`
}

func (t *Term) String() string {
	switch {
	case t.Name != "":
		return t.Name + t.Repetition
	case t.Literal != "":
		return t.Literal + t.Repetition
	case t.Token != "":
		return "<" + t.Token + ">" + t.Repetition
	case t.Group != nil:
		return t.Group.String() + t.Repetition
	default:
		panic("??")
	}
}

// LookaheadAssertion enum.
type LookaheadAssertion rune

func (l *LookaheadAssertion) Capture(tokens []string) error { // nolint
	rn := tokens[0][0]
	switch rn {
	case '!', '=':
		*l = LookaheadAssertion(rn)

	default:
		panic(rn)
	}
	return nil
}

// Lookahead assertion enums.
const (
	LookaheadAssertionNone     LookaheadAssertion = 0
	LookaheadAssertionNegative LookaheadAssertion = '!'
	LookaheadAssertionPositive LookaheadAssertion = '='
)

// SubExpression is an expression inside parentheses ( ... )
type SubExpression struct {
	Lookahead LookaheadAssertion `"(" ("?" @("!" | "="))?`
	Expr      *Expression        `@@ ")"`
}

func (s *SubExpression) String() string {
	out := "("
	if s.Lookahead != LookaheadAssertionNone {
		out += "?" + string(s.Lookahead)
	}
	out += s.Expr.String() + ")"
	return out
}

// A Sequence of terms.
type Sequence struct {
	Terms []*Term `@@+`
}

func (s *Sequence) String() (out string) {
	for i, term := range s.Terms {
		if i > 0 {
			out += " "
		}
		out += term.String()
	}
	return
}

// Expression is a set of alternatives separated by "|" in the EBNF.
type Expression struct {
	Alternatives []*Sequence `@@ ( "|" @@ )*`
}

func (e *Expression) String() (out string) {
	for i, seq := range e.Alternatives {
		if i > 0 {
			out += " | "
		}
		out += seq.String()
	}
	return
}

// Production of the grammar.
type Production struct {
	Production string      `@Ident "="`
	Expression *Expression `@@ "."`
}

// EBNF itself.
type EBNF struct {
	Productions []*Production `@@*`
}

func (e *EBNF) String() (out string) {
	for i, production := range e.Productions {
		out += fmt.Sprintf("%s = %s .", production.Production, production.Expression)
		if i < len(e.Productions)-1 {
			out += "\n"
		}
	}
	return
}

// ParseString string into EBNF.
func ParseString(ebnf string) (*EBNF, error) {
	out := &EBNF{}
	return out, parser.ParseString("", ebnf, out)
}

// Parse io.Reader into EBNF.
func Parse(r io.Reader) (*EBNF, error) {
	out := &EBNF{}
	return out, parser.Parse("", r, out)
}
