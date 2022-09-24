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

type lexerTestCase struct {
	name string
	def  lexer.Definition
}

func lexerTestCases(reflectionDef lexer.Definition, generatedDef lexer.Definition) []lexerTestCase {
	var cases []lexerTestCase
	if reflectionDef != nil {
		cases = append(cases, lexerTestCase{name: "Reflection", def: reflectionDef})
	}
	if generatedDef != nil {
		cases = append(cases, lexerTestCase{name: "Generated", def: generatedDef})
	}
	return cases
}

func runLexerTestCases(cases []lexerTestCase, t *testing.T, f func(t *testing.T, def lexer.Definition)) {
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			f(t, test.def)
		})
	}
}

func runLexerBenchmarkCases(cases []lexerTestCase, b *testing.B, f func(b *testing.B, def lexer.Definition)) {
	b.Helper()
	for _, test := range cases {
		b.Run(test.name, func(b *testing.B) {
			b.Helper()
			b.ResetTimer()
			f(b, test.def)
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	data, err := json.MarshalIndent(internal.InterpolatedRules, "", "  ")
	require.NoError(t, err)
	unmarshalledRules := lexer.Rules{}
	err = json.Unmarshal(data, &unmarshalledRules)
	require.NoError(t, err)
	require.Equal(t, internal.InterpolatedRules, unmarshalledRules)
}

func TestStatefulLexer(t *testing.T) {
	tests := []struct {
		name   string
		rules  lexer.Rules
		gendef lexer.Definition
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
			rules: internal.HeredocWithWhitespaceRules,
			// TODO: support backreferences in generated lexer
			//gendef: internal.GeneratedHeredocWithWhitespaceLexer,
			input: `
				<<END
				hello world
				END
			`,
			tokens: []string{"\n\t\t\t\t", "<<END", "\n\t\t\t\t", "hello", " ", "world", "\n\t\t\t\t", "END", "\n\t\t\t"},
		},
		{name: "BackslashIsntABackRef",
			// TODO: support backreferences in generated lexer
			//gendef: internal.GeneratedHeredocWithWhitespaceLexer,
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
			rules:  internal.InterpolatedWithWhitespaceRules,
			gendef: internal.GeneratedInterpolatedWithWhitespaceLexer,
			input:  `"hello ${user + "??" + "${nested}"}"`,
			tokens: []string{"\"", "hello ", "${", "user", " ", "+", " ", "\"", "??", "\"", " ", "+", " ", "\"", "${", "nested", "}", "\"", "}", "\""},
		},
		{name: "Return",
			rules:  internal.ReferenceRules,
			gendef: internal.GeneratedReferenceLexer,
			input:  `hello.world `,
			tokens: []string{"hello", ".", "world"},
		},
		{name: "NoMatchLongest",
			rules:  internal.ARules,
			gendef: internal.GeneratedALexer,
			input:  `a apple`,
			tokens: []string{"a", "a", "pple"},
		},
		{name: "MatchLongest",
			rules:  internal.ARules,
			lngst:  true,
			input:  `a apple`,
			tokens: []string{"a", "apple"},
		},
		{name: "NoMatchNoMutatorError",
			// TODO: support error handling in generated lexer
			rules: lexer.Rules{
				"Root": {
					{"NoMatch", "", nil},
				},
			},
			input: "hello",
			err:   "1:1: rule \"NoMatch\" did not match any input",
		},
		{name: "NoMatchPushError",
			// TODO: support error handling in generated lexer
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
			cases := lexerTestCases(def, test.gendef)
			runLexerTestCases(cases, t, func(t *testing.T, def lexer.Definition) {
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

	def, err := lexer.New(internal.InterpolatedRules)
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
	cases := lexerTestCases(lexer.MustStateful(internal.InterpolatedRules), internal.GeneratedInterpolatedLexer)
	runLexerTestCases(cases, t, func(t *testing.T, def lexer.Definition) {
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
	})
}

func TestHereDoc(t *testing.T) {
	type Heredoc struct {
		Idents []string `Heredoc @Ident* End`
	}

	type AST struct {
		Doc *Heredoc `@@`
	}

	// TODO: add support for backreferences to generated lexer
	//cases := lexerTestCases(lexer.MustStateful(internal.HeredocRules), internal.GeneratedHeredocLexer)
	cases := lexerTestCases(lexer.MustStateful(internal.HeredocRules), nil)
	runLexerTestCases(cases, t, func(t *testing.T, def lexer.Definition) {
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
	})
}

func BenchmarkStateful(b *testing.B) {
	source := strings.Repeat(`"hello ${user + "${last}"}"`, 100)
	cases := lexerTestCases(lexer.MustStateful(internal.InterpolatedRules), internal.GeneratedInterpolatedLexer)
	runLexerBenchmarkCases(cases, b, func(b *testing.B, def lexer.Definition) {
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
	})
}

func BenchmarkStatefulBackrefs(b *testing.B) {
	source := strings.Repeat(`
	<<END
	hello world
	END
`, 100)
	// TODO: add support for backreferences to generated lexer
	//cases := lexerTestCases(lexer.MustStateful(internal.HeredocRules), internal.GeneratedHeredocLexer)
	cases := lexerTestCases(lexer.MustStateful(internal.HeredocRules), nil)
	runLexerBenchmarkCases(cases, b, func(b *testing.B, def lexer.Definition) {
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
	})
}

func BenchmarkStatefulBASIC(b *testing.B) {
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
	cases := lexerTestCases(lexer.MustStateful(internal.BasicRules), internal.GeneratedBasicLexer)
	runLexerBenchmarkCases(cases, b, func(b *testing.B, def lexer.Definition) {
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
	})
}
