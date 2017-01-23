package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var benchInput = "hello world 123 hello world 123"

func BenchmarkTextScannerLexer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lex := TextScannerLexer.Lex(strings.NewReader(benchInput))
		ConsumeAll(lex)
	}
}

func BenchmarkRegexpLexer(b *testing.B) {
	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := def.Lex(strings.NewReader(benchInput))
		ConsumeAll(lex)
	}
}

func BenchmarkEBNFLexer(b *testing.B) {
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
		lex := def.Lex(strings.NewReader(benchInput))
		ConsumeAll(lex)
	}
}
