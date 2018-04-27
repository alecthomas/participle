package main

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

var (
	sqlArg = kingpin.Arg("sql", "SQL to parse.").Required().String()

	sqlLexer = lexer.Unquote(lexer.Upper(lexer.Must(lexer.Regexp(`(\s+)`+
		`|(?P<Keyword>(?i)SELECT|FROM|TOP|DISTINCT|ALL|WHERE|GROUP|BY|HAVING|UNION|MINUS|EXCEPT|INTERSECT|ORDER|LIMIT|OFFSET|TRUE|FALSE|NULL|IS|NOT|ANY|SOME|BETWEEN|AND|OR|LIKE|AS|IN)`+
		`|(?P<Ident>[a-zA-Z_][a-zA-Z0-9_]*)`+
		`|(?P<Number>[-+]?\d*\.?\d+([eE][-+]?\d+)?)`+
		`|(?P<String>'[^']*'|"[^"]*")`+
		`|(?P<Operators><>|!=|<=|>=|[-+*/%,.()=<>])`,
	)), "Keyword"), "String")
	sqlParser = participle.MustBuild(&Expression{}, sqlLexer)
)

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "TRUE"
	return nil
}

// Select based on http://www.h2database.com/html/grammar.html
type Select struct {
	Top        *Term             `"SELECT" [ "TOP" @@ ]`
	Distinct   bool              `[  @"DISTINCT"`
	All        bool              ` | @"ALL" ]`
	Expression *SelectExpression `@@`
	From       *From             `"FROM" @@`
	Limit      *Expression       `[ "LIMIT" @@ ]`
	Offset     *Expression       `[ "OFFSET" @@ ]`
	GroupBy    *Expression       `[ "GROUP" "BY" @@ ]`
}

type From struct {
	TableExpressions []*TableExpression `@@ { "," @@ }`
	Where            *Expression        `[ "WHERE" @@ ]`
}

type TableExpression struct {
	Table  string        `( @Ident { "." @Ident }`
	Select *Select       `  | "(" @@ ")"`
	Values []*Expression `  | "VALUES" "(" @@ { "," @@ } ")")`
	As     string        `[ "AS" @Ident ]`
}

type SelectExpression struct {
	All         bool                 `  @"*"`
	Expressions []*AliasedExpression `| @@ { "," @@ }`
}

type AliasedExpression struct {
	Expression *Expression `@@`
	As         string      `[ "AS" @Ident ]`
}

type Expression struct {
	And *AndCondition `@@ { "OR" @@ }`
}

type AndCondition struct {
	Or []*Condition `@@ { "AND" @@ }`
}

type Condition struct {
	Operand *ConditionOperand `  @@`
	Not     *Condition        `| "NOT" @@`
	Exists  *Select           `| "EXISTS" "(" @@ ")"`
}

type ConditionOperand struct {
	Operand      *Operand      `@@`
	ConditionRHS *ConditionRHS `[ @@ ]`
}

type ConditionRHS struct {
	Compare *Compare `  @@`
	Is      *Is      `| "IS" @@`
	Between *Between `| "BETWEEN" @@`
	In      *In      `| "IN" "(" @@ ")"`
	Like    *Like    `| "LIKE" @@`
}

type Compare struct {
	Operator string         `@( "<>" | "<=" | ">=" | "=" | "<" | ">" | "!=" )`
	Operand  *Operand       `(  @@`
	Select   *CompareSelect ` | @@ )`
}

type CompareSelect struct {
	All    bool    `(  @"ALL"`
	Any    bool    ` | @"ANY"`
	Some   bool    ` | @"SOME" )`
	Select *Select `"(" @@ ")"`
}

type Like struct {
	Not     bool     `[ @"NOT" ]`
	Operand *Operand `@@`
}

type Is struct {
	Not          bool     `[ @"NOT" ]`
	Null         bool     `( @"NULL"`
	DistinctFrom *Operand `  | "DISTINCT" "FROM" @@ )`
}

type Between struct {
	Start *Operand `@@`
	End   *Operand `"AND" @@`
}

type In struct {
	Select      *Select       `  @@`
	Expressions []*Expression `| @@ { "," @@ }`
}

type Operand struct {
	Summand []*Summand `@@ { "|" "|" @@ }`
}

type Summand struct {
	LHS *Factor `@@`
	Op  string  `[ @("+" | "-")`
	RHS *Factor `  @@ ]`
}

type Factor struct {
	LHS *Term  `@@`
	Op  string `[ @("*" | "/" | "%")`
	RHS *Term  `  @@ ]`
}

type Term struct {
	Select        *Select     `  @@`
	SymbolRef     *SymbolRef  `| @@`
	Value         *Value      `| @@`
	SubExpression *Expression `| "(" @@ ")"`
}

type SymbolRef struct {
	Symbol     string        `@Ident @{ "." Ident }`
	Parameters []*Expression `[ "(" @@ { "," @@ } ")" ]`
}

type Value struct {
	Negated bool `[ @"-" | "+" ]`

	Wildcard bool     `(  @"*"`
	Number   *float64 ` | @Number`
	String   *string  ` | @String`
	Boolean  *Boolean ` | @("TRUE" | "FALSE")`
	Null     bool     ` | @"NULL"`
	Array    *Array   ` | @@ )`
}

type Array struct {
	Expressions []*Expression `"(" @@ { "," @@ } ")"`
}

func main() {
	kingpin.Parse()
	sql := &Expression{}
	err := sqlParser.ParseString(*sqlArg, sql)
	kingpin.FatalIfError(err, "")
	repr.Println(sql, repr.Indent("  "), repr.OmitEmpty(true))
}
