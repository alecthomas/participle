package participle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/lexer"
)

func TestProductionCapture(t *testing.T) {
	type testCapture struct {
		A string `@Test`
	}

	_, err := Build(&testCapture{})
	require.Error(t, err)
}

func TestTermCapture(t *testing.T) {
	type grammar struct {
		A string `@{"."}`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{"..."}

	err := parser.ParseString("...", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestParseScalar(t *testing.T) {
	type grammar struct {
		A string `@"one"`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	err := parser.ParseString("one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{"one"}, actual)
}

func TestParseGroup(t *testing.T) {
	type grammar struct {
		A string `@("one" | "two")`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	err := parser.ParseString("one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{"one"}, actual)

	actual = &grammar{}
	err = parser.ParseString("two", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{"two"}, actual)
}

func TestParseAlternative(t *testing.T) {
	type grammar struct {
		A string `@"one" |`
		B string `@"two"`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	err := parser.ParseString("one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{A: "one"}, actual)

	actual = &grammar{}
	err = parser.ParseString("two", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{B: "two"}, actual)
}

func TestParseSequence(t *testing.T) {
	type grammar struct {
		A string `@"one"`
		B string `@"two"`
		C string `@"three"`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{"one", "two", "three"}
	err := parser.ParseString("one two three", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &grammar{}
	expected = &grammar{}
	err = parser.ParseString("moo", actual)
	require.Error(t, err)
	require.Equal(t, expected, actual)
}

func TestNested(t *testing.T) {
	type nestedInner struct {
		B string `@"one"`
		C string `@"two"`
	}

	type testNested struct {
		A *nestedInner `@@`
	}

	parser := mustTestParser(t, &testNested{})

	actual := &testNested{}
	expected := &testNested{A: &nestedInner{B: "one", C: "two"}}
	err := parser.ParseString("one two", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRepititionNoMatch(t *testing.T) {
	type grammar struct {
		A []string `{ @"." }`
	}
	parser := mustTestParser(t, &grammar{})

	expected := &grammar{}
	actual := &grammar{}
	err := parser.ParseString(``, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRepitition(t *testing.T) {
	type grammar struct {
		A []string `{ @"." }`
	}
	parser := mustTestParser(t, &grammar{})

	expected := &grammar{A: []string{".", ".", "."}}
	actual := &grammar{}
	err := parser.ParseString(`...`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestRepititionAcrossFields(t *testing.T) {
	type testRepitition struct {
		A []string `{ @"." }`
		B *string  `(@"b" |`
		C *string  ` @"c")`
	}

	parser := mustTestParser(t, &testRepitition{})

	b := "b"
	c := "c"

	actual := &testRepitition{}
	expected := &testRepitition{
		A: []string{".", ".", "."},
		B: &b,
	}
	err := parser.ParseString("...b", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &testRepitition{}
	expected = &testRepitition{
		A: []string{".", ".", "."},
		C: &c,
	}
	err = parser.ParseString("...c", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &testRepitition{}
	expected = &testRepitition{
		B: &b,
	}
	err = parser.ParseString("b", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

type Group struct {
	Expression *Expression `"(" @@ ")"`
}

type EBNFOption struct {
	Expression *Expression `"[" @@ "]"`
}

type Repetition struct {
	Expression *Expression `"{" @@ "}"`
}

type Literal struct {
	Start string `@String`
}

type Range struct {
	Start string `@String`
	End   string `"…" @String`
}

type Term struct {
	Name       string      `@Ident |`
	Literal    *Literal    `@@ |`
	Range      *Range      `@@ |`
	Group      *Group      `@@ |`
	Option     *EBNFOption `@@ |`
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
			{
				Name: "Production",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Name: "name"},
									{Literal: &Literal{Start: "="}},
									{
										Option: &EBNFOption{
											Expression: &Expression{
												Alternatives: []*Sequence{
													{
														Terms: []*Term{
															{Name: "Expression"},
														},
													},
												},
											},
										},
									},
									{Literal: &Literal{Start: "."}},
								},
							},
						},
					},
				},
			},
			{
				Name: "Expression",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Name: "Alternative"},
									{
										Repetition: &Repetition{
											Expression: &Expression{
												Alternatives: []*Sequence{
													{
														Terms: []*Term{
															{Literal: &Literal{Start: "|"}},
															{Name: "Alternative"},
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
			{
				Name: "Alternative",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Name: "Term"},
									{
										Repetition: &Repetition{
											Expression: &Expression{
												Alternatives: []*Sequence{
													{
														Terms: []*Term{
															{Name: "Term"},
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
			{
				Name: "Term",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{Terms: []*Term{{Name: "name"}}},
							{
								Terms: []*Term{
									{Name: "token"},
									{
										Option: &EBNFOption{
											Expression: &Expression{
												Alternatives: []*Sequence{
													{
														Terms: []*Term{
															{Literal: &Literal{Start: "…"}},
															{Name: "token"},
														},
													},
												},
											},
										},
									},
								},
							},
							{Terms: []*Term{{Literal: &Literal{Start: "@@"}}}},
							{Terms: []*Term{{Name: "Group"}}},
							{Terms: []*Term{{Name: "EBNFOption"}}},
							{Terms: []*Term{{Name: "Repetition"}}},
						},
					},
				},
			},
			{
				Name: "Group",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Literal: &Literal{Start: "("}},
									{Name: "Expression"},
									{Literal: &Literal{Start: ")"}},
								},
							},
						},
					},
				},
			},
			{
				Name: "EBNFOption",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Literal: &Literal{Start: "["}},
									{Name: "Expression"},
									{Literal: &Literal{Start: "]"}},
								},
							},
						},
					},
				},
			},
			{
				Name: "Repetition",
				Expression: []*Expression{
					{
						Alternatives: []*Sequence{
							{
								Terms: []*Term{
									{Literal: &Literal{Start: "{"}},
									{Name: "Expression"},
									{Literal: &Literal{Start: "}"}},
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
Term        = name | token [ "…" token ] | "@@" | Group | EBNFOption | Repetition .
Group       = "(" Expression ")" .
EBNFOption      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
`), actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestParseTokenCapture(t *testing.T) {
}

func TestParseOptional(t *testing.T) {
	type testOptional struct {
		A string `[ @"a" @"b" ]`
		B string `@"c"`
	}

	parser := mustTestParser(t, &testOptional{})

	expected := &testOptional{B: "c"}
	actual := &testOptional{}
	err := parser.ParseString(`c`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
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
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func mustTestParser(t *testing.T, grammar interface{}, options ...Option) *Parser {
	t.Helper()
	parser, err := Build(grammar, options...)
	require.NoError(t, err)
	return parser
}

func BenchmarkEBNFParser(b *testing.B) {
	parser, err := Build(&EBNF{})
	require.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		actual := &EBNF{}
		parser.ParseString(strings.TrimSpace(`
Production  = name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = name | token [ "…" token ] | "@@" | Group | EBNFOption | Repetition .
Group       = "(" Expression ")" .
EBNFOption      = "[" Expression "]" .
Repetition  = "{" Expression "}" .

`), actual)
	}
}

func TestRepeatAcrossFields(t *testing.T) {
	type grammar struct {
		A string `{ @("." ">") |`
		B string `  @("," "<") }`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{A: ".>.>.>.>", B: ",<,<,<"}

	err := parser.ParseString(".>,<.>.>,<.>,<", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestPosInjection(t *testing.T) {
	type subgrammar struct {
		Pos lexer.Position
		B   string `@{ "," }`
	}
	type grammar struct {
		Pos lexer.Position
		A   string      `@{ "." }`
		B   *subgrammar `@@`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{
		Pos: lexer.Position{
			Offset: 3,
			Line:   1,
			Column: 4,
		},
		A: "...",
		B: &subgrammar{
			B: ",,,",
			Pos: lexer.Position{
				Offset: 6,
				Line:   1,
				Column: 7,
			},
		},
	}

	err := parser.ParseString("   ...,,,", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

type parseableCount int

func (c *parseableCount) Capture(values []string) error {
	*c += parseableCount(len(values))
	return nil
}

func TestCaptureInterface(t *testing.T) {
	type grammar struct {
		Count parseableCount `{ @"a" }`
	}

	parser := mustTestParser(t, &grammar{})
	actual := &grammar{}
	expected := &grammar{Count: 3}
	err := parser.ParseString("a a a", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestLiteralTypeConstraint(t *testing.T) {
	type grammar struct {
		Literal string `@"123456":String`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{Literal: "123456"}
	err := parser.ParseString(`"123456"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	err = parser.ParseString(`123456`, actual)
	require.Error(t, err)
}

type nestedCapture struct {
	Tokens []string
}

func (n *nestedCapture) Capture(tokens []string) error {
	n.Tokens = tokens
	return nil
}

func TestStructCaptureInterface(t *testing.T) {
	type grammar struct {
		Capture *nestedCapture `@String`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{Capture: &nestedCapture{Tokens: []string{"hello"}}}
	err = parser.ParseString(`"hello"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

type parseableStruct struct {
	Tokens []string
}

func (p *parseableStruct) Parse(lex lexer.PeekingLexer) error {
	tokens, err := lexer.ConsumeAll(lex)
	if err != nil {
		return err
	}
	for _, t := range tokens {
		p.Tokens = append(p.Tokens, t.Value)
	}
	return nil
}

func TestParseable(t *testing.T) {
	type grammar struct {
		Inner *parseableStruct `@@`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{Inner: &parseableStruct{Tokens: []string{"hello", "123", "world", ""}}}
	err = parser.ParseString(`hello 123 "world"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIncrementInt(t *testing.T) {
	type grammar struct {
		Field int `@"." { @"." }`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{4}
	err = parser.ParseString(`. . . .`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIncrementUint(t *testing.T) {
	type grammar struct {
		Field uint `@"." { @"." }`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{4}
	err = parser.ParseString(`. . . .`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestIncrementFloat(t *testing.T) {
	type grammar struct {
		Field float32 `@"." { @"." }`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{4}
	err = parser.ParseString(`. . . .`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestStringConcat(t *testing.T) {
	type grammar struct {
		Field string `@"." { @"." }`
	}

	parser, err := Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{"...."}
	err = parser.ParseString(`. . . .`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestParseIntSlice(t *testing.T) {
	type grammar struct {
		Field []int `@Int { @Int }`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{[]int{1, 2, 3, 4}}
	err := parser.ParseString(`1 2 3 4`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestEmptyStructErrorsNotPanicsIssue21(t *testing.T) {
	type grammar struct {
		Foo struct{} `@@`
	}
	_, err := Build(&grammar{})
	require.Error(t, err)
}

func TestMultipleTokensIntoScalar(t *testing.T) {
	var grammar struct {
		Field int `@("-" Int)`
	}
	p, err := Build(&grammar)
	require.NoError(t, err)
	err = p.ParseString(`- 10`, &grammar)
	require.NoError(t, err)
	require.Equal(t, -10, grammar.Field)
}
