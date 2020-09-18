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
	"strconv"
	"text/scanner"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
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
	lhs := next(lex)
	for {
		op := peek(lex)
		if op == nil || info[op.Op].Priority < minPrec {
			break
		}
		nextMinPrec := info[op.Op].Priority
		if !info[op.Op].RightAssociative {
			nextMinPrec++
		}
		next(lex)
		rhs := parseExpr(lex, nextMinPrec)
		lhs = parseOp(op, lhs, rhs)
	}
	return lhs
}

func parseOp(op *Expr, lhs *Expr, rhs *Expr) *Expr {
	op.Left = lhs
	op.Right = rhs
	return op
}

func next(lex *lexer.PeekingLexer) *Expr {
	e := peek(lex)
	if e == nil {
		return e
	}
	_, _ = lex.Next()
	switch e.Op {
	case "(":
		return next(lex)
	}
	return e
}

func peek(lex *lexer.PeekingLexer) *Expr {
	t, err := lex.Peek(0)
	if err != nil {
		panic(err)
	}
	if t.EOF() {
		return nil
	}
	switch t.Type {
	case scanner.Int:
		n, err := strconv.ParseInt(t.Value, 10, 64)
		if err != nil {
			panic(err)
		}
		ni := int(n)
		return &Expr{Terminal: &ni}

	case ')':
		_, _ = lex.Next()
		return nil

	default:
		return &Expr{Op: t.Value}
	}
}

var parser = participle.MustBuild(&Expr{})

func main() {
	e := &Expr{}
	err := parser.ParseString("", `(1 + 3) * 2 ^ 2 + 1`, e)
	if err != nil {
		panic(err)
	}
	fmt.Println(e)
	repr.Println(e)
}
