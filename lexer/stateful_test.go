package lexer_test

import (
	"encoding/json"
	"log"
	"strings"
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/internal"
	"github.com/alecthomas/repr"
)

var interpolatedRules = lexer.Rules{
	"Root": {
		{`String`, `"`, lexer.Push("String")},
	},
	"String": {
		{"Escaped", `\\.`, nil},
		{"StringEnd", `"`, lexer.Pop()},
		{"Expr", `\${`, lexer.Push("Expr")},
		{"Char", `[^$"\\]+`, nil},
	},
	"Expr": {
		lexer.Include("Root"),
		{`whitespace`, `\s+`, nil},
		{`Oper`, `[-+/*%]`, nil},
		{"Ident", `\w+`, nil},
		{"ExprEnd", `}`, lexer.Pop()},
	},
}

func TestMarshalUnmarshal(t *testing.T) {
	data, err := json.MarshalIndent(interpolatedRules, "", "  ")
	require.NoError(t, err)
	unmarshalledRules := lexer.Rules{}
	err = json.Unmarshal(data, &unmarshalledRules)
	require.NoError(t, err)
	require.Equal(t, interpolatedRules, unmarshalledRules)
}

func TestStatefulLexer(t *testing.T) {
	tests := []struct {
		name   string
		rules  lexer.Rules
		lngst  bool
		input  string
		tokens []string
		err    string
	}{
		{name: "BackrefNoGroups",
			input: `hello`,
			err:   `1:1: rule "Backref": invalid backref expansion: "\\1": invalid group 1 from parent with 0 groups`,
			rules: lexer.Rules{"Root": {{"Backref", `\1`, nil}}},
		},
		{name: "BackrefInvalidGroups",
			input: `<<EOF EOF`,
			err:   "1:6: rule \"End\": invalid backref expansion: \"\\\\b\\\\2\\\\b\": invalid group 2 from parent with 2 groups",
			rules: lexer.Rules{
				"Root": {
					{"Heredoc", `<<(\w+)\b`, lexer.Push("Heredoc")},
				},
				"Heredoc": {
					{"End", `\b\2\b`, lexer.Pop()},
				},
			},
		},
		{name: "Heredoc",
			rules: lexer.Rules{
				"Root": {
					{"Heredoc", `<<(\w+\b)`, lexer.Push("Heredoc")},
					lexer.Include("Common"),
				},
				"Heredoc": {
					{"End", `\b\1\b`, lexer.Pop()},
					lexer.Include("Common"),
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
			rules: lexer.Rules{
				"Root": {
					{"JustOne", `(\\\\1)`, lexer.Push("Convoluted")},
				},
				"Convoluted": {
					{"ConvolutedMatch", `\\\1`, nil},
				},
			},
			input:  `\\1\\\1`,
			tokens: []string{`\\1`, `\\\1`},
		},
		{name: "Recursive",
			rules: lexer.Rules{
				"Root": {
					{`String`, `"`, lexer.Push("String")},
				},
				"String": {
					{"Escaped", `\\.`, nil},
					{"StringEnd", `"`, lexer.Pop()},
					{"Expr", `\${`, lexer.Push("Expr")},
					{"Char", `[^$"\\]+`, nil},
				},
				"Expr": {
					lexer.Include("Root"),
					{`Whitespace`, `\s+`, nil},
					{`Oper`, `[-+/*%]`, nil},
					{"Ident", `\w+`, nil},
					{"ExprEnd", `}`, lexer.Pop()},
				},
			},
			input:  `"hello ${user + "??" + "${nested}"}"`,
			tokens: []string{"\"", "hello ", "${", "user", " ", "+", " ", "\"", "??", "\"", " ", "+", " ", "\"", "${", "nested", "}", "\"", "}", "\""},
		},
		{name: "Return",
			rules: lexer.Rules{
				"Root": {
					{"Ident", `\w+`, lexer.Push("Reference")},
					{"whitespace", `\s+`, nil},
				},
				"Reference": {
					{"Dot", `\.`, nil},
					{"Ident", `\w+`, nil},
					lexer.Return(),
				},
			},
			input:  `hello.world `,
			tokens: []string{"hello", ".", "world"},
		},
		{name: "NoMatchLongest",
			rules: lexer.Rules{
				"Root": {
					{"A", `a`, nil},
					{"Ident", `\w+`, nil},
					{"whitespace", `\s+`, nil},
				},
			},
			input:  `a apple`,
			tokens: []string{"a", "a", "pple"},
		},
		{name: "MatchLongest",
			rules: lexer.Rules{
				"Root": {
					{"A", `a`, nil},
					{"Ident", `\w+`, nil},
					{"whitespace", `\s+`, nil},
				},
			},
			lngst:  true,
			input:  `a apple`,
			tokens: []string{"a", "apple"},
		},
		{name: "NoMatchNoMutatorError",
			rules: lexer.Rules{
				"Root": {
					{"NoMatch", "", nil},
				},
			},
			input: "hello",
			err:   "1:1: rule \"NoMatch\" did not match any input",
		},
		{name: "NoMatchPushError",
			rules: lexer.Rules{
				"Root": {
					{"NoMatch", "", lexer.Push("Sub")},
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
			var opts []lexer.Option
			if test.lngst {
				opts = append(opts, lexer.MatchLongest())
			}
			def, err := lexer.New(test.rules, opts...)
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

	def, err := lexer.New(interpolatedRules)
	if err != nil {
		log.Fatal(err)
	}
	parser, err := participle.Build[String](participle.Lexer(def))
	if err != nil {
		log.Fatal(err)
	}

	actual, err := parser.ParseString("", `"hello ${user + "??"}"`)
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
	def, err := lexer.New(lexer.Rules{
		"Root": {
			{`String`, `"`, lexer.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, lexer.Pop()},
			{"Expr", `\${`, lexer.Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			lexer.Include("Root"),
			{`whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, lexer.Pop()},
		},
	})
	require.NoError(t, err)
	parser, err := participle.Build[String](participle.Lexer(def))
	require.NoError(t, err)

	actual, err := parser.ParseString("", `"hello ${user + "${last}"}"`)
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

	def, err := lexer.New(lexer.Rules{
		"Root": {
			{"Heredoc", `<<(\w+\b)`, lexer.Push("Heredoc")},
			lexer.Include("Common"),
		},
		"Heredoc": {
			{"End", `\b\1\b`, lexer.Pop()},
			lexer.Include("Common"),
		},
		"Common": {
			{"whitespace", `\s+`, nil},
			{"Ident", `\w+`, nil},
		},
	})
	require.NoError(t, err)
	parser, err := participle.Build[AST](participle.Lexer(def))
	require.NoError(t, err)

	expected := &AST{
		Doc: &Heredoc{
			Idents: []string{"hello", "world"},
		},
	}
	actual, err := parser.ParseString("", `
		<<END
		hello world
		END
	`)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func BenchmarkStateful(b *testing.B) {
	source := strings.Repeat(`"hello ${user + "${last}"}"`, 100)
	def := lexer.Must(lexer.New(interpolatedRules))
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
	def, err := lexer.New(lexer.Rules{
		"Root": {
			{"Heredoc", `<<(\w+\b)`, lexer.Push("Heredoc")},
			lexer.Include("Common"),
		},
		"Heredoc": {
			{"End", `\b\1\b`, lexer.Pop()},
			lexer.Include("Common"),
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

func basicBenchmark(b *testing.B, def lexer.Definition) {
	b.Helper()
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
	b.ReportMetric(float64(len(source)), "B")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex, err := def.(lexer.StringDefinition).LexString("", source)
		if err != nil {
			b.Fatal(err)
		}
		count := 0
		var token lexer.Token
		for !token.EOF() {
			token, err = lex.Next()
			if err != nil {
				b.Fatal(err)
			}
			count++
		}
		if count != 11101 {
			b.Fatalf("%d != 6601", count)
		}
	}
}

func BenchmarkStatefulBASIC(b *testing.B) {
	def, err := lexer.New(lexer.Rules{"Root": []lexer.Rule{
		{"String", `"(\\"|[^"])*"`, nil},
		{"Number", `[-+]?(\d*\.)?\d+`, nil},
		{"Ident", `[a-zA-Z_]\w*`, nil},
		{"Punct", `[!-/:-@[-` + "`" + `{-~]+`, nil},
		{"EOL", `\n`, nil},
		{"Comment", `(?i)rem[^\n]*\n`, nil},
		{"Whitespace", `[ \t]+`, nil},
	}})
	require.NoError(b, err)
	basicBenchmark(b, def)
}

func BenchmarkStatefulGeneratedBASIC(b *testing.B) {
	basicBenchmark(b, internal.GeneratedBasicLexer)
}
