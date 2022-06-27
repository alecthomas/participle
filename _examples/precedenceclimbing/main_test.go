package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	actual, err := parser.ParseString("", `1 + 2 - 3 * (4 + 2)`)
	require.NoError(t, err)
	expected := expr(
		expr(intp(1), "+", intp(2)),
		"-",
		expr(intp(3),
			"*",
			expr(intp(4), "+", intp(2))))
	require.Equal(t, expected, actual)
}

func expr(l *Expr, op string, r *Expr) *Expr { return &Expr{Left: l, Op: op, Right: r} }
func intp(n int) *Expr                       { return &Expr{Terminal: &n} }
