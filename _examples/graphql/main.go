package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
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
	Cases []string `"{" { @Ident } "}"`
}

type Schema struct {
	Fields []*Field `"schema" "{" { @@ } "}"`
}

type Type struct {
	Name       string   `"type" @Ident`
	Implements string   `[ "implements" @Ident ]`
	Fields     []*Field `"{" { @@ } "}"`
}

type Field struct {
	Name       string      `@Ident`
	Arguments  []*Argument `[ "(" [ @@ { "," @@ } ] ")" ]`
	Type       *TypeRef    `":" @@`
	Annotation string      `[ "@" @Ident ]`
}

type Argument struct {
	Name    string   `@Ident`
	Type    *TypeRef `":" @@`
	Default *Value   `[ "=" @@ ]`
}

type TypeRef struct {
	Array       *TypeRef `(   "[" @@ "]"`
	Type        string   `  | @Ident )`
	NonNullable bool     `[ @"!" ]`
}

type Value struct {
	Symbol string `@Ident`
}

var (
	graphQLLexer = lexer.Must(ebnf.New(`
Comment = ("#" | "//") { "\u0000"…"\uffff"-"\n" } .
Ident = (alpha | "_") { "_" | alpha | digit } .
Number = ("." | digit) {"." | digit} .
Whitespace = " " | "\t" | "\n" | "\r" .
Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .

alpha = "a"…"z" | "A"…"Z" .
digit = "0"…"9" .
`))

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
		err = parser.Parse(r, ast)
		r.Close()
		repr.Println(ast)
		ctx.FatalIfErrorf(err)
	}
}
