package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	_, err := parser.ParseString("", `
Production  = name "=" [ Expression ] "." .
  Expression  = Alternative { "|" Alternative } .
  Alternative = Term { Term } .
  Term        = name | token [ "…" token ] | Group | Option | Repetition .
  Group       = "(" Expression ")" .
  Option      = "[" Expression "]" .
  Repetition  = "{" Expression "}" .`)
	require.NoError(t, err)
}
