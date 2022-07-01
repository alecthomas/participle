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
		Visibility string `@"public"?`
		Class      string `"class" @Ident`
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

	ast, err := p.ParseString("", `public class A;`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Decls: []*decl{
		{Class: &cls{Visibility: "public", Class: "A"}},
	}}, ast)
	_, err = p.ParseString("", `public union A;`)
	require.NoError(t, err)
	_, err = p.ParseString("", `public struct Bar;`)
	require.EqualError(t, err, `1:8: unexpected token "struct" (expected "union" <ident>)`)
	_, err = p.ParseString("", `public class 1;`)
	require.EqualError(t, err, `1:14: unexpected token "1" (expected <ident>)`)
}

func TestErrorWrap(t *testing.T) {
	expected := errors.New("badbad")
	err := participle.Wrapf(lexer.Position{Line: 1, Column: 1}, expected, "bad: %s", "thing")
	require.Equal(t, expected, errors.Unwrap(err))
	require.Equal(t, "1:1: bad: thing: badbad", err.Error())
}
