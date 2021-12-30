package participle_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestUpper(t *testing.T) {
	var grammar struct {
		Text string `@Ident`
	}
	def := lexer.MustSimple([]lexer.SimpleRule{
		{"Whitespace", `\s+`},
		{"Ident", `\w+`},
	})
	parser := mustTestParser(t, &grammar, participle.Lexer(def), participle.Upper("Ident"))
	actual, err := parser.Lex("", strings.NewReader("hello world"))
	require.NoError(t, err)

	expected := []lexer.Token{
		{Type: -3, Value: "HELLO", Pos: lexer.Position{Filename: "", Offset: 0, Line: 1, Column: 1}},
		{Type: -2, Value: " ", Pos: lexer.Position{Filename: "", Offset: 5, Line: 1, Column: 6}},
		{Type: -3, Value: "WORLD", Pos: lexer.Position{Filename: "", Offset: 6, Line: 1, Column: 7}},
		{Type: lexer.EOF, Value: "", Pos: lexer.Position{Filename: "", Offset: 11, Line: 1, Column: 12}},
	}

	require.Equal(t, expected, actual)
}

func TestUnquote(t *testing.T) {
	var grammar struct {
		Text string `@Ident`
	}
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `\w+`},
		{"String", `\"(?:[^\"]|\\.)*\"`},
		{"RawString", "`[^`]*`"},
	})
	parser := mustTestParser(t, &grammar, participle.Lexer(lex), participle.Unquote("String", "RawString"))
	actual, err := parser.Lex("", strings.NewReader("hello world \"quoted\\tstring\" `backtick quotes`"))
	require.NoError(t, err)
	expected := []lexer.Token{
		{Type: -3, Value: "hello", Pos: lexer.Position{Line: 1, Column: 1}},
		{Type: -3, Value: "world", Pos: lexer.Position{Offset: 6, Line: 1, Column: 7}},
		{Type: -4, Value: "quoted\tstring", Pos: lexer.Position{Offset: 12, Line: 1, Column: 13}},
		{Type: -5, Value: "backtick quotes", Pos: lexer.Position{Offset: 29, Line: 1, Column: 30}},
		{Type: lexer.EOF, Value: "", Pos: lexer.Position{Offset: 46, Line: 1, Column: 47}},
	}
	require.Equal(t, expected, actual)
}
