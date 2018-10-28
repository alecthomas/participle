// nolint: golint
package main

import (
	"io"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// Parse a BASIC program.
func Parse(r io.Reader) (*Program, error) {
	program := &Program{}
	err := basicParser.Parse(r, program)
	if err != nil {
		return nil, err
	}
	program.init()
	return program, nil
}

type Program struct {
	Pos lexer.Position

	Commands []*Command `{ @@ }`

	Table map[int]*Command
}

type Command struct {
	Pos lexer.Position

	Index int

	Line int `@Number`

	Remark *Remark `(   @@`
	Input  *Input  `  | @@`
	Let    *Let    `  | @@`
	Goto   *Goto   `  | @@`
	If     *If     `  | @@`
	Print  *Print  `  | @@`
	Call   *Call   `  | @@ ) EOL`
}

type Remark struct {
	Pos lexer.Position

	Comment string `@Comment`
}

type Call struct {
	Pos lexer.Position

	Name string        `@Ident`
	Args []*Expression `"(" [ @@ { "," @@ } ] ")"`
}

type Print struct {
	Pos lexer.Position

	Expression *Expression `"PRINT" @@`
}

type Input struct {
	Pos lexer.Position

	Variable string `"INPUT" @Ident`
}

type Let struct {
	Pos lexer.Position

	Variable string      `"LET" @Ident`
	Value    *Expression `"=" @@`
}

type Goto struct {
	Pos lexer.Position

	Line int `"GOTO" @Number`
}

type If struct {
	Pos lexer.Position

	Condition *Expression `"IF" @@`
	Line      int         `"THEN" @Number`
}

type Operator string

func (o *Operator) Capture(s []string) error {
	*o = Operator(strings.Join(s, ""))
	return nil
}

type Value struct {
	Pos lexer.Position

	Number        *float64    `  @Number`
	Variable      *string     `| @Ident`
	String        *string     `| @String`
	Call          *Call       `| @@`
	Subexpression *Expression `| "(" @@ ")"`
}

type Factor struct {
	Pos lexer.Position

	Base     *Value `@@`
	Exponent *Value `[ "^" @@ ]`
}

type OpFactor struct {
	Pos lexer.Position

	Operator Operator `@("*" | "/")`
	Factor   *Factor  `@@`
}

type Term struct {
	Pos lexer.Position

	Left  *Factor     `@@`
	Right []*OpFactor `{ @@ }`
}

type OpTerm struct {
	Pos lexer.Position

	Operator Operator `@("+" | "-")`
	Term     *Term    `@@`
}

type Cmp struct {
	Pos lexer.Position

	Left  *Term     `@@`
	Right []*OpTerm `{ @@ }`
}

type OpCmp struct {
	Pos lexer.Position

	Operator Operator `@("=" | "<" "=" | ">" "=" | "<" | ">" | "!" "=")`
	Cmp      *Cmp     `@@`
}

type Expression struct {
	Pos lexer.Position

	Left  *Cmp     `@@`
	Right []*OpCmp `{ @@ }`
}
