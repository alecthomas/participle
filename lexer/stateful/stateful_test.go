package stateful

import (
	"log"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle"
)

// An example of parsing nested expressions within strings.
func ExampleNew() {
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

	def, err := New(Rules{
		"Root": {
			{`String`, `"`, Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"End", `"`, Pop()},
			{"Expr", `\${`, Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"End", `}`, Pop()},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	parser, err := participle.Build(&String{}, participle.Lexer(def),
		participle.Elide("ExprWhitespace"))
	if err != nil {
		log.Fatal(err)
	}

	actual := &String{}
	err = parser.ParseString(`"hello ${user + "??"}"`, actual)
	if err != nil {
		log.Fatal(err)
	}
	repr.Println(actual)
}

type String struct {
	Fragments []*Fragment `"\"" @@* "\""`
}

type Fragment struct {
	Escaped string `(  @StringEscaped`
	Expr    *Expr  ` | "${" @@ "}"`
	Text    string ` | @StringChar)`
}

type Expr struct {
	Left  *Terminal `@@`
	Op    string    `( @ExprOper`
	Right *Terminal `  @@)?`
}

type Terminal struct {
	String *String `  @@`
	Ident  string  `| @ExprIdent`
}

func TestStateful(t *testing.T) {
	def, err := New(Rules{
		"Root": {
			{`String`, `"`, Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"End", `"`, Pop()},
			{"Expr", `\${`, Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"End", `}`, Pop()},
		},
	})
	require.NoError(t, err)
	parser, err := participle.Build(&String{}, participle.Lexer(def),
		participle.Elide("ExprWhitespace"))
	require.NoError(t, err)

	actual := &String{}
	err = parser.ParseString(`"hello ${user + "${last}"}"`, actual)
	require.NoError(t, err)
	expected := &String{
		Fragments: []*Fragment{
			{Text: "hello "},
			{Expr: &Expr{
				Left: &Terminal{Ident: "user"},
				Op:   "+",
				Right: &Terminal{
					String: &String{
						Fragments: []*Fragment{{
							Expr: &Expr{
								Left: &Terminal{Ident: "last"},
							},
						}},
					},
				},
			}},
		},
	}
	require.Equal(t, expected, actual)
}
