// Package main implements a parser for Thrift files (https://thrift.apache.org/)
//
// It parses namespaces, exceptions, services, structs, consts, typedefs and enums, but is easily
// extensible to more.
//
// It also supports annotations and method throws.
package main

import (
	"os"

	"github.com/alecthomas/participle"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	files = kingpin.Arg("thrift", "Thrift files.").Required().Strings()
)

type Namespace struct {
	Language  string `"namespace" @Ident`
	Namespace string `@Ident { @"." @Ident }`
}

type Type struct {
	Name    string `@Ident { @"." @Ident }`
	TypeOne *Type  `[ "<" @@ [ ","`
	TypeTwo *Type  `           @@ ] ">" ]`
}

type Annotation struct {
	Key   string   `@Ident { @"." @Ident }`
	Value *Literal `[ "=" @@ ]`
}

type Field struct {
	ID          string        `@Int ":"`
	Requirement string        `@[ "optional" | "required" ]`
	Type        *Type         `@@`
	Name        string        `@Ident`
	Default     *Literal      `[ "=" @@ ]`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ] [ ";" ]`
}

type Exception struct {
	Name        string        `"exception" @Ident "{"`
	Fields      []*Field      `@@ { @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Struct struct {
	Union       bool          `( "struct" | @"union" )`
	Name        string        `@Ident "{"`
	Fields      []*Field      `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Argument struct {
	ID   string `@Int ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Throw struct {
	ID   string `@Int ":"`
	Type *Type  `@@`
	Name string `@Ident`
}

type Method struct {
	ReturnType  *Type         `@@`
	Name        string        `@Ident`
	Arguments   []*Argument   `"(" [ @@ { "," @@ } ] ")"`
	Throws      []*Throw      `[ "throws" "(" @@ { "," @@ } ")" ]`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Service struct {
	Name        string        `"service" @Ident`
	Extends     string        `[ "extends" @Ident { @"." @Ident } ]`
	Methods     []*Method     `"{" { @@ [ ";" ] } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

// Literal is a "union" type, where only one matching value will be present.
type Literal struct {
	Str       *string    `  @String`
	Float     *float64   `| @Float`
	Int       *int64     `| @Int`
	Bool      *string    `| @( "true" | "false" )`
	Reference *string    `| @Ident { @"." @Ident }`
	Minus     *Literal   `| "-" @@`
	List      []*Literal `| "[" { @@ [ "," ] } "]"`
	Map       []*MapItem `| "{" { @@ [ "," ] } "}"`
}

type MapItem struct {
	Key   *Literal `@@ ":"`
	Value *Literal `@@`
}

type Case struct {
	Name        string        `@Ident`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
	Value       *Literal      `[ "=" @@ ] [ "," | ";" ]`
}

type Enum struct {
	Name        string        `"enum" @Ident "{"`
	Cases       []*Case       `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Typedef struct {
	Type *Type  `"typedef" @@`
	Name string `@Ident`
}

type Const struct {
	Type  *Type    `"const" @@`
	Name  string   `@Ident`
	Value *Literal `"=" @@ [ ";" ]`
}

// Thrift files consist of a set of top-level directives and definitions.
//
// The grammar
type Thrift struct {
	Includes   []string     `{ "include" @String`
	Namespaces []*Namespace `  | @@`
	Structs    []*Struct    `  | @@`
	Exceptions []*Exception `  | @@`
	Services   []*Service   `  | @@`
	Enums      []*Enum      `  | @@`
	Typedefs   []*Typedef   `  | @@`
	Consts     []*Const     `  | @@ }`
}

func main() {
	kingpin.Parse()

	parser, err := participle.Build(&Thrift{}, nil)
	kingpin.FatalIfError(err, "")

	for _, file := range *files {
		thrift := &Thrift{}
		r, err := os.Open(file)
		kingpin.FatalIfError(err, "")
		err = parser.Parse(r, thrift)
		kingpin.FatalIfError(err, "")
	}
}
