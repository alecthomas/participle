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
	lexer, err := def.Lex(strings.NewReader("hello\n123 456\n⌘orld"))
	require.NoError(t, err)
	tokens, err := ConsumeAll(lexer)
	require.NoError(t, err)
	require.Equal(t, []Token{
		{Type: -2, Value: "hello", Pos: Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		{Type: -4, Value: "123", Pos: Position{Filename: "", Offset: 6, Line: 2, Column: 1}},
		{Type: -4, Value: "456", Pos: Position{Filename: "", Offset: 10, Line: 2, Column: 5}},
		{Type: -2, Value: "⌘orld", Pos: Position{Filename: "", Offset: 14, Line: 3, Column: 1}},
		{Type: EOF, Value: "", Pos: Position{Filename: "", Offset: 21, Line: 3, Column: 6}},
	}, tokens)
	lexer, err = def.Lex(strings.NewReader("hello ?"))
	require.NoError(t, err)
	_, err = ConsumeAll(lexer)
	require.Error(t, err)
}

func BenchmarkRegexpLexer(b *testing.B) {
	b.ReportAllocs()
	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
	require.NoError(b, err)
	r := strings.NewReader(strings.Repeat("hello world 123 hello world 123", 100))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex, _ := def.Lex(r)
		for {
			token, _ := lex.Next()
			if token.Type == EOF {
				break
			}
		}
		r.Seek(0, 0)
	}
}
