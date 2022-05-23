package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	expr := &Expression{}
	err := parser.ParseString("", `1 + 2 / 3 * (1 + 2)`, expr)
	require.NoError(t, err)
	repr.Println(expr)
}
