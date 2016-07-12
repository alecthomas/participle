package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/parser"
	"gopkg.in/alecthomas/kingpin.v2"
)

// E --> T {( "+" | "-" ) T}
// T --> F {( "*" | "/" ) F}
// F --> P ["^" F]
// P --> v | "(" E ")" | "-" T

type Value struct {
	Number        *float64    `@(Float|Int)`
	Subexpression *Expression `| "(" @@ ")"`
}

func (v *Value) Eval() float64 {
	if v.Number != nil {
		return *v.Number
	}
	return v.Subexpression.Eval()
}

type Factor struct {
	Base     *Value `@@`
	Exponent *Value `[ "^" @@ ]`
}

func (f *Factor) Eval() float64 {
	return f.Base.Eval()
}

type OpFactor struct {
	Operator string  `@("*" | "/")`
	Factor   *Factor `@@`
}

type Term struct {
	Left  *Factor     `@@`
	Right []*OpFactor `{ @@ }`
}

func (t *Term) Eval() float64 {
	return t.Left.Eval()
}

type OpTerm struct {
	Operator string `@("+" | "-")`
	Term     *Term  `@@`
}

type Expression struct {
	Left  *Term     `@@`
	Right []*OpTerm `{ @@ }`
}

func (e *Expression) Eval() float64 {
	return e.Left.Eval()
}

func main() {
	kingpin.Parse()

	p, err := parser.Parse(&Expression{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Expression{}
	err = p.Parse(os.Stdin, expr)
	kingpin.FatalIfError(err, "")

	json.NewEncoder(os.Stdout).Encode(expr)
	fmt.Println(expr.Eval())
}
