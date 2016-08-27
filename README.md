# A dead simple parser package for Go [![](https://godoc.org/github.com/alecthomas/participle?status.svg)](http://godoc.org/github.com/alecthomas/participle) [![Build Status](https://travis-ci.org/alecthomas/participle.svg?branch=master)](https://travis-ci.org/alecthomas/participle)

The goals of this package are:

1. Provide a simple, idiomatic and elegant way to define parsers.
2. Allow generation of very fast parsers from this definition.

A grammar is an annotated Go structure that source is parsed into.
Conceptually it operates similarly to how the JSON package works; annotations
on the struct define how this mapping occurs.

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
- `<expr> <expr> ...` Match expressions.
- `<expr> | <expr>` Match one of the alternatives.

Notes:

- Each struct is a single production, with each field applied in sequence.
- `@<expr>` is the mechanism for capturing matches into the field.

## Capturing

Prefixing any expression in the grammar with `@` will capture matching values
for that expression into the corresponding field.

For example:

```go
// The grammar definition.
type Grammar struct {
  Hello string `@Ident`
}

// The source text to parse.
source := "world"

// The resulting AST.
result := &String{
  Hello: "world",
}
```


For slice and string fields, each instance of `@` will accumulate into the
field (including repeated patterns). Accumulation into other types is not
supported.

A successful capture match into a boolean field will set the field to true.

For integer and floating point types, a successful capture will be parsed
with `strconv.ParseInt()` and `strconv.ParseBool()` respectively.

Custom control of how values are captured into fields can be achieved by a field type
implementing the the `Parseable` interface (`Parse(values []string) error`).

## Lexing

Participle operates on tokens and thus relies on a lexer to convert character
streams to tokens. A default lexer based on the
[text/scanner](https://golang.org/pkg/text/scanner/) package is included.

To use your own Lexer you will need to implement two interfaces:
[LexerDefinition](https://godoc.org/github.com/alecthomas/participle#LexerDefinition)
and [Lexer](https://godoc.org/github.com/alecthomas/participle#Lexer).

## Example

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

There are also more [examples](_examples) included in the source.
