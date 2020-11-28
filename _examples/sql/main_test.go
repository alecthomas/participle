package main

import (
	"testing"

	"github.com/alecthomas/repr"
	"github.com/stretchr/testify/require"
)

func TestExe(t *testing.T) {
	sel := &Select{}
	err := parser.ParseString("", `SELECT * FROM table WHERE attr = 10`, sel)
	require.NoError(t, err)
	repr.Println(sel)
}
