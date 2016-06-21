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

func TestTermReference(t *testing.T) {
	type testTermReference struct {
		A string `@{"."}`
	}

	parser, err := Parse(&testTermReference{}, nil)
	assert.NoError(t, err)

	actual := &testTermReference{}
	expected := &testTermReference{"..."}

	err = parser.ParseString("...", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
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
	err = parser.ParseString("one two three", actual)
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
	err = parser.ParseString("one two", actual)
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
	err = parser.ParseString("one two one two", actual)
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
	actual = &testRepitition{}
	expected = &testRepitition{
		C: &c,
	}
	err = parser.ParseString("c", actual)
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

func TestRange(t *testing.T) {
	type testRange struct {
		A string `@"!" … "/"`
	}

	parser, err := Parse(&testRange{}, nil)
	assert.NoError(t, err)

	actual := &testRange{}
	expected := &testRange{"+"}
	err = parser.ParseString("+", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	err = parser.ParseString("1", actual)
	assert.Error(t, err)
}

type Group struct {
	Expression *Expression `"(" @@ ")"`
}

type Option struct {
	Expression *Expression `"[" @@ "]"`
}

type Repetition struct {
	Expression *Expression `"{" @@ "}"`
}

type TokenRange struct {
	Start string  `@String` // Lexer token "String"
	End   *string `[ "…" @String ]`
}

type Term struct {
	Name       *string     `@Ident |`
	TokenRange *TokenRange `@@ |`
	Group      *Group      `@@ |`
	Option     *Option     `@@ |`
	Repetition *Repetition `@@`
}

type Expression struct {
	Alternatives []*Term `@@ { "|" @@ }`
}

type Production struct {
	Name       string        `@Ident "="`
	Expression []*Expression `@@ { @@ } "."`
}

type EBNF struct {
	Productions []*Production `{ @@ }`
}

func TestEBNF(t *testing.T) {
	parser, err := Parse(&EBNF{}, nil)
	assert.NoError(t, err)

	expected := &EBNF{
		Productions: []*Production{
			&Production{
				Name: "A",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Term{
							{TokenRange: &TokenRange{Start: "a"}},
						},
					},
					&Expression{
						Alternatives: []*Term{
							{TokenRange: &TokenRange{Start: "b"}},
						},
					},
					&Expression{
						Alternatives: []*Term{
							&Term{
								Option: &Option{
									Expression: &Expression{
										Alternatives: []*Term{
											&Term{
												TokenRange: &TokenRange{
													Start: "c",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	actual := &EBNF{}

	err = parser.ParseString(`A = "a" "b" [ "c" ].`, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseType(t *testing.T) {
}

func TestParseExpression(t *testing.T) {
}

func TestParseTokenReference(t *testing.T) {
}

func TestParseOptional(t *testing.T) {
	type testOptional struct {
		A string `@[ "a" "b" ]`
		B string `@"c"`
	}

	parser, err := Parse(&testOptional{}, nil)
	assert.NoError(t, err)

	expected := &testOptional{B: "c"}
	actual := &testOptional{}
	err = parser.ParseString(`c`, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseRepitition(t *testing.T) {
}

func TestParseQuotedStringOrRange(t *testing.T) {
}

func TestParseQuotedString(t *testing.T) {
}

func TestHello(t *testing.T) {
	type testHello struct {
		Hello string `@"hello"`
		To    string `@String`
	}

	parser, err := Parse(&testHello{}, nil)
	assert.NoError(t, err)

	expected := &testHello{"hello", "Bobby Brown"}
	actual := &testHello{}
	err = parser.ParseString(`hello "Bobby Brown"`, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
