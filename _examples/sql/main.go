// nolint: govet
package main

import (
	"github.com/alecthomas/kong"
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
	"github.com/alecthomas/repr"
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
	Value         *Value      `| @@`
	SymbolRef     *SymbolRef  `| @@`
	SubExpression *Expression `| "(" @@ ")"`
}

type SymbolRef struct {
	Symbol     string        `@Ident @{ "." Ident }`
	Parameters []*Expression `[ "(" @@ { "," @@ } ")" ]`
}

type Value struct {
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

var (
	cli struct {
		SQL string `arg:"" required:"" help:"SQL to parse."`
	}

	sqlLexer = lexer.Must(ebnf.New(`
Comment = "--" { "\u0000"…"\uffff"-"\n" } .
Ident = (alpha | "_") { "_" | alpha | digit } .
String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
Number = [ "-" | "+" ] ("." | digit) {"." | digit} .
Punct = "!"…"/" | ":"…"@" | "["…`+"\"`\""+` | "{"…"~" .
Whitespace = " " | "\t" | "\n" | "\r" .

alpha = "a"…"z" | "A"…"Z" .
digit = "0"…"9" .
any = "\u0000"…"\uffff" .
	`, ebnf.Elide("Whitespace", "Comment")))
	sqlParser = participle.MustBuild(
		&Select{},
		participle.Lexer(sqlLexer),
		participle.Unquote("String"),
		participle.CaseInsensitive("Ident"),
		// Need to solve left recursion detection first, if possible.
		// participle.UseLookahead(),
	)
)

func main() {
	ctx := kong.Parse(&cli)
	sql := &Select{}
	err := sqlParser.ParseString(cli.SQL, sql)
	ctx.FatalIfErrorf(err)
	repr.Println(sql, repr.Indent("  "), repr.OmitEmpty(true))
}
