package lexer

import (
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
)

func TestRegexp(t *testing.T) {
	def, err := Regexp(`(?P<Ident>[⌘a-z]+)|(\s+)|(?P<Number>\d+)`)
	assert.NilError(t, err)
	assert.DeepEqual(t, map[string]rune{
		"EOF":    -1,
		"Ident":  -2,
		"Number": -4,
	}, def.Symbols())
	lexer := def.Lex(strings.NewReader("hello\n123 456\n⌘orld"))
	tokens, err := ConsumeAll(lexer)
	assert.NilError(t, err)
	assert.DeepEqual(t, []Token{
		{Type: -2, Value: "hello", Pos: Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		{Type: -4, Value: "123", Pos: Position{Filename: "", Offset: 6, Line: 2, Column: 1}},
		{Type: -4, Value: "456", Pos: Position{Filename: "", Offset: 10, Line: 2, Column: 5}},
		{Type: -2, Value: "⌘orld", Pos: Position{Filename: "", Offset: 14, Line: 3, Column: 1}},
		{Type: -1, Value: "<<EOF>>", Pos: Position{Filename: "", Offset: 21, Line: 3, Column: 6}},
	}, tokens)
	_, err = ConsumeAll(def.Lex(strings.NewReader("hello ?")))
	assert.Check(t, err != nil)
}

func BenchmarkRegexpLexer(b *testing.B) {
	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
	assert.NilError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := def.Lex(strings.NewReader("hello world 123 hello world 123"))
		ConsumeAll(lex)
	}
}
