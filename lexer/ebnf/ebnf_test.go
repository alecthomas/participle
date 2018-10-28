package ebnf

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/lexer"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	type entry struct {
		options []Option
		source  string
		tokens  []string
		roots   []string
		fail    bool
	}
	tests := []struct {
		name      string
		grammar   string
		cases     []entry
		failBuild bool
	}{
		{
			name:      "BadEBNF",
			grammar:   "Production = helper .",
			failBuild: true,
		},
		{
			name:    "EmptyProductionErrorsWithInput",
			grammar: `Extra = .`,
			cases: []entry{{
				source: "a",
				fail:   true,
			}},
		},
		{
			name:    "ExtraInputErrors",
			grammar: `Extra = "b" .`,
			cases: []entry{{
				source: "ba",
				fail:   true,
			}},
		},
		{
			name:    "TokenMatch",
			grammar: `Token = "token" .`,
			cases: []entry{{
				source: `token`,
				tokens: []string{"token"},
				roots:  []string{"Token"},
			}},
		},
		{
			name:    "TokenNoMatch",
			grammar: `Token = "token" .`,
			cases: []entry{{
				source: `toke`,
				fail:   true,
			}},
		},
		{
			name:    "RangeMatch",
			grammar: `Range = "a" … "z" .`,
			cases: []entry{{
				source: "x",
				tokens: []string{"x"},
			}},
		},
		{
			name:    "RangeNoMatch",
			grammar: `Range = "a" … "z" .`,
			cases: []entry{{
				source: "A",
				fail:   true,
			}},
		},
		{
			name:    "Alternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			cases: []entry{{
				source: "a",
				tokens: []string{"a"},
			}},
		},
		{
			name:    "2ndAlternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			cases: []entry{{
				source: "b",
				tokens: []string{"b"},
			}},
		},
		{
			name:    "3rdAlternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			cases: []entry{{
				source: "c",
				tokens: []string{"c"},
			}},
		},
		{
			name:    "AlternativeDoesNotMatch",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			cases: []entry{{
				source: "d",
				fail:   true,
			}},
		},
		{
			name:    "Group",
			grammar: `Group = ("token") .`,
			cases: []entry{{
				source: "token",
				tokens: []string{"token"},
			}},
		},
		{
			name:    "OptionWithInnerMatch",
			grammar: `Option = [ "t" ] .`,
			cases: []entry{{
				source: "t",
				tokens: []string{"t"},
			}},
		},
		{
			name:    "OptionWithNoInnerMatch",
			grammar: `Option = [ "t" ] .`,
			cases: []entry{{
				source: "",
			}},
		},
		{
			name:    "RangeWithExclusion",
			grammar: `Option = "a"…"z"-"f"…"g"-"z"-"y" .`,
			cases: []entry{{
				source: "y",
				fail:   true,
			}},
		},
		{
			name: "Ident",
			grammar: `
			Identifier = alpha { alpha | number } .
			Whitespace = "\n" | "\r" | "\t" | " " .

			alpha = "a"…"z" | "A"…"Z" | "_" .
			number = "0"…"9" .
			`,
			cases: []entry{{
				source: `some id withCase andNumb3rs a`,
				tokens: []string{"some", " ", "id", " ", "withCase", " ", "andNumb3rs", " ", "a"},
			}},
		},
		{
			name: "Rewind",
			grammar: `
			Comment = "//" .
			Operator = "/" .
			Whitespace = " " .
			`,
			cases: []entry{{
				source: "//",
				tokens: []string{"//"},
			}, {
				source: "/ /",
				tokens: []string{"/", " ", "/"},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, entry := range test.cases {
				defi, err := New(test.grammar, entry.options...)
				if test.failBuild {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				def := defi.(*ebnfLexerDefinition)
				if entry.roots != nil {
					roots := []string{}
					for sym := range def.symbols {
						if sym != "EOF" {
							roots = append(roots, sym)
						}
					}
					require.Equal(t, entry.roots, roots)
				}
				lexer, err := def.Lex(strings.NewReader(entry.source))
				require.NoError(t, err)
				tokens, err := readAllTokens(lexer)
				if entry.fail {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, entry.tokens, tokens)
			}
		})
	}
}

func readAllTokens(lex lexer.Lexer) (out []string, err error) {
	for {
		token, err := lex.Next()
		if err != nil {
			return nil, err
		}
		if token.EOF() {
			return out, nil
		}
		out = append(out, token.Value)
	}
}

func BenchmarkEBNFLexer(b *testing.B) {
	b.ReportAllocs()
	def, err := New(`
Identifier = alpha { alpha | digit } .
Whitespace = "\n" | "\r" | "\t" | " " .
Number = digit { digit } .

alpha = "a"…"z" | "A"…"Z" | "_" .
digit = "0"…"9" .
`)
	require.NoError(b, err)
	r := strings.NewReader(strings.Repeat("hello world 123 hello world 123", 100))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex, _ := def.Lex(r)
		for {
			token, _ := lex.Next()
			if token.Type == lexer.EOF {
				break
			}
		}
		_, _ = r.Seek(0, 0)
	}
}
