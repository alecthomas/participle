package main

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	ast := &EBNF{}
	err := parser.ParseString("", `
Production  = name "=" [ Expression ] "." .
  Expression  = Alternative { "|" Alternative } .
  Alternative = Term { Term } .
  Term        = name | token [ "…" token ] | Group | Option | Repetition .
  Group       = "(" Expression ")" .
  Option      = "[" Expression "]" .
  Repetition  = "{" Expression "}" .`, ast)
	assert.NoError(t, err)
}
