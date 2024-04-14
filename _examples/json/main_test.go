package main

import (
	"os"
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestParse(t *testing.T) {
	src, err := os.ReadFile("./test.json")
	require.NoError(t, err)
	_, err = Parse(src)
	require.NoError(t, err)
}
