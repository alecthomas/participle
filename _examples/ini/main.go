package main

import (
	"os"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"

	"github.com/alecthomas/repr"
)

// A custom lexer for INI files. This illustrates a relatively complex Regexp lexer, as well
// as use of the Unquote filter, which unquotes string tokens.
var iniLexer = lexer.Must(stateful.NewSimple([]stateful.Rule{
	{`Ident`, `[a-zA-Z][a-zA-Z_\d]*`, nil},
	{`String`, `"(?:\\.|[^"])*"`, nil},
	{`Float`, `\d+(?:\.\d+)?`, nil},
	{`Punct`, `[][=]`, nil},
	{"comment", `[#;][^\n]*`, nil},
	{"whitespace", `\s+`, nil},
}))

type INI struct {
	Properties []*Property `@@*`
	Sections   []*Section  `@@*`
}

type Section struct {
	Identifier string      `"[" @Ident "]"`
	Properties []*Property `@@*`
}

type Property struct {
	Key   string `@Ident "="`
	Value *Value `@@`
}

type Value struct {
	String *string  `  @String`
	Number *float64 `| @Float`
}

func main() {
	parser, err := participle.Build(&INI{},
		participle.Lexer(iniLexer),
		participle.Unquote("String"),
	)
	if err != nil {
		panic(err)
	}
	ini := &INI{}
	err = parser.Parse(os.Stdin, ini)
	if err != nil {
		panic(err)
	}
	repr.Println(ini, repr.Indent("  "), repr.OmitEmpty(true))
}
