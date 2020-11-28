package main

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
}
