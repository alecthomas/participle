package main

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

// TODO: Custom lexer to support dates.

const example = `
// This is a TOML document.

title = "TOML Example"

[owner]
name = "Tom Preston-Werner"
// dob = 1979-05-27T07:32:00-08:00 // First class dates

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ]
connection_max = 5000
enabled = true

[servers]

  // Indentation (tabs and/or spaces) is allowed but not required
  [servers.alpha]
  ip = "10.0.0.1"
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ]

// Line breaks are OK when inside arrays
hosts = [
  "alpha",
  "omega"
]
`

type TOML struct {
	Pos lexer.Position

	Entries []*Entry `{ @@ }`
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
	String  *string  `  @String`
	Bool    *bool    `| (@"true" | "false")`
	Integer *int64   `| @Int`
	Float   *float64 `| @Float`
	List    []*Value `| "[" [ @@ { "," @@ } ] "]"`
}

type Section struct {
	Name   string   `"[" @(Ident { "." Ident }) "]"`
	Fields []*Field `{ @@ }`
}

func main() {
	parser, err := participle.Build(&TOML{})
	kingpin.FatalIfError(err, "")
	toml := &TOML{}
	err = parser.ParseString(example, toml)
	kingpin.FatalIfError(err, "")
	repr.Println(toml)
}
