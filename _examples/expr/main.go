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

type Operator int

const (
	OpMul Operator = iota
	OpDiv
	OpAdd
	OpSub
)

var operatorMap = map[string]Operator{"+": OpAdd, "-": OpSub, "*": OpMul, "/": OpDiv}

func (o *Operator) Capture(s []string) error {
	*o = operatorMap[s[0]]
	return nil
}

func (o Operator) Eval(l, r float64) float64 {
	switch o {
	case OpMul:
		return l * r
	case OpDiv:
		return l / r
	case OpAdd:
		return l + r
	case OpSub:
		return l - r
	}
	panic("unsupported operator")
}

func (o Operator) String() string {
	switch o {
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpSub:
		return "-"
	case OpAdd:
		return "+"
	}
	panic("unsupported operator")
}

// E --> T {( "+" | "-" ) T}
// T --> F {( "*" | "/" ) F}
// F --> P ["^" F]
// P --> v | "(" E ")" | "-" T

type Value struct {
	Number        *float64    `@(Float|Int)`
	Subexpression *Expression `| "(" @@ ")"`
}

func (v *Value) String() string {
	if v.Number != nil {
		return fmt.Sprintf("%g", *v.Number)
	}
	return "(" + v.Subexpression.String() + ")"
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

func (f *Factor) String() string {
	out := f.Base.String()
	if f.Exponent != nil {
		out += " ^ " + f.Exponent.String()
	}
	return out
}

func (f *Factor) Eval() float64 {
	b := f.Base.Eval()
	if f.Exponent != nil {
		return math.Pow(b, f.Exponent.Eval())
	}
	return b
}

type OpFactor struct {
	Operator Operator `@("*" | "/")`
	Factor   *Factor  `@@`
}

func (o *OpFactor) String() string {
	return fmt.Sprintf("%s %s", o.Operator, o.Factor)
}

type Term struct {
	Left  *Factor     `@@`
	Right []*OpFactor `{ @@ }`
}

func (t *Term) String() string {
	out := []string{t.Left.String()}
	for _, r := range t.Right {
		out = append(out, r.String())
	}
	return strings.Join(out, " ")
}

func (t *Term) Eval() float64 {
	n := t.Left.Eval()
	for _, r := range t.Right {
		n = r.Operator.Eval(n, r.Factor.Eval())
	}
	return n
}

type OpTerm struct {
	Operator Operator `@("+" | "-")`
	Term     *Term    `@@`
}

func (o *OpTerm) String() string {
	return fmt.Sprintf("%s %s", o.Operator, o.Term)
}

type Expression struct {
	Left  *Term     `@@`
	Right []*OpTerm `{ @@ }`
}

func (e *Expression) String() string {
	out := []string{e.Left.String()}
	for _, r := range e.Right {
		out = append(out, r.String())
	}
	return strings.Join(out, " ")
}

func (e *Expression) Eval() float64 {
	l := e.Left.Eval()
	for _, r := range e.Right {
		l = r.Operator.Eval(l, r.Term.Eval())
	}
	return l
}

func main() {
	kingpin.CommandLine.Help = "A basic expression parser and evaluator."
	kingpin.Parse()

	parser, err := participle.Build(&Expression{}, nil)
	kingpin.FatalIfError(err, "")

	expr := &Expression{}
	err = parser.ParseString(strings.Join(*exprArgs, " "), expr)
	kingpin.FatalIfError(err, "")

	if *astFlag {
		json.NewEncoder(os.Stdout).Encode(expr)
	} else {
		fmt.Println(expr, "=", expr.Eval())
	}
}
