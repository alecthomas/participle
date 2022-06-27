package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	sel, err := parser.ParseString("", `SELECT * FROM table WHERE attr = 10`)
	require.NoError(t, err)
	repr.Println(sel)
}
