// +build !go1.11

package lexer

import (
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/require"
)

func TestLexBacktickString(t *testing.T) {
	lexer := LexString("", "`hello\\nworld`")
	token := lexer.Next()
	// See https://github.com/golang/go/issues/23675.  Go 1.11 fixes token type into RawString.
	require.Equal(t, Token{Type: scanner.String, Value: "hello\\nworld", Pos: Position{Line: 1, Column: 1}}, token)
}
