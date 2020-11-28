package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"
)

func TestExe(t *testing.T) {
	ini := &INI{}
	err := parser.ParseString("", `
global = 1

[section]
value = "str"
`, ini)
	require.NoError(t, err)
	repr.Println(ini)
}
