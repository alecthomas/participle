package main

import "github.com/alecthomas/kong"

var (
	version string = "dev"
	cli     struct {
		Version kong.VersionFlag
		Gen     struct {
			Lexer genLexerCmd `cmd:""`
		} `cmd:"" help:"Generate code to accelerate Participle."`
	}
)

func main() {
	kctx := kong.Parse(&cli,
		kong.Description(`A command-line tool for Participle.`),
		kong.Vars{"version": version},
	)
	err := kctx.Run()
	kctx.FatalIfErrorf(err)
}
