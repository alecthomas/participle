// Package main implements a parser for CSV files.
package main

//go:generate antlr2participle CSV.g4 --name=csv

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/_examples/csv/csv"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	Lexer  = lexer.MustStateful(csv.Rules)
	Parser = participle.MustBuild(
		&csv.CsvFile{},
		participle.Lexer(Lexer),
		participle.UseLookahead(2),
	)
)

func main() {
	kingpin.Parse()

	res := &csv.CsvFile{}
	err := Parser.Parse("", os.Stdin, res)
	kingpin.FatalIfError(err, "")

	repr.Println(res)
}
