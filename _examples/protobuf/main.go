// nolint: govet, golint
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

type Proto struct {
	Pos lexer.Position

	Entries []*Entry `{ @@ { ";" } }`
}

type Entry struct {
	Pos lexer.Position

	Syntax  string   `  "syntax" "=" @String`
	Package string   `| "package" @(Ident { "." Ident })`
	Import  string   `| "import" @String`
	Message *Message `| @@`
	Service *Service `| @@`
	Enum    *Enum    `| @@`
	Option  *Option  `| "option" @@`
	Extend  *Extend  `| @@`
}

type Option struct {
	Pos lexer.Position

	Name  string  `( "(" @Ident @{ "." Ident } ")" | @Ident @{ "." @Ident } )`
	Attr  *string `[ "." @Ident { "." @Ident } ]`
	Value *Value  `"=" @@`
}

type Value struct {
	Pos lexer.Position

	String    *string  `  @String`
	Number    *float64 `| @Float`
	Int       *int64   `| @Int`
	Bool      *bool    `| (@"true" | "false")`
	Reference *string  `| @Ident @{ "." Ident }`
	Map       *Map     `| @@`
	Array     *Array   `| @@`
}

type Array struct {
	Pos lexer.Position

	Elements []*Value `"[" [ @@ { [ "," ] @@ } ] "]"`
}

type Map struct {
	Pos lexer.Position

	Entries []*MapEntry `"{" [ @@ { [ "," ] @@ } ] "}"`
}

type MapEntry struct {
	Pos lexer.Position

	Key   *Value `@@`
	Value *Value `[ ":" ] @@`
}

type Extensions struct {
	Pos lexer.Position

	Extensions []Range `"extensions" @@ { "," @@ }`
}

type Reserved struct {
	Pos lexer.Position

	Reserved []Range `"reserved" @@ { "," @@ }`
}

type Range struct {
	Ident string `  @String`
	Start int    `| ( @Int`
	End   *int   `  [ "to" ( @Int`
	Max   bool   `           | @"max" ) ] )`
}

type Extend struct {
	Pos lexer.Position

	Reference string   `"extend" @Ident { "." @Ident }`
	Fields    []*Field `"{" { @@ [ ";" ] } "}"`
}

type Service struct {
	Pos lexer.Position

	Name  string          `"service" @Ident`
	Entry []*ServiceEntry `"{" { @@ [ ";" ] } "}"`
}

type ServiceEntry struct {
	Pos lexer.Position

	Option *Option `  "option" @@`
	Method *Method `| @@`
}

type Method struct {
	Pos lexer.Position

	Name              string    `"rpc" @Ident`
	StreamingRequest  bool      `"(" [ @"stream" ]`
	Request           *Type     `    @@ ")"`
	StreamingResponse bool      `"returns" "(" [ @"stream" ]`
	Response          *Type     `              @@ ")"`
	Options           []*Option `[ "{" { "option" @@ ";" } "}" ]`
}

type Enum struct {
	Pos lexer.Position

	Name   string       `"enum" @Ident`
	Values []*EnumEntry `"{" { @@ { ";" } } "}"`
}

type EnumEntry struct {
	Pos lexer.Position

	Value  *EnumValue `  @@`
	Option *Option    `| "option" @@`
}

type EnumValue struct {
	Pos lexer.Position

	Key   string `@Ident`
	Value int    `"=" @( [ "-" ] Int )`

	Options []*Option `[ "[" @@ { "," @@ } "]" ]`
}

type Message struct {
	Pos lexer.Position

	Name    string          `"message" @Ident`
	Entries []*MessageEntry `"{" { @@ } "}"`
}

type MessageEntry struct {
	Pos lexer.Position

	Enum       *Enum       `( @@`
	Option     *Option     ` | "option" @@`
	Message    *Message    ` | @@`
	Oneof      *Oneof      ` | @@`
	Extend     *Extend     ` | @@`
	Reserved   *Reserved   ` | @@`
	Extensions *Extensions ` | @@`
	Field      *Field      ` | @@ ) { ";" }`
}

type Oneof struct {
	Pos lexer.Position

	Name    string        `"oneof" @Ident`
	Entries []*OneofEntry `"{" { @@ { ";" } } "}"`
}

type OneofEntry struct {
	Pos lexer.Position

	Field  *Field  `  @@`
	Option *Option `| "option" @@`
}

type Field struct {
	Pos lexer.Position

	Optional bool `[   @"optional"`
	Required bool `  | @"required"`
	Repeated bool `  | @"repeated" ]`

	Type *Type  `@@`
	Name string `@Ident`
	Tag  int    `"=" @Int`

	Options []*Option `[ "[" @@ { "," @@ } "]" ]`
}

type Scalar int

const (
	None Scalar = iota
	Double
	Float
	Int32
	Int64
	Uint32
	Uint64
	Sint32
	Sint64
	Fixed32
	Fixed64
	SFixed32
	SFixed64
	Bool
	String
	Bytes
)

var scalarToString = map[Scalar]string{
	None: "None", Double: "Double", Float: "Float", Int32: "Int32", Int64: "Int64", Uint32: "Uint32",
	Uint64: "Uint64", Sint32: "Sint32", Sint64: "Sint64", Fixed32: "Fixed32", Fixed64: "Fixed64",
	SFixed32: "SFixed32", SFixed64: "SFixed64", Bool: "Bool", String: "String", Bytes: "Bytes",
}

func (s Scalar) GoString() string { return scalarToString[s] }

var stringToScalar = map[string]Scalar{
	"double": Double, "float": Float, "int32": Int32, "int64": Int64, "uint32": Uint32, "uint64": Uint64,
	"sint32": Sint32, "sint64": Sint64, "fixed32": Fixed32, "fixed64": Fixed64, "sfixed32": SFixed32,
	"sfixed64": SFixed64, "bool": Bool, "string": String, "bytes": Bytes,
}

func (s *Scalar) Parse(lex lexer.PeekingLexer) error {
	token, err := lex.Peek(0)
	if err != nil {
		return err
	}
	v, ok := stringToScalar[token.Value]
	if !ok {
		return participle.NextMatch
	}
	_, err = lex.Next()
	if err != nil {
		return err
	}
	*s = v
	return nil
}

type Type struct {
	Pos lexer.Position

	Scalar    Scalar   `  @@`
	Map       *MapType `| @@`
	Reference string   `| @(Ident { "." Ident })`
}

type MapType struct {
	Pos lexer.Position

	Key   *Type `"map" "<" @@`
	Value *Type `"," @@ ">"`
}

var (
	parser = participle.MustBuild(&Proto{}, participle.UseLookahead(0))

	cli struct {
		Files []string `required existingfile arg help:"Protobuf files."`
	}
)

func main() {
	ctx := kong.Parse(&cli)

	for _, file := range cli.Files {
		fmt.Println(file)
		proto := &Proto{}
		r, err := os.Open(file)
		ctx.FatalIfErrorf(err, "")
		err = parser.Parse(r, proto)
		repr.Println(proto, repr.Hide(&lexer.Position{}))
		ctx.FatalIfErrorf(err, "")
	}
}
