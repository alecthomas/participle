package codegen_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/experimental/codegen"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

var (
	benchmarkInput = `"` + strings.Repeat(`hello ${name} world what's the song that you're singing, come on get ${emotion}`, 1000) + `"`
	exprLexer      = stateful.Must(stateful.Rules{
		"Root": {
			{`String`, `"`, stateful.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, stateful.Pop()},
			{"Expr", `\${`, stateful.Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			stateful.Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, stateful.Pop()},
			stateful.Return(),
		},
	})
)

func TestGenerate(t *testing.T) {
	w := &bytes.Buffer{}
	err := codegen.GenerateLexer(w, "codegen_test", exprLexer)
	require.NoError(t, err)
	t.Log(w.String())
	// cmd := exec.Command("pbcopy")
	// cmd.Stdin = strings.NewReader(source)
	// err = cmd.Run()
	// require.NoError(t, err)
}

func BenchmarkStatefulGenerated(b *testing.B) {
	b.ReportAllocs()
	slex := Lexer.(lexer.StringDefinition)
	start := time.Now()
	for i := 0; i < b.N; i++ {
		lex, err := slex.LexString("", benchmarkInput)
		if err != nil {
			b.Fatal(err)
		}
		for {
			t, err := lex.Next()
			if err != nil {
				b.Fatal(err)
			}
			if t.EOF() {
				break
			}
		}
	}
	b.ReportMetric(float64(len(benchmarkInput)*b.N)*float64(time.Since(start)/time.Second)/1024/1024, "MiB/s")
}

func BenchmarkStatefulRegex(b *testing.B) {
	b.ReportAllocs()
	input := []byte(benchmarkInput)
	start := time.Now()
	for i := 0; i < b.N; i++ {
		lex, err := exprLexer.LexBytes("", input)
		if err != nil {
			b.Fatal(err)
		}
		for {
			t, err := lex.Next()
			if err != nil {
				b.Fatal(err)
			}
			if t.EOF() {
				break
			}
		}
	}
	b.ReportMetric(float64(len(benchmarkInput)*b.N)/float64(time.Since(start)/time.Second)/1024/1024, "MiB/s")
}
