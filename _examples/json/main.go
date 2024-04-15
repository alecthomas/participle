// nolint: golint, dupl
package main

import (
	"os"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	jsonLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "Comment", Pattern: `\/\/[^\n]*`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "Punct", Pattern: `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`},
		{Name: "Null", Pattern: "null"},
		{Name: "True", Pattern: "true"},
		{Name: "False", Pattern: "false"},
		{Name: "EOL", Pattern: `[\n\r]+`},
		{Name: "Whitespace", Pattern: `[ \t]+`},
	})

	jsonParser = participle.MustBuild[Json](
		participle.Lexer(jsonLexer),
		participle.Unquote("String"),
		participle.Elide("Whitespace", "EOL"),
		participle.UseLookahead(2),
	)

	cli struct {
		File string `arg:"" type:"existingfile" help:"File to parse."`
	}
)

// Parse a Json string.
func Parse(data []byte) (*Json, error) {
	json, err := jsonParser.ParseBytes("", data)
	if err != nil {
		return nil, err
	}
	return json, nil
}

type Json struct {
	Pos lexer.Position

	Object *Object `parser:"@@ |"`
	Array  *Array  `parser:"@@ |"`
	Number *string `parser:"@Number |"`
	String *string `parser:"@String |"`
	False  *string `parser:"@False |"`
	True   *string `parser:"@True |"`
	Null   *string `parser:"@Null"`
}

type Object struct {
	Pos lexer.Position

	Pairs []*Pair `parser:"'{' @@ (',' @@)* '}'"`
}

type Pair struct {
	Pos lexer.Position

	Key   string `parser:"@String ':'"`
	Value *Json  `parser:"@@"`
}

type Array struct {
	Pos lexer.Position

	Items []*Json `parser:"'[' @@ (',' @@)* ']'"`
}

func main() {
	ctx := kong.Parse(&cli)
	data, err := os.ReadFile(cli.File)
	ctx.FatalIfErrorf(err)

	res, err := Parse(data)
	ctx.FatalIfErrorf(err)
	ctx.Printf("res is: %v", res)
}
