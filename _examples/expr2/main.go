package main

import (
	"strings"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
)

// Based on http://www.craftinginterpreters.com/parsing-expressions.html

// expression     → equality ;
// equality       → comparison ( ( "!=" | "==" ) comparison )* ;
// comparison     → addition ( ( ">" | ">=" | "<" | "<=" ) addition )* ;
// addition       → multiplication ( ( "-" | "+" ) multiplication )* ;
// multiplication → unary ( ( "/" | "*" ) unary )* ;
// unary          → ( "!" | "-" ) unary
//                | primary ;
// primary        → NUMBER | STRING | "false" | "true" | "nil"
//                | "(" expression ")" ;

type Expression struct {
	Equality *Equality `@@`
}

type Equality struct {
	Comparison *Comparison `@@`
	Op         string      `( @( "!" "=" | "=" "=" )`
	Next       *Equality   `  @@ )*`
}

type Comparison struct {
	Addition *Addition   `@@`
	Op       string      `( @( ">" | ">" "=" | "<" | "<" "=" )`
	Next     *Comparison `  @@ )*`
}

type Addition struct {
	Multiplication *Multiplication `@@`
	Op             string          `( @( "-" | "+" )`
	Next           *Addition       `  @@ )*`
}

type Multiplication struct {
	Unary *Unary          `@@`
	Op    string          `( @( "/" | "*" )`
	Next  *Multiplication `  @@ )*`
}

type Unary struct {
	Op      string   `  ( @( "!" | "-" )`
	Unary   *Unary   `    @@ )`
	Primary *Primary `| @@`
}

type Primary struct {
	Number        *float64    `  @Float | @Int`
	String        *string     `| @String`
	Bool          *Boolean    `| @( "true" | "false" )`
	Nil           bool        `| @"nil"`
	SubExpression *Expression `| "(" @@ ")" `
}

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

var parser = participle.MustBuild[Expression](participle.UseLookahead(2))

func main() {
	var cli struct {
		Expr []string `arg required help:"Expression to parse."`
	}
	ctx := kong.Parse(&cli)

	expr, err := parser.ParseString("", strings.Join(cli.Expr, " "))
	ctx.FatalIfErrorf(err)

	repr.Println(expr)
}
