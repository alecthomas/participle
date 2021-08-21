package main

import (
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
)

type Generic struct {
	Params []string `"<" (@Ident ","?)+ ">" (?= ("(" | ")" | "]" | ":" | ";" | "," | "." | "?" | "=" "=" | "!" "="))`
}

type Call struct {
	Params []*Expr `( @@ ","?)*`
}

type Terminal struct {
	Ident  string `  @Ident`
	Number int    `| @Int`
	Sub    *Expr  `| "(" @@ ")"`
}

type Expr struct {
	Terminal *Terminal `@@`

	Generic *Generic `( @@`
	RHS     *RHS     `  | @@ )?`

	Call      *Call `(   "(" @@ ")"`
	Reference *Expr `  | "." @@ )?`
}

type RHS struct {
	Oper string `@("<" | ">" | "=" "=" | "!" "=" | "+" | "-" | "*" | "/")`
	RHS  *Expr  `@@`
}

var parser = participle.MustBuild[Expr](participle.UseLookahead(1024))

func main() {
	expr, err := parser.ParseString("", "hello < world * (1 + 3)")
	if err != nil {
		panic(err)
	}
	repr.Println(expr)
	expr, err = parser.ParseString("", "type<int, string>.method(1, 2, 3)")
	if err != nil {
		panic(err)
	}
	repr.Println(expr)
}
