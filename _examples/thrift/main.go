// Package main implements a parser for Thrift files (https://thrift.apache.org/)
//
// It parses namespaces, exceptions, services, structs, consts, typedefs and enums, but is easily
// extensible to more.
//
// It also supports annotations and method throws.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Namespace struct {
	Pos       lexer.Position
	Language  string `"namespace" @Ident`
	Namespace string `@Ident ( @"." @Ident )*`
}

type Type struct {
	Pos     lexer.Position
	Name    string `@Ident ( @"." @Ident )*`
	TypeOne *Type  `( "<" @@ ( ","`
	TypeTwo *Type  `           @@ )? ">" )?`
}

type Annotation struct {
	Pos   lexer.Position
	Key   string   `@Ident ( @"." @Ident )*`
	Value *Literal `( "=" @@ )?`
}

type Field struct {
	Pos         lexer.Position
	ID          string        `@Number ":"`
	Requirement string        `@( "optional" | "required" )?`
	Type        *Type         `@@`
	Name        string        `@Ident`
	Default     *Literal      `( "=" @@ )?`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )? ";"?`
}

type Exception struct {
	Pos         lexer.Position
	Name        string        `"exception" @Ident "{"`
	Fields      []*Field      `@@ @@* "}"`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
}

type Struct struct {
	Pos         lexer.Position
	Union       bool          `( "struct" | @"union" )`
	Name        string        `@Ident "{"`
	Fields      []*Field      `@@* "}"`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
}

type Argument struct {
	Pos  lexer.Position
	ID   string `@Number ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Throw struct {
	Pos  lexer.Position
	ID   string `@Number ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Method struct {
	Pos         lexer.Position
	ReturnType  *Type         `@@`
	Name        string        `@Ident`
	Arguments   []*Argument   `"(" ( @@ ( "," @@ )* )? ")"`
	Throws      []*Throw      `( "throws" "(" @@ ( "," @@ )* ")" )?`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
}

type Service struct {
	Pos         lexer.Position
	Name        string        `"service" @Ident`
	Extends     string        `( "extends" @Ident ( @"." @Ident )* )?`
	Methods     []*Method     `"{" ( @@ ";"? )* "}"`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
}

// Literal is a "union" type, where only one matching value will be present.
type Literal struct {
	Pos       lexer.Position
	Str       *string    `  @String`
	Number    *float64   `| @Number`
	Bool      *string    `| @( "true" | "false" )`
	Reference *string    `| @Ident ( @"." @Ident )*`
	Minus     *Literal   `| "-" @@`
	List      []*Literal `| "[" ( @@ ","? )* "]"`
	Map       []*MapItem `| "{" ( @@ ","? )* "}"`
}

func (l *Literal) GoString() string {
	switch {
	case l.Str != nil:
		return fmt.Sprintf("%q", *l.Str)
	case l.Number != nil:
		return fmt.Sprintf("%v", *l.Number)
	case l.Bool != nil:
		return fmt.Sprintf("%v", *l.Bool)
	case l.Reference != nil:
		return fmt.Sprintf("%s", *l.Reference)
	case l.Minus != nil:
		return fmt.Sprintf("-%v", l.Minus)
	case l.List != nil:
		parts := []string{}
		for _, e := range l.List {
			parts = append(parts, e.GoString())
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	case l.Map != nil:
		parts := []string{}
		for _, e := range l.Map {
			parts = append(parts, e.GoString())
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	}
	panic("unsupported?")
}

type MapItem struct {
	Pos   lexer.Position
	Key   *Literal `@@ ":"`
	Value *Literal `@@`
}

func (m *MapItem) GoString() string {
	return fmt.Sprintf("%v: %v", m.Key, m.Value)
}

type Case struct {
	Pos         lexer.Position
	Name        string        `@Ident`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
	Value       *Literal      `( "=" @@ )? ( "," | ";" )?`
}

type Enum struct {
	Pos         lexer.Position
	Name        string        `"enum" @Ident "{"`
	Cases       []*Case       `@@* "}"`
	Annotations []*Annotation `( "(" @@ ( "," @@ )* ")" )?`
}

type Typedef struct {
	Pos  lexer.Position
	Type *Type  `"typedef" @@`
	Name string `@Ident`
}

type Const struct {
	Pos   lexer.Position
	Type  *Type    `"const" @@`
	Name  string   `@Ident`
	Value *Literal `"=" @@ ";"?`
}

type Entry struct {
	Pos        lexer.Position
	Includes   []string     `  "include" @String`
	Namespaces []*Namespace `| @@`
	Structs    []*Struct    `| @@`
	Exceptions []*Exception `| @@`
	Services   []*Service   `| @@`
	Enums      []*Enum      `| @@`
	Typedefs   []*Typedef   `| @@`
	Consts     []*Const     `| @@`
}

// Thrift files consist of a set of top-level directives and definitions.
//
// The grammar
type Thrift struct {
	Pos     lexer.Position
	Entries []*Entry `@@*`
}

var (
	def = lexer.MustSimple([]lexer.SimpleRule{
		{"Number", `\d+`},
		{"Ident", `\w+`},
		{"String", `"[^"]*"`},
		{"Whitespace", `\s+`},
		{"Punct", `[,.<>(){}=:]`},
		{"Comment", `//.*`},
	})
	parser = participle.MustBuild(&Thrift{},
		participle.Lexer(def),
		participle.Unquote(),
		participle.Elide("Whitespace"),
	)
)

func main() {
	var cli struct {
		Gen   bool     `help:"Generate lexer."`
		Files []string `help:"Thrift files."`
	}

	ctx := kong.Parse(&cli)

	if cli.Gen {
		w, err := os.Create("lexer_gen.go")
		ctx.FatalIfErrorf(err)
		defer w.Close()
		err = lexer.ExperimentalGenerateLexer(w, "main", def)
		ctx.FatalIfErrorf(err)
		return
	}

	for _, file := range cli.Files {
		thrift := &Thrift{}
		r, err := os.Open(file)
		ctx.FatalIfErrorf(err, "")
		err = parser.Parse("", r, thrift)
		ctx.FatalIfErrorf(err, "")
		repr.Println(thrift)
	}
}
