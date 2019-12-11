package regex

import (
	"strings"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/lexer"
)

func TestLexer(t *testing.T) {
	d, err := New(`
		Ident = [[:alpha:]]\w*
		Whitespace = \s+
		Equal = =
	`)
	require.NoError(t, err)
	l, err := d.Lex(strings.NewReader("hello = world"))
	require.NoError(t, err)
	actual, err := lexer.ConsumeAll(l)
	require.NoError(t, err)
	repr.Println(actual, repr.IgnoreGoStringer())
	expected := []lexer.Token{
		{Type: -2, Value: "hello", Pos: lexer.Position{Line: 1, Column: 1}},
		{Type: -3, Value: " ", Pos: lexer.Position{Offset: 5, Line: 1, Column: 6}},
		{Type: -4, Value: "=", Pos: lexer.Position{Offset: 6, Line: 1, Column: 7}},
		{Type: -3, Value: " ", Pos: lexer.Position{Offset: 7, Line: 1, Column: 8}},
		{Type: -2, Value: "world", Pos: lexer.Position{Offset: 8, Line: 1, Column: 9}},
		{Type: -1, Pos: lexer.Position{Offset: 13, Line: 1, Column: 14}},
	}
	require.Equal(t, expected, actual)
}
