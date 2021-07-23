package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"
)

func TestExe(t *testing.T) {
	expr := &Expr{}
	err := parser.ParseString("", `1 + 2 / 3 * (1 + 2)`, expr)
	require.NoError(t, err)
	repr.Println(expr)
}
