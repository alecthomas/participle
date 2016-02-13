package parser

import (
	"testing"

	"github.com/alecthomas/assert"
)

func TestParseTagSequence(t *testing.T) {
	actual := parseTag(`"a" "b"`)
	expected := &expression{
		[]*alternative{
			{[]interface{}{&quotedString{"a"}, &quotedString{"b"}}},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagRange(t *testing.T) {
	actual := parseTag(`"a" … "b"`)
	expected := &expression{
		[]*alternative{
			{[]interface{}{
				&stringRange{&quotedString{"a"}, &quotedString{"b"}}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagAlternative(t *testing.T) {
	actual := parseTag(`"a" | "b"`)
	expected := &expression{
		[]*alternative{
			{[]interface{}{&quotedString{"a"}}},
			{[]interface{}{&quotedString{"b"}}},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagOptional(t *testing.T) {
	actual := parseTag(`[ "a" ]`)
	expected := &expression{
		[]*alternative{{
			[]interface{}{
				&optional{
					&expression{
						[]*alternative{
							{[]interface{}{&quotedString{"a"}}},
						},
					},
				},
			},
		}},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagGroup(t *testing.T) {
	actual := parseTag(`( "a" @ )`)
	expected := &expression{
		[]*alternative{{
			[]interface{}{
				&group{
					&expression{
						[]*alternative{
							{[]interface{}{&quotedString{"a"}, &self{}}},
						},
					},
				},
			},
		}},
	}
	assert.Equal(t, expected, actual)
}

func TestParseTagRepitition(t *testing.T) {
	actual := parseTag(`{ "a" | @ }`)
	expected := &expression{
		[]*alternative{{
			[]interface{}{
				&repitition{
					&expression{
						[]*alternative{
							{[]interface{}{&quotedString{"a"}}},
							{[]interface{}{&self{}}},
						},
					},
				},
			},
		}},
	}
	assert.Equal(t, expected, actual)
}
