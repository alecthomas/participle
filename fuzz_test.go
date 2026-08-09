package participle_test

import (
	"testing"

	"github.com/alecthomas/participle/v2"
)

// FuzzParse exercises the participle parser itself with arbitrary input,
// ensuring that no grammar panics or hangs on malformed input.
//
// Run with: go test -fuzz=FuzzParse -fuzztime 30s ./...
func FuzzParse(f *testing.F) {
	type grammar struct {
		Expr struct {
			Left  int      `@Int`
			Op    string   `@("+" | "-" | "*" | "/")`
			Right []int    `(@Int)*`
		} `@@`
	}
	parser, err := participle.Build[grammar]()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range []string{
		"1+2",
		"",
		"(",
		"1 / 0",
		"\xff\xfe",
		"abc",
		"1+-2*3/4",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		//nolint:errcheck // err is the expected outcome of fuzzing.
		_, _ = parser.ParseString("", in)
	})
}