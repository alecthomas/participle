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

	scan, err := lexStruct(reflect.TypeOf(testScanner{}))
	require.NoError(t, err)
	t12 := lexer.Token{Type: scanner.Int, Value: "12", Pos: lexer.Position{Line: 1, Column: 1}}
	t34 := lexer.Token{Type: scanner.Int, Value: "34", Pos: lexer.Position{Line: 2, Column: 1}}
	require.Equal(t, t12, mustPeek(scan))
	require.Equal(t, 0, scan.field)
	require.Equal(t, t12, mustNext(scan))

	require.Equal(t, t34, mustPeek(scan))
	require.Equal(t, 0, scan.field)
	require.Equal(t, t34, mustNext(scan))
	require.Equal(t, 1, scan.field)

	require.True(t, mustNext(scan).EOF())
}

func TestStructLexer(t *testing.T) {
	g := struct {
		A string `"a"|`
		B string `"b"`
	}{}

	gt := reflect.TypeOf(g)
	r, err := lexStruct(gt)
	require.NoError(t, err)
	f := []structLexerField{}
	s := ""
	for {
		_, err := r.Peek()
		require.NoError(t, err)
		rn, err := r.Next()
		require.NoError(t, err)
		if rn.EOF() {
			break
		}
		f = append(f, r.Field())
		s += rn.String()
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
	indexes, err := collectFieldIndexes(typ)
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 0}, {0, 1}, {1}}, indexes)
}

func mustPeek(scan *structLexer) lexer.Token {
	token, err := scan.Peek()
	if err != nil {
		panic(err)
	}
	return token
}

func mustNext(scan *structLexer) lexer.Token { // nolint: interfacer
	token, err := scan.Next()
	if err != nil {
		panic(err)
	}
	return token
}
