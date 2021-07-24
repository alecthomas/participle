// Package main implements a parser for FlatBuffers files.
package main

//go:generate antlr2participle FlatBuffers.g4 --name=flatbuffers

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2/_examples/flatbuffers/flatbuffers"
)

func main() {
	kingpin.Parse()

	res := &flatbuffers.Schema{}
	err := flatbuffers.Parser.Parse("", os.Stdin, res)
	kingpin.FatalIfError(err, "")

	repr.Println(res)
}
