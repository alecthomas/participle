package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestCustomExprParser(t *testing.T) {
	type testCase struct {
		src      string
		expected Expr
	}

	for _, c := range []testCase{
		{`1`, ExprNumber{1}},
		{`1.5`, ExprNumber{1.5}},
		{`"a"`, ExprString{"a"}},
		{`(1)`, ExprParens{ExprNumber{1}}},
		{`1+1`, ExprBinary{ExprNumber{1}, "+", ExprNumber{1}}},
		{`1-1`, ExprBinary{ExprNumber{1}, "-", ExprNumber{1}}},
		{`1*1`, ExprBinary{ExprNumber{1}, "*", ExprNumber{1}}},
		{`1/1`, ExprBinary{ExprNumber{1}, "/", ExprNumber{1}}},
		{`1%1`, ExprBinary{ExprNumber{1}, "%", ExprNumber{1}}},
		{`a - -b`, ExprBinary{ExprIdent{"a"}, "-", ExprUnary{"-", ExprIdent{"b"}}}},
		{
			`a + b - c * d / e % f`,
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
		},
		{
			`a * b + c * d`,
			ExprBinary{
				ExprBinary{ExprIdent{"a"}, "*", ExprIdent{"b"}},
				"+",
				ExprBinary{ExprIdent{"c"}, "*", ExprIdent{"d"}},
			},
		},
		{
			`(a + b) * (c + d)`,
			ExprBinary{
				ExprParens{ExprBinary{ExprIdent{"a"}, "+", ExprIdent{"b"}}},
				"*",
				ExprParens{ExprBinary{ExprIdent{"c"}, "+", ExprIdent{"d"}}},
			},
		},
	} {
		actual, err := parser.ParseString("", c.src)
		require.NoError(t, err)
		require.Equal(t, c.expected, actual.X)
	}
}
