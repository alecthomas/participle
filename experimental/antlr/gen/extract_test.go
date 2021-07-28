package gen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtract(t *testing.T) {
	tt := []struct {
		structs []*Struct
		result  string
	}{
		{
			structs: []*Struct{
				{
					Name: "Foo",
					Fields: StructFields{
						{
							Name:    "A",
							RawType: "*string",
						},
						{
							Name: "B",
							SubType: &Struct{
								Name: "Bar",
								Fields: StructFields{
									{
										Name:    "C",
										RawType: "bool",
									},
								},
							},
						},
					},
				},
				{
					Name: "Baz",
					Fields: StructFields{
						{
							Name:    "D",
							RawType: "*string",
						},
						{
							Name: "E",
							SubType: &Struct{
								Name: "Bang",
								Fields: StructFields{
									{
										Name:    "F",
										RawType: "bool",
									},
								},
							},
						},
					},
				},
			},
			result: "type Foo struct {\nA *string ``\nB []*Bar ``\n}\ntype Bar struct {\nC bool ``\n}\ntype Baz struct {\nD *string ``\nE []*Bang ``\n}\ntype Bang struct {\nF bool ``\n}",
		},
	}

	for _, test := range tt {
		extracted := NewTypeExtractor(test.structs).Extract()
		results := make([]string, len(extracted))
		for i, ex := range extracted {
			results[i] = NewPrinter(false).Visit(ex)
		}

		require.Equal(t, test.result, strings.Join(results, "\n"))
	}
}
