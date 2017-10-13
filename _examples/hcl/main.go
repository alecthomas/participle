// Package main implements a parser for HashiCorp's HCL configuration syntax.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Number float64

func (i *Number) Parse(s string) error {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*i = Number(n)
	return nil
}

type Bool bool

func (b *Bool) Capture(v []string) error { *b = v[0] == "true"; return nil }

type Value struct {
	Boolean    *Bool    `  @('true'|'false')`
	Identifier *string  `| @Ident { @"." @Ident }`
	String     *string  `| @(String|Char|RawString)`
	Number     *Number  `| @(Float|Int)`
	Array      []*Value `| '[' { @@ [ ',' ] } ']'`
}

func (l *Value) GoString() string {
	switch {
	case l.Boolean != nil:
		return fmt.Sprintf("%v", *l.Boolean)
	case l.Identifier != nil:
		return fmt.Sprintf("`%s`", *l.Identifier)
	case l.String != nil:
		return fmt.Sprintf("%q", *l.String)
	case l.Number != nil:
		return fmt.Sprintf("%v", *l.Number)
	case l.Array != nil:
		out := []string{}
		for _, v := range l.Array {
			out = append(out, v.GoString())
		}
		return fmt.Sprintf("[]*Value{ %s }", strings.Join(out, ", "))
	}
	panic("??")
}

type Entry struct {
	Key   *Value `@@`
	Value *Value `( '=' @@`
	Block *Block `| @@ )`
}

type Block struct {
	Parameters []*Value `{ @@ }`
	Entries    []*Entry `'{' { @@ } '}'`
}

type Config struct {
	Entries []*Entry `{ @@ }`
}

func main() {
	kingpin.Parse()

	parser, err := participle.Build(&Config{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Config{}
	err = parser.Parse(os.Stdin, expr)
	kingpin.FatalIfError(err, "")

	repr.Println(expr)
}
