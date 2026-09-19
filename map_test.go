package participle_test

import (
	"strings"
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestUpper(t *testing.T) {
	type grammar struct {
		Text string `@Ident`
	}
	def := lexer.MustSimple([]lexer.SimpleRule{
		{"Whitespace", `\s+`},
		{"Ident", `\w+`},
	})
	parser := mustTestParser[grammar](t, participle.Lexer(def), participle.Upper("Ident"))
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
	type grammar struct {
		Text string `@Ident`
	}
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `\w+`},
		{"String", `\"(?:[^\"]|\\.)*\"`},
		{"RawString", "`[^`]*`"},
	})
	parser := mustTestParser[grammar](t, participle.Lexer(lex), participle.Unquote("String", "RawString"))
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

func TestUnquoteByteEscapes(t *testing.T) {
	type grammar struct {
		Text string `@String`
	}
	parser := mustTestParser[grammar](t, participle.Unquote())
	for _, tc := range []struct {
		name, input, want string
	}{
		{"hexadecimal byte", `"\xff"`, "\xff"},
		{"octal byte", `"\377"`, "\xff"},
		{"UTF-8 bytes", `"\xc3\xbf"`, "ÿ"},
		{"mixed byte and Unicode", `"a\xff\u00ff"`, "a\xffÿ"},
		{"Unicode escape", `"\u00ff"`, "ÿ"},
		{"literal Unicode", `"ÿ"`, "ÿ"},
		{"ASCII escapes", `"\x41\101"`, "AA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parser.ParseString("", tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, actual.Text)
		})
	}
}

func TestUnquoteShortToken(t *testing.T) {
	type grammar struct {
		Text string `@String`
	}
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"String", `.`},
	})
	parser := mustTestParser[grammar](t, participle.Lexer(lex), participle.Unquote("String"))
	defer func() {
		recovered := recover()
		require.NotZero(t, recovered)
	}()
	_, _ = parser.Lex("", strings.NewReader(`"`))
	t.Fatal("expected panic on token too short to be unquoted")
}
