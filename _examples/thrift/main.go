// Package main implements a parser for Thrift files (https://thrift.apache.org/)
//
// It parses namespaces, exceptions, services, structs, and enums, but is easily extensible to more.
//
// It also supports annotations and method throws.
package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"gopkg.in/alecthomas/kingpin.v2"
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
	Requirement string        `@( "optional" | "required" )`
	Type        *Type         `@@`
	Name        string        `@Ident`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Exception struct {
	Name        string        `"exception" @Ident "{"`
	Fields      []*Field      `@@ { @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

type Struct struct {
	Name        string        `"struct" @Ident "{"`
	Fields      []*Field      `@@ { @@ } "}"`
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
	Name        string        `"service" @Ident "{"`
	Methods     []*Method     `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
}

// Literal is a "union" type, where only one matching value will be present.
type Literal struct {
	Str   *string  `  @String`
	Float *float64 `| @Float`
	Int   *int64   `| @Int`
	Bool  *string  `| @( "true" | "false" )`
}

type Case struct {
	Name        string        `@Ident`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
	Value       *Literal      `[ "=" @@ ]`
}

type Enum struct {
	Name        string        `"enum" @Ident "{"`
	Cases       []*Case       `{ @@ } "}"`
	Annotations []*Annotation `[ "(" @@ { "," @@ } ")" ]`
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
	Enums      []*Enum      `  | @@ }`
}

func main() {
	kingpin.Parse()

	parser, err := participle.Parse(&Thrift{}, nil)
	kingpin.FatalIfError(err, "")

	thrift := &Thrift{}
	err = parser.ParseString(strings.TrimSpace(`

namespace go user
namespace py gen.user
namespace java org.swapoff.user

include "../common/base.thrift"
include "profile.thrift"

enum Enum {
	AUTO (auto)
	VALUE = 1.3
	ANOTHER = 2
}

struct User {
	1: optional list<profile.Profile> name (inject="profile")
}

exception Exception {
	1: optional string message
}

exception Bad {
	1: optional string message
}

service Service {
 	User someMethod(1: User strct)
 	    throws (1: Exception exc, 2: Bad exc)
 	    (url="/method")
 	User anotherMethod(1: string str)
 	    throws (1: Exception exc, 2: Bad exc)
 	    (url="/method")
} (url="/service")

`), thrift)
	kingpin.FatalIfError(err, "")

	json.NewEncoder(os.Stdout).Encode(thrift)
}
