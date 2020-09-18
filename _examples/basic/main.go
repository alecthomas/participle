// nolint: golint, dupl
package main

import (
	"os"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

var (
	basicLexer = lexer.Must(stateful.NewSimple([]stateful.Rule{
		{"Comment", `(?i)rem[^\n]*`, nil},
		{"String", `"(\\"|[^"])*"`, nil},
		{"Number", `[-+]?(\d*\.)?\d+`, nil},
		{"Ident", `[a-zA-Z_]\w*`, nil},
		{"Punct", `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`, nil},
		{"EOL", `[\n\r]+`, nil},
		{"whitespace", `[ \t]+`, nil},
	}))

	basicParser = participle.MustBuild(&Program{},
		participle.Lexer(basicLexer),
		participle.CaseInsensitive("Ident"),
		participle.Unquote("String"),
		participle.UseLookahead(2),
	)

	cli struct {
		File string `arg:"" type:"existingfile" help:"File to parse."`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	r, err := os.Open(cli.File)
	ctx.FatalIfErrorf(err)
	defer r.Close()
	program, err := Parse(r)
	ctx.FatalIfErrorf(err)

	funcs := map[string]Function{
		"ADD": func(args ...interface{}) (interface{}, error) {
			return args[0].(float64) + args[1].(float64), nil
		},
	}
	err = program.Evaluate(os.Stdin, os.Stdout, funcs)
	ctx.FatalIfErrorf(err)
}
