package lexer_test

import (
	"math/rand"
	"testing"

	require "github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2/lexer"
)

type staticLexer struct {
	tokens []lexer.Token
}

func (s *staticLexer) Next() (lexer.Token, error) {
	if len(s.tokens) == 0 {
		return lexer.EOFToken(lexer.Position{}), nil
	}
	t := s.tokens[0]
	s.tokens = s.tokens[1:]
	return t, nil
}

func TestUpgrade(t *testing.T) {
	t0 := lexer.Token{Type: 1, Value: "moo"}
	ts := lexer.Token{Type: 3, Value: " "}
	t1 := lexer.Token{Type: 2, Value: "blah"}
	tokens := []lexer.Token{t0, ts, t1}
	l, err := lexer.Upgrade(&staticLexer{tokens: tokens}, 3)
	require.NoError(t, err)
	require.Equal(t, t0, *l.Peek())
	require.Equal(t, t0, *l.Peek())
	require.Equal(t, tokens, l.Range(0, 3))
}

func TestUpgradePooled(t *testing.T) {
	t0 := lexer.Token{Type: 1, Value: "moo"}
	ts := lexer.Token{Type: 3, Value: " "}
	t1 := lexer.Token{Type: 2, Value: "blah"}
	tokens := []lexer.Token{t0, ts, t1}
	l, err := lexer.UpgradePooled(&staticLexer{tokens: tokens}, 3)
	defer lexer.PutBackPooledPeekingLexer(l)
	require.NoError(t, err)
	require.Equal(t, t0, *l.Peek())
	require.Equal(t, t0, *l.Peek())
	require.Equal(t, tokens, l.Range(0, 3))
}

func TestPeekingLexer_Peek_Next_Checkpoint(t *testing.T) {
	slexdef := lexer.MustSimple([]lexer.SimpleRule{
		{"Ident", `\w+`},
		{"Whitespace", `\s+`},
	})
	slex, err := slexdef.LexString("", `hello world last`)
	require.NoError(t, err)
	plex, err := lexer.Upgrade(slex, slexdef.Symbols()["Whitespace"])
	require.NoError(t, err)
	expected := []lexer.Token{
		{Type: -2, Value: "hello", Pos: lexer.Position{Line: 1, Column: 1}},
		{Type: -3, Value: " ", Pos: lexer.Position{Line: 1, Column: 6, Offset: 5}},
		{Type: -2, Value: "world", Pos: lexer.Position{Line: 1, Column: 7, Offset: 6}},
		{Type: -3, Value: " ", Pos: lexer.Position{Line: 1, Column: 12, Offset: 11}},
		{Type: -2, Value: "last", Pos: lexer.Position{Line: 1, Column: 13, Offset: 12}},
	}
	checkpoint := plex.Checkpoint
	require.Equal(t, expected[0], *plex.Next())
	require.Equal(t, expected[2], *plex.Peek(), "should have skipped whitespace")
	plex.Checkpoint = checkpoint
	require.Equal(t, expected[0], *plex.Peek(), "should have reverted to pre-Next state")
}

func BenchmarkPeekingLexer_Peek(b *testing.B) {
	tokens := []lexer.Token{{Type: 1, Value: "x"}, {Type: 3, Value: " "}, {Type: 2, Value: "y"}}
	l, err := lexer.Upgrade(&staticLexer{tokens: tokens}, 3)
	require.NoError(b, err)
	l.Next()
	t := l.Peek()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t = l.Peek()
		if t.EOF() {
			return
		}
	}
	require.Equal(b, lexer.Token{Type: 2, Value: "y"}, *t)
}

func generateTokenArray(sz int) []lexer.Token {
	tokens := make([]lexer.Token, sz)
	for i := range tokens {
		tokens[i].Type = lexer.TokenType(rand.Int() % 3)
		tokens[i].Value = string([]byte{byte('a' + (rand.Int() % 26))})
	}
	return tokens
}

func BenchmarkPeekingLexer_Upgrade(b *testing.B) {
	tokens := generateTokenArray(10000)
	testLexer := &staticLexer{tokens: tokens}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l, err := lexer.Upgrade(testLexer)
		require.NoError(b, err)
		l.Next()
		l.Peek()
	}
}

func BenchmarkPeekingLexer_UpgradePooled(b *testing.B) {
	tokens := generateTokenArray(10000)
	testLexer := &staticLexer{tokens: tokens}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l, err := lexer.UpgradePooled(testLexer)
		require.NoError(b, err)
		l.Next()
		l.Peek()
		lexer.PutBackPooledPeekingLexer(l)
	}
}
