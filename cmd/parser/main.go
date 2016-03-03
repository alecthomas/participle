// Production  = name "=" [ Expression ] "." .
// Expression  = Alternative { "|" Alternative } .
// Alternative = Term { Term } .
// Term        = name | token [ "…" token ] | Group | Option | Repetition .
// Group       = "(" Expression ")" .
// Option      = "[" Expression "]" .
// Repetition  = "{" Expression "}" .
package main

import (
	"fmt"

	"github.com/alecthomas/parser"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Lexer struct {
	Identifier string      `("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`
	String     string      `"\"" {"\\" . | .} "\""`
	Whitespace parser.Skip `" " | "\t" | "\n" | "\r"`
}

type EBNF struct {
	Productions []*Production
}

type Production struct {
	Name       string      `@Identifier "="`
	Expression *Expression `[ @@ ] "."`
}

type Expression struct {
	Alternatives []*Alternative `@@ { "|" @@ }`
}

type Alternative struct {
	Term Term
}

type Term struct {
	Name       *string     `@Identifier |`
	TokenRange *TokenRange `@@ |`
	Group      *Group      `@@ |`
	Option     *Option     `@@ |`
	Repetition *Repetition
}

type Group struct {
	Expression *Expression `"(" @@ ")"`
}

type Option struct {
	Expression *Expression `"[" @@ "]"`
}

type Repetition struct {
	Expression *Expression `"{" @@ "}"`
}

type TokenRange struct {
	Start string  `@String` // Lexer token "String"
	End   *string ` [ "…" @String ]`
}

func main() {
	kingpin.Parse()
	p, err := parser.New(Lexer{}, EBNF{})
	kingpin.FatalIfError(err, "")
	i, err := p.ParseString(`a = "1" .`)
	kingpin.FatalIfError(err, "")
	ebnf := i.(EBNF)
	fmt.Println(ebnf)
}
