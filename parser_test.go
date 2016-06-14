package parser

import (
	"testing"

	"github.com/alecthomas/assert"
)

func TestProductionReference(t *testing.T) {
	type testReference struct {
		A string `@Test`
	}

	_, err := Parse(&testReference{}, nil)
	assert.Error(t, err)
}

func TestParseScalar(t *testing.T) {
	type testScalar struct {
		A string `@"one"`
	}

	parser, err := Parse(&testScalar{}, nil)
	assert.NoError(t, err)

	actual := &testScalar{}
	err = parser.ParseString("one", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testScalar{"one"}, actual)
}

func TestParseGroup(t *testing.T) {
	type testGroup struct {
		A string `@("one" | "two")`
	}

	parser, err := Parse(&testGroup{}, nil)
	assert.NoError(t, err)

	actual := &testGroup{}
	err = parser.ParseString("one", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testGroup{"one"}, actual)

	actual = &testGroup{}
	err = parser.ParseString("two", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testGroup{"two"}, actual)
}

func TestParseAlternative(t *testing.T) {
	type testAlternative struct {
		A string `@"one" |`
		B string `@"two"`
	}

	parser, err := Parse(&testAlternative{}, nil)
	assert.NoError(t, err)

	actual := &testAlternative{}
	err = parser.ParseString("one", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testAlternative{A: "one"}, actual)

	actual = &testAlternative{}
	err = parser.ParseString("two", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testAlternative{B: "two"}, actual)
}

func TestParseSequence(t *testing.T) {
	type testSequence struct {
		A string `@"one"`
		B string `@"two"`
		C string `@"three"`
	}

	parser, err := Parse(&testSequence{}, nil)
	assert.NoError(t, err)

	actual := &testSequence{}
	expected := &testSequence{"one", "two", "three"}
	err = parser.ParseString("onetwothree", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	actual = &testSequence{}
	expected = &testSequence{}
	err = parser.ParseString("moo", actual)
	assert.Error(t, err)
	assert.Equal(t, expected, actual)
}

func TestNested(t *testing.T) {
	type nestedInner struct {
		B string `@"one"`
		C string `@"two"`
	}
	type testNested struct {
		A *nestedInner `@@`
	}

	type testAccumulateNested struct {
		A []*nestedInner `@@ { @@ }`
	}

	parser, err := Parse(&testNested{}, nil)
	assert.NoError(t, err)

	actual := &testNested{}
	expected := &testNested{A: &nestedInner{B: "one", C: "two"}}
	err = parser.ParseString("onetwo", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAccumulateNested(t *testing.T) {
	type nestedInner struct {
		B string `@"one"`
		C string `@"two"`
	}
	type testAccumulateNested struct {
		A []*nestedInner `@@ { @@ }`
	}

	parser, err := Parse(&testAccumulateNested{}, nil)
	assert.NoError(t, err)

	actual := &testAccumulateNested{}
	expected := &testAccumulateNested{A: []*nestedInner{{B: "one", C: "two"}, {B: "one", C: "two"}}}
	err = parser.ParseString("onetwoonetwo", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRepitition(t *testing.T) {
	type testRepitition struct {
		A []string `{ @"." }`
		B *string  `(@"b" |`
		C *string  ` @"c")`
	}

	parser, err := Parse(&testRepitition{}, nil)
	assert.NoError(t, err)

	actual := &testRepitition{}
	b := "b"
	c := "c"
	expected := &testRepitition{
		A: []string{".", ".", "."},
		B: &b,
	}
	err = parser.ParseString("...b", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	actual = &testRepitition{}
	expected = &testRepitition{
		A: []string{".", ".", "."},
		C: &c,
	}
	err = parser.ParseString("...c", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAccumulateString(t *testing.T) {
	type testAccumulateString struct {
		A string `@"." { @"." }`
	}

	parser, err := Parse(&testAccumulateString{}, nil)
	assert.NoError(t, err)

	actual := &testAccumulateString{}
	expected := &testAccumulateString{
		A: "...",
	}
	err = parser.ParseString("...", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
