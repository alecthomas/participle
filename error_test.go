package participle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alecthomas/participle/v2"
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
	assert.EqualError(t, err, `1:8: unexpected token "struct" (expected "union")`)
	err = p.ParseString("", `public class 1;`, ast)
	assert.EqualError(t, err, `1:14: unexpected token "1" (expected <ident>)`)
}
