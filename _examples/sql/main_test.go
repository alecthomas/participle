package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	sel := &Select{}
	err := parser.ParseString("", `SELECT * FROM table WHERE attr = 10`, sel)
	assert.NoError(t, err)
	repr.Println(sel)
}
