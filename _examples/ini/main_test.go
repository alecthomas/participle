package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	ini := &INI{}
	err := parser.ParseString("", `
global = 1

[section]
value = "str"
`, ini)
	assert.NoError(t, err)
	repr.Println(ini)
}
