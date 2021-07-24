package internal_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2/lexer"
)

var (
	testInput      = `hello ${name} world what's the song that you're singing, come on get ${emotion}`
	benchmarkInput = `"` + strings.Repeat(testInput, 1000) + `"`
	exprLexer      = lexer.MustStateful(lexer.Rules{
		"Root": {
			{`String`, `"`, lexer.Push("String")},
		},
		"String": {
			{"Escaped", `\\.`, nil},
			{"StringEnd", `"`, lexer.Pop()},
			{"Expr", `\${`, lexer.Push("Expr")},
			{"Char", `[^$"\\]+`, nil},
		},
		"Expr": {
			lexer.Include("Root"),
			{`Whitespace`, `\s+`, nil},
			{`Oper`, `[-+/*%]`, nil},
			{"Ident", `\w+`, nil},
			{"ExprEnd", `}`, lexer.Pop()},
		},
	})
)

func TestGenerate(t *testing.T) {
	w, err := os.Create("codegen_gen_test.go")
	require.NoError(t, err)
	defer w.Close()
	err = lexer.ExperimentalGenerateLexer(w, "internal_test", exprLexer)
	require.NoError(t, err)
	err = exec.Command("gofmt", "-w", "codegen_gen_test.go").Run()
	require.NoError(t, err)
	// cmd.Stdin = strings.NewReader(source)
	// err = cmd.Run()
	// require.NoError(t, err)
}

func TestIdentical(t *testing.T) {
	lex, err := exprLexer.LexString("", `"`+testInput+`"`)
	require.NoError(t, err)
	expected, err := lexer.ConsumeAll(lex)
	require.NoError(t, err)

	lex, err = Lexer.Lex("", strings.NewReader(`"`+testInput+`"`))
	require.NoError(t, err)
	actual, err := lexer.ConsumeAll(lex)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
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
	start := time.Now()
	for i := 0; i < b.N; i++ {
		lex, err := exprLexer.LexString("", benchmarkInput)
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
