package lexer_test

import (
	"strings"
	"testing"
	"text/scanner"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestLexer(t *testing.T) {
	lex, err := lexer.Upgrade(lexer.LexString("", "hello world"))
	require.NoError(t, err)
	helloPos := lexer.Position{Offset: 0, Line: 1, Column: 1}
	worldPos := lexer.Position{Offset: 6, Line: 1, Column: 7}
	eofPos := lexer.Position{Offset: 11, Line: 1, Column: 12}
	require.Equal(t, lexer.Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lex.Peek())
	require.Equal(t, lexer.Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lex.Peek())
	require.Equal(t, lexer.Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lex.Next())
	require.Equal(t, lexer.Token{Type: scanner.Ident, Value: "world", Pos: worldPos}, lex.Peek())
	require.Equal(t, lexer.Token{Type: scanner.Ident, Value: "world", Pos: worldPos}, lex.Next())
	require.Equal(t, lexer.Token{Type: scanner.EOF, Value: "", Pos: eofPos}, lex.Peek())
	require.Equal(t, lexer.Token{Type: scanner.EOF, Value: "", Pos: eofPos}, lex.Next())
}

func TestLexString(t *testing.T) {
	lex := lexer.LexString("", "\"hello world\"")
	token, err := lex.Next()
	require.NoError(t, err)
	require.Equal(t, token, lexer.Token{Type: scanner.String, Value: "\"hello world\"", Pos: lexer.Position{Line: 1, Column: 1}})
}

func TestLexSingleString(t *testing.T) {
	lex := lexer.LexString("", "`hello world`")
	token, err := lex.Next()
	require.NoError(t, err)
	require.Equal(t, lexer.Token{Type: scanner.RawString, Value: "`hello world`", Pos: lexer.Position{Line: 1, Column: 1}}, token)
	lex = lexer.LexString("", `'\U00008a9e'`)
	token, err = lex.Next()
	require.NoError(t, err)
	require.Equal(t, lexer.Token{Type: scanner.Char, Value: `'\U00008a9e'`, Pos: lexer.Position{Line: 1, Column: 1}}, token)
}

func TestNewTextScannerLexerDefault(t *testing.T) {
	type grammar struct {
		Text string `@String @Ident`
	}
	p, err := participle.Build[grammar](participle.Lexer(lexer.NewTextScannerLexer(nil)), participle.Unquote())
	require.NoError(t, err)
	g, err := p.ParseString("", `"hello" world`)
	require.NoError(t, err)
	require.Equal(t, "helloworld", g.Text)
}

func TestNewTextScannerLexer(t *testing.T) {
	type grammar struct {
		Text string `(@String | @Comment | @Ident)+`
	}
	p, err := participle.Build[grammar](participle.Lexer(lexer.NewTextScannerLexer(func(s *scanner.Scanner) {
		s.Mode &^= scanner.SkipComments
	})), participle.Unquote())
	require.NoError(t, err)
	g, err := p.ParseString("", `"hello" /* comment */ world`)
	require.NoError(t, err)
	require.Equal(t, "hello/* comment */world", g.Text)
}

func BenchmarkTextScannerLexer(b *testing.B) {
	input := strings.Repeat("hello world 123 hello world 123", 100)
	r := strings.NewReader(input)
	b.ReportMetric(float64(len(input)), "B")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lex, _ := lexer.TextScannerLexer.Lex("", r)
		for {
			token, _ := lex.Next()
			if token.Type == lexer.EOF {
				break
			}
		}
		_, _ = r.Seek(0, 0)
	}
}
