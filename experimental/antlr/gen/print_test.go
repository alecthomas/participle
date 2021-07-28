package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Credit to https://gist.github.com/getify/3667624

func TestDquoEscaper(t *testing.T) {
	tt := []struct {
		input  string
		output string
	}{
		{`ab`, `ab`},
		{`a"b`, `a\"b`},
		{`a\"b`, `a\"b`},
		{`a\\"b`, `a\\\"b`},
		{`a\\\"b`, `a\\\"b`},
		{`a"b"c`, `a\"b\"c`},
		{`a""b`, `a\"\"b`},
		{`""`, `\"\"`},
	}

	for _, test := range tt {
		assert.Equal(t, test.output, dquoEscaper.ReplaceAllString(test.input, "\\$1$2"))
	}
}
