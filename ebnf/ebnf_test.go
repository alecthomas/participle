package ebnf

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestEBNF(t *testing.T) {
	input := parser.String()
	t.Log(input)
	ast, err := ParseString(input)
	require.NoError(t, err, input)
	require.Equal(t, input, ast.String())
}

func TestEBNFRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"NegatedLiteral", `Term = ~"a" .`},
		{"NegatedToken", `Term = ~<ident> .`},
		{"NegatedProduction", `Term = ~Production .`},
		{"NegatedGroup", `Term = ~("a" | "b") .`},
		{"NegatedRepetition", `Term = ~"a"+ .`},
		{"PositiveLookahead", `Term = (?= "a") "b" .`},
		{"NegativeLookahead", `Term = (?! "a") "b" .`},
		// Verbatim output of Parser.String() for the grammar in
		// TestEBNF_Other in the root package.
		{"ParserOutput", `Grammar = ((?= "good") <ident>) | ((?! "bad" | "worse") <ident>) | ~("anything" | "but") .`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ast, err := ParseString(test.input)
			require.NoError(t, err, test.input)
			require.Equal(t, test.input, ast.String())
		})
	}
}
