package parser

import (
	"testing"

	"github.com/alecthomas/assert"
)

func TestParseTagSequence(t *testing.T) {
	actual := parseTag(`"a" "b"`)
	expected := expression{
		alternative{str("a"), str("b")},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagRange(t *testing.T) {
	actual := parseTag(`"a" … "b"`)
	expected := expression{
		alternative{srange{str("a"), str("b")}},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagAlternative(t *testing.T) {
	actual := parseTag(`"a" | "b"`)
	expected := expression{
		alternative{str("a")},
		alternative{str("b")},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagOptional(t *testing.T) {
	actual := parseTag(`[ "a" ]`)
	expected := expression{alternative{
		optional{{str("a")}},
	}}
	assert.Equal(t, expected, actual)
}

func TestParseTagGroup(t *testing.T) {
	actual := parseTag(`( "a" @ )`)
	expected := expression{
		alternative{
			group{
				alternative{str("a"), self{}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagRepitition(t *testing.T) {
	actual := parseTag(`{ "a" | @ }`)
	expected := expression{
		alternative{
			repitition{
				alternative{str("a")},
				alternative{self{}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestParseComplexTag(t *testing.T) {
	actual := parseTag(`("a"…"z" | "A"…"Z" | "_") {"a"…"z" | "A"…"Z" | "0"…"9" | "_"}`)
	expected := expression{
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
	}
	assert.Equal(t, expected, actual)
}
