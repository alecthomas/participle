package main

import (
	"encoding/json"
	"os"
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	r, err := os.Open("github-webhook.json")
	require.NoError(t, err)
	input := map[string]interface{}{}
	err = json.NewDecoder(r).Decode(&input)
	require.NoError(t, err)

	ast := pathExpr{}
	err = parser.ParseString(``, `check_run.check_suite.pull_requests[0].url`, &ast)
	require.NoError(t, err)

	result, err := match(input, ast)
	require.NoError(t, err)
	require.Equal(t, "https://api.github.com/repos/Codertocat/Hello-World/pulls/2", result)
}
