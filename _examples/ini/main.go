package main

import (
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer/stateful"

	"github.com/alecthomas/repr"
)

// A custom lexer for INI files. This illustrates a relatively complex Regexp lexer, as well
// as use of the Unquote filter, which unquotes string tokens.
var iniLexer = stateful.MustSimple([]stateful.Rule{
	{`Ident`, `[a-zA-Z][a-zA-Z_\d]*`, nil},
	{`String`, `"(?:\\.|[^"])*"`, nil},
	{`Float`, `\d+(?:\.\d+)?`, nil},
	{`Punct`, `[][=]`, nil},
	{"comment", `[#;][^\n]*`, nil},
	{"whitespace", `\s+`, nil},
})

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

var parser = participle.MustBuild(&INI{},
	participle.Lexer(iniLexer),
	participle.Unquote("String"),
)

func main() {
	ini := &INI{}
	err := parser.Parse("", os.Stdin, ini)
	if err != nil {
		panic(err)
	}
	repr.Println(ini, repr.Indent("  "), repr.OmitEmpty(true))
}
