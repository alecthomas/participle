package lexer

import (
	"regexp"
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
			source: `some id withCase andNumb3rs`,
			tokens: []string{"some", " ", "id", " ", "withCase", " ", "andNumb3rs"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defi, err := Build(test.grammar)
			if test.failBuild {
				require.Error(t, err, "lexer")
				return
			}
			require.NoError(t, err, "lexer")
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
				require.Error(t, err, "lexer")
			} else {
				require.NoError(t, err, "lexer")
			}
			require.Equal(t, test.tokens, tokens)
		})
	}
}

func readAllTokens(lex Lexer) (out []string, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			if perr, ok := msg.(error); ok {
				err = perr
			} else {
				panic(msg)
			}
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

func BenchmarkBuilder(b *testing.B) {
	grammar := `
	Identifier = alpha { alpha | number } .
	Whitespace = "\n" | "\r" | "\t" | " " .

	alpha = "a"…"z" | "A"…"Z" | "_" .
	number = "0"…"9" .
	`
	source := `some id withCase andNumb3rs`
	def, err := Build(grammar)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := def.Lex(strings.NewReader(source))
		for !lexer.Next().EOF() {
		}
	}
}

func BenchmarkScanner(b *testing.B) {
	source := `some id withCase andNumb3rs`
	for i := 0; i < b.N; i++ {
		lexer := LexString(source)
		for !lexer.Next().EOF() {
		}
	}
}

func BenchmarkRegex(b *testing.B) {
	re := regexp.MustCompile(`(?P<Identifier>[A-Za-z_][A-Za-z0-9_]*)|(?P<Whitespace>[[:space:]])`)
	source := `some id withCase andNumb3rs`
	for i := 0; i < b.N; i++ {
		re.FindAllStringSubmatch(source, -1)
	}
}
