package participle_test

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"

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

	var err error
	_, err = p.ParseString("", `public class A;`)
	assert.NoError(t, err)
	_, err = p.ParseString("", `public union A;`)
	assert.NoError(t, err)
	_, err = p.ParseString("", `public struct Bar;`)
	assert.Equal(t, err.Error(), `1:8: unexpected token "struct" (expected "union" <ident>)`)
	_, err = p.ParseString("", `public class 1;`)
	assert.Equal(t, err.Error(), `1:14: unexpected token "1" (expected <ident>)`)
}

func TestErrorWrap(t *testing.T) {
	expected := errors.New("badbad")
	err := participle.Wrapf(lexer.Position{Line: 1, Column: 1}, expected, "bad: %d", 10)
	assert.Equal(t, expected, errors.Unwrap(err))
	assert.Equal(t, "1:1: bad: 10: badbad", err.Error())
}

func TestAnnotateError(t *testing.T) {
	orig := errors.New("an error")
	err := participle.AnnotateError(lexer.Position{Line: 1, Column: 1}, orig)
	assert.Equal(t, "1:1: an error", err.Error())
}
