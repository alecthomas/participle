package lexer

import (
	"strings"

	"github.com/stretchr/testify/require"

	"testing"
)

func TestRegexp(t *testing.T) {
	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
	require.NoError(t, err)
	require.Equal(t, map[string]rune{
		"EOF":        -1,
		"Ident":      -2,
		"Whitespace": -3,
		"Number":     -4,
	}, def.Symbols())
	lexer := def.Lex(strings.NewReader("hello\n123 456\nworld"))
	tokens := ReadAll(lexer)
	require.Equal(t, []Token{
		Token{Type: -2, Value: "hello", Pos: Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		Token{Type: -3, Value: "\n", Pos: Position{Filename: "", Offset: 5, Line: 1, Column: 6}},
		Token{Type: -4, Value: "123", Pos: Position{Filename: "", Offset: 6, Line: 2, Column: 1}},
		Token{Type: -3, Value: " ", Pos: Position{Filename: "", Offset: 9, Line: 2, Column: 4}},
		Token{Type: -4, Value: "456", Pos: Position{Filename: "", Offset: 10, Line: 2, Column: 5}},
		Token{Type: -3, Value: "\n", Pos: Position{Filename: "", Offset: 13, Line: 2, Column: 8}},
		Token{Type: -2, Value: "world", Pos: Position{Filename: "", Offset: 14, Line: 3, Column: 1}},
		Token{Type: -1, Value: "<<EOF>>", Pos: Position{Filename: "", Offset: 19, Line: 3, Column: 6}},
	}, tokens)
	require.Panics(t, func() {
		ReadAll(def.Lex(strings.NewReader("hello ?")))
	})
}
