package main

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/repr"
)

type operatorPrec struct{ Left, Right int }

var operatorPrecs = map[string]operatorPrec{
	"+": {1, 1},
	"-": {1, 1},
	"*": {3, 2},
	"/": {5, 4},
	"%": {7, 6},
}

type (
	Expr interface{ expr() }

	ExprIdent  struct{ Name string }
	ExprString struct{ Value string }
	ExprNumber struct{ Value float64 }
	ExprParens struct{ Sub Expr }

	ExprUnary struct {
		Op  string
		Sub Expr
	}

	ExprBinary struct {
		Lhs Expr
		Op  string
		Rhs Expr
	}
)

func (ExprIdent) expr()  {}
func (ExprString) expr() {}
func (ExprNumber) expr() {}
func (ExprParens) expr() {}
func (ExprUnary) expr()  {}
func (ExprBinary) expr() {}

func parseExprAny(lex *lexer.PeekingLexer) (Expr, error) { return parseExprPrec(lex, 0) }

func parseExprAtom(lex *lexer.PeekingLexer) (Expr, error) {
	switch peek := lex.Peek(); {
	case peek.Type == scanner.Ident:
		return ExprIdent{lex.Next().Value}, nil
	case peek.Type == scanner.String:
		val, err := strconv.Unquote(lex.Next().Value)
		if err != nil {
			return nil, err
		}
		return ExprString{val}, nil
	case peek.Type == scanner.Int || peek.Type == scanner.Float:
		val, err := strconv.ParseFloat(lex.Next().Value, 64)
		if err != nil {
			return nil, err
		}
		return ExprNumber{val}, nil
	case peek.Value == "(":
		_ = lex.Next()
		inner, err := parseExprAny(lex)
		if err != nil {
			return nil, err
		}
		if lex.Peek().Value != ")" {
			return nil, fmt.Errorf("expected closing ')'")
		}
		_ = lex.Next()
		return ExprParens{inner}, nil
	default:
		return nil, participle.NextMatch
	}
}

func parseExprPrec(lex *lexer.PeekingLexer, minPrec int) (Expr, error) {
	var lhs Expr
	if peeked := lex.Peek(); peeked.Value == "-" || peeked.Value == "!" {
		op := lex.Next().Value
		atom, err := parseExprAtom(lex)
		if err != nil {
			return nil, err
		}
		lhs = ExprUnary{op, atom}
	} else {
		atom, err := parseExprAtom(lex)
		if err != nil {
			return nil, err
		}
		lhs = atom
	}

	for {
		peek := lex.Peek()
		prec, isOp := operatorPrecs[peek.Value]
		if !isOp || prec.Left < minPrec {
			break
		}
		op := lex.Next().Value
		rhs, err := parseExprPrec(lex, prec.Right)
		if err != nil {
			return nil, err
		}
		lhs = ExprBinary{lhs, op, rhs}
	}
	return lhs, nil
}

type Expression struct {
	X Expr `@@`
}

var parser = participle.MustBuild(&Expression{}, participle.ParseTypeWith(parseExprAny))

func main() {
	var cli struct {
		Expr []string `arg required help:"Expression to parse."`
	}
	ctx := kong.Parse(&cli)

	expr := &Expression{}
	err := parser.ParseString("", strings.Join(cli.Expr, " "), expr)
	ctx.FatalIfErrorf(err)

	repr.Println(expr)
}
