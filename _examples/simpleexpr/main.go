package main

import (
	"github.com/alecthomas/kong"
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
)

type Expr struct {
	Lhs   *Value  `@@`
	Opers []*Oper `@@*`
}

type Oper struct {
	Op  string `@( "!" "=" | ("!"|"="|"<"|">") "="? | "+" | "-" | "/" | "*" )`
	Rhs *Value `@@`
}

type Value struct {
	Number        *float64 `  @Float | @Int`
	String        *string  `| @String`
	Bool          *bool    `| ( @"true" | "false" )`
	Nil           bool     `| @"nil"`
	SubExpression *Expr    `| "(" @@ ")" `
}

var (
	cli struct {
		Expr string `arg:"" help:"Expression."`
	}
	parser = participle.MustBuild(&Expr{})
)

func main() {
	kctx := kong.Parse(&cli, kong.Description(`
A simple expression parser that does not capture precedence at all. Precedence
must be applied at the evaluation phase.

The advantage of this approach over expr1, which does encode precedence in
the parser, is that it is significantly less complex and less nested. The
advantage of this over the "precedenceclimbing" example is that no custom
parsing is required.
`))
	expr := &Expr{}
	err := parser.ParseString("", cli.Expr, expr)
	kctx.FatalIfErrorf(err)
	repr.Println(expr)
}
