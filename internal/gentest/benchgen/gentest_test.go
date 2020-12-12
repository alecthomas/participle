package benchgen

import (
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/internal/gentest"
)

func TestParser(t *testing.T) {
	p := participle.MustBuild(&AST{}, participle.Lexer(Lexer), participle.Elide("whitespace"))
	actual := &AST{}
	err := p.ParseString("", gentest.BenchmarkInput, actual)
	require.NoError(t, err)
	expected := &AST{Entries: []*Entry{
		{Key: "string", Value: &Value{String: "\"hello world\""}},
		{Key: "number", Value: &Value{Number: 1234}},
	}}
	require.Equal(t, expected, actual, repr.String(actual))
}

func TestInvalidInput(t *testing.T) {
	p := participle.MustBuild(&AST{}, participle.Lexer(Lexer), participle.Elide("whitespace"))
	actual := &AST{}
	err := p.ParseString("", `
		string = "str"
		number =
	`, actual)
	require.EqualError(t, err, "4:2: unexpected token \"<EOF>\" (expected Value)")
}

func Benchmark(b *testing.B) {
	b.ReportAllocs()
	p := participle.MustBuild(&AST{}, participle.Lexer(Lexer), participle.Elide("whitespace"))
	input := strings.Repeat(gentest.BenchmarkInput, 1000)
	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ast := &AST{}
		err := p.ParseString("", input, ast)
		if err != nil {
			b.Fatal(err)
		}
		if len(ast.Entries) != 2000 {
			b.Fatal(len(ast.Entries))
		}
	}
	b.ReportMetric(float64(len(input)*b.N)*float64(time.Since(start)/time.Second)/1024/1024, "MiB/s")
}
