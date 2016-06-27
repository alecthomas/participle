package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/parser"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Group struct {
	Expression *Expression `parser:"\"(\" @@ \")\"" json:",omitempty"`
}

type Option struct {
	Expression *Expression `parser:"\"[\" @@ \"]\"" json:",omitempty"`
}

type Repetition struct {
	Expression *Expression `parser:"\"{\" @@ \"}\"" json:",omitempty"`
}

type Literal struct {
	Start string  `parser:"@String"` // Lexer token \"String\" json:",omitempty""
	End   *string `parser:"[ \"…\" @String ]" json:",omitempty"`
}

type Term struct {
	Name       string      `parser:"@Ident |" json:",omitempty"`
	Literal    *Literal    `parser:"@@ |" json:",omitempty"`
	Group      *Group      `parser:"@@ |" json:",omitempty"`
	Option     *Option     `parser:"@@ |" json:",omitempty"`
	Repetition *Repetition `parser:"@@" json:",omitempty"`
}

type Sequence struct {
	Terms []*Term `parser:"@@ { @@ }" json:",omitempty"`
}

type Expression struct {
	Alternatives []*Sequence `parser:"@@ { \"|\" @@ }" json:",omitempty"`
}

type Production struct {
	Name       string        `parser:"@Ident \"=\"" json:",omitempty"`
	Expression []*Expression `parser:"@@ { @@ } \".\"" json:",omitempty"`
}

type EBNF struct {
	Productions []*Production `parser:"{ @@ }" json:",omitempty"`
}

func main() {
	kingpin.CommandLine.Help = `An EBNF parser compatible with Go's exp/ebnf. The grammar is
in the form:

  Production  = name "=" [ Expression ] "." .
  Expression  = Alternative { "|" Alternative } .
  Alternative = Term { Term } .
  Term        = name | token [ "…" token ] | Group | Option | Repetition .
  Group       = "(" Expression ")" .
  Option      = "[" Expression "]" .
  Repetition  = "{" Expression "}" .
`
	kingpin.Parse()

	parser, err := parser.Parse(&EBNF{}, nil)
	kingpin.FatalIfError(err, "")

	ebnf := &EBNF{}
	err = parser.Parse(os.Stdin, ebnf)
	kingpin.FatalIfError(err, "")

	bytes, _ := json.MarshalIndent(ebnf, "", "  ")
	fmt.Printf("%s\n", bytes)
}
