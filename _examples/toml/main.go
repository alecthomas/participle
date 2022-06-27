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
	tomlLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"DateTime", `\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-\d\d:\d\d)?`},
		{"Date", `\d\d\d\d-\d\d-\d\d`},
		{"Time", `\d\d:\d\d:\d\d(\.\d+)?`},
		{"Ident", `[a-zA-Z_][a-zA-Z_0-9]*`},
		{"String", `"[^"]*"`},
		{"Number", `[-+]?[.0-9]+\b`},
		{"Punct", `\[|]|[-!()+/*=,]`},
		{"comment", `#[^\n]+`},
		{"whitespace", `\s+`},
	})
	tomlParser = participle.MustBuild[TOML](
		participle.Lexer(tomlLexer),
		participle.Unquote("String"),
	)

	cli struct {
		File string `help:"TOML file to parse." arg:""`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	r, err := os.Open(cli.File)
	ctx.FatalIfErrorf(err)
	defer r.Close()
	toml, err := tomlParser.Parse(cli.File, r)
	ctx.FatalIfErrorf(err)
	repr.Println(toml)
}
