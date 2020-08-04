package main

import (
	"log"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

type Terminal struct {
	String *String `  @@`
	Ident  string  `| @ExprIdent`
}

type Expr struct {
	Left  *Terminal `@@`
	Op    string    `( @ExprOper`
	Right *Terminal `  @@)?`
}

type Fragment struct {
	Escaped string `(  @StringEscaped`
	Expr    *Expr  ` | "${" @@ "}"`
	Text    string ` | @StringChar)`
}

type String struct {
	Fragments []*Fragment `"\"" @@* "\""`
}

var (
	def = lexer.Must(stateful.New(stateful.Rules{
		"Root": {
			{`String`, `"`, stateful.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"End", `"`, stateful.Pop()},
			{"Expr", `\${`, stateful.Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			stateful.Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"End", `}`, stateful.Pop()},
		},
	}))
	parser = participle.MustBuild(&String{}, participle.Lexer(def),
		participle.Elide("ExprWhitespace"))
)

func main() {
	actual := &String{}
	err := parser.ParseString(`"hello ${first + "${last}"}"`, actual)
	if err != nil {
		log.Fatal(err)
	}
	repr.Println(actual)
}
