package gentest_test

import (
	"os"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/experimental/codegen"
	"github.com/alecthomas/participle/v2/internal/gentest"
	"github.com/alecthomas/participle/v2/internal/gentest/benchgen"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	p := participle.MustBuild(&benchgen.AST{}, participle.Lexer(gentest.Lexer))
	w, err := os.Create("benchgen/benchgen.parser.go")
	require.NoError(t, err)
	defer w.Close()
	err = p.Generate(w)
	require.NoError(t, err)
	w, err = os.Create("benchgen/benchgen.lexer.go")
	require.NoError(t, err)
	defer w.Close()
	err = codegen.GenerateLexer(w, "benchgen", gentest.Lexer)
	require.NoError(t, err)
}
