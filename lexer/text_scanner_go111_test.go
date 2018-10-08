// +build go1.11

package lexer

import (
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/require"
)

func TestLexBacktickString(t *testing.T) {
	lexer := LexString("`hello\\nworld`")
	token, err := lexer.Next()
	require.NoError(t, err)
	require.Equal(t, Token{Type: scanner.RawString, Value: "hello\\nworld", Pos: Position{Line: 1, Column: 1}}, token)
}
