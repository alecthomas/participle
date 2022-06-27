package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	expr, err := parser.ParseString("", `1 + 2 / 3 * (1 + 2)`)
	repr.Println(expr)
	require.NoError(t, err)
}
