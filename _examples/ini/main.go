package main

import (
	"os"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

// A custom lexer for INI files. This illustrates a relatively complex Regexp lexer, as well
// as use of the Unquote filter, which unquotes string tokens.
var iniLexer = lexer.Unquote(lexer.Must(lexer.Regexp(
	`(?m)` +
		`(\s+)` +
		`|(^#.*$)` +
		`|(?P<Ident>[a-zA-Z][a-zA-Z_\d]*)` +
		`|(?P<String>"(?:\\.|[^"])*")` +
		`|(?P<Number>\d+(?:\.\d+)?)` +
		`|(?P<Punct>[][=])`,
)))

// Value is either a string or a number.
type Value struct {
	String *string  `  @String`
	Number *float64 `| @Number`
}

type Entry struct {
	Key   string `@Ident "="`
	Value *Value `@@`
}
type Section struct {
	Name    string   `"[" @Ident "]"`
	Entries []*Entry `{ @@ }`
}

type INI struct {
	Sections []*Section `{ @@ }`
}

func main() {
	parser, err := participle.Build(&INI{}, iniLexer)
	if err != nil {
		panic(err)
	}
	ini := &INI{}
	err = parser.Parse(os.Stdin, ini)
	if err != nil {
		panic(err)
	}
	repr.Println(ini, repr.Indent("  "))
}
