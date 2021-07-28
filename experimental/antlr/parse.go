package antlr

import (
	"io"

	"github.com/alecthomas/participle/v2/experimental/antlr/ast"
)

// Parse generates a walkable AST from an Antlr grammar file.
func Parse(filename string, r io.Reader) (dst *ast.AntlrFile, err error) {
	dst = &ast.AntlrFile{}
	err = ast.Parser.Parse(filename, r, dst)
	return
}
