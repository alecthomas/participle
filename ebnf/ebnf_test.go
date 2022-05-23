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
