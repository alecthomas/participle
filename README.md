# A lexer and parser package for Go

The goals of this package are:

1. Provide an idiomatic and elegant way to define parsers.
2. Allow generation of very fast parsers from this definition.

A grammar is a Go structure that source is decoded into, conceptually similar to how the
JSON package works. Annotations on the grammar structures define how this mapping occurs.

Note that annotations in the grammar *deliberately* do not follow the Go tag format conventions.
This allows the grammar syntax to remain clear and simple to maintain.

## Annotation syntax

- `@<term>` Capture term into the field.
  - `@@` Recursively capture using the fields own type.
  - `@Identifier` Map token of the given name onto the field.
- `{ ... }` Match 0 or more times.
- `( ... )` Group.
- `[ ... ]` Optional.
- `"..."` Match the literal.
- `"."…"."` Match rune in range.
- `.` Period matches any single character.
- `... | ...` Match one of the alternatives.

Notes:

- Each struct is a single production, with each field applied in sequence.
- Syntax can apply across fields.
- `@<term>` is the mechanism for extracting matches.
- For slice fields, each instance of `@@` or `@Identifier` will accumulate into the slice, including
repeated patterns. Accumulation into maps is not supported.
- Optional and alternatives should map to pointer fields, so that non-nil means the value was selected.

## Examples

Here is an example of defining a lexer and parser for a form of EBNF:

```go
// Production  = name "=" [ Expression ] "." .
// Expression  = Alternative { "|" Alternative } .
// Alternative = Term { Term } .
// Term        = name | token [ "…" token ] | "@@" | Group | Option | Repetition .
// Group       = "(" Expression ")" .
// Option      = "[" Expression "]" .
// Repetition  = "{" Expression "}" .

func main() {
  p := parser.New(&Lexer{}, &EBNF{})
  g, err := p.Parse(`Hello = "Hello" | "Hola" | "Kon'nichiwa"`)
  if err != nil {
    panic(err)
  }
  ebnf := g.(*EBNF)
  // ...
}

// Lexer definition.
//
// Lexer tokens can be referenced from the grammar by name in their tags.
type Lexer struct {
  Identifier string      `("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`
  String     string      `"\"" {"\\" . | .} "\""`
  Whitespace lexer.Skip  `" " | "\t" | "\n" | "\r"`
}

type EBNF struct {
  Productions []*Production
}

type Production struct {
  Name       string      `@Identifier "="`
  Expression *Expression `[ @@ ] "."`
}

type Expression struct {
  Alternatives []*Alternative `@@ { "|" @@ }`
}

type Alternative struct {
  Term Term
}

type Term struct {
  Name       *string       `@Identifier |`
  TokenRange *TokenRange   `@@ |`
  Group      *Group        `@@ |`
  Option     *Option       `@@ |`
  Repetition *Repetition   `@@`
}

type Group struct {
  Expression *Expression `"(" @@ ")"`
}

type Option struct {
  Expression *Expression `"[" @@ "]"`
}

type Repetition struct {
  Expression *Expression `"{" @@ "}"`
}

type TokenRange struct {
  Start string  `@String` // Lexer token "String"
  End   *string ` [ "…" @String ]`
}
```


Here is an example JSON grammar:

```go
type Lexer struct {
  Boolean    string      `"true" | "false"`
  Null       string      `"null"`
  Number     float64     `"0"…"9" {"0"…"9"}`
  String     string      `"\"" {"\\" . | .} "\""`
  Whitespace lexer.Skip  `" " | "\t" | "\n" | "\r"`
}

type JSON struct {
  Number  *float64 `@Number |`
  String  *string  `@String |`
  Boolean *bool    `@Boolean |`
  Null    *bool    `@Null |`
  Array   *Array   `@@ |`
  Object  *Object
}

type Array struct {
  Elements []*JSON `"[" [ @@ { "," @@ } ] "]"`
}

type Object struct {
  Elements []*KeyValue `"{" [ @@ { "," @@ } ] "}"`
}

type KeyValue struct {
  Key   string `@String ":"`
  Value *JSON
}
```

A rudimentary parser for import and package lines in Go:

```go
type Lexer struct {
  Identifier string      `("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`
  String     string      `"\"" {"\\" . | .} "\""`
  Whitespace lexer.Skip  `" " | "\t" | "\r"`
  EOL        rune        `"\n" | ";"`
}

type Go struct {
  Package string `"package" @Identifier EOL`
  Imports *Imports
}

type Imports struct {
  Imports []string `"import" ("(" { @String } ")" | @String )`
}

type Import string {
  Alias string `[ @"." | @Identifier ]`
}
```


INI file:

```go
type INI struct {
  Sections []*Section
}

type Section struct {
  Name string `"[" @@Identifier "]" EOL`
  Value []*Value
}

type Value struct {
  Key   string  `@@Identifier "="`
  Value string  `@@Value EOL`
}
```
