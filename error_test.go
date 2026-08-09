package participle_test

import (
	"errors"
	"testing"

	require "github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestErrorReporting(t *testing.T) {
	type cls struct {
		Visibility string   `@"public"?`
		Class      string   `"class" @Ident`
		Bases      []string `('(' @Ident (',' @Ident)+ ')')?`
	}
	type union struct {
		Visibility string `@"public"?`
		Union      string `"union" @Ident`
	}
	type decl struct {
		Class *cls   `(  @@`
		Union *union ` | @@ )`
	}
	type grammar struct {
		Decls []*decl `( @@ ";" )*`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(5))

	ast, err := p.ParseString("", `public class A(B, C); class D; public union A;`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Decls: []*decl{
		{Class: &cls{Visibility: "public", Class: "A", Bases: []string{"B", "C"}}},
		{Class: &cls{Class: "D"}},
		{Union: &union{Visibility: "public", Union: "A"}},
	}}, ast)

	_, err = p.ParseString("", `public struct Bar;`)
	require.EqualError(t, err, `1:8: unexpected token "struct" of type Ident (expected "union" <ident>)`)
	_, err = p.ParseString("", `public class 1;`)
	require.EqualError(t, err, `1:14: unexpected token "1" of type Int (expected <ident> ("(" <ident> ("," <ident>)+ ")")?)`)
	_, err = p.ParseString("", `public class A(B,C,);`)
	require.EqualError(t, err, `1:20: unexpected token ")" (expected <ident>)`)
}

func TestMoreThanOneErrors(t *testing.T) {
	type unionMatchAtLeastOnce struct {
		Ident  string  `( @Ident `
		String string  `| @String+ `
		Float  float64 `| @Float )`
	}
	type union struct {
		Ident  string  `( @Ident `
		String string  `| @String `
		Float  float64 `| @Float )`
	}

	pAtLeastOnce := mustTestParser[unionMatchAtLeastOnce](t, participle.Unquote("String"))
	p := mustTestParser[union](t, participle.Unquote("String"))

	ast, err := pAtLeastOnce.ParseString("", `"a string" "two strings"`)
	require.NoError(t, err)
	require.Equal(t, &unionMatchAtLeastOnce{String: "a stringtwo strings"}, ast)

	_, err = p.ParseString("", `102`)
	require.EqualError(t, err, `1:1: unexpected token "102" of type Int`)

	_, err = pAtLeastOnce.ParseString("", `102`)
	// ensure we don't get a "+1:1: sub-expression <string>+ must match at least once" error
	require.EqualError(t, err, `1:1: unexpected token "102" of type Int`)
}

func TestErrorWrap(t *testing.T) {
	expected := errors.New("badbad")
	err := participle.Wrapf(lexer.Position{Line: 1, Column: 1}, expected, "bad: %s", "thing")
	require.Equal(t, expected, errors.Unwrap(err))
	require.Equal(t, "1:1: bad: thing: badbad", err.Error())
}

// TestUnexpectedTokenErrorReportsType verifies that unexpected token errors
// include the symbolic name of the token type, per #265. A keyword that
// overlaps an identifier is easy to mistake for a valid token.
func TestUnexpectedTokenErrorReportsTokenType(t *testing.T) {
	lex := lexer.MustStateful(lexer.Rules{
		"Root": {
			{"whitespace", `\s+`, nil},
			{"Keyword", `(group|for|func)\b`, nil},
			{"Ident", `[a-zA-Z]\w*`, nil},
			{"Punct", `\+`, nil},
		},
	})
	type expression struct {
		LHS string `parser:"@Ident"`
		Op  string `parser:"@'+'"`
		RHS string `parser:"@Ident"`
	}
	p := mustTestParser[expression](t, participle.Lexer(lex), participle.Elide("whitespace"))

	_, err := p.ParseString("", `group + two`)
	require.EqualError(t, err, `1:1: unexpected token "group" of type Keyword`)

	_, err = p.ParseString("", `two + group`)
	require.EqualError(t, err, `1:7: unexpected token "group" of type Keyword (expected <ident>)`)
}

// TestUnexpectedTokenErrorWithoutSymbolicName verifies that tokens without a
// symbolic name (such as literals) do not include a token type in the error.
func TestUnexpectedTokenErrorWithoutTokenType(t *testing.T) {
	type grammar struct {
		Value string `@('.' | ',')`
	}
	p := mustTestParser[grammar](t)

	_, err := p.ParseString("", `-`)
	require.EqualError(t, err, `1:1: unexpected token "-"`)
}

// TestUnexpectedTokenErrorForParseable verifies that the token type is
// reported for parsers using the Parseable interface (NextMatch path).
func TestUnexpectedTokenErrorForParseable(t *testing.T) {
	p := mustTestParser[parseableReject](t)

	_, err := p.ParseString("", `hello`)
	require.EqualError(t, err, `1:1: unexpected token "hello" of type Ident`)
}

// parseableReject always rejects input via NextMatch.
type parseableReject struct{}

func (*parseableReject) Parse(_ *lexer.PeekingLexer) error { return participle.NextMatch }
