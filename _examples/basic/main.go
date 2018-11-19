// nolint: golint, dupl
package main

import (
	"os"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var (
	basicLexer = lexer.Must(ebnf.New(`
		Comment = ("REM" | "rem" ) { "\u0000"…"\uffff"-"\n"-"\r" } .
		Ident = (alpha | "_") { "_" | alpha | digit } .
		String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
		Number = [ "-" | "+" ] ("." | digit) { "." | digit } .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
		EOL = ( "\n" | "\r" ) { "\n" | "\r" }.
		Whitespace = ( " " | "\t" ) { " " | "\t" } .

		alpha = "a"…"z" | "A"…"Z" .
		digit = "0"…"9" .
		any = "\u0000"…"\uffff" .
	`))

	basicParser = participle.MustBuild(&Program{},
		participle.Lexer(basicLexer),
		participle.CaseInsensitive("Ident"),
		participle.Unquote("String"),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
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
