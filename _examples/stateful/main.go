package main

import (
	"log"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
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
	def = lexer.MustStateful(lexer.Rules{
		"Root": {
			{`String`, `"`, lexer.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, lexer.Pop()},
			{"Expr", `\${`, lexer.Push("Expr")},
			{"Char", `\$|[^$"\\]+`, nil},
		},
		"Expr": {
			lexer.Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, lexer.Pop()},
		},
	})
	parser = participle.MustBuild[String](participle.Lexer(def),
		participle.Elide("Whitespace"))
)

func main() {
	actual, err := parser.ParseString("", `"hello $(world) ${first + "${last}"}"`)
	repr.Println(actual)
	if err != nil {
		log.Fatal(err)
	}
}
