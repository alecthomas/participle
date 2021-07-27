package antlr

import (
	"testing"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
	"github.com/stretchr/testify/assert"
)

func TestLexableToRegex(t *testing.T) {
	tt := []struct {
		name   string
		code   string
		rule   string
		result string
		debug  bool
	}{
		{
			code: `
			grammar json;

			NUMBER
				: '-'? INT ('.' [0-9] +)? EXP?
				;

			fragment INT
				: '0' | [1-9] [0-9]*
				;
		 
			fragment EXP
				: [Ee] [+\-]? INT
				;
		 	`,
			rule:   "NUMBER",
			result: `-?(0|[1-9][0-9]*)(\.[0-9]+)?([Ee][+\-]?(0|[1-9][0-9]*))?`,
		},
		{
			code: `
			grammar json;

			fragment UNICODE
				: 'u' HEX HEX HEX HEX
				;

			fragment HEX
				: [0-9a-fA-F]
				;
		 	`,
			rule:   "UNICODE",
			result: `u[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]`,
		},
		{
			code: `
			grammar json;

			fragment ESC
				: '\\' (["\\/bfnrt] | UNICODE)
				;
			fragment UNICODE
			: 'u' HEX HEX HEX HEX
			;
			fragment HEX
			: [0-9a-fA-F]
			;
		 	`,
			rule:   "ESC",
			result: `\\(["\\/bfnrt]|u[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F])`,
		},
		{
			code: `
			grammar json;

			STRING
				: '"' (ESC | SAFECODEPOINT)* '"'
				;
			fragment ESC
				: '\\' (["\\/bfnrt] | UNICODE)
				;
			fragment UNICODE
				: 'u' HEX HEX HEX HEX
				;
			fragment HEX
				: [0-9a-fA-F]
				;
			fragment SAFECODEPOINT
				: ~ ["\\\u0000-\u001F]
				;
		 	`,
			rule:   "STRING",
			result: `"(\\(["\\/bfnrt]|u[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F])|[^"\\\x{0000}-\x{001F}])*"`,
		},
		{
			code: `
			grammar json;

			NOT_RANGE
				: ~ 'a'..'f'
				;
		 	`,
			rule:   "NOT_RANGE",
			result: `[^a-f]`,
		},
		{
			code: `
			grammar json;

			STRING:              'N'? '\'' (~'\'' | '\'\'')* '\'';
		 	`,
			rule:   "STRING",
			result: `N?'([^']|'')*'`,
		},
		{
			code: `
			grammar tsql;

			LOCAL_ID:           '@' ([A-Z_$@#0-9] | FullWidthLetter)*;

			fragment FullWidthLetter
			: '\u00c0'..'\u00d6'
			| '\u00d8'..'\u00f6'
			;
		 	`,
			rule:   "LOCAL_ID",
			result: `@([A-Z_$@#0-9]|([\x{00c0}-\x{00d6}]|[\x{00d8}-\x{00f6}]))*`,
		},
		{
			name: "negating a group",
			code: `
			grammar foo;

			Bar: ~( 'a' | 'b');
		 	`,
			rule:   "Bar",
			result: `[^ab]`,
		},
		{
			name: "negating a group with nested lexer tokens",
			code: `
			grammar foo;

			Bar: ( 'a' | 'b' );
			Baz: ~( 'c' | Bar );
		 	`,
			rule:   "Baz",
			result: `[^cab]`,
		},
		{
			name: "negating a group with escaped elements",
			code: `
			grammar foo;

			Bar: ~( '\\' | '"' | '\n' | '\r' ) | '\\' ~('\n' | '\r');
		 	`,
			rule:   "Bar",
			result: `([^\\"\n\r]|\\[^\n\r])`,
		},
		{
			name: "not excessive group nesting",
			code: `
			grammar foo;

			HexNumber: '0x' HexDigit+;
			fragment HexDigit: [a-f] | [A-F] | Digit;
			fragment Digit: [0-9];
		 	`,
			rule:   "HexNumber",
			result: `0x([a-f]|[A-F]|[0-9])+`,
		},
		{
			name: "group where necessary",
			code: `
			grammar foo;

			Abc: '@' A B C;
			fragment A: 'a' | 'A';
			fragment B: 'b' | 'B';
			fragment C: 'c' | 'C';
		 	`,
			rule:   "Abc",
			result: `@(a|A)(b|B)(c|C)`,
		},
		{
			name: "further nesting",
			code: `
			grammar foo;

			Abc: '@' A B C;
			fragment A: 'a' | 'A' | 'q' Zeros ('x'|'y') Newline;
			fragment B: 'b' | 'B';
			fragment C: 'c' | 'C';
			fragment Newline: '\r' | '\n';
			fragment Zeros: '0'? '0'?;
		 	`,
			rule:   "Abc",
			result: `@(a|A|q0?0?(x|y)(\r|\n))(b|B)(c|C)`,
		},
	}

	p := ast.MustBuildParser(&ast.AntlrFile{})
	for _, test := range tt {
		dst := &ast.AntlrFile{}
		if err := p.ParseString("", test.code, dst); err != nil {
			t.Fatal(err)
		}

		lrs := dst.LexRules()
		rm := map[string]*ast.LexerRule{}
		for _, lr := range lrs {
			rm[lr.Name] = lr
		}
		lv := NewLexerVisitor(rm)
		lv.debug = test.debug
		lexables := map[string]gen.LexerRule{}
		for _, v := range lrs {
			lexables[v.Name] = lv.Visit(v)
		}
		assert.Equal(t, test.result, lexables[test.rule].Content, "%s", test.rule)
	}
}
