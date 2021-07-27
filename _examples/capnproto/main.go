// Package main implements a parser for CapnProto files.
package main

//go:generate antlr2participle CapnProto.g4 --name=capnproto --explode-literals

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/_examples/capnproto/capnproto"
)

var Parser = participle.MustBuild(
	&capnproto.Document{},
	participle.Lexer(capnproto.Lexer),
	participle.UseLookahead(7), // Parsing fails with a lower lookahead.
)

func main() {
	kingpin.Parse()

	res := &capnproto.Document{}
	err := Parser.Parse("", os.Stdin, res)
	kingpin.FatalIfError(err, "")

	repr.Println(res)
}
