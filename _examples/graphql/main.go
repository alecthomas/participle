package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/participle/v2/lexer/stateful"
)

type File struct {
	Entries []*Entry `@@*`
}

type Entry struct {
	Type   *Type   `  @@`
	Schema *Schema `| @@`
	Enum   *Enum   `| @@`
	Scalar string  `| "scalar" @Ident`
}

type Enum struct {
	Name  string   `"enum" @Ident`
	Cases []string `"{" @Ident* "}"`
}

type Schema struct {
	Fields []*Field `"schema" "{" @@* "}"`
}

type Type struct {
	Name       string   `"type" @Ident`
	Implements string   `( "implements" @Ident )?`
	Fields     []*Field `"{" @@* "}"`
}

type Field struct {
	Name       string      `@Ident`
	Arguments  []*Argument `( "(" ( @@ ( "," @@ )* )? ")" )?`
	Type       *TypeRef    `":" @@`
	Annotation string      `( "@" @Ident )?`
}

type Argument struct {
	Name    string   `@Ident`
	Type    *TypeRef `":" @@`
	Default *Value   `( "=" @@ )?`
}

type TypeRef struct {
	Array       *TypeRef `(   "[" @@ "]"`
	Type        string   `  | @Ident )`
	NonNullable bool     `@"!"?`
}

type Value struct {
	Symbol string `@Ident`
}

var (
	graphQLLexer = lexer.Must(stateful.NewSimple([]stateful.Rule{
		{"Comment", `(?:#|//)[^\n]*\n?`, nil},
		{"Ident", `[a-zA-Z]\w*`, nil},
		{"Number", `(?:\d*\.)?\d+`, nil},
		{"Punct", `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`, nil},
		{"Whitespace", `[ \t\n\r]+`, nil},
	}))
	parser = participle.MustBuild(&File{},
		participle.Lexer(graphQLLexer),
		participle.Elide("Comment", "Whitespace"),
		participle.UseLookahead(2),
	)
)

var cli struct {
	EBNF  bool     `help"Dump EBNF."`
	Files []string `arg:"" optional:"" type:"existingfile" help:"GraphQL schema files to parse."`
}

func main() {
	ctx := kong.Parse(&cli)
	if cli.EBNF {
		fmt.Println(parser.String())
		ctx.Exit(0)
	}
	for _, file := range cli.Files {
		ast := &File{}
		r, err := os.Open(file)
		ctx.FatalIfErrorf(err)
		err = parser.Parse("", r, ast)
		r.Close()
		repr.Println(ast)
		ctx.FatalIfErrorf(err)
	}
}
