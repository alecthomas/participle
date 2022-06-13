package main

import (
	"strings"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/repr"
)

type (
	ExprString struct {
		Value string `@String`
	}

	ExprNumber struct {
		Value float64 `@Int | @Float`
	}

	ExprIdent struct {
		Name string `@Ident`
	}

	ExprParens struct {
		Inner ExprPrecAll `"(" @@ ")"`
	}

	ExprUnary struct {
		Op   string      `@("-" | "!")`
		Expr ExprOperand `@@`
	}

	ExprAddSub struct {
		Head ExprPrec2       `@@`
		Tail []ExprAddSubExt `@@+`
	}

	ExprAddSubExt struct {
		Op   string    `@("+" | "-")`
		Expr ExprPrec2 `@@`
	}

	ExprMulDiv struct {
		Head ExprPrec3       `@@`
		Tail []ExprMulDivExt `@@+`
	}

	ExprMulDivExt struct {
		Op   string    `@("*" | "/")`
		Expr ExprPrec3 `@@`
	}

	ExprRem struct {
		Head ExprOperand  `@@`
		Tail []ExprRemExt `@@+`
	}

	ExprRemExt struct {
		Op   string      `@"%"`
		Expr ExprOperand `@@`
	}

	ExprPrecAll interface{ exprPrecAll() }
	ExprPrec2   interface{ exprPrec2() }
	ExprPrec3   interface{ exprPrec3() }
	ExprOperand interface{ exprOperand() }
)

// These expression types can be matches as individual operands
func (ExprIdent) exprOperand()  {}
func (ExprNumber) exprOperand() {}
func (ExprString) exprOperand() {}
func (ExprParens) exprOperand() {}
func (ExprUnary) exprOperand()  {}

// These expression types can be matched at precedence level 3
func (ExprIdent) exprPrec3()  {}
func (ExprNumber) exprPrec3() {}
func (ExprString) exprPrec3() {}
func (ExprParens) exprPrec3() {}
func (ExprUnary) exprPrec3()  {}
func (ExprRem) exprPrec3()    {}

// These expression types can be matched at precedence level 2
func (ExprIdent) exprPrec2()  {}
func (ExprNumber) exprPrec2() {}
func (ExprString) exprPrec2() {}
func (ExprParens) exprPrec2() {}
func (ExprUnary) exprPrec2()  {}
func (ExprRem) exprPrec2()    {}
func (ExprMulDiv) exprPrec2() {}

// These expression types can be matched at the minimum precedence level
func (ExprIdent) exprPrecAll()  {}
func (ExprNumber) exprPrecAll() {}
func (ExprString) exprPrecAll() {}
func (ExprParens) exprPrecAll() {}
func (ExprUnary) exprPrecAll()  {}
func (ExprRem) exprPrecAll()    {}
func (ExprMulDiv) exprPrecAll() {}
func (ExprAddSub) exprPrecAll() {}

type Expression struct {
	X ExprPrecAll `@@`
}

var parser = participle.MustBuild(&Expression{},
	// This grammar requires enough lookahead to see the entire expression before
	// it can select the proper binary expression type - in other words, we only
	// know that `1 * 2 * 3 * 4` isn't the left-hand side of an addition or subtraction
	// expression until we know for sure that no `+` or `-` operator follows it
	participle.UseLookahead(99999),
	// Register the ExprOperand union so we can parse individual operands
	participle.ParseUnion[ExprOperand](ExprUnary{}, ExprIdent{}, ExprNumber{}, ExprString{}, ExprParens{}),
	// Register the ExprPrec3 union so we can parse expressions at precedence level 3
	participle.ParseUnion[ExprPrec3](ExprRem{}, ExprUnary{}, ExprIdent{}, ExprNumber{}, ExprString{}, ExprParens{}),
	// Register the ExprPrec2 union so we can parse expressions at precedence level 2
	participle.ParseUnion[ExprPrec2](ExprMulDiv{}, ExprRem{}, ExprUnary{}, ExprIdent{}, ExprNumber{}, ExprString{}, ExprParens{}),
	// Register the ExprPrecAll union so we can parse expressions at the minimum precedence level
	participle.ParseUnion[ExprPrecAll](ExprAddSub{}, ExprMulDiv{}, ExprRem{}, ExprUnary{}, ExprIdent{}, ExprNumber{}, ExprString{}, ExprParens{}),
)

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
