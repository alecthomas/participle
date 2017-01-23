package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	def := Must(Regexp(`(?P<Whitespace>\s+)|(?P<Ident>\w+)`))

	// Remove whitespace and upper case all other tokens.
	mapper := Map(def, func(t *Token) *Token {
		if t.Type == def.Symbols()["Whitespace"] {
			return nil
		}
		t.Value = strings.ToUpper(t.Value)
		return t
	})

	mappingLexer := mapper.Lex(strings.NewReader("hello world"))
	actual, err := ConsumeAll(mappingLexer)
	require.NoError(t, err)

	expected := []Token{
		Token{Type: -3, Value: "HELLO", Pos: Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		Token{Type: -3, Value: "WORLD", Pos: Position{Filename: "", Offset: 6, Line: 1, Column: 7}},
		Token{Type: -1, Value: "<<EOF>>", Pos: Position{Filename: "", Offset: 11, Line: 1, Column: 12}},
	}

	require.Equal(t, expected, actual)
}
