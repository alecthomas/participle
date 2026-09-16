package participle_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type mappedIdentifiers struct {
	Values []string `@Ident*`
}

func mappedIdentifierLexer() lexer.Definition {
	return lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Ident", Pattern: `[a-z]+`},
		{Name: "Other", Pattern: `[0-9]+`},
	})
}

func TestMapOrderAndOriginalTokenType(t *testing.T) {
	definition := mappedIdentifierLexer()
	symbols := definition.Symbols()
	var calls []string
	parser := participle.MustBuild[mappedIdentifiers](
		participle.Lexer(definition),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			calls = append(calls, "specific")
			token.Value += "/specific"
			return token, nil
		}, "Ident"),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			if token.EOF() {
				return token, nil
			}
			calls = append(calls, "global")
			token.Type = symbols["Other"]
			token.Value += "/global"
			return token, nil
		}),
	)

	tokens, err := parser.Lex("", strings.NewReader("hello"))
	require.NoError(t, err)
	require.Equal(t, []string{"global", "specific"}, calls)
	require.Equal(t, "hello/global/specific", tokens[0].Value)
	require.Equal(t, symbols["Other"], tokens[0].Type)
}

func TestMapStopsAtFirstError(t *testing.T) {
	expected := errors.New("mapper failed")
	specificCalled := false
	parser := participle.MustBuild[mappedIdentifiers](
		participle.Lexer(mappedIdentifierLexer()),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			specificCalled = true
			return token, nil
		}, "Ident"),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			return token, expected
		}),
	)

	_, err := parser.Lex("", strings.NewReader("hello"))
	require.True(t, errors.Is(err, expected))
	require.False(t, specificCalled)
}

// A global mapper is also selected as the token-specific mapper for EOF, so
// preserve the existing behaviour of applying it twice to that token.
func TestMapEOFBehaviour(t *testing.T) {
	var eofCalls int
	parser := participle.MustBuild[mappedIdentifiers](
		participle.Lexer(mappedIdentifierLexer()),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			if token.EOF() {
				eofCalls++
			}
			return token, nil
		}),
	)

	_, err := parser.Lex("", strings.NewReader("hello"))
	require.NoError(t, err)
	require.Equal(t, 2, eofCalls)
}

// constantLexer deliberately does no scanning or allocation, isolating the
// mapping wrapper installed by Build.
type constantLexer struct{}

func (constantLexer) Next() (lexer.Token, error) {
	return lexer.Token{Type: -2, Value: "identifier"}, nil
}

type constantLexerDefinition struct{}

func (constantLexerDefinition) Symbols() map[string]lexer.TokenType {
	return map[string]lexer.TokenType{"Ident": -2}
}

func (constantLexerDefinition) Lex(string, io.Reader) (lexer.Lexer, error) {
	return constantLexer{}, nil
}

var mappedTokenSink lexer.Token

func BenchmarkMapper(b *testing.B) {
	identity := func(token lexer.Token) (lexer.Token, error) { return token, nil }
	parser := participle.MustBuild[mappedIdentifiers](
		participle.Lexer(constantLexerDefinition{}),
		participle.Map(identity),
		participle.Map(identity, "Ident"),
	)
	stream, err := parser.Lexer().Lex("", strings.NewReader(""))
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		mappedTokenSink, err = stream.Next()
		if err != nil {
			b.Fatal(err)
		}
	}
}
