package parser

import (
	"strings"
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

	parser := mustTestParser(t, &testTermReference{})

	actual := &testTermReference{}
	expected := &testTermReference{"..."}

	err := parser.ParseString("...", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseScalar(t *testing.T) {
	type testScalar struct {
		A string `@"one"`
	}

	parser := mustTestParser(t, &testScalar{})

	actual := &testScalar{}
	err := parser.ParseString("one", actual)
	assert.NoError(t, err)
	assert.Equal(t, &testScalar{"one"}, actual)
}

func TestParseGroup(t *testing.T) {
	type testGroup struct {
		A string `@("one" | "two")`
	}

	parser := mustTestParser(t, &testGroup{})

	actual := &testGroup{}
	err := parser.ParseString("one", actual)
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

	parser := mustTestParser(t, &testAlternative{})

	actual := &testAlternative{}
	err := parser.ParseString("one", actual)
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

	parser := mustTestParser(t, &testSequence{})

	actual := &testSequence{}
	expected := &testSequence{"one", "two", "three"}
	err := parser.ParseString("one two three", actual)
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

	parser := mustTestParser(t, &testNested{})

	actual := &testNested{}
	expected := &testNested{A: &nestedInner{B: "one", C: "two"}}
	err := parser.ParseString("one two", actual)
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

	parser := mustTestParser(t, &testAccumulateNested{})

	actual := &testAccumulateNested{}
	expected := &testAccumulateNested{A: []*nestedInner{{B: "one", C: "two"}, {B: "one", C: "two"}}}
	err := parser.ParseString("one two one two", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRepitition(t *testing.T) {
	type testRepitition struct {
		A []string `{ @"." }`
		B *string  `(@"b" |`
		C *string  ` @"c")`
	}

	parser := mustTestParser(t, &testRepitition{})

	actual := &testRepitition{}
	b := "b"
	c := "c"
	expected := &testRepitition{
		A: []string{".", ".", "."},
		B: &b,
	}
	err := parser.ParseString("...b", actual)
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

	parser := mustTestParser(t, &testAccumulateString{})

	actual := &testAccumulateString{}
	expected := &testAccumulateString{
		A: "...",
	}
	err := parser.ParseString("...", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRange(t *testing.T) {
	type testRange struct {
		A string `@"!" … "/"`
	}

	parser := mustTestParser(t, &testRange{})

	actual := &testRange{}
	expected := &testRange{"+"}
	err := parser.ParseString("+", actual)
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

type Literal struct {
	Start string  `@String` // Lexer token "String"
	End   *string `[ "…" @String ]`
}

type Term struct {
	Name       string      `@Ident |`
	Literal    *Literal    `@@ |`
	Group      *Group      `@@ |`
	Option     *Option     `@@ |`
	Repetition *Repetition `@@`
}

type Sequence struct {
	Terms []*Term `@@ { @@ }`
}

type Expression struct {
	Alternatives []*Sequence `@@ { "|" @@ }`
}

type Production struct {
	Name       string        `@Ident "="`
	Expression []*Expression `@@ { @@ } "."`
}

type EBNF struct {
	Productions []*Production `{ @@ }`
}

func TestEBNF(t *testing.T) {
	parser := mustTestParser(t, &EBNF{})

	expected := &EBNF{
		Productions: []*Production{
			&Production{
				Name: "Production",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Name: "name"},
									&Term{Literal: &Literal{Start: "="}},
									&Term{
										Option: &Option{
											Expression: &Expression{
												Alternatives: []*Sequence{
													&Sequence{
														Terms: []*Term{
															&Term{Name: "Expression"},
														},
													},
												},
											},
										},
									},
									&Term{Literal: &Literal{Start: "."}},
								},
							},
						},
					},
				},
			},
			&Production{
				Name: "Expression",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Name: "Alternative"},
									&Term{
										Repetition: &Repetition{
											Expression: &Expression{
												Alternatives: []*Sequence{
													&Sequence{
														Terms: []*Term{
															&Term{Literal: &Literal{Start: "|"}},
															&Term{Name: "Alternative"},
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
			},
			&Production{
				Name: "Alternative",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Name: "Term"},
									&Term{
										Repetition: &Repetition{
											Expression: &Expression{
												Alternatives: []*Sequence{
													&Sequence{
														Terms: []*Term{
															&Term{Name: "Term"},
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
			},
			&Production{
				Name: "Term",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{Terms: []*Term{&Term{Name: "name"}}},
							&Sequence{
								Terms: []*Term{
									&Term{Name: "token"},
									&Term{
										Option: &Option{
											Expression: &Expression{
												Alternatives: []*Sequence{
													&Sequence{
														Terms: []*Term{
															&Term{Literal: &Literal{Start: "…"}},
															&Term{Name: "token"},
														},
													},
												},
											},
										},
									},
								},
							},
							&Sequence{Terms: []*Term{&Term{Literal: &Literal{Start: "@@"}}}},
							&Sequence{Terms: []*Term{&Term{Name: "Group"}}},
							&Sequence{Terms: []*Term{&Term{Name: "Option"}}},
							&Sequence{Terms: []*Term{&Term{Name: "Repetition"}}},
						},
					},
				},
			},
			&Production{
				Name: "Group",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Literal: &Literal{Start: "("}},
									&Term{Name: "Expression"},
									&Term{Literal: &Literal{Start: ")"}},
								},
							},
						},
					},
				},
			},
			&Production{
				Name: "Option",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Literal: &Literal{Start: "["}},
									&Term{Name: "Expression"},
									&Term{Literal: &Literal{Start: "]"}},
								},
							},
						},
					},
				},
			},
			&Production{
				Name: "Repetition",
				Expression: []*Expression{
					&Expression{
						Alternatives: []*Sequence{
							&Sequence{
								Terms: []*Term{
									&Term{Literal: &Literal{Start: "{"}},
									&Term{Name: "Expression"},
									&Term{Literal: &Literal{Start: "}"}},
								},
							},
						},
					},
				},
			},
		},
	}
	actual := &EBNF{}
	err := parser.ParseString(strings.TrimSpace(`
Production  = name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = name | token [ "…" token ] | "@@" | Group | Option | Repetition .
Group       = "(" Expression ")" .
Option      = "[" Expression "]" .
Repetition  = "{" Expression "}" .

`), actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseType(t *testing.T) {
}

func TestParseExpression(t *testing.T) {
	type testNestA struct {
		A string `":" @{ "a" }`
	}
	type testNestB struct {
		B string `";" @{ "b" }`
	}
	type testExpression struct {
		A *testNestA `@@ |`
		B *testNestB `@@`
	}

	parser := mustTestParser(t, &testExpression{})

	expected := &testExpression{
		B: &testNestB{
			B: "b",
		},
	}
	actual := &testExpression{}
	err := parser.ParseString(";b", actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseTokenReference(t *testing.T) {
}

func TestParseOptional(t *testing.T) {
	type testOptional struct {
		A string `@[ "a" "b" ]`
		B string `@"c"`
	}

	parser := mustTestParser(t, &testOptional{})

	expected := &testOptional{B: "c"}
	actual := &testOptional{}
	err := parser.ParseString(`c`, actual)
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

	parser := mustTestParser(t, &testHello{})

	expected := &testHello{"hello", "Bobby Brown"}
	actual := &testHello{}
	err := parser.ParseString(`hello "Bobby Brown"`, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func mustTestParser(t *testing.T, grammar interface{}) *Parser {
	parser, err := Parse(grammar, nil)
	assert.NoError(t, err)
	return parser
}
