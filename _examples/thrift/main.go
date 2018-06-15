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

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

var (
	files = kingpin.Arg("thrift", "Thrift files.").Required().Strings()
)

type Namespace struct {
	Pos       lexer.Position
	Language  string `"namespace" @Ident`
	Namespace string `@Ident { @"." @Ident }`
}

type Type struct {
	Pos     lexer.Position
	Name    string `@Ident { @"." @Ident }`
	TypeOne *Type  `[ "<" @@ [ ","`
	TypeTwo *Type  `           @@ ] ">" ]`
}

type Annotation struct {
	Pos   lexer.Position
	Key   string   `@Ident { @"." @Ident }`
	Value *Literal `[ "=" @@ ]`
}

type Field struct {
	Pos         lexer.Position
	ID          string        `@Int ":"`
	Requirement string        `@[ "optional" | "required" ]`
	Type        *Type         `@@`
	Name        string        `@Ident`
	Default     *Literal      `[ "=" @@ ]`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ] [ ";" ]`
}

type Exception struct {
	Pos         lexer.Position
	Name        string        `"exception" @Ident "{"`
	Fields      []*Field      `@@ { @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Struct struct {
	Pos         lexer.Position
	Union       bool          `( "struct" | @"union" )`
	Name        string        `@Ident "{"`
	Fields      []*Field      `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Argument struct {
	Pos  lexer.Position
	ID   string `@Int ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Throw struct {
	Pos  lexer.Position
	ID   string `@Int ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Method struct {
	Pos         lexer.Position
	ReturnType  *Type         `@@`
	Name        string        `@Ident`
	Arguments   []*Argument   `"(" [ @@ { "," @@ } ] ")"`
	Throws      []*Throw      `[ "throws" "(" @@ { "," @@ } ")" ]`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Service struct {
	Pos         lexer.Position
	Name        string        `"service" @Ident`
	Extends     string        `[ "extends" @Ident { @"." @Ident } ]`
	Methods     []*Method     `"{" { @@ [ ";" ] } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

// Literal is a "union" type, where only one matching value will be present.
type Literal struct {
	Pos       lexer.Position
	Str       *string    `  @String`
	Float     *float64   `| @Float`
	Int       *int64     `| @Int`
	Bool      *string    `| @( "true" | "false" )`
	Reference *string    `| @Ident { @"." @Ident }`
	Minus     *Literal   `| "-" @@`
	List      []*Literal `| "[" { @@ [ "," ] } "]"`
	Map       []*MapItem `| "{" { @@ [ "," ] } "}"`
}

func (l *Literal) GoString() string {
	switch {
	case l.Str != nil:
		return fmt.Sprintf("%q", *l.Str)
	case l.Float != nil:
		return fmt.Sprintf("%v", *l.Float)
	case l.Int != nil:
		return fmt.Sprintf("%v", *l.Int)
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
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
	Value       *Literal      `[ "=" @@ ] [ "," | ";" ]`
}

type Enum struct {
	Pos         lexer.Position
	Name        string        `"enum" @Ident "{"`
	Cases       []*Case       `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
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
	Value *Literal `"=" @@ [ ";" ]`
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
	Entries []*Entry `{ @@ }`
}

func main() {
	kingpin.Parse()

	parser, err := participle.Build(&Thrift{})
	kingpin.FatalIfError(err, "")

	for _, file := range *files {
		thrift := &Thrift{}
		r, err := os.Open(file)
		kingpin.FatalIfError(err, "")
		err = parser.Parse(r, thrift)
		kingpin.FatalIfError(err, "")
		repr.Println(thrift)
	}
}
