// Package participle constructs parsers from definitions in struct tags and parses directly into
// those structs. The approach is philosophically similar to how other marshallers work in Go,
// "unmarshalling" an instance of a grammar into a struct.
//
// The supported annotation syntax is:
//
//     - `@<expr>` Capture subexpression into the field.
//     - `@@` Recursively capture using the fields own type.
//     - `@Identifier` Match token of the given name and capture it.
//     - `{ ... }` Match 0 or more times.
//     - `( ... )` Group.
//     - `[ ... ]` Optional.
//     - `"..."` Match the literal.
//     - `<expr> | <expr>` Match one of the alternatives.
//
// Here's an example of an EBNF grammar.
//
//     type Group struct {
//         Expression *Expression `"(" @@ ")"`
//     }
//
//     type Option struct {
//         Expression *Expression `"[" @@ "]"`
//     }
//
//     type Repetition struct {
//         Expression *Expression `"{" @@ "}"`
//     }
//
//     type Literal struct {
//         Start string `@String` // lexer.Lexer token "String"
//         End   string `[ "…" @String ]`
//     }
//
//     type Term struct {
//         Name       string      `@Ident |`
//         Literal    *Literal    `@@ |`
//         Group      *Group      `@@ |`
//         Option     *Option     `@@ |`
//         Repetition *Repetition `@@`
//     }
//
//     type Sequence struct {
//         Terms []*Term `@@ { @@ }`
//     }
//
//     type Expression struct {
//         Alternatives []*Sequence `@@ { "|" @@ }`
//     }
//
//     type Expressions []*Expression
//
//     type Production struct {
//         Name        string      `@Ident "="`
//         Expressions Expressions `@@ { @@ } "."`
//     }
//
//     type EBNF struct {
//         Productions []*Production `{ @@ }`
//     }
package participle
