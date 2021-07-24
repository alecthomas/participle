package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2/_examples/json/json"
)

func TestExe(t *testing.T) {
	r, err := os.Open("testdata/github-webhook.json")
	require.NoError(t, err)

	res := &json.Json{}

	err = json.Parser.Parse("github-webhook.json", r, res)
	require.NoError(t, err)
}
