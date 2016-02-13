// Production  = name "=" [ Expression ] "." .
// Expression  = Alternative { "|" Alternative } .
// Alternative = Term { Term } .
// Term        = name | token [ "…" token ] | Group | Option | Repetition .
// Group       = "(" Expression ")" .
// Option      = "[" Expression "]" .
// Repetition  = "{" Expression "}" .
package main

type EBNF struct {
	Productions []Production
}

type Production struct {
	Name       *Name       `@ "="`
	Expression *Expression `[ @ ] "."`
}

type Expression struct {
	Atlernatives []*Alternative `@ { "|" @ }`
}

type Alternative struct {
	Term Term
}

type Term struct {
	Name       *Name       `|`
	TokenRange *TokenRange `|`
	Group      *Group      `|`
	Option     *Option     `|`
	Repitition *Repitition
}

type Group struct {
	Expression *Expression `"(" @ ")"`
}

type Option struct {
	Expression *Expression `"[" @ "]"`
}

type Repitition struct {
	Expression *Expression `"{" @ "}"`
}

type TokenRange struct {
	Start *Token
	End   *Token ` [ "…" @ ]`
}

type Token struct {
	Token string `"\"" {"\\" . | .} "\""`
}

type Name struct {
	Name string `("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`
}
