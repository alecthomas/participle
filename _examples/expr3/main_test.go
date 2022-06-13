package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestExpressionParser(t *testing.T) {
	type testCase struct {
		src      string
		expected ExprPrecAll
	}

	for _, c := range []testCase{
		{`1`, ExprNumber{1}},
		{`1.5`, ExprNumber{1.5}},
		{`"a"`, ExprString{`"a"`}},
		{`(1)`, ExprParens{ExprNumber{1}}},
		{`1 + 1`, ExprAddSub{ExprNumber{1}, []ExprAddSubExt{{"+", ExprNumber{1}}}}},
		{`1 - 1`, ExprAddSub{ExprNumber{1}, []ExprAddSubExt{{"-", ExprNumber{1}}}}},
		{`1 * 1`, ExprMulDiv{ExprNumber{1}, []ExprMulDivExt{{"*", ExprNumber{1}}}}},
		{`1 / 1`, ExprMulDiv{ExprNumber{1}, []ExprMulDivExt{{"/", ExprNumber{1}}}}},
		{`1 % 1`, ExprRem{ExprNumber{1}, []ExprRemExt{{"%", ExprNumber{1}}}}},
		{
			`a + b - c * d / e % f`,
			ExprAddSub{
				ExprIdent{"a"},
				[]ExprAddSubExt{
					{"+", ExprIdent{"b"}},
					{"-", ExprMulDiv{
						ExprIdent{"c"},
						[]ExprMulDivExt{
							{"*", ExprIdent{Name: "d"}},
							{"/", ExprRem{
								ExprIdent{"e"},
								[]ExprRemExt{{"%", ExprIdent{"f"}}},
							}},
						},
					}},
				},
			},
		},
	} {
		var actual Expression
		require.NoError(t, parser.ParseString("<test>", c.src, &actual))
		require.Equal(t, c.expected, actual.X)
	}
}
