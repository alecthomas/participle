package antlr

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

func TestComputedLexerBody(t *testing.T) {
	tt := []struct {
		name   string
		code   string
		result string
	}{
		{
			name: "shortest literals should sort to the top",
			code: `grammar Test; cmd: ABC | FG; ABC: 'de'; FG: 'hijk';`,
			result: fmt.Sprintf(
				"{%s, %s, nil},\n{%s, %s, nil},\n",
				`"ABC"`, "`de`",
				`"FG"`, "`hijk`",
			),
		},
		{
			name: "additional lexer elements from literals in parser rules",
			code: `grammar Test; STRING: '"' ~'"'* '"'; cmd: '+' STRING;`,
			result: fmt.Sprintf(
				"{%s, %s, nil},\n{%s, %s, nil},\n",
				`"STRING"`, "`\"[^\"]*\"`",
				`"XXX__LITERAL_Plus"`, "`\\+`",
			),
		},
		{
			name: "additional lexer elements from undeclared lexer tokens in parser rules",
			code: `grammar Test; STRING: '"' ~'"'* '"'; cmd: PLUS STRING;`,
			result: fmt.Sprintf(
				"{%s, %s, nil},\n{%s, %s, nil},\n",
				`"STRING"`, "`\"[^\"]*\"`",
				`"PLUS"`, "`PLUS`",
			),
		},
	}

	p := ast.MustBuildParser(&ast.AntlrFile{})
	for _, test := range tt {
		dst := &ast.AntlrFile{}
		if err := p.ParseString("", test.code, dst); err != nil {
			t.Fatal(err)
		}

		lexRules, _, _, err := compute(dst)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, test.result, lexRules)
	}
}

func TestComputedParseObjects(t *testing.T) {
	tt := []struct {
		name   string
		code   string
		result string
	}{
		{
			name: "shortest literals should sort to the top",
			code: `grammar foo;

				bar : baz (',' baz)* '\r'? '\n' ;

				baz
					: 'a'
					| 'b'
					|
					;`,
			result: "type Bar struct {\nBaz []*Baz `@@? ( ',' @@? )* '\\r'? '\\n'`\n}\ntype Baz struct {\nAB *string `@( 'a' | 'b' )`\n}",
		},
	}

	p := ast.MustBuildParser(&ast.AntlrFile{})
	for _, test := range tt {
		dst := &ast.AntlrFile{}
		if err := p.ParseString("", test.code, dst); err != nil {
			t.Fatal(err)
		}

		_, parseObjs, _, err := compute(dst)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, test.result, parseObjs)
	}

}

func TestConvertWholeGrammar(t *testing.T) {
	tt := []struct {
		grammar string
		rules   string
		structs string
		root    string
	}{
		{
			grammar: "json",
			root:    "json",
		},
	}

	p := ast.MustBuildParser(&ast.AntlrFile{})
	for _, test := range tt {
		b, err := ioutil.ReadFile("./testdata/" + test.grammar + ".g4")
		if err != nil {
			t.Fatal(err)
		}

		dst := &ast.AntlrFile{}
		if err := p.ParseBytes(test.grammar, b, dst); err != nil {
			t.Fatal(err)
		}

		lexRules, parseObjs, root, err := compute(dst)
		if err != nil {
			t.Fatal(err)
		}

		g := goldie.New(t)
		g.Assert(t, test.grammar+"-rules", []byte(lexRules))
		g.Assert(t, test.grammar+"-structs", []byte(parseObjs))
		assert.Equal(t, test.root, root.Name)
	}
}
