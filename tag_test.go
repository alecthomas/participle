package parser

import (
	"testing"

	"github.com/alecthomas/assert"
)

var tagParserTestData = []struct {
	actual   string
	expected expression
}{
	{
		`"a" "b"`,
		expression{
			alternative{str("a"), str("b")},
		},
	},
	{
		`"a" … "b"`,
		expression{
			alternative{srange{str("a"), str("b")}},
		},
	},
	{
		`"a" | "b"`,
		expression{
			alternative{str("a")},
			alternative{str("b")},
		},
	},
	{
		`[ "a" ]`,
		expression{
			alternative{optional{{str("a")}}},
		},
	},
	{
		`( "a" @ )`,
		expression{
			alternative{
				group{
					alternative{str("a"), self{}},
				},
			},
		},
	},
	{
		`{ "a" | @ }`,
		expression{
			alternative{
				repitition{
					alternative{str("a")},
					alternative{self{}},
				},
			},
		},
	},
	{
		`("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`,
		expression{
			alternative{
				group{
					alternative{srange{start: str("a"), end: str("z")}},
					alternative{srange{start: str("A"), end: str("Z")}},
					alternative{str("_")},
				},
				repitition{
					alternative{srange{start: str("a"), end: str("z")}},
					alternative{srange{start: str("A"), end: str("Z")}},
					alternative{srange{start: str("0"), end: str("9")}},
					alternative{str("_")},
				},
			},
		},
	},
}

func TestTagParser(t *testing.T) {
	for _, data := range tagParserTestData {
		actual := parseTag(data.actual)
		assert.Equal(t, data.expected, actual)
	}
}
