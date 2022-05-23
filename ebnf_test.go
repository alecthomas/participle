package participle_test

import (
	"strings"
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestEBNF(t *testing.T) {
	parser := mustTestParser(t, &EBNF{})
	expected := `
EBNF = Production* .
Production = <ident> "=" Expression+ "." .
Expression = Sequence ("|" Sequence)* .
Sequence = Term+ .
Term = <ident> | Literal | Range | Group | LookaheadGroup | EBNFOption | Repetition | Negation .
Literal = <string> .
Range = <string> "…" <string> .
Group = "(" Expression ")" .
LookaheadGroup = "(" "?" ("=" | "!") Expression ")" .
EBNFOption = "[" Expression "]" .
Repetition = "{" Expression "}" .
Negation = "!" Expression .
`
	require.Equal(t, strings.TrimSpace(expected), parser.String())
}

func TestEBNF_Other(t *testing.T) {
	type Grammar struct {
		PositiveLookahead string `  (?= 'good') @Ident`
		NegativeLookahead string `| (?! 'bad' | "worse") @Ident`
		Negation          string `| !("anything" | 'but')`
	}

	parser := mustTestParser(t, &Grammar{})
	expected := `Grammar = ((?= "good") <ident>) | ((?! "bad" | "worse") <ident>) | ~("anything" | "but") .`
	require.Equal(t, expected, parser.String())
}
