package participle_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEBNF(t *testing.T) {
	parser := mustTestParser(t, &EBNF{})
	expected := `
EBNF = Production* .
Production = <ident> "=" Expression+ "." .
Expression = Sequence ("|" Sequence)* .
Sequence = Term+ .
Term = <ident> | Literal | Range | Group | EBNFOption | Repetition .
Literal = <string> .
Range = <string> "…" <string> .
Group = "(" Expression ")" .
EBNFOption = "[" Expression "]" .
Repetition = "{" Expression "}" .
`
	require.Equal(t, strings.TrimSpace(expected), parser.String())
}
