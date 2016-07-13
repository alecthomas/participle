// Package main implements
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	astFlag  = kingpin.Flag("ast", "Print AST for expression.").Bool()
	exprArgs = kingpin.Arg("expression", "Expression to evaluate.").Required().Strings()
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
	b := f.Base.Eval()
	if f.Exponent != nil {
		return math.Pow(b, f.Exponent.Eval())
	}
	return b
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
	n := t.Left.Eval()
	for _, r := range t.Right {
		switch r.Operator {
		case "*":
			n *= r.Factor.Eval()
		case "/":
			n /= r.Factor.Eval()
		}
	}
	return n
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
	l := e.Left.Eval()
	for _, r := range e.Right {
		switch r.Operator {
		case "+":
			l += r.Term.Eval()
		case "-":
			l -= r.Term.Eval()
		}
	}
	return l
}

func main() {
	kingpin.CommandLine.Help = "A basic expression parser and evaluator."
	kingpin.Parse()

	parser, err := participle.Parse(&Expression{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Expression{}
	err = parser.ParseString(strings.Join(*exprArgs, " "), expr)
	kingpin.FatalIfError(err, "")

	if *astFlag {
		json.NewEncoder(os.Stdout).Encode(expr)
	} else {
		fmt.Println(expr.Eval())
	}
}
