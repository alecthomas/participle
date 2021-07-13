package ebnf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEBNF(t *testing.T) {
	input := parser.String()
	t.Log(input)
	ast, err := ParseString(input)
	require.NoError(t, err, input)
	require.Equal(t, input, ast.String())
}
