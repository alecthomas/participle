// Package main implements a parser for HashiCorp's HCL configuration syntax.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
)

type Bool bool

func (b *Bool) Capture(v []string) error { *b = v[0] == "true"; return nil }

type Value struct {
	Boolean    *Bool    `  @("true"|"false")`
	Identifier *string  `| @Ident ( @"." @Ident )*`
	String     *string  `| @(String|Char|RawString)`
	Number     *float64 `| @(Float|Int)`
	Array      []*Value `| "[" ( @@ ","? )* "]"`
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
	Key   string `@Ident`
	Value *Value `( "=" @@`
	Block *Block `  | @@ )`
}

type Block struct {
	Parameters []*Value `@@*`
	Entries    []*Entry `"{" @@* "}"`
}

type Config struct {
	Entries []*Entry `@@*`
}

var parser = participle.MustBuild[Config](participle.Unquote())

func main() {
	expr, err := parser.Parse("", os.Stdin)
	if err != nil {
		panic(err)
	}
	repr.Println(expr)
}
