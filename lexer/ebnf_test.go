package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name      string
		grammar   string
		source    string
		tokens    []string
		roots     []string
		failBuild bool
		fail      bool
	}{
		{
			name:      "BadEBNF",
			grammar:   "Production = helper .",
			failBuild: true,
		},
		{
			name:    "EmptyProductionErrorsWithInput",
			grammar: `Extra = .`,
			source:  "a",
			fail:    true,
		},
		{
			name:    "ExtraInputErrors",
			grammar: `Extra = "b" .`,
			source:  "ba",
			tokens:  []string{"b"},
			fail:    true,
		},
		{
			name:    "TokenMatch",
			grammar: `Token = "token" .`,
			source:  `token`,
			tokens:  []string{"token"},
			roots:   []string{"Token"},
		},
		{
			name:    "TokenNoMatch",
			grammar: `Token = "token" .`,
			source:  `toke`,
			fail:    true,
		},
		{
			name:    "RangeMatch",
			grammar: `Range = "a" … "z" .`,
			source:  "x",
			tokens:  []string{"x"},
		},
		{
			name:    "RangeNoMatch",
			grammar: `Range = "a" … "z" .`,
			source:  "A",
			fail:    true,
		},
		{
			name:    "Alternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			source:  "a",
			tokens:  []string{"a"},
		},
		{
			name:    "2ndAlternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			source:  "b",
			tokens:  []string{"b"},
		},
		{
			name:    "3rdAlternative",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			source:  "c",
			tokens:  []string{"c"},
		},
		{
			name:    "AlternativeDoesNotMatch",
			grammar: `Alternatives = "a" | "b" | "c" .`,
			source:  "d",
			fail:    true,
		},
		{
			name:    "Group",
			grammar: `Group = ("token") .`,
			source:  "token",
			tokens:  []string{"token"},
		},
		{
			name:    "OptionWithInnerMatch",
			grammar: `Option = [ "t" ] .`,
			source:  "t",
			tokens:  []string{"t"},
		},
		{
			name:    "OptionWithNoInnerMatch",
			grammar: `Option = [ "t" ] .`,
			source:  "",
		},
		{
			name: "Identifier",
			grammar: `
			Identifier = alpha { alpha | number } .
			Whitespace = "\n" | "\r" | "\t" | " " .

			alpha = "a"…"z" | "A"…"Z" | "_" .
			number = "0"…"9" .
			`,
			source: `some id withCase andNumb3rs a`,
			tokens: []string{"some", " ", "id", " ", "withCase", " ", "andNumb3rs", " ", "a"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defi, err := EBNF(test.grammar)
			if test.failBuild {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			def := defi.(*ebnfLexerDefinition)
			if test.roots != nil {
				roots := []string{}
				for sym := range def.symbols {
					if sym != "EOF" {
						roots = append(roots, sym)
					}
				}
				require.Equal(t, test.roots, roots)
			}
			// repr.Println(def, repr.Indent("  "))
			tokens, err := readAllTokens(def.Lex(strings.NewReader(test.source)))
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.tokens, tokens)
		})
	}
}

func readAllTokens(lex Lexer) (out []string, err error) {
	defer func() {
		if msg, ok := recover().(error); ok {
			err = msg
		}
	}()
	for {
		token := lex.Next()
		if token.EOF() {
			return
		}
		out = append(out, token.Value)
	}
}

func BenchmarkEBNFLexer(b *testing.B) {
	b.ReportAllocs()
	def, err := EBNF(`
Identifier = alpha { alpha | digit } .
Whitespace = "\n" | "\r" | "\t" | " " .
Number = digit { digit } .

alpha = "a"…"z" | "A"…"Z" | "_" .
digit = "0"…"9" .
`)
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := def.Lex(strings.NewReader("hello world 123 hello world 123"))
		ConsumeAll(lex)
	}
}
