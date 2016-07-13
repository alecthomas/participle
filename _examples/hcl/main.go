// Package main implements a parser for HashiCorp's HCL configuration syntax.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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

func (b *Bool) Parse(v []interface{}) error {
	if len(v) != 1 {
		return fmt.Errorf("bool can only accept one value")
	}
	s, ok := v[0].(string)
	if !ok {
		return fmt.Errorf("bool must be either 'true' or 'false' but got %q", v[0])
	}
	switch s {
	case "true":
		*b = true
	case "false":
		*b = false
	default:
		return fmt.Errorf("invalid boolean value %q", s)
	}
	return nil
}

type Literal struct {
	Boolean    *Bool   `parser:"@('true'|'false')" json:"boolean,omitempty"`
	Identifier *string `parser:"| @Ident" json:"identifier,omitempty"`
	String     *string `parser:"| @(String|Char|RawString)" json:"string,omitempty"`
	Number     *Number `parser:"| @(Float|Int)" json:"number,omitempty"`
}

type BlockHeader struct {
	Parameters []*Literal `parser:"{ @@ } '{'" json:"parameters,omitempty"`
	Body       *Block     `parser:"@@ '}'" json:"body,omitempty"`
}

type Value struct {
	Literal *Literal   `parser:"@@ |" json:"literal,omitempty"`
	Array   []*Literal `parser:"'[' @@ {',' @@} ']'" json:"array,omitempty"`
}

type Assignment struct {
	Attribute *Value       `parser:"'=' @@ |" json:"attribute,omitempty"`
	Block     *BlockHeader `parser:"@@" json:"block,omitempty"`
}

type Entry struct {
	Key   string      `parser:"@Ident" json:"key,omitempty"`
	Value *Assignment `parser:"@@" json:"value,omitempty"`
}

type Block struct {
	Entries []*Entry `parser:"{ @@ }" json:"entries,omitempty"`
}

func main() {
	kingpin.Parse()

	parser, err := participle.Parse(&Block{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Block{}
	err = parser.Parse(os.Stdin, expr)
	kingpin.FatalIfError(err, "")

	json.NewEncoder(os.Stdout).Encode(expr)
}
