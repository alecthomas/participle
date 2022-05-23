package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	sel := &Select{}
	err := parser.ParseString("", `SELECT * FROM table WHERE attr = 10`, sel)
	require.NoError(t, err)
	repr.Println(sel)
}
