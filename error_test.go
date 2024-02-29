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
	require.EqualError(t, err, `1:8: unexpected token "struct" (expected "union" <ident>)`)
	_, err = p.ParseString("", `public class 1;`)
	require.EqualError(t, err, `1:14: unexpected token "1" (expected <ident> ("(" <ident> ("," <ident>)+ ")")?)`)
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
	require.EqualError(t, err, `1:1: unexpected token "102"`)

	_, err = pAtLeastOnce.ParseString("", `102`)
	// ensure we don't get a "+1:1: sub-expression <string>+ must match at least once" error
	require.EqualError(t, err, `1:1: unexpected token "102"`)
}

func TestErrorWrap(t *testing.T) {
	expected := errors.New("badbad")
	err := participle.Wrapf(lexer.Position{Line: 1, Column: 1}, expected, "bad: %s", "thing")
	require.Equal(t, expected, errors.Unwrap(err))
	require.Equal(t, "1:1: bad: thing: badbad", err.Error())
}
