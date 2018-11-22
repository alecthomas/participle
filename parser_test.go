package participle

import (
	"fmt"
	"math"
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
	source := strings.TrimSpace(`
Production  = name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = name | token [ "…" token ] | "@@" | Group | EBNFOption | Repetition .
Group       = "(" Expression ")" .
EBNFOption      = "[" Expression "]" .
Repetition  = "{" Expression "}" .

`)
	for i := 0; i < b.N; i++ {
		actual := &EBNF{}
		_ = parser.ParseString(source, actual)
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

type posMixin struct {
	Pos lexer.Position
}

func TestMixinPosIsPopulated(t *testing.T) {
	var grammar struct {
		posMixin

		Int int `@Int`
	}

	p := mustTestParser(t, &grammar)
	err := p.ParseString("10", &grammar)
	require.NoError(t, err)
	require.Equal(t, 10, grammar.Int)
	require.Equal(t, 1, grammar.Pos.Column)
	require.Equal(t, 1, grammar.Pos.Line)
}

type testParserMixin struct {
	A string `@Ident`
	B string `@Ident`
}

func TestMixinFieldsAreParsed(t *testing.T) {
	var grammar struct {
		testParserMixin
		C string `@Ident`
	}
	p := mustTestParser(t, &grammar)
	err := p.ParseString("one two three", &grammar)
	require.NoError(t, err)
	require.Equal(t, "one", grammar.A)
	require.Equal(t, "two", grammar.B)
	require.Equal(t, "three", grammar.C)
}

func TestNestedOptional(t *testing.T) {
	type grammar struct {
		Args []string `"(" [ @Ident { "," @Ident } ] ")"`
	}
	p := mustTestParser(t, &grammar{})
	actual := &grammar{}
	err := p.ParseString(`()`, actual)
	require.NoError(t, err)
	err = p.ParseString(`(a)`, actual)
	require.NoError(t, err)
	err = p.ParseString(`(a, b, c)`, actual)
	require.NoError(t, err)
	err = p.ParseString(`(1)`, actual)
	require.Error(t, err)
}

type captureableWithPosition struct {
	Pos   lexer.Position
	Value string
}

func (c *captureableWithPosition) Capture(values []string) error {
	c.Value = strings.Join(values, " ")
	return nil
}

func TestIssue35(t *testing.T) {
	type grammar struct {
		Value *captureableWithPosition `@Ident`
	}
	p := mustTestParser(t, &grammar{})
	actual := &grammar{}
	err := p.ParseString(`hello`, actual)
	require.NoError(t, err)
	expected := &grammar{Value: &captureableWithPosition{
		Pos:   lexer.Position{Column: 1, Offset: 0, Line: 1},
		Value: "hello",
	}}
	require.Equal(t, expected, actual)
}

func TestInvalidNumbers(t *testing.T) {
	type grammar struct {
		Int8    int8    `  "int8" @Int`
		Int16   int16   `| "int16" @Int`
		Int32   int32   `| "int32" @Int`
		Int64   int64   `| "int64" @Int`
		Uint8   uint8   `| "uint8" @Int`
		Uint16  uint16  `| "uint16" @Int`
		Uint32  uint32  `| "uint32" @Int`
		Uint64  uint64  `| "uint64" @Int`
		Float32 float32 `| "float32" @Float`
		Float64 float64 `| "float64" @Float`
	}

	p := mustTestParser(t, &grammar{})

	tests := []struct {
		name     string
		input    string
		expected *grammar
		err      bool
	}{
		{name: "ValidInt8", input: "int8 127", expected: &grammar{Int8: 127}},
		{name: "InvalidInt8", input: "int8 129", err: true},
		{name: "ValidInt16", input: "int16 32767", expected: &grammar{Int16: 32767}},
		{name: "InvalidInt16", input: "int16 32768", err: true},
		{name: "ValidInt32", input: fmt.Sprintf("int32 %d", math.MaxInt32), expected: &grammar{Int32: math.MaxInt32}},
		{name: "InvalidInt32", input: fmt.Sprintf("int32 %d", math.MaxInt32+1), err: true},
		{name: "ValidInt64", input: fmt.Sprintf("int64 %d", math.MaxInt64), expected: &grammar{Int64: math.MaxInt64}},
		{name: "InvalidInt64", input: "int64 9223372036854775808", err: true},
		{name: "ValidFloat64", input: "float64 1234.5", expected: &grammar{Float64: 1234.5}},
		{name: "InvalidFloat64", input: "float64 asdf", err: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := &grammar{}
			err := p.ParseString(test.input, actual)
			if test.err {
				require.Error(t, err, fmt.Sprintf("%#v", actual))
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, actual)
			}
		})
	}
}

// We'd like this to work, but it can wait.

func TestPartialAST(t *testing.T) {
	type grammar struct {
		Succeed string `@Ident`
		Fail    string `@"foo"`
	}
	p := mustTestParser(t, &grammar{})
	actual := &grammar{}
	err := p.ParseString(`foo bar`, actual)
	require.Error(t, err)
	expected := &grammar{Succeed: "foo"}
	require.Equal(t, expected, actual)
}

func TestCaseInsensitive(t *testing.T) {
	type grammar struct {
		Select string `"select":Keyword @Ident`
	}

	lex := lexer.Must(lexer.Regexp(
		`(?i)(?P<Keyword>SELECT)` +
			`|(?P<Ident>\w+)` +
			`|(\s+)`,
	))

	p := mustTestParser(t, &grammar{}, Lexer(lex), CaseInsensitive("Keyword"))
	actual := &grammar{}
	err := p.ParseString(`SELECT foo`, actual)
	expected := &grammar{"foo"}
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &grammar{}
	err = p.ParseString(`select foo`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestTokenAfterRepeatErrors(t *testing.T) {
	type grammar struct {
		Text string `{ @Ident } "foo"`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString(``, ast)
	require.Error(t, err)
}

func TestEOFAfterRepeat(t *testing.T) {
	type grammar struct {
		Text string `{ @Ident }`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString(``, ast)
	require.NoError(t, err)
}

func TestTrailing(t *testing.T) {
	type grammar struct {
		Text string `@Ident`
	}
	p := mustTestParser(t, &grammar{})
	err := p.ParseString(`foo bar`, &grammar{})
	require.Error(t, err)
}
