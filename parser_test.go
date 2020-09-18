package participle_test

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

func TestProductionCapture(t *testing.T) {
	type testCapture struct {
		A string `@Test`
	}

	_, err := participle.Build(&testCapture{})
	require.Error(t, err)
}

func TestTermCapture(t *testing.T) {
	type grammar struct {
		A string `@{"."}`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	expected := &grammar{"..."}

	err := parser.ParseString("", "...", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestParseScalar(t *testing.T) {
	type grammar struct {
		A string `@"one"`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	err := parser.ParseString("", "one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{"one"}, actual)
}

func TestParseGroup(t *testing.T) {
	type grammar struct {
		A string `@("one" | "two")`
	}

	parser := mustTestParser(t, &grammar{})

	actual := &grammar{}
	err := parser.ParseString("", "one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{"one"}, actual)

	actual = &grammar{}
	err = parser.ParseString("", "two", actual)
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
	err := parser.ParseString("", "one", actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{A: "one"}, actual)

	actual = &grammar{}
	err = parser.ParseString("", "two", actual)
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
	err := parser.ParseString("", "one two three", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &grammar{}
	expected = &grammar{}
	err = parser.ParseString("", "moo", actual)
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
	err := parser.ParseString("", "one two", actual)
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
	err := parser.ParseString("", "one two one two", actual)
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
	err := parser.ParseString("", ``, actual)
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
	err := parser.ParseString("", `...`, actual)
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
	err := parser.ParseString("", "...b", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &testRepitition{}
	expected = &testRepitition{
		A: []string{".", ".", "."},
		C: &c,
	}
	err = parser.ParseString("", "...c", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &testRepitition{}
	expected = &testRepitition{
		B: &b,
	}
	err = parser.ParseString("", "b", actual)
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
	err := parser.ParseString("", "...", actual)
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

func TestEBNFParser(t *testing.T) {
	parser := mustTestParser(t, &EBNF{}, participle.Unquote())

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
	err := parser.ParseString("", strings.TrimSpace(`
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
	err := parser.ParseString("", ";b", actual)
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
	err := parser.ParseString("", `c`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestHello(t *testing.T) {
	type testHello struct {
		Hello string `@"hello"`
		To    string `@String`
	}

	parser := mustTestParser(t, &testHello{}, participle.Unquote())

	expected := &testHello{"hello", `Bobby Brown`}
	actual := &testHello{}
	err := parser.ParseString("", `hello "Bobby Brown"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func mustTestParser(t *testing.T, grammar interface{}, options ...participle.Option) *participle.Parser {
	t.Helper()
	parser, err := participle.Build(grammar, options...)
	require.NoError(t, err)
	return parser
}

func BenchmarkEBNFParser(b *testing.B) {
	parser, err := participle.Build(&EBNF{})
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
		_ = parser.ParseString("", source, actual)
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

	err := parser.ParseString("", ".>,<.>.>,<.>,<", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestPosInjection(t *testing.T) {
	type subgrammar struct {
		Pos    lexer.Position
		B      string `@{ "," }`
		EndPos lexer.Position
	}
	type grammar struct {
		Pos    lexer.Position
		A      string      `@{ "." }`
		B      *subgrammar `@@`
		C      string      `@"."`
		EndPos lexer.Position
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
			EndPos: lexer.Position{
				Offset: 9,
				Line:   1,
				Column: 10,
			},
		},
		C: ".",
		EndPos: lexer.Position{
			Offset: 10,
			Line:   1,
			Column: 11,
		},
	}

	err := parser.ParseString("", "   ...,,,.", actual)
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
	err := parser.ParseString("", "a a a", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

type unmarshallableCount int

func (u *unmarshallableCount) UnmarshalText(text []byte) error {
	*u += unmarshallableCount(len(text))
	return nil
}

func TestTextUnmarshalerInterface(t *testing.T) {
	type grammar struct {
		Count unmarshallableCount `{ @"a" }`
	}

	parser := mustTestParser(t, &grammar{})
	actual := &grammar{}
	expected := &grammar{Count: 3}
	err := parser.ParseString("", "a a a", actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestLiteralTypeConstraint(t *testing.T) {
	type grammar struct {
		Literal string `@"123456":String`
	}

	parser := mustTestParser(t, &grammar{}, participle.Unquote())

	actual := &grammar{}
	expected := &grammar{Literal: "123456"}
	err := parser.ParseString("", `"123456"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	err = parser.ParseString("", `123456`, actual)
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

	parser, err := participle.Build(&grammar{}, participle.Unquote())
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{Capture: &nestedCapture{Tokens: []string{"hello"}}}
	err = parser.ParseString("", `"hello"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

type parseableStruct struct {
	Tokens []string
}

func (p *parseableStruct) Parse(lex *lexer.PeekingLexer) error {
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

	parser, err := participle.Build(&grammar{}, participle.Unquote())
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{Inner: &parseableStruct{Tokens: []string{"hello", "123", "world", ""}}}
	err = parser.ParseString("", `hello 123 "world"`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestStringConcat(t *testing.T) {
	type grammar struct {
		Field string `@"." { @"." }`
	}

	parser, err := participle.Build(&grammar{})
	require.NoError(t, err)

	actual := &grammar{}
	expected := &grammar{"...."}
	err = parser.ParseString("", `. . . .`, actual)
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
	err := parser.ParseString("", `1 2 3 4`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestEmptyStructErrorsNotPanicsIssue21(t *testing.T) {
	type grammar struct {
		Foo struct{} `@@`
	}
	_, err := participle.Build(&grammar{})
	require.Error(t, err)
}

func TestMultipleTokensIntoScalar(t *testing.T) {
	var grammar struct {
		Field int `@("-" Int)`
	}
	p, err := participle.Build(&grammar)
	require.NoError(t, err)
	err = p.ParseString("", `- 10`, &grammar)
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
	err := p.ParseString("", "10", &grammar)
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
	err := p.ParseString("", "one two three", &grammar)
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
	err := p.ParseString("", `()`, actual)
	require.NoError(t, err)
	err = p.ParseString("", `(a)`, actual)
	require.NoError(t, err)
	err = p.ParseString("", `(a, b, c)`, actual)
	require.NoError(t, err)
	err = p.ParseString("", `(1)`, actual)
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
	err := p.ParseString("", `hello`, actual)
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
		// nolint: scopelint
		t.Run(test.name, func(t *testing.T) {
			actual := &grammar{}
			err := p.ParseString("", test.input, actual)
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
	err := p.ParseString("", `foo bar`, actual)
	require.Error(t, err)
	expected := &grammar{Succeed: "foo"}
	require.Equal(t, expected, actual)
}

func TestCaseInsensitive(t *testing.T) {
	type grammar struct {
		Select string `"select":Keyword @Ident`
	}

	// lex := lexer.Must(lexer.Regexp(
	// 	`(?i)(?P<Keyword>SELECT)` +
	// 		`|(?P<Ident>\w+)` +
	// 		`|(\s+)`,
	// ))
	lex := lexer.Must(stateful.NewSimple([]stateful.Rule{
		{"Keyword", `(?i)SELECT`, nil},
		{"Ident", `\w+`, nil},
		{"whitespace", `\s+`, nil},
	}))

	p := mustTestParser(t, &grammar{}, participle.Lexer(lex), participle.CaseInsensitive("Keyword"))
	actual := &grammar{}
	err := p.ParseString("", `SELECT foo`, actual)
	expected := &grammar{"foo"}
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	actual = &grammar{}
	err = p.ParseString("", `select foo`, actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestTokenAfterRepeatErrors(t *testing.T) {
	type grammar struct {
		Text string `{ @Ident } "foo"`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", ``, ast)
	require.Error(t, err)
}

func TestEOFAfterRepeat(t *testing.T) {
	type grammar struct {
		Text string `{ @Ident }`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", ``, ast)
	require.NoError(t, err)
}

func TestTrailing(t *testing.T) {
	type grammar struct {
		Text string `@Ident`
	}
	p := mustTestParser(t, &grammar{})
	err := p.ParseString("", `foo bar`, &grammar{})
	require.Error(t, err)
}

func TestModifiers(t *testing.T) {
	nonEmptyGrammar := &struct {
		A string `@( ("x"? "y"? "z"?)! "b" )`
	}{}
	tests := []struct {
		name     string
		grammar  interface{}
		input    string
		expected string
		fail     bool
	}{
		{name: "NonMatchingOptionalNonEmpty",
			input:   "b",
			fail:    true,
			grammar: nonEmptyGrammar},
		{name: "NonEmptyMatch",
			input:    "x b",
			expected: "xb",
			grammar:  nonEmptyGrammar},
		{name: "NonEmptyMatchAll",
			input:    "x y z b",
			expected: "xyzb",
			grammar:  nonEmptyGrammar},
		{name: "NonEmptyMatchSome",
			input:    "x z b",
			expected: "xzb",
			grammar:  nonEmptyGrammar},
		{name: "MatchingOptional",
			input:    "a b",
			expected: "ab",
			grammar: &struct {
				A string `@( "a"? "b" )`
			}{}},
		{name: "NonMatchingOptionalIsSkipped",
			input:    "b",
			expected: "b",
			grammar: &struct {
				A string `@( "a"? "b" )`
			}{}},
		{name: "MatchingOneOrMore",
			input:    "a a a a a",
			expected: "aaaaa",
			grammar: &struct {
				A string `@( "a"+ )`
			}{}},
		{name: "NonMatchingOneOrMore",
			input: "",
			fail:  true,
			grammar: &struct {
				A string `@( "a"+ )`
			}{}},
		{name: "MatchingZeroOrMore",
			input: "aaaaaaa",
			fail:  true,
			grammar: &struct {
				A string `@( "a"* )`
			}{}},
		{name: "NonMatchingZeroOrMore",
			input: "",
			grammar: &struct {
				A string `@( "a"* )`
			}{}},
	}
	for _, test := range tests {
		// nolint: scopelint
		t.Run(test.name, func(t *testing.T) {
			p := mustTestParser(t, test.grammar)
			err := p.ParseString("", test.input, test.grammar)
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				actual := reflect.ValueOf(test.grammar).Elem().FieldByName("A").String()
				require.Equal(t, test.expected, actual)
			}
		})
	}
}

func TestStreamingParser(t *testing.T) {
	type token struct {
		Str string `  @Ident`
		Num int    `| @Int`
	}
	parser := mustTestParser(t, &token{})

	tokens := make(chan *token, 128)
	err := parser.ParseString("", `hello 10 11 12 world`, tokens)
	actual := []*token{}
	for token := range tokens {
		actual = append(actual, token)
	}
	expected := []*token{
		{Str: "hello", Num: 0},
		{Str: "", Num: 10},
		{Str: "", Num: 11},
		{Str: "", Num: 12},
		{Str: "world", Num: 0},
	}
	require.Equal(t, expected, actual)
	require.NoError(t, err)
}

func TestIssue60(t *testing.T) {
	type grammar struct {
		A string `@("one" | | "two")`
	}
	_, err := participle.Build(&grammar{})
	require.Error(t, err)
}

type Issue62Bar struct {
	A int
}

func (x *Issue62Bar) Parse(lex *lexer.PeekingLexer) error {
	token, err := lex.Next()
	if err != nil {
		return err
	}
	x.A, err = strconv.Atoi(token.Value)
	return err
}

type Issue62Foo struct {
	Bars []Issue62Bar `parser:"@@+"`
}

func TestIssue62(t *testing.T) {
	_, err := participle.Build(&Issue62Foo{})
	require.NoError(t, err)
}

// nolint: structcheck
func TestIssue71(t *testing.T) {
	type Sub struct {
		name string `@Ident`
	}
	type grammar struct {
		pattern *Sub `@@`
	}

	_, err := participle.Build(&grammar{})
	require.Error(t, err)
}

func TestAllowTrailing(t *testing.T) {
	type G struct {
		Name string `@Ident`
	}

	p, err := participle.Build(&G{})
	require.NoError(t, err)

	g := &G{}
	err = p.ParseString("", `hello world`, g)
	require.Error(t, err)
	err = p.ParseString("", `hello world`, g, participle.AllowTrailing(true))
	require.NoError(t, err)
	require.Equal(t, &G{"hello"}, g)
}

func TestDisjunctionErrorReporting(t *testing.T) {
	type statement struct {
		Add    bool `  @"add"`
		Remove bool `| @"remove"`
	}
	type grammar struct {
		Statements []*statement `"{" ( @@ )* "}"`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", `{ add foo }`, ast)
	// TODO: This should produce a more useful error. This is returned by sequence.Parse().
	require.EqualError(t, err, `1:7: unexpected token "foo" (expected "}")`)
}

func TestCustomInt(t *testing.T) {
	type MyInt int
	type G struct {
		Value MyInt `@Int`
	}

	p, err := participle.Build(&G{})
	require.NoError(t, err)

	g := &G{}
	err = p.ParseString("", `42`, g)
	require.NoError(t, err)
	require.Equal(t, &G{42}, g)
}

func TestBoolIfSet(t *testing.T) {
	type G struct {
		Value bool `@"true"?`
	}

	p, err := participle.Build(&G{})
	require.NoError(t, err)

	g := &G{}
	err = p.ParseString("", `true`, g)
	require.NoError(t, err)
	require.Equal(t, &G{true}, g)
	err = p.ParseString("", ``, g)
	require.NoError(t, err)
	require.Equal(t, &G{false}, g)
}

func TestCustomBoolIfSet(t *testing.T) {
	type MyBool bool
	type G struct {
		Value MyBool `@"true"?`
	}

	p, err := participle.Build(&G{})
	require.NoError(t, err)

	g := &G{}
	err = p.ParseString("", `true`, g)
	require.NoError(t, err)
	require.Equal(t, &G{true}, g)
	err = p.ParseString("", ``, g)
	require.NoError(t, err)
	require.Equal(t, &G{false}, g)
}

func TestPointerToList(t *testing.T) {
	type grammar struct {
		List *[]string `@Ident*`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", `foo bar`, ast)
	require.NoError(t, err)
	l := []string{"foo", "bar"}
	require.Equal(t, &grammar{List: &l}, ast)
}

// I'm not sure if this is a problem that should be solved like this.

// func TestMatchHydratesNullFields(t *testing.T) {
// 	type grammar struct {
// 		List []string `"{" @Ident* "}"`
// 	}
// 	p := mustTestParser(t, &grammar{})
// 	ast := &grammar{}
// 	err := p.ParseString(`{}`, ast)
// 	require.NoError(t, err)
// 	require.NotNil(t, ast.List)
// }

func TestNegation(t *testing.T) {
	type grammar struct {
		EverythingUntilSemicolon *[]string `@!';'* @';'`
	}
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", `hello world ;`, ast)
	require.NoError(t, err)
	require.Equal(t, &[]string{"hello", "world", ";"}, ast.EverythingUntilSemicolon)

	err = p.ParseString("", `hello world`, ast)
	require.Error(t, err)
}

func TestNegationWithPattern(t *testing.T) {
	type grammar struct {
		EverythingMoreComplex *[]string `@!(';' String)* @';' @String`
	}

	p := mustTestParser(t, &grammar{}, participle.Unquote())
	// j, err := json.MarshalIndent(p.root, "", "  ")
	// log.Print(j)
	// log.Print(stringer(p.root))
	ast := &grammar{}
	err := p.ParseString("", `hello world ; "some-str"`, ast)
	require.NoError(t, err)
	require.Equal(t, &[]string{"hello", "world", ";", `some-str`}, ast.EverythingMoreComplex)

	err = p.ParseString("", `hello ; world ; "hey"`, ast)
	require.NoError(t, err)
	require.Equal(t, &[]string{"hello", ";", "world", ";", `hey`}, ast.EverythingMoreComplex)

	err = p.ParseString("", `hello ; world ;`, ast)
	require.Error(t, err)
}

func TestNegationWithDisjunction(t *testing.T) {
	type grammar struct {
		EverythingMoreComplex *[]string `@!(';' | ',')* @(';' | ',')`
	}

	// Note: we need more lookahead since (';' String) needs some before failing to match
	p := mustTestParser(t, &grammar{})
	ast := &grammar{}
	err := p.ParseString("", `hello world ;`, ast)
	require.NoError(t, err)
	require.Equal(t, &[]string{"hello", "world", ";"}, ast.EverythingMoreComplex)

	err = p.ParseString("", `hello world , `, ast)
	require.NoError(t, err)
	require.Equal(t, &[]string{"hello", "world", ","}, ast.EverythingMoreComplex)

}
