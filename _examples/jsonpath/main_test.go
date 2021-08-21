package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	r, err := os.Open("github-webhook.json")
	assert.NoError(t, err)
	input := map[string]interface{}{}
	err = json.NewDecoder(r).Decode(&input)
	assert.NoError(t, err)

	ast := pathExpr{}
	err = parser.ParseString(``, `check_run.check_suite.pull_requests[0].url`, &ast)
	assert.NoError(t, err)

	result, err := match(input, ast)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.github.com/repos/Codertocat/Hello-World/pulls/2", result)
}
