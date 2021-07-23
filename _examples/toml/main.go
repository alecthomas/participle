package main

import (
	"os"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type TOML struct {
	Pos lexer.Position

	Entries []*Entry `@@*`
}

type Entry struct {
	Field   *Field   `  @@`
	Section *Section `| @@`
}

type Field struct {
	Key   string `@Ident "="`
	Value *Value `@@`
}

type Value struct {
	String   *string  `  @String`
	DateTime *string  `| @DateTime`
	Date     *string  `| @Date`
	Time     *string  `| @Time`
	Bool     *bool    `| (@"true" | "false")`
	Number   *float64 `| @Number`
	List     []*Value `| "[" ( @@ ( "," @@ )* )? "]"`
}

type Section struct {
	Name   string   `"[" @(Ident ( "." Ident )*) "]"`
	Fields []*Field `@@*`
}

var (
	tomlLexer = lexer.Must(lexer.NewSimple([]lexer.Rule{
		{"DateTime", `\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-\d\d:\d\d)?`, nil},
		{"Date", `\d\d\d\d-\d\d-\d\d`, nil},
		{"Time", `\d\d:\d\d:\d\d(\.\d+)?`, nil},
		{"Ident", `[a-zA-Z_][a-zA-Z_0-9]*`, nil},
		{"String", `"[^"]*"`, nil},
		{"Number", `[-+]?[.0-9]+\b`, nil},
		{"Punct", `\[|]|[-!()+/*=,]`, nil},
		{"comment", `#[^\n]+`, nil},
		{"whitespace", `\s+`, nil},
	}))
	tomlParser = participle.MustBuild(&TOML{},
		participle.Lexer(
			tomlLexer,
		),
		participle.Unquote("String"),
	)

	cli struct {
		File string `help:"TOML file to parse." arg:""`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	toml := &TOML{}
	r, err := os.Open(cli.File)
	ctx.FatalIfErrorf(err)
	defer r.Close()
	err = tomlParser.Parse(cli.File, r, toml)
	ctx.FatalIfErrorf(err)
	repr.Println(toml)
}
