// Package main implements a parser for JSON.
package main

//go:generate antlr2participle json.g4 --name=json

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2/_examples/json/json"
)

func main() {
	kingpin.Parse()

	j := &json.Json{}
	err := json.Parser.Parse("", os.Stdin, j)
	kingpin.FatalIfError(err, "")

	repr.Println(j)
}
