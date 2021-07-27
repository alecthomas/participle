package antlr

import (
	"testing"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/stretchr/testify/assert"
)

func TestCheckOptional(t *testing.T) {
	tt := []struct {
		name string
		code string
	}{
		{
			name: "empty alternate",
			code: `
			principal_id:
			| id_
			| PUBLIC
			;`,
		},
		{
			name: "empty alternate middle",
			code: `
			rule
			: 'a'
			|
			| 'b'
			;
			`,
		},
		{
			name: "empty alternate last",
			code: `
			rule
			: 'a'
			| 'b'
			|
			;
			`,
		},
	}

	p := ast.MustBuildParser(&ast.ParserRule{})
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			dst := &ast.ParserRule{}
			if err := p.ParseString("", test.code, dst); err != nil {
				t.Fatal(err)
			}

			v := new(OptionalChecker)
			assert.True(t, v.RuleIsOptional(dst))
		})
	}
}
