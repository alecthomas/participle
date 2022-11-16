// Package main shows an example of how to add precedence climbing to a Participle parser.
//
// Precedence climbing is an approach to parsing expressions that efficiently
// produces compact parse trees.
//
// In contrast, naive recursive descent expression parsers produce parse trees proportional in
// complexity to the number of operators supported. This impacts both readability and
// performance.
//
// It is based on https://eli.thegreenplace.net/2012/08/02/parsing-expressions-by-precedence-climbing
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type opInfo struct {
	RightAssociative bool
	Priority         int
}

var info = map[string]opInfo{
	"+": {Priority: 1},
	"-": {Priority: 1},
	"*": {Priority: 2},
	"/": {Priority: 2},
	"^": {RightAssociative: true, Priority: 3},
}

type Expr struct {
	Terminal *int

	Left  *Expr
	Op    string
	Right *Expr
}

func (e *Expr) String() string {
	if e.Left != nil {
		return fmt.Sprintf("(%s %s %s)", e.Left, e.Op, e.Right)
	}
	return fmt.Sprintf("%d", *e.Terminal)
}

func (e *Expr) Parse(lex *lexer.PeekingLexer) error {
	*e = *parseExpr(lex, 0)
	return nil
}

// (1 + 2) * 3
func parseExpr(lex *lexer.PeekingLexer, minPrec int) *Expr {
	lhs := parseAtom(lex)
	for {
		tok := peek(lex)
		if tok.EOF() || !isOp(rune(tok.Type)) || info[tok.Value].Priority < minPrec {
			break
		}
		op := tok.Value
		nextMinPrec := info[op].Priority
		if !info[op].RightAssociative {
			nextMinPrec++
		}
		lex.Next()
		rhs := parseExpr(lex, nextMinPrec)
		lhs = parseOp(op, lhs, rhs)
	}
	return lhs
}
func parseAtom(lex *lexer.PeekingLexer) *Expr {
	tok := peek(lex)
	if tok.Type == '(' {
		lex.Next()
		val := parseExpr(lex, 1)
		if peek(lex).Value != ")" {
			panic("unmatched (")
		}
		lex.Next()
		return val
	} else if tok.EOF() {
		panic("unexpected EOF")
	} else if isOp(rune(tok.Type)) {
		panic("expected a terminal not " + tok.String())
	} else {
		lex.Next()
		n, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			panic("invalid number " + tok.Value)
		}
		in := int(n)
		return &Expr{Terminal: &in}
	}
}

func isOp(rn rune) bool {
	return strings.ContainsRune("+-*/^", rn)
}

func peek(lex *lexer.PeekingLexer) *lexer.Token {
	return lex.Peek()
}

func parseOp(op string, lhs *Expr, rhs *Expr) *Expr {
	return &Expr{
		Op:    op,
		Left:  lhs,
		Right: rhs,
	}
}

var parser = participle.MustBuild[Expr]()

func main() {
	e, err := parser.ParseString("", strings.Join(os.Args[1:], " "))
	fmt.Println(e)
	repr.Println(e)
	if err != nil {
		panic(err)
	}
}
