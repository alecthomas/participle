package conformance_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var conformanceLexer = lexer.MustStateful(lexer.Rules{
	"Root": {
		{"ExprTest", `EXPRTEST:`, lexer.Push("ExprTest")},
		{"LiteralTest", `LITTEST:`, lexer.Push("LiteralTest")},
		{"CaseInsensitiveTest", `CITEST:`, lexer.Push("CaseInsensitiveTest")},
		{"WordBoundaryTest", `WBTEST:`, lexer.Push("WordBoundaryTest")},
	},
	"ExprTest": {
		{"ExprString", `"`, lexer.Push("ExprString")},
		// {"ExprHeredoc", `<<(\w+)`, lexer.Push("ExprHeredoc")},
	},
	"ExprString": {
		{"ExprEscaped", `\\.`, nil},
		{"ExprStringEnd", `"`, lexer.Pop()},
		{"Expr", `\${`, lexer.Push("Expr")},
		{"ExprChar", `[^$"\\]+`, nil},
	},
	"Expr": {
		lexer.Include("ExprTest"),
		{`Whitespace`, `\s+`, nil},
		{`ExprOper`, `[-+/*%]`, nil},
		{"Ident", `\w+`, lexer.Push("ExprReference")},
		{"ExprEnd", `}`, lexer.Pop()},
	},
	"ExprReference": {
		{"ExprDot", `\.`, nil},
		{"Ident", `\w+`, nil},
		lexer.Return(),
	},
	// "ExprHeredoc": {
	// 	{"ExprHeredocEnd", `\1`, lexer.Pop()},
	// 	lexer.Include("Expr"),
	// },
	"LiteralTest": {
		{`LITOne`, `ONE`, nil},
		{`LITKeyword`, `SELECT|FROM|WHERE|LIKE`, nil},
		{"Ident", `\w+`, nil},
		{"Whitespace", `\s+`, nil},
	},
	"CaseInsensitiveTest": {
		{`ABCWord`, `[aA][bB][cC]`, nil},
		{`CIKeyword`, `(?i)(SELECT|from|WHERE|LIKE)`, nil},
		{"Ident", `\w+`, nil},
		{"Whitespace", `\s+`, nil},
	},
	"WordBoundaryTest": {
		{`WBKeyword`, `\b(?:abc|xyz)\b`, nil},
		{"Slash", `/`, nil},
		{"Ident", `\w+`, nil},
		{"Whitespace", `\s+`, nil},
	},
})

type token struct {
	Type  string
	Value string
}

func testLexer(t *testing.T, lex lexer.Definition) {
	t.Helper()
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{"ExprPush", `EXPRTEST:"${"Hello ${name + "!"}"}"`, []token{
			{"ExprString", "\""},
			{"Expr", "${"},
			{"ExprString", "\""},
			{"ExprChar", "Hello "},
			{"Expr", "${"},
			{"Ident", "name"},
			{"Whitespace", " "},
			{"ExprOper", "+"},
			{"Whitespace", " "},
			{"ExprString", "\""},
			{"ExprChar", "!"},
			{"ExprStringEnd", "\""},
			{"ExprEnd", "}"},
			{"ExprStringEnd", "\""},
			{"ExprEnd", "}"},
			{"ExprStringEnd", "\""},
		}},
		{"ExprReference", `EXPRTEST:"${user.name}"`, []token{
			{"ExprString", "\""},
			{"Expr", "${"},
			{"Ident", "user"},
			{"ExprDot", "."},
			{"Ident", "name"},
			{"ExprEnd", "}"},
			{"ExprStringEnd", "\""},
		}},
		// TODO(alecthomas): Once backreferences are supported, this will work.
		// 		{"Backref", `EXPRTEST:<<EOF
		// heredoc
		// EOF`, []token{
		// 			{"ExprHeredoc", "<<EOF"},
		// 			{"Whitespace", "\n"},
		// 			{"Ident", "heredoc"},
		// 			{"Whitespace", "\n"},
		// 			{"ExprHeredocEnd", "EOF"},
		// 		}},
		{"CaseInsensitiveSimple", `CITEST:hello aBC world`, []token{
			{"Ident", "hello"},
			{"Whitespace", " "},
			{"ABCWord", "aBC"},
			{"Whitespace", " "},
			{"Ident", "world"},
		}},
		{"CaseInsensitiveMixed", `CITEST:hello SeLeCt FROM world where END`, []token{
			{"Ident", "hello"},
			{"Whitespace", " "},
			{"CIKeyword", "SeLeCt"},
			{"Whitespace", " "},
			{"CIKeyword", "FROM"},
			{"Whitespace", " "},
			{"Ident", "world"},
			{"Whitespace", " "},
			{"CIKeyword", "where"},
			{"Whitespace", " "},
			{"Ident", "END"},
		}},
		{"OneLiteralAtEnd", `LITTEST:ONE`, []token{
			{"LITOne", "ONE"},
		}},
		{"KeywordLiteralAtEnd", `LITTEST:SELECT`, []token{
			{"LITKeyword", "SELECT"},
		}},
		{"LiteralMixed", `LITTEST:hello ONE test LIKE world`, []token{
			{"Ident", "hello"},
			{"Whitespace", " "},
			{"LITOne", "ONE"},
			{"Whitespace", " "},
			{"Ident", "test"},
			{"Whitespace", " "},
			{"LITKeyword", "LIKE"},
			{"Whitespace", " "},
			{"Ident", "world"},
		}},
		{"WordBoundarySlash", `WBTEST:xyz/hello world`, []token{
			{"WBKeyword", "xyz"},
			{"Slash", "/"},
			{"Ident", "hello"},
			{"Whitespace", " "},
			{"Ident", "world"},
		}},
		{"WordBoundaryWhitespace", `WBTEST:abchello xyz world`, []token{
			{"Ident", "abchello"},
			{"Whitespace", " "},
			{"WBKeyword", "xyz"},
			{"Whitespace", " "},
			{"Ident", "world"},
		}},
		{"WordBoundaryStartEnd", `WBTEST:xyz`, []token{
			{"WBKeyword", "xyz"},
		}},
	}
	symbols := lexer.SymbolsByRune(lex)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l, err := lex.Lex(test.name, strings.NewReader(test.input))
			assert.NoError(t, err)
			tokens, err := lexer.ConsumeAll(l)
			assert.NoError(t, err)
			actual := make([]token, 0, len(tokens))
			for i, t := range tokens {
				if (i == 0 && strings.HasSuffix(t.Value, "TEST:")) || t.Type == lexer.EOF {
					continue
				}
				actual = append(actual, token{Type: symbols[t.Type], Value: t.Value})
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestLexerConformanceGenerated(t *testing.T) {
	genLexer(t)
	args := []string{"test", "-run", "TestLexerConformanceGeneratedInternal", "-tags", "generated"}
	// Propagate test flags.
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Value.String() != f.DefValue && f.Name != "test.run" {
			args = append(args, fmt.Sprintf("-%s=%s", f.Name, f.Value.String()))
		}
	})
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	assert.NoError(t, err)
}

func TestLexerConformance(t *testing.T) {
	testLexer(t, conformanceLexer)
}

func genLexer(t *testing.T) {
	t.Helper()
	lexerJSON, err := json.Marshal(conformanceLexer)
	assert.NoError(t, err)
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	generatedConformanceLexer := filepath.Join(cwd, "conformance_lexer_gen.go")
	t.Cleanup(func() {
		_ = os.Remove(generatedConformanceLexer)
	})
	cmd := exec.Command(
		"../../../scripts/participle",
		"gen", "lexer", "conformance",
		"--tags", "generated",
		"--name", "GeneratedConformance",
		"--output", generatedConformanceLexer)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	w, err := cmd.StdinPipe()
	assert.NoError(t, err)
	defer w.Close()
	err = cmd.Start()
	assert.NoError(t, err)
	_, err = w.Write(lexerJSON)
	assert.NoError(t, err)
	err = w.Close()
	assert.NoError(t, err)
	err = cmd.Wait()
	assert.NoError(t, err)
}
