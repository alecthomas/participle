// Package main generates a Participle parser from Antlr4 grammar files.
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/alecthomas/participle/v2/antlr"
	"github.com/alecthomas/participle/v2/antlr/ast"
)

var cli struct {
	Grammar []string `arg required help:"The .g4 files to process.  Globs accepted."`
	Name    string   `short:"n" help:"Override the generated package name."`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Description("Generate a Participle parser from Antlr4 grammar files."),
		kong.UsageOnError(),
	)

	var parsed *ast.AntlrFile
	for _, pat := range cli.Grammar {
		matches, err := filepath.Glob(pat)
		ctx.FatalIfErrorf(err)

		for _, m := range matches {
			ctx.Printf("Reading: %s", m)
			fd, err := os.Open(m)
			ctx.FatalIfErrorf(err)

			a, err := antlr.Parse(m, fd)
			ctx.FatalIfErrorf(err)

			err = fd.Close()
			ctx.FatalIfErrorf(err)

			parsed, err = parsed.Merge(a)
			ctx.FatalIfErrorf(err)
		}
	}

	buf := new(bytes.Buffer)
	err := antlr.ParticipleFromAntlr(parsed, buf)
	ctx.FatalIfErrorf(err)

	name := parsed.Grammar.Name
	if cli.Name != "" {
		name = cli.Name
	}

	err = os.MkdirAll(name, 0777)
	ctx.FatalIfErrorf(err)

	fd, err := os.Create(filepath.Join(name, "parser.go"))
	ctx.FatalIfErrorf(err)
	defer func() {
		ctx.FatalIfErrorf(fd.Close())
	}()

	_, err = io.Copy(fd, buf)
	ctx.FatalIfErrorf(err)
}
