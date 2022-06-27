package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	ini, err := parser.ParseString("", `
global = 1

[section]
value = "str"
`)
	require.NoError(t, err)
	repr.Println(ini)
}
