package main

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/alecthomas/parser"
	"gopkg.in/alecthomas/kingpin.v2"
)

// E --> T {( "+" | "-" ) T}
// T --> F {( "*" | "/" ) F}
// F --> P ["^" F]
// P --> v | "(" E ")" | "-" T

type Number float64

func (i *Number) Parse(s string) error {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*i = Number(n)
	return nil
}

type Value struct {
	Number        *Number     `parser:"@Int |" json:"number,omitempty"`
	Subexpression *Expression `parser:"'(' @@ ')' |" json:"subexpression,omitempty"`
}

type Factor struct {
	Base     *Value `parser:"@@" json:"base,omitempty"`
	Exponent *Value `parser:"[ '^' @@ ]" json:"exponent,omitempty"`
}

type OpFactor struct {
	Operator string  `parser:"@('*' | '/')" json:"operator,omitempty"`
	Factor   *Factor `parser:"@@" json:"factor,omitempty"`
}

type Term struct {
	Left  *Factor     `parser:"@@" json:"left,omitempty"`
	Right []*OpFactor `parser:"{ @@ }" json:"right,omitempty"`
}

type OpTerm struct {
	Operator string `parser:"@('+' | '-')" json:"operator,omitempty"`
	Term     *Term  `parser:"@@" json:"term,omitempty"`
}

type Expression struct {
	Left  *Term     `parser:"@@" json:"left,omitempty"`
	Right []*OpTerm `parser:"{ @@ }" json:"right,omitempty"`
}

func main() {
	kingpin.Parse()

	p, err := parser.Parse(&Expression{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Expression{}
	err = p.Parse(os.Stdin, expr)
	kingpin.FatalIfError(err, "")

	json.NewEncoder(os.Stdout).Encode(expr)
}
