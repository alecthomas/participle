// nolint: golint, stylecheck
package gentest

import "github.com/alecthomas/participle/v2/lexer/stateful"

// BenchmarkInput used by shared benchmarks.
const BenchmarkInput = `
string = "hello world"
number = 1234
`

// Lexer used by generated and non-generated code.
var Lexer = stateful.MustSimple([]stateful.Rule{
	{"Int", `\d+`, nil},
	{"Ident", `[a-zA-Z_][a-zA-Z_0-9]*`, nil},
	{"String", `"(\\.|[^"])*"`, nil},
	{"Operator", `=`, nil},
	{"whitespace", `\s+`, nil},
})
