package antlr

import (
	"testing"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
	"github.com/stretchr/testify/assert"
)

func TestStructGenFromParserRule(t *testing.T) {
	tt := []struct {
		name   string
		code   string
		result string
		debug  bool
	}{
		{
			name:   "labeled",
			code:   `rule: a='b';`,
			result: "type Rule struct {\nA bool `@'b'`\n}",
		},
		{
			name:   "lexer rule match",
			code:   `rule: DIGIT;`,
			result: "type Rule struct {\nDigit *string `@DIGIT`\n}",
		},
		{
			name:   "lexer rule or",
			code:   `rule: DIGIT | OTHER;`,
			result: "type Rule struct {\nDigit *string `@DIGIT`\nOther *string `| @OTHER`\n}",
		},
		{
			name: "lexer references",
			code: `
			xml_common_directives
			: ',' (BINARY_BASE64 | TYPE | ROOT ('(' STRING ')')?)
			;`,
			result: "type XmlCommonDirectives struct {\nBinaryBase64 *string `',' ( @BINARY_BASE64`\nType *string `| @TYPE`\nRoot *string `| @ROOT`\nString *string `( '(' @STRING ')' )? )`\n}",
		},
		{
			name: "parser references",
			code: `
			optimize_for_arg
			: LOCAL_ID (UNKNOWN | '=' (constant | NULL_))
			;`,
			result: "type OptimizeForArg struct {\nLocalId *string `@LOCAL_ID`\nUnknown *string `( @UNKNOWN`\nConstant *Constant `| '=' ( @@`\nNull *string `| @NULL_ ) )`\n}",
		},
		{
			name: "some groups",
			code: `
			asterisk
			: (table_name '.')? '*'
			| (INSERTED | DELETED) '.' '*'
			;`,
			result: "type Asterisk struct {\nTableName *TableName `( @@ '.' )? '*'`\nInserted *string `| ( @INSERTED`\nDeleted *string `| @DELETED ) '.' '*'`\n}",
		},
		{
			name: "labeled group",
			code: `
			expression
			: primitive_expression
			| expression op=('*' | '/' | '%') expr
			;`,
			result: "type Expression struct {\nPrimitiveExpression *PrimitiveExpression `@@`\nExpression *Expression `| @@`\nOp *string `@( '*' | '/' | '%' )`\nExpr *Expr `@@`\n}",
		},
		{
			name: "boolean-captured top-level literals",
			code: `
			value
			: STRING
			| NUMBER
			| obj
			| arr
			| 'true'
			| 'false'
			| 'null'
			;`,
			result: "type Value struct {\nString *string `@STRING`\nNumber *string `| @NUMBER`\nObj *Obj `| @@`\nArr *Arr `| @@`\nTrue bool `| @'true'`\nFalse bool `| @'false'`\nNull bool `| @'null'`\n}",
		},
		{
			name:   "grouped boolean-captured top-level literals",
			code:   `value: 'a' | 'b' 'c';`,
			result: "type Value struct {\nABC *string `@( 'a' | 'b' 'c' )`\n}",
		},
		{
			name: "merge adjacent",
			code: `
			obj: '{' pair (',' pair)* '}';`,
			result: "type Obj struct {\nPair []*Pair `'{' @@ ( ',' @@ )* '}'`\n}",
		},
		{
			name: "merge adjacent",
			code: `
			rule: apple (',' apple)* 'a';`,
			result: "type Rule struct {\nApple []*Apple `@@ ( ',' @@ )* 'a'`\n}",
		},
		{
			name: "don't merge across alternatives",
			code: `
			rule
			: 'a' apple+
			| 'b' apple
			;`,
			result: "type Rule struct {\nApple []*Apple `'a' @@+`\nApple2 *Apple `| 'b' @@`\n}",
		},
		{
			name: "zero or more",
			code: `
			obj: '{' pair* '}';`,
			result: "type Obj struct {\nPair []*Pair `'{' @@* '}'`\n}",
		},
		{
			name: "one or more",
			code: `
			obj: '{' pair+ '}';`,
			result: "type Obj struct {\nPair []*Pair `'{' @@+ '}'`\n}",
		},
		{
			name: "zero or more group",
			code: `
			obj: '{' ( 'a' pair )* '}';`,
			result: "type Obj struct {\nPair []*Pair `'{' ( 'a' @@ )* '}'`\n}",
		},
		{
			name: "one or more group",
			code: `
			obj: '{' ( 'a' pair )+ '}';`,
			result: "type Obj struct {\nPair []*Pair `'{' ( 'a' @@ )+ '}'`\n}",
		},
		{
			name: "various; real-world example",
			code: `
			obj
			: '{' pair (',' pair)* '}'
			| '{' '}'
			;`,
			result: "type Obj struct {\nPair []*Pair `'{' @@ ( ',' @@ )* '}' | '{' '}'`\n}",
		},
		{
			name: "don't merge captures with different labels",
			code: `
			expression_elem
 			: leftAlias=column_alias eq='=' leftAssignment=expression
 			| expressionAs=expression as_column_alias?
 			;`,
			result: "type ExpressionElem struct {\nLeftAlias *ColumnAlias `@@`\nEq bool `@'='`\nLeftAssignment *Expression `@@`\nExpressionAs *Expression `| @@`\nAsColumnAlias *AsColumnAlias `@@?`\n}",
		},
		{
			name: "negation",
			code: `
			thing
			: ~TOKEN
			| ~';'
			;`,
			result: "type Thing struct {\nNotToken *string `@!TOKEN`\nNotSemi *string `| @!';'`\n}",
		},
		{
			name: "nested struct definition",
			code: `
			rule
			: THING argument=(DECIMAL | STRING | LOCAL_ID)*
			;`,
			result: "type Rule struct {\nThing *string `@THING`\nArgument []*struct{\n\tDecimal *string `@DECIMAL`\n\tString *string `| @STRING`\n\tLocalId *string `| @LOCAL_ID`\n} `@@*`\n}",
		},
		{
			name: "multi-nested struct definition",
			code: `
			raiseerror_statement
			: RAISERROR DECIMAL (',' argument=(DECIMAL | STRING | LOCAL_ID))*
			;`,
			result: "type RaiseerrorStatement struct {\nRaiserror *string `@RAISERROR`\nDecimal *string `@DECIMAL`\nDecimalStringLocalId []*struct{\n\tArgument *string `',' @(`\n\tDecimal *string `@DECIMAL`\n\tString *string `| @STRING`\n\tLocalId *string `| @LOCAL_ID )`\n} `@@*`\n}",
		},
		{
			name: "orphan literals preceding nested struct definition",
			code: `
			create_or_alter_event_session: '-' (DECIMAL|STRING)* ;`,
			result: "type CreateOrAlterEventSession struct {\nDecimalString []*struct{\n\tDecimal *string `@DECIMAL`\n\tString *string `| @STRING`\n} `'-' @@*`\n}",
		},
		{
			name: "empty alternate",
			code: `
			principal_id:
			| id_
			| PUBLIC
			;`,
			result: "type PrincipalId struct {\nId *Id `@@`\nPublic *string `| @PUBLIC`\n}",
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
			result: "type Rule struct {\nAB *string `@( 'a' | 'b' )`\n}",
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
			result: "type Rule struct {\nAB *string `@( 'a' | 'b' )`\n}",
		},
		{
			name: "optional rule is optional",
			code: `
			rule
			: FOO
			| 'a' abracadabra
			| BAR
			;`,
			result: "type Rule struct {\nFoo *string `@FOO`\nAbracadabra *Abracadabra `| 'a' @@?`\nBar *string `| @BAR`\n}",
		},
		{
			name: "optional rule is optional inside a multi-group",
			code: `
			rule
			: FOO
			| ( 'a' abracadabra )*
			| BAR
			;`,
			result: "type Rule struct {\nFoo *string `@FOO`\nAbracadabra []*Abracadabra `| ( 'a' @@? )*`\nBar *string `| @BAR`\n}",
		},
		{
			name: "optional rule is optional when merged",
			code: `
			rule
			: 'a' abracadabra+ abracadabra
			;`,
			result: "type Rule struct {\nAbracadabra []*Abracadabra `'a' @@* @@?`\n}",
		},
		{
			name: "optional rule is optional when merged, even in subgroup",
			code: `
			rule
			: abracadabra (',' abracadabra)* 'a'
			;`,
			result: "type Rule struct {\nAbracadabra []*Abracadabra `@@? ( ',' @@? )* 'a'`\n}",
		},
		{
			name:   "duplicate field name",
			code:   `rule: FOO thing | BAR thing;`,
			result: "type Rule struct {\nFoo *string `@FOO`\nThing *Thing `@@`\nBar *string `| @BAR`\nThing2 *Thing `@@`\n}",
		},
		{
			name: "simple top-level literal match",
			code: `
			sign
			: '+'
			| '-'
			;`,
			result: "type Sign struct {\nPlusMinus *string `@( '+' | '-' )`\n}",
		},
		{
			name: "sub-struct tag data should not trail onto the next field",
			code: `
			rule
			: (A B)+
			| C
			;
			`,
			result: "type Rule struct {\nAB []*struct{\n\tA *string `@A`\n\tB *string `@B`\n} `@@+`\nC *string `| @C`\n}",
		},
		{
			name: "don't leave behind tag data after a sub-struct",
			code: `
			rule
			: ( C
			| (A B)+ )
			;
			`,
			result: "type Rule struct {\nC *string `( @C`\nAB []*struct{\n\tA *string `@A`\n\tB *string `@B`\n} `| @@+ )`\n}",
		},
		{
			name: "capturing multiple lexer rules should result in a string slice",
			code: `
			rule: '#' (~THING)* THING;
			`,
			result: "type Rule struct {\nNotThing []*string `'#' ( @!THING )*`\nThing *string `@THING`\n}",
		},
	}

	p := ast.MustBuildParser(&ast.ParserRule{})
	for _, test := range tt {
		// t.Run(test.name, func(t *testing.T) {
		dst := &ast.ParserRule{}
		if err := p.ParseString("", test.code, dst); err != nil {
			t.Fatal(err)
		}

		v := NewStructVisitor(map[string]bool{
			"abracadabra": true,
		}, map[string]struct{}{})
		v.debug = test.debug
		v.Visit(dst)

		new(gen.FieldRenamer).VisitStruct(v.Result)

		assert.Equal(t, test.result, new(gen.Printer).Visit(v.Result), test.name)
		// })
	}
}
