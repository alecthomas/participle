# A parser package for Go [![](https://godoc.org/github.com/alecthomas/participle?status.svg)](http://godoc.org/github.com/alecthomas/participle) [![Build Status](https://travis-ci.org/alecthomas/participle.png)](https://travis-ci.org/alecthomas/participle)

The goals of this package are:

1. Provide an idiomatic and elegant way to define parsers.
2. Allow generation of very fast parsers from this definition.

A grammar is a Go structure that source is parsed into. Conceptually it operates similarly to how
the JSON package works; annotations on the struct define how this mapping occurs.

Note that if a struct field is not keyed with "parser", the entire struct tag will be
used as the grammar fragment. This allows the grammar syntax to remain clear and simple to maintain.

## Annotation syntax

- `@<expr>` Capture expression into the field.
- `@@` Recursively capture using the fields own type.
- `<identifier>` Match named lexer token.
- `{ ... }` Match 0 or more times.
- `( ... )` Group.
- `[ ... ]` Optional.
- `"..."` Match the literal.
- `"."…"."` Match rune in range.
- `.` Period matches any single character.
- `<expr> <expr> ...` Match expressions.
- `<expr> | <expr>` Match one of the alternatives.

Notes:

- Each struct is a single production, with each field applied in sequence.
- `@<expr>` is the mechanism for extracting matches.
- For slice and string fields, each instance of `@` will accumulate into the field, including
  repeated patterns. Accumulation into other types is not supported.

## Examples

Here is an example of defining a parser for the form of EBNF used by `exp/ebnf`:

```go
package main

import (
  "fmt"
  "os"

  "github.com/alecthomas/participle"
)

type Group struct {
  Expression *Expression `'(' @@ ')'`
}

type Option struct {
  Expression *Expression `'[' @@ ']'`
}

type Repetition struct {
  Expression *Expression `'{' @@ '}'`
}

type Literal struct {
  Start string `@String`
  End   string `[ '…' @String ]`
}

type Term struct {
  Name       string      `@Ident |`
  Literal    *Literal    `@@ |`
  Group      *Group      `@@ |`
  Option     *Option     `@@ |`
  Repetition *Repetition `@@`
}

type Sequence struct {
  Terms []*Term `@@ { @@ }`
}

type Expression struct {
  Alternatives []*Sequence `@@ { '|' @@ }`
}

type Expressions []*Expression

type Production struct {
  Name        string      `@Ident '='`
  Expressions Expressions `@@ { @@ } '.'`
}

type EBNF struct {
  Productions []*Production `{ @@ }`
}

func main() {
  parser, err := participle.Parse(&EBNF{}, nil)
  if err != nil { panic(err) }

  ebnf := &EBNF{}
  err = parser.Parse(os.Stdin, ebnf)
  if err != nil { panic(err) }

  json.NewEncoder(os.Stdout).Encode(ebnf)
}
```
