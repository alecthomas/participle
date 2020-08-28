package stateful

import (
	"log"
	"strings"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

func TestStatefulLexer(t *testing.T) {
	tests := []struct {
		name   string
		rules  Rules
		input  string
		tokens []string
		err    string
	}{
		{name: "BackrefNoGroups",
			input: `hello`,
			err:   `1:1: rule "Backref": invalid backref expansion: "\\1": invalid group 1 from parent with 0 groups`,
			rules: Rules{"Root": {{"Backref", `\1`, nil}}},
		},
		{name: "BackrefInvalidGroups",
			input: `<<EOF EOF`,
			err:   "1:6: rule \"End\": invalid backref expansion: \"\\\\b\\\\2\\\\b\": invalid group 2 from parent with 2 groups",
			rules: Rules{
				"Root": {
					{"Heredoc", `<<(\w+)\b`, Push("Heredoc")},
				},
				"Heredoc": {
					{"End", `\b\2\b`, Pop()},
				},
			},
		},
		{name: "Heredoc",
			rules: Rules{
				"Root": {
					{"Heredoc", `<<(\w+\b)`, Push("Heredoc")},
					Include("Common"),
				},
				"Heredoc": {
					{"End", `\b\1\b`, Pop()},
					Include("Common"),
				},
				"Common": {
					{"Whitespace", `\s+`, nil},
					{"Ident", `\w+`, nil},
				},
			},
			input: `
				<<END
				hello world
				END
			`,
			tokens: []string{"\n\t\t\t\t", "<<END", "\n\t\t\t\t", "hello", " ", "world", "\n\t\t\t\t", "END", "\n\t\t\t", ""},
		},
		{name: "Recursive",
			rules: Rules{
				"Root": {
					{`String`, `"`, Push("String")},
				},
				"String": {
					{"Escaped", `\\.`, nil},
					{"StringEnd", `"`, Pop()},
					{"Expr", `\${`, Push("Expr")},
					{"Char", `[^$"\\]+`, nil},
				},
				"Expr": {
					Include("Root"),
					{`Whitespace`, `\s+`, nil},
					{`Oper`, `[-+/*%]`, nil},
					{"Ident", `\w+`, nil},
					{"ExprEnd", `}`, Pop()},
				},
			},
			input:  `"hello ${user + "??" + "${nested}"}"`,
			tokens: []string{"\"", "hello ", "${", "user", " ", "+", " ", "\"", "??", "\"", " ", "+", " ", "\"", "${", "nested", "}", "\"", "}", "\"", ""},
		},
	}
	// nolint: scopelint
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			def, err := New(test.rules)
			require.NoError(t, err)
			lex, err := def.Lex(strings.NewReader(test.input))
			require.NoError(t, err)
			tokens, err := lexer.ConsumeAll(lex)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				actual := []string{}
				for _, token := range tokens {
					actual = append(actual, token.Value)
				}
				require.Equal(t, test.tokens, actual)
			}
		})
	}
}

// An example of parsing nested expressions within strings.
func ExampleNew() {
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

	def, err := New(Rules{
		"Root": {
			{`String`, `"`, Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, Pop()},
			{"Expr", `\${`, Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			Include("Root"),
			{`whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, Pop()},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	parser, err := participle.Build(&String{}, participle.Lexer(def))
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
	Escaped string `(  @Escaped`
	Expr    *Expr  ` | "${" @@ "}"`
	Text    string ` | @Char)`
}

type Expr struct {
	Left  *Terminal `@@`
	Op    string    `( @Oper`
	Right *Terminal `  @@)?`
}

type Terminal struct {
	String *String `  @@`
	Ident  string  `| @Ident`
}

func TestStateful(t *testing.T) {
	def, err := New(Rules{
		"Root": {
			{`String`, `"`, Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, Pop()},
			{"Expr", `\${`, Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			Include("Root"),
			{`whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, Pop()},
		},
	})
	require.NoError(t, err)
	parser, err := participle.Build(&String{}, participle.Lexer(def))
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

func TestHereDoc(t *testing.T) {
	type Heredoc struct {
		Idents []string `Heredoc @Ident* End`
	}

	type AST struct {
		Doc *Heredoc `@@`
	}

	def, err := New(Rules{
		"Root": {
			{"Heredoc", `<<(\w+\b)`, Push("Heredoc")},
			Include("Common"),
		},
		"Heredoc": {
			{"End", `\b\1\b`, Pop()},
			Include("Common"),
		},
		"Common": {
			{"whitespace", `\s+`, nil},
			{"Ident", `\w+`, nil},
		},
	})
	require.NoError(t, err)
	parser, err := participle.Build(&AST{},
		participle.Lexer(def),
	)
	require.NoError(t, err)

	expected := &AST{
		Doc: &Heredoc{
			Idents: []string{"hello", "world"},
		},
	}
	actual := &AST{}
	err = parser.ParseString(`
		<<END
		hello world
		END
	`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
