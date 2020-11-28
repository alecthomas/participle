package main

import (
	"log"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer/stateful"
)

type Terminal struct {
	String *String `  @@`
	Ident  string  `| @Ident`
}

type Expr struct {
	Left  *Terminal `@@`
	Op    string    `( @Oper`
	Right *Terminal `  @@)?`
}

type Fragment struct {
	Escaped string `(  @Escaped`
	Expr    *Expr  ` | "${" @@ "}"`
	Text    string ` | @Char)`
}

type String struct {
	Fragments []*Fragment `"\"" @@* "\""`
}

var (
	def = stateful.Must(stateful.Rules{
		"Root": {
			{`String`, `"`, stateful.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, stateful.Pop()},
			{"Expr", `\${`, stateful.Push("Expr")},
			{"Char", `\$|[^$"\\]+`, nil},
		},
		"Expr": {
			stateful.Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, stateful.Pop()},
		},
	})
	parser = participle.MustBuild(&String{}, participle.Lexer(def),
		participle.Elide("Whitespace"))
)

func main() {
	actual := &String{}
	err := parser.ParseString("", `"hello $(world) ${first + "${last}"}"`, actual)
	if err != nil {
		log.Fatal(err)
	}
	repr.Println(actual)
}
