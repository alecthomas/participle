package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	expr, err := parser.ParseString("", `1 + 2 / 3 * (1 + 2)`)
	repr.Println(expr)
	require.NoError(t, err)
}

func toPtr[T any](x T) *T {
	return &x
}

func TestExe_BoolFalse(t *testing.T) {
	got, err := parser.ParseString("", `1 + false`)

	expected := &Expression{
		Equality: &Equality{
			Comparison: &Comparison{
				Addition: &Addition{
					Multiplication: &Multiplication{
						Unary: &Unary{
							Primary: &Primary{
								Number: toPtr(float64(1)),
							},
						},
					},
					Op: "+",
					Next: &Addition{
						Multiplication: &Multiplication{
							Unary: &Unary{
								Primary: &Primary{
									Bool: toPtr(Boolean(false)),
								},
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, err)
	require.Equal(t, expected, got)
}
