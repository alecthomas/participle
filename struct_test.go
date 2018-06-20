package participle

import (
	"reflect"
	"testing"
	"text/scanner"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/lexer"
)

func TestStructLexerTokens(t *testing.T) {
	type testScanner struct {
		A string `12`
		B string `34`
	}

	scan := lexStruct(reflect.TypeOf(testScanner{}))
	t12 := lexer.Token{Type: scanner.Int, Value: "12", Pos: lexer.Position{Line: 1, Column: 1}}
	t34 := lexer.Token{Type: scanner.Int, Value: "34", Pos: lexer.Position{Line: 2, Column: 1}}
	require.Equal(t, t12, scan.Peek())
	require.Equal(t, 0, scan.field)
	require.Equal(t, t12, scan.Next())

	require.Equal(t, t34, scan.Peek())
	require.Equal(t, 0, scan.field)
	require.Equal(t, t34, scan.Next())
	require.Equal(t, 1, scan.field)

	require.True(t, scan.Next().EOF())
}

func TestStructLexer(t *testing.T) {
	g := struct {
		A string `"a"|`
		B string `"b"`
	}{}

	gt := reflect.TypeOf(g)
	r := lexStruct(gt)
	f := []structLexerField{}
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
	require.Equal(t, `a|b`, s)
	f0 := r.GetField(0)
	f1 := r.GetField(1)
	require.Equal(t, []structLexerField{f0, f0, f1}, f)
}

type testEmbeddedIndexes struct {
	A string `@String`
	B string `@String`
}

func TestCollectFieldIndexes(t *testing.T) {
	var grammar struct {
		testEmbeddedIndexes
		C string `@String`
	}
	typ := reflect.TypeOf(grammar)
	indexes := collectFieldIndexes(typ)
	require.Equal(t, [][]int{{0, 0}, {0, 1}, {1}}, indexes)
}
