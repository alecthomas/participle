package stateful

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var interpolatedRules = Rules{
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
}

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
			tokens: []string{"\n\t\t\t\t", "<<END", "\n\t\t\t\t", "hello", " ", "world", "\n\t\t\t\t", "END", "\n\t\t\t"},
		},
		{name: "BackslashIsntABackRef",
			rules: Rules{
				"Root": {
					{"JustOne", `(\\\\1)`, Push("Convoluted")},
				},
				"Convoluted": {
					{"ConvolutedMatch", `\\\1`, nil},
				},
			},
			input:  `\\1\\\1`,
			tokens: []string{`\\1`, `\\\1`},
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
			tokens: []string{"\"", "hello ", "${", "user", " ", "+", " ", "\"", "??", "\"", " ", "+", " ", "\"", "${", "nested", "}", "\"", "}", "\""},
		},
		{name: "Return",
			rules: Rules{
				"Root": {
					{"Ident", `\w+`, Push("Reference")},
					{"whitespace", `\s+`, nil},
				},
				"Reference": {
					{"Dot", `\.`, nil},
					{"Ident", `\w+`, nil},
					Return(),
				},
			},
			input:  `hello.world `,
			tokens: []string{"hello", ".", "world"},
		},
		{name: "NoMatchNoMutatorError",
			rules: Rules{
				"Root": {
					{"NoMatch", "", nil},
				},
			},
			input: "hello",
			err:   "1:1: rule \"NoMatch\" did not match any input",
		},
		{name: "NoMatchPushError",
			rules: Rules{
				"Root": {
					{"NoMatch", "", Push("Sub")},
				},
				"Sub": {
					{"Ident", `\w+`, nil},
				},
			},
			input: "hello",
			err:   "1:1: rule \"NoMatch\": did not consume any input",
		},
	}
	// nolint: scopelint
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			def, err := New(test.rules)
			require.NoError(t, err)
			lex, err := def.Lex("", strings.NewReader(test.input))
			require.NoError(t, err)
			tokens, err := lexer.ConsumeAll(lex)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				actual := []string{}
				for _, token := range tokens {
					if token.EOF() {
						break
					}
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

	def, err := New(interpolatedRules)
	if err != nil {
		log.Fatal(err)
	}
	parser, err := participle.Build(&String{}, participle.Lexer(def))
	if err != nil {
		log.Fatal(err)
	}

	actual := &String{}
	err = parser.ParseString("", `"hello ${user + "??"}"`, actual)
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
	err = parser.ParseString("", `"hello ${user + "${last}"}"`, actual)
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
	err = parser.ParseString("", `
		<<END
		hello world
		END
	`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func BenchmarkStateful(b *testing.B) {
	source := strings.Repeat(`"hello ${user + "${last}"}"`, 100)
	def := lexer.Must(New(interpolatedRules))
	b.ReportMetric(float64(len(source)), "B")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lex, err := def.Lex("", strings.NewReader(source))
		if err != nil {
			b.Fatal(err)
		}
		tokens, err := lexer.ConsumeAll(lex)
		if err != nil {
			b.Fatal(err)
		}
		if len(tokens) != 1201 {
			b.Fatalf("%d != 1201", len(tokens))
		}
	}
}

func BenchmarkStatefulBackrefs(b *testing.B) {
	source := strings.Repeat(`
	<<END
	hello world
	END
`, 100)
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
	require.NoError(b, err)
	b.ReportMetric(float64(len(source)), "B")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lex, err := def.Lex("", strings.NewReader(source))
		if err != nil {
			b.Fatal(err)
		}
		tokens, err := lexer.ConsumeAll(lex)
		if err != nil {
			b.Fatal(err)
		}
		if len(tokens) != 401 {
			b.Fatalf("%d != 401", len(tokens))
		}
	}
}

func BenchmarkStatefulBasic(b *testing.B) {
	source := strings.Repeat(`
 5  REM inputting the argument
10  PRINT "Factorial of:"
20  INPUT A
30  LET B = 1
35  REM beginning of the loop
40  IF A <= 1 THEN 80
50  LET B = B * A
60  LET A = A - 1
70  GOTO 40
75  REM prints the result
80  PRINT B
	`, 100)
	def, err := NewSimple([]Rule{
		{"String", `"(\\"|[^"])*"`, nil},
		{"Number", `[-+]?(\d*\.)?\d+`, nil},
		{"Ident", `[a-zA-Z_]\w*`, nil},
		{"Punct", `[!-/:-@[-` + "`" + `{-~]+`, nil},
		{"EOL", `\n`, nil},
		{"comment", `(?i)rem[^\n]*\n`, nil},
		{"whitespace", `[ \t]+`, nil},
	})
	require.NoError(b, err)
	b.ReportMetric(float64(len(source)), "B")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lex, err := def.Lex("", strings.NewReader(source))
		if err != nil {
			b.Fatal(err)
		}
		tokens, err := lexer.ConsumeAll(lex)
		if err != nil {
			b.Fatal(err)
		}
		if len(tokens) != 6601 {
			b.Fatalf("%d != 401", len(tokens))
		}
	}
}

func TestZeroCopyBytesReader(t *testing.T) {
	s := []byte("hello")
	b := bytes.NewReader(s)
	def := MustSimple([]Rule{{"Ident", `\w+`, nil}})
	l, err := def.Lex("", b)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%p", s), fmt.Sprintf("%p", l.(*Lexer).data))
}

func TestZeroCopyBytesBuffer(t *testing.T) {
	s := []byte("hello")
	b := bytes.NewBuffer(s)
	def := MustSimple([]Rule{{"Ident", `\w+`, nil}})
	l, err := def.Lex("", b)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%p", s), fmt.Sprintf("%p", l.(*Lexer).data))
}
