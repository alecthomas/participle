package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/participle/v2"
)

var cli struct {
	JSON bool `help:"Display AST as JSON."`
}

type Group struct {
	Expression *Expression `"(" @@ ")"`
}

func (g *Group) String() string {
	return fmt.Sprintf("( %s )", g.Expression)
}

type Option struct {
	Expression *Expression `"[" @@ "]"`
}

func (o *Option) String() string {
	return fmt.Sprintf("[ %s ]", o.Expression)
}

type Repetition struct {
	Expression *Expression `"{" @@ "}"`
}

func (r *Repetition) String() string {
	return fmt.Sprintf("{ %s }", r.Expression)
}

type Literal struct {
	Start string `@String` // Lexer token "String"
	End   string `( "…" @String )?`
}

func (l *Literal) String() string {
	if l.End != "" {
		return fmt.Sprintf("%q … %q", l.Start, l.End)
	}
	return fmt.Sprintf("%q", l.Start)
}

type Term struct {
	Name       string      `@Ident |`
	Literal    *Literal    `@@ |`
	Group      *Group      `@@ |`
	Option     *Option     `@@ |`
	Repetition *Repetition `@@`
}

func (t *Term) String() string {
	switch {
	case t.Name != "":
		return t.Name
	case t.Literal != nil:
		return t.Literal.String()
	case t.Group != nil:
		return t.Group.String()
	case t.Option != nil:
		return t.Option.String()
	case t.Repetition != nil:
		return t.Repetition.String()
	default:
		panic("wut")
	}
}

type Sequence struct {
	Terms []*Term `@@+`
}

func (s *Sequence) String() string {
	terms := []string{}
	for _, term := range s.Terms {
		terms = append(terms, term.String())
	}
	return strings.Join(terms, " ")
}

type Expression struct {
	Alternatives []*Sequence `@@ ( "|" @@ )*`
}

func (e *Expression) String() string {
	sequences := []string{}
	for _, sequence := range e.Alternatives {
		sequences = append(sequences, sequence.String())
	}
	return strings.Join(sequences, " | ")
}

type Expressions []*Expression

func (e Expressions) String() string {
	expressions := []string{}
	for _, expression := range e {
		expressions = append(expressions, expression.String())
	}
	return strings.Join(expressions, " ")
}

type Production struct {
	Name        string      `@Ident "="`
	Expressions Expressions `@@+ "."`
}

func (p *Production) String() string {
	expressions := []string{}
	for _, expression := range p.Expressions {
		expressions = append(expressions, expression.String())
	}
	return fmt.Sprintf("%s = %s .", p.Name, strings.Join(expressions, " "))
}

type EBNF struct {
	Productions []*Production `@@*`
}

func (e *EBNF) String() string {
	w := bytes.NewBuffer(nil)
	for _, production := range e.Productions {
		fmt.Fprintf(w, "%s\n", production)
	}
	return w.String()
}

var parser = participle.MustBuild[EBNF]()

func main() {
	help := `An EBNF parser compatible with Go"s exp/ebnf. The grammar is
in the form:

  Production  = name "=" [ Expression ] "." .
  Expression  = Alternative { "|" Alternative } .
  Alternative = Term { Term } .
  Term        = name | token [ "…" token ] | Group | Option | Repetition .
  Group       = "(" Expression ")" .
  Option      = "[" Expression "]" .
  Repetition  = "{" Expression "}" .
`
	ctx := kong.Parse(&cli, kong.Description(help))

	ebnf, err := parser.Parse("", os.Stdin)
	ctx.FatalIfErrorf(err, "")

	if cli.JSON {
		bytes, _ := json.MarshalIndent(ebnf, "", "  ")
		fmt.Printf("%s\n", bytes)
	} else {
		fmt.Print(ebnf)
	}
}
