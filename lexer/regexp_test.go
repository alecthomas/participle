package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexp(t *testing.T) {
	def, err := Regexp(`(?P<Ident>[⌘a-z]+)|(\s+)|(?P<Number>\d+)`)
	require.NoError(t, err)
	require.Equal(t, map[string]rune{
		"EOF":    -1,
		"Ident":  -2,
		"Number": -4,
	}, def.Symbols())
	lexer := def.Lex(strings.NewReader("hello\n123 456\n⌘orld"))
	tokens, err := ConsumeAll(lexer)
	require.NoError(t, err)
	require.Equal(t, []Token{
		{Type: -2, Value: "hello", Pos: Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		{Type: -4, Value: "123", Pos: Position{Filename: "", Offset: 6, Line: 2, Column: 1}},
		{Type: -4, Value: "456", Pos: Position{Filename: "", Offset: 10, Line: 2, Column: 5}},
		{Type: -2, Value: "⌘orld", Pos: Position{Filename: "", Offset: 14, Line: 3, Column: 1}},
		{Type: -1, Value: "<<EOF>>", Pos: Position{Filename: "", Offset: 21, Line: 3, Column: 6}},
	}, tokens)
	_, err = ConsumeAll(def.Lex(strings.NewReader("hello ?")))
	require.Error(t, err)
}

func BenchmarkRegexpLexer(b *testing.B) {
	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := def.Lex(strings.NewReader("hello world 123 hello world 123"))
		ConsumeAll(lex)
	}
}
