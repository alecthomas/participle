package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	expr := &Expression{}
	err := parser.ParseString("", `1 + 2 / 3 * (1 + 2)`, expr)
	assert.NoError(t, err)
	repr.Println(expr)
}
