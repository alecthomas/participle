package participle_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/scanner"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
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

func TestCustomExprParser(t *testing.T) {
	type Wrapper struct {
		Expr Expr `@@`
	}

	exprParser := mustTestParser(t, &Wrapper{}, participle.UseCustom(parseExprAny))

	requireParseExpr := func(src string, expected Wrapper) {
		t.Helper()
		var actual Wrapper
		require.NoError(t, exprParser.ParseString("", src, &actual))
		require.Equal(t, expected, actual)
	}

	requireParseExpr(`1`, Wrapper{ExprNumber{1}})
	requireParseExpr(`1.5`, Wrapper{ExprNumber{1.5}})
	requireParseExpr(`"a"`, Wrapper{ExprString{"a"}})
	requireParseExpr(`(1)`, Wrapper{ExprParens{ExprNumber{1}}})
	requireParseExpr(`1+1`, Wrapper{ExprBinary{ExprNumber{1}, "+", ExprNumber{1}}})
	requireParseExpr(`1-1`, Wrapper{ExprBinary{ExprNumber{1}, "-", ExprNumber{1}}})
	requireParseExpr(`1*1`, Wrapper{ExprBinary{ExprNumber{1}, "*", ExprNumber{1}}})
	requireParseExpr(`1/1`, Wrapper{ExprBinary{ExprNumber{1}, "/", ExprNumber{1}}})
	requireParseExpr(`1%1`, Wrapper{ExprBinary{ExprNumber{1}, "%", ExprNumber{1}}})
	requireParseExpr(`a - -b`, Wrapper{ExprBinary{ExprIdent{"a"}, "-", ExprUnary{"-", ExprIdent{"b"}}}})

	requireParseExpr(`a + b - c * d / e % f`, Wrapper{
		ExprBinary{
			ExprIdent{"a"}, "+", ExprBinary{
				ExprIdent{"b"}, "-", ExprBinary{
					ExprIdent{"c"}, "*", ExprBinary{
						ExprIdent{"d"}, "/", ExprBinary{
							ExprIdent{"e"}, "%", ExprIdent{"f"},
						},
					},
				},
			},
		},
	})

	requireParseExpr(`a * b + c * d`, Wrapper{
		ExprBinary{
			ExprBinary{ExprIdent{"a"}, "*", ExprIdent{"b"}},
			"+",
			ExprBinary{ExprIdent{"c"}, "*", ExprIdent{"d"}},
		},
	})

	requireParseExpr(`(a + b) * (c + d)`, Wrapper{
		ExprBinary{
			ExprParens{ExprBinary{ExprIdent{"a"}, "+", ExprIdent{"b"}}},
			"*",
			ExprParens{ExprBinary{ExprIdent{"c"}, "+", ExprIdent{"d"}}},
		},
	})

	require.Equal(t, "Wrapper = Expr .", exprParser.String())
}

type (
	MemberString struct {
		Value string `@String`
	}

	MemberNumber struct {
		Value float64 `@Int | @Float`
	}

	MemberIdent struct {
		Name string `@Ident`
	}

	MemberParens struct {
		Inner ExprPrecAll `"(" @@ ")"`
	}

	MemberUnary struct {
		Op   string      `@("-" | "!")`
		Expr ExprOperand `@@`
	}

	MemberBinAddSub struct {
		Head ExprPrec2             `@@`
		Tail []MemberBinAddSubCont `@@+`
	}

	MemberBinAddSubCont struct {
		Op   string    `@("+" | "-")`
		Expr ExprPrec2 `@@`
	}

	MemberBinMulDiv struct {
		Head ExprPrec3             `@@`
		Tail []MemberBinMulDivCont `@@+`
	}

	MemberBinMulDivCont struct {
		Op   string    `@("*" | "/")`
		Expr ExprPrec3 `@@`
	}

	MemberBinRem struct {
		Head ExprOperand        `@@`
		Tail []MemberBinRemCont `@@+`
	}

	MemberBinRemCont struct {
		Op   string      `@"%"`
		Expr ExprOperand `@@`
	}

	ExprPrecAll interface{ exprPrecAll() }
	ExprPrec1   interface{ exprPrec1() }
	ExprPrec2   interface{ exprPrec2() }
	ExprPrec3   interface{ exprPrec3() }
	ExprOperand interface{ exprOperand() }
)

// These expression types can be matches as individual operands
func (MemberIdent) exprOperand()  {}
func (MemberNumber) exprOperand() {}
func (MemberString) exprOperand() {}
func (MemberParens) exprOperand() {}
func (MemberUnary) exprOperand()  {}

// These expression types can be matched at precedence level 3
func (MemberIdent) exprPrec3()  {}
func (MemberNumber) exprPrec3() {}
func (MemberString) exprPrec3() {}
func (MemberParens) exprPrec3() {}
func (MemberUnary) exprPrec3()  {}
func (MemberBinRem) exprPrec3() {}

// These expression types can be matched at precedence level 2
func (MemberIdent) exprPrec2()     {}
func (MemberNumber) exprPrec2()    {}
func (MemberString) exprPrec2()    {}
func (MemberParens) exprPrec2()    {}
func (MemberUnary) exprPrec2()     {}
func (MemberBinRem) exprPrec2()    {}
func (MemberBinMulDiv) exprPrec2() {}

// These expression types can be matched at precedence level 1
func (MemberIdent) exprPrec1()     {}
func (MemberNumber) exprPrec1()    {}
func (MemberString) exprPrec1()    {}
func (MemberParens) exprPrec1()    {}
func (MemberUnary) exprPrec1()     {}
func (MemberBinRem) exprPrec1()    {}
func (MemberBinMulDiv) exprPrec1() {}
func (MemberBinAddSub) exprPrec1() {}

// These are all of the expression types
func (MemberIdent) exprPrecAll()     {}
func (MemberNumber) exprPrecAll()    {}
func (MemberString) exprPrecAll()    {}
func (MemberParens) exprPrecAll()    {}
func (MemberUnary) exprPrecAll()     {}
func (MemberBinRem) exprPrecAll()    {}
func (MemberBinMulDiv) exprPrecAll() {}
func (MemberBinAddSub) exprPrecAll() {}

func TestUnionExprParser(t *testing.T) {
	type Wrapper struct {
		Expr ExprPrecAll `@@`
	}

	var (
		withExprOperand = participle.UseUnion[ExprOperand](
			MemberUnary{}, MemberIdent{}, MemberNumber{}, MemberString{}, MemberParens{})

		withExprPrec3 = participle.UseUnion[ExprPrec3](
			MemberBinRem{}, MemberUnary{}, MemberIdent{}, MemberNumber{}, MemberString{}, MemberParens{})

		withExprPrec2 = participle.UseUnion[ExprPrec2](
			MemberBinMulDiv{}, MemberBinRem{}, MemberUnary{}, MemberIdent{}, MemberNumber{}, MemberString{}, MemberParens{})

		withExprPrec1 = participle.UseUnion[ExprPrec1](
			MemberBinAddSub{}, MemberBinMulDiv{}, MemberBinRem{}, MemberUnary{}, MemberIdent{}, MemberNumber{}, MemberString{}, MemberParens{})

		withExprPrecAll = participle.UseUnion[ExprPrecAll](
			MemberBinAddSub{}, MemberBinMulDiv{}, MemberBinRem{}, MemberUnary{}, MemberIdent{}, MemberNumber{}, MemberString{}, MemberParens{})
	)

	exprParser := mustTestParser(t, &Wrapper{}, withExprPrecAll, withExprPrec1, withExprPrec2, withExprPrec3, withExprOperand)

	requireParseExpr := func(src string, expected Wrapper) {
		t.Helper()
		var actual Wrapper
		require.NoError(t, exprParser.ParseString("", src, &actual))
		require.Equal(t, expected, actual)
	}

	requireParseExpr(`1`, Wrapper{MemberNumber{1}})
	requireParseExpr(`1.5`, Wrapper{MemberNumber{1.5}})
	requireParseExpr(`"a"`, Wrapper{MemberString{`"a"`}})
	requireParseExpr(`(1)`, Wrapper{MemberParens{MemberNumber{1}}})
	requireParseExpr(`1 + 1`, Wrapper{MemberBinAddSub{MemberNumber{1}, []MemberBinAddSubCont{{"+", MemberNumber{1}}}}})
	requireParseExpr(`1 - 1`, Wrapper{MemberBinAddSub{MemberNumber{1}, []MemberBinAddSubCont{{"-", MemberNumber{1}}}}})
	requireParseExpr(`1 * 1`, Wrapper{MemberBinMulDiv{MemberNumber{1}, []MemberBinMulDivCont{{"*", MemberNumber{1}}}}})
	requireParseExpr(`1 / 1`, Wrapper{MemberBinMulDiv{MemberNumber{1}, []MemberBinMulDivCont{{"/", MemberNumber{1}}}}})
	requireParseExpr(`1 % 1`, Wrapper{MemberBinRem{MemberNumber{1}, []MemberBinRemCont{{"%", MemberNumber{1}}}}})
	requireParseExpr(`a - -b`, Wrapper{MemberBinAddSub{MemberIdent{"a"}, []MemberBinAddSubCont{{"-", MemberUnary{"-", MemberIdent{"b"}}}}}})

	requireParseExpr(`a + b - c * d / e % f`, Wrapper{
		Expr: MemberBinAddSub{
			MemberIdent{"a"},
			[]MemberBinAddSubCont{
				{"+", MemberIdent{"b"}},
				{"-", MemberBinMulDiv{
					MemberIdent{"c"},
					[]MemberBinMulDivCont{
						{"*", MemberIdent{Name: "d"}},
						{"/", MemberBinRem{
							MemberIdent{"e"},
							[]MemberBinRemCont{
								{"%", MemberIdent{"f"}},
							},
						}},
					},
				}},
			},
		},
	})

	requireParseExpr(`a * b + c * d`, Wrapper{
		MemberBinAddSub{
			MemberBinMulDiv{MemberIdent{"a"}, []MemberBinMulDivCont{{"*", MemberIdent{"b"}}}},
			[]MemberBinAddSubCont{{
				"+",
				MemberBinMulDiv{MemberIdent{"c"}, []MemberBinMulDivCont{{"*", MemberIdent{"d"}}}},
			}},
		},
	})

	requireParseExpr(`(a + b) * (c + d)`, Wrapper{
		MemberBinMulDiv{
			MemberParens{
				MemberBinAddSub{MemberIdent{"a"}, []MemberBinAddSubCont{{"+", MemberIdent{"b"}}}},
			},
			[]MemberBinMulDivCont{
				{"*", MemberParens{
					MemberBinAddSub{MemberIdent{"c"}, []MemberBinAddSubCont{{"+", MemberIdent{"d"}}}}},
				},
			},
		},
	})

	require.Equal(t, strings.TrimSpace(`
Wrapper = ExprPrecAll .
ExprPrecAll = MemberBinAddSub | MemberBinMulDiv | MemberBinRem | MemberUnary | MemberIdent | MemberNumber | MemberString | MemberParens .
MemberBinAddSub = ExprPrec2 MemberBinAddSubCont+ .
ExprPrec2 = MemberBinMulDiv | MemberBinRem | MemberUnary | MemberIdent | MemberNumber | MemberString | MemberParens .
MemberBinMulDiv = ExprPrec3 MemberBinMulDivCont+ .
ExprPrec3 = MemberBinRem | MemberUnary | MemberIdent | MemberNumber | MemberString | MemberParens .
MemberBinRem = ExprOperand MemberBinRemCont+ .
ExprOperand = MemberUnary | MemberIdent | MemberNumber | MemberString | MemberParens .
MemberUnary = ("-" | "!") ExprOperand .
MemberIdent = <ident> .
MemberNumber = <int> | <float> .
MemberString = <string> .
MemberParens = "(" ExprPrecAll ")" .
MemberBinRemCont = "%" ExprOperand .
MemberBinMulDivCont = ("*" | "/") ExprPrec3 .
MemberBinAddSubCont = ("+" | "-") ExprPrec2 .
	`), exprParser.String())
}
