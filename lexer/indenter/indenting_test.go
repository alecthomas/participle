package indenter

import (
	"strings"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	input = `
if true
  print something
else
  print other
  print thing
  if more
    print more
  else
    print last
`
	def = lexer.Must(New(lexer.MustSimple([]lexer.Rule{
		{"Whitespace", `\s+`, nil},
		{"Ident", `\w+`, nil},
	})))
)

type Pythonish struct {
	Statements []*Stmt `@@*`
}

type Stmt struct {
	If    *IfStmt    `  @@`
	Print *PrintStmt `| @@`
}

type IfStmt struct {
	If    string `"if" @Ident`
	Block *Block `@@`
	Else  *Block `"else" @@`
}

type PrintStmt struct {
	Arg string `"print" @Ident`
}

type Block struct {
	Stmt []*Stmt `Indent @@+ Dedent`
}

func TestIndenterNotSupported(t *testing.T) {
	_, err := New(lexer.DefaultDefinition)
	require.Error(t, err)
}

func TestIndenterLexer(t *testing.T) {
	lex, err := def.Lex("", strings.NewReader(input))
	require.NoError(t, err)
	tokens, err := lexer.ConsumeAll(lex)
	require.NoError(t, err)
	repr.Println(tokens)
	require.Equal(t, []lexer.Token{}, tokens)
}

func TestIndentedGrammar(t *testing.T) {
	parser := participle.MustBuild(&Pythonish{}, participle.Lexer(def))
	grammar := &Pythonish{}
	err := parser.ParseString("", input, grammar)
	require.NoError(t, err)
}
