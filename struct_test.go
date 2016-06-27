package parser

import (
	"reflect"
	"testing"
	"text/scanner"

	"github.com/alecthomas/assert"
)

func TestStructLexerTokens(t *testing.T) {
	type testScanner struct {
		A string `12`
		B string `34`
	}

	scan := lexStruct(reflect.TypeOf(testScanner{}))
	t12 := Token{scanner.Int, "12"}
	t34 := Token{scanner.Int, "34"}
	assert.Equal(t, t12, scan.Peek())
	assert.Equal(t, 0, scan.field)
	assert.Equal(t, t12, scan.Next())

	assert.Equal(t, t34, scan.Peek())
	assert.Equal(t, 0, scan.field)
	assert.Equal(t, t34, scan.Next())
	assert.Equal(t, 1, scan.field)

	assert.Equal(t, EOFToken, scan.Next())
}

func TestStructLexer(t *testing.T) {
	g := struct {
		A string `"a"|`
		B string `"b"`
	}{}

	gt := reflect.TypeOf(g)
	r := lexStruct(gt)
	f := []reflect.StructField{}
	s := ""
	for {
		r.Peek()
		rn := r.Next()
		if rn.EOF() {
			break
		}
		f = append(f, r.Field())
		s += string(rn.String())
	}
	assert.Equal(t, `a|b`, s)
	f0 := gt.Field(0)
	f1 := gt.Field(1)
	assert.Equal(t, []reflect.StructField{f0, f0, f1}, f)
}
