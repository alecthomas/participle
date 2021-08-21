package ebnf

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestEBNF(t *testing.T) {
	input := parser.String()
	t.Log(input)
	ast, err := ParseString(input)
	assert.NoError(t, err, input)
	assert.Equal(t, input, ast.String())
}
