package participle_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))

	var err error
	ast := &grammar{}
	err = p.ParseString("", `public class A;`, ast)
	assert.NoError(t, err)
	err = p.ParseString("", `public union A;`, ast)
	assert.NoError(t, err)
	err = p.ParseString("", `public struct Bar;`, ast)
	assert.EqualError(t, err, `1:8: unexpected token "struct" (expected "union" <ident>)`)
	err = p.ParseString("", `public class 1;`, ast)
	assert.EqualError(t, err, `1:14: unexpected token "1" (expected <ident>)`)
}

func TestErrorWrap(t *testing.T) {
	expected := errors.New("badbad")
	err := participle.Wrapf(lexer.Position{Line: 1, Column: 1}, expected, "bad: %d", 10)
	require.Equal(t, expected, errors.Unwrap(err))
	require.Equal(t, "1:1: bad: 10: badbad", err.Error())
}

func TestAnnotateError(t *testing.T) {
	orig := errors.New("an error")
	err := participle.AnnotateError(lexer.Position{Line: 1, Column: 1}, orig)
	require.Equal(t, "1:1: an error", err.Error())
}
