package codegen_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
	"github.com/alecthomas/participle/lexer/stateful/codegen"
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
		},
	})
)

func TestGenerate(t *testing.T) {
	w := &bytes.Buffer{}
	err := codegen.Generate(w, "codegen_test", exprLexer)
	require.NoError(t, err)
	source := w.String()
	// cmd := exec.Command("pbcopy")
	// cmd.Stdin = strings.NewReader(source)
	// err = cmd.Run()
	// require.NoError(t, err)

	formatted := &bytes.Buffer{}
	cmd := exec.Command("gofmt", "-s")
	cmd.Stdin = strings.NewReader(source)
	cmd.Stdout = formatted
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err, source)

	// cmd = exec.Command("pbcopy")
	// cmd.Stdin = formatted
	// err = cmd.Run()
	// require.NoError(t, err)
}

func BenchmarkStatefulGenerated(b *testing.B) {
	b.ReportAllocs()
	b.ReportMetric(float64(len(benchmarkInput)), "B")
	slex := Lexer.(interface {
		LexString(string, string) (lexer.Lexer, error)
	})
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
}

func BenchmarkStatefulRegex(b *testing.B) {
	b.ReportAllocs()
	b.ReportMetric(float64(len(benchmarkInput)), "B")
	input := []byte(benchmarkInput)
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
}
