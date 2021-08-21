package participle_test

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestProductionCapture(t *testing.T) {
	type testCapture struct {
		A string `@Test`
	}

	_, err := participle.Build[testCapture]()
	assert.Error(t, err)
}

func TestTermCapture(t *testing.T) {
	type grammar struct {
		A string `@"."*`
	}

	parser := mustTestParser[grammar](t)

	expected := &grammar{"..."}

	actual, err := parser.ParseString("", "...")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseScalar(t *testing.T) {
	type grammar struct {
		A string `@"one"`
	}

	parser := mustTestParser[grammar](t)

	actual, err := parser.ParseString("", "one")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{"one"}, actual)
}

func TestParseGroup(t *testing.T) {
	type grammar struct {
		A string `@("one" | "two")`
	}

	parser := mustTestParser[grammar](t)

	actual, err := parser.ParseString("", "one")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{"one"}, actual)

	actual, err = parser.ParseString("", "two")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{"two"}, actual)
}

func TestParseAlternative(t *testing.T) {
	type grammar struct {
		A string `@"one" |`
		B string `@"two"`
	}

	parser := mustTestParser[grammar](t)

	actual, err := parser.ParseString("", "one")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{A: "one"}, actual)

	actual, err = parser.ParseString("", "two")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{B: "two"}, actual)
}

func TestParseSequence(t *testing.T) {
	type grammar struct {
		A string `@"one"`
		B string `@"two"`
		C string `@"three"`
	}

	parser := mustTestParser[grammar](t)

	expected := &grammar{"one", "two", "three"}
	actual, err := parser.ParseString("", "one two three")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	expected = &grammar{}
	actual, err = parser.ParseString("", "moo")
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

	parser := mustTestParser[testNested](t)

	expected := &testNested{A: &nestedInner{B: "one", C: "two"}}
	actual, err := parser.ParseString("", "one two")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAccumulateNested(t *testing.T) {
	type nestedInner struct {
		B string `@"one"`
		C string `@"two"`
	}
	type testAccumulateNested struct {
		A []*nestedInner `@@+`
	}

	parser := mustTestParser[testAccumulateNested](t)

	expected := &testAccumulateNested{A: []*nestedInner{{B: "one", C: "two"}, {B: "one", C: "two"}}}
	actual, err := parser.ParseString("", "one two one two")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRepetitionNoMatch(t *testing.T) {
	type grammar struct {
		A []string `@"."*`
	}
	parser := mustTestParser[grammar](t)

	expected := &grammar{}
	actual, err := parser.ParseString("", ``)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRepetition(t *testing.T) {
	type grammar struct {
		A []string `@"."*`
	}
	parser := mustTestParser[grammar](t)

	expected := &grammar{A: []string{".", ".", "."}}
	actual, err := parser.ParseString("", `...`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRepetitionAcrossFields(t *testing.T) {
	type testRepetition struct {
		A []string `@"."*`
		B *string  `(@"b" |`
		C *string  ` @"c")`
	}

	parser := mustTestParser[testRepetition](t)

	b := "b"
	c := "c"

	expected := &testRepetition{
		A: []string{".", ".", "."},
		B: &b,
	}
	actual, err := parser.ParseString("", "...b")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	expected = &testRepetition{
		A: []string{".", ".", "."},
		C: &c,
	}
	actual, err = parser.ParseString("", "...c")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	expected = &testRepetition{
		B: &b,
	}
	actual, err = parser.ParseString("", "b")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAccumulateString(t *testing.T) {
	type testAccumulateString struct {
		A string `@"."+`
	}

	parser := mustTestParser[testAccumulateString](t)

	expected := &testAccumulateString{
		A: "...",
	}
	actual, err := parser.ParseString("", "...")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

type Group struct {
	Expression *Expression `"(" @@ ")"`
}

type LookaheadGroup struct {
	Expression *Expression `"(" "?" ("=" | "!") @@ ")"`
}

type EBNFOption struct {
	Expression *Expression `"[" @@ "]"`
}

type Repetition struct {
	Expression *Expression `"{" @@ "}"`
}

type Negation struct {
	Expression *Expression `"!" @@`
}

type Literal struct {
	Start string `@String`
}

type Range struct {
	Start string `@String`
	End   string `"…" @String`
}

type Term struct {
	Name           string          `@Ident |`
	Literal        *Literal        `@@ |`
	Range          *Range          `@@ |`
	Group          *Group          `@@ |`
	LookaheadGroup *LookaheadGroup `@@ |`
	Option         *EBNFOption     `@@ |`
	Repetition     *Repetition     `@@ |`
	Negation       *Negation       `@@`
}

type Sequence struct {
	Terms []*Term `@@+`
}

type Expression struct {
	Alternatives []*Sequence `@@ ( "|" @@ )*`
}

type Production struct {
	Name       string        `@Ident "="`
	Expression []*Expression `@@+ "."`
}

type EBNF struct {
	Productions []*Production `@@*`
}

func TestEBNFParser(t *testing.T) {
	parser := mustTestParser[EBNF](t, participle.Unquote())

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
	actual, err := parser.ParseString("", strings.TrimSpace(`
Production  = name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = name | token [ "…" token ] | "@@" | Group | EBNFOption | Repetition .
Group       = "(" Expression ")" .
EBNFOption      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
`))
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseExpression(t *testing.T) {
	type testNestA struct {
		A string `":" @"a"*`
	}
	type testNestB struct {
		B string `";" @"b"*`
	}
	type testExpression struct {
		A *testNestA `@@ |`
		B *testNestB `@@`
	}

	parser := mustTestParser[testExpression](t)

	expected := &testExpression{
		B: &testNestB{
			B: "b",
		},
	}
	actual, err := parser.ParseString("", ";b")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseOptional(t *testing.T) {
	type testOptional struct {
		A string `( @"a" @"b" )?`
		B string `@"c"`
	}

	parser := mustTestParser[testOptional](t)

	expected := &testOptional{B: "c"}
	actual, err := parser.ParseString("", `c`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestHello(t *testing.T) {
	type testHello struct {
		Hello string `@"hello"`
		To    string `@String`
	}

	parser := mustTestParser[testHello](t, participle.Unquote())

	expected := &testHello{"hello", `Bobby Brown`}
	actual, err := parser.ParseString("", `hello "Bobby Brown"`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func mustTestParser[T any](t *testing.T, options ...participle.Option) *participle.Parser[T] {
	t.Helper()
	parser, err := participle.Build[T](options...)
	assert.NoError(t, err)
	return parser
}

func BenchmarkEBNFParser(b *testing.B) {
	parser, err := participle.Build[EBNF]()
	assert.NoError(b, err)
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
		_, _ = parser.ParseString("", source)
	}
}

func TestRepeatAcrossFields(t *testing.T) {
	type grammar struct {
		A string `( @("." ">") |`
		B string `  @("," "<") )*`
	}

	parser := mustTestParser[grammar](t)

	expected := &grammar{A: ".>.>.>.>", B: ",<,<,<"}

	actual, err := parser.ParseString("", ".>,<.>.>,<.>,<")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestPosInjection(t *testing.T) {
	type subgrammar struct {
		Pos    lexer.Position
		B      string `@","*`
		EndPos lexer.Position
	}
	type grammar struct {
		Pos    lexer.Position
		A      string      `@"."*`
		B      *subgrammar `@@`
		C      string      `@"."`
		EndPos lexer.Position
	}

	parser := mustTestParser[grammar](t)

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

	actual, err := parser.ParseString("", "   ...,,,.")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

type parseableCount int

func (c *parseableCount) Capture(values []string) error {
	*c += parseableCount(len(values))
	return nil
}

func TestCaptureInterface(t *testing.T) {
	type grammar struct {
		Count parseableCount `@"a"*`
	}

	parser := mustTestParser[grammar](t)
	expected := &grammar{Count: 3}
	actual, err := parser.ParseString("", "a a a")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
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

	parser := mustTestParser[grammar](t)
	expected := &grammar{Count: 3}
	actual, err := parser.ParseString("", "a a a")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestLiteralTypeConstraint(t *testing.T) {
	type grammar struct {
		Literal string `@"123456":String`
	}

	parser := mustTestParser[grammar](t, participle.Unquote())

	expected := &grammar{Literal: "123456"}
	actual, err := parser.ParseString("", `"123456"`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = parser.ParseString("", `123456`)
	assert.Error(t, err)
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

	parser, err := participle.Build[grammar](participle.Unquote())
	assert.NoError(t, err)

	expected := &grammar{Capture: &nestedCapture{Tokens: []string{"hello"}}}
	actual, err := parser.ParseString("", `"hello"`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
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

	parser, err := participle.Build[grammar](participle.Unquote())
	assert.NoError(t, err)

	expected := &grammar{Inner: &parseableStruct{Tokens: []string{"hello", "123", "world", ""}}}
	actual, err := parser.ParseString("", `hello 123 "world"`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestStringConcat(t *testing.T) {
	type grammar struct {
		Field string `@"."+`
	}

	parser, err := participle.Build[grammar]()
	assert.NoError(t, err)

	expected := &grammar{"...."}
	actual, err := parser.ParseString("", `. . . .`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestParseIntSlice(t *testing.T) {
	type grammar struct {
		Field []int `@Int+`
	}

	parser := mustTestParser[grammar](t)

	expected := &grammar{[]int{1, 2, 3, 4}}
	actual, err := parser.ParseString("", `1 2 3 4`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestEmptyStructErrorsNotPanicsIssue21(t *testing.T) {
	type grammar struct {
		Foo struct{} `@@`
	}
	_, err := participle.Build[grammar]()
	assert.Error(t, err)
}

func TestMultipleTokensIntoScalar(t *testing.T) {
	type grammar struct {
		Field int `@("-" Int)`
	}
	p, err := participle.Build[grammar]()
	assert.NoError(t, err)
	actual, err := p.ParseString("", `- 10`)
	assert.NoError(t, err)
	assert.Equal(t, -10, actual.Field)
}

type posMixin struct {
	Pos lexer.Position
}

func TestMixinPosIsPopulated(t *testing.T) {
	type grammar struct {
		posMixin

		Int int `@Int`
	}

	p := mustTestParser[grammar](t)
	actual, err := p.ParseString("", "10")
	assert.NoError(t, err)
	assert.Equal(t, 10, actual.Int)
	assert.Equal(t, 1, actual.Pos.Column)
	assert.Equal(t, 1, actual.Pos.Line)
}

type testParserMixin struct {
	A string `@Ident`
	B string `@Ident`
}

func TestMixinFieldsAreParsed(t *testing.T) {
	type grammar struct {
		testParserMixin
		C string `@Ident`
	}
	p := mustTestParser[grammar](t)
	actual, err := p.ParseString("", "one two three")
	assert.NoError(t, err)
	assert.Equal(t, "one", actual.A)
	assert.Equal(t, "two", actual.B)
	assert.Equal(t, "three", actual.C)
}

func TestNestedOptional(t *testing.T) {
	type grammar struct {
		Args []string `"(" [ @Ident ( "," @Ident )* ] ")"`
	}
	p := mustTestParser[grammar](t)
	_, err := p.ParseString("", `()`)
	assert.NoError(t, err)
	_, err = p.ParseString("", `(a)`)
	assert.NoError(t, err)
	_, err = p.ParseString("", `(a, b, c)`)
	assert.NoError(t, err)
	_, err = p.ParseString("", `(1)`)
	assert.Error(t, err)
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

	p := mustTestParser[grammar](t)

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
		{name: "InvalidInt32", input: fmt.Sprintf("int32 %d", int64(math.MaxInt32+1)), err: true},
		{name: "ValidInt64", input: fmt.Sprintf("int64 %d", int64(math.MaxInt64)), expected: &grammar{Int64: math.MaxInt64}},
		{name: "InvalidInt64", input: "int64 9223372036854775808", err: true},
		{name: "ValidFloat64", input: "float64 1234.5", expected: &grammar{Float64: 1234.5}},
		{name: "InvalidFloat64", input: "float64 asdf", err: true},
	}
	for _, test := range tests {
		// nolint: scopelint
		t.Run(test.name, func(t *testing.T) {
			actual, err := p.ParseString("", test.input)
			if test.err {
				assert.Error(t, err, fmt.Sprintf("%#v", actual))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, actual)
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
	p := mustTestParser[grammar](t)
	actual, err := p.ParseString("", `foo bar`)
	assert.Error(t, err)
	expected := &grammar{Succeed: "foo"}
	assert.Equal(t, expected, actual)
}

func TestCaseInsensitive(t *testing.T) {
	type grammar struct {
		Select string `"select":Keyword @Ident`
	}

	// lex := lexer.MustStateful(lexer.Regexp(
	// 	`(?i)(?P<Keyword>SELECT)` +
	// 		`|(?P<Ident>\w+)` +
	// 		`|(\s+)`,
	// ))
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"Keyword", `(?i)SELECT`},
		{"Ident", `\w+`},
		{"whitespace", `\s+`},
	})

	p := mustTestParser[grammar](t, participle.Lexer(lex), participle.CaseInsensitive("Keyword"))
	actual, err := p.ParseString("", `SELECT foo`)
	expected := &grammar{"foo"}
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = p.ParseString("", `select foo`)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestTokenAfterRepeatErrors(t *testing.T) {
	type grammar struct {
		Text string `@Ident* "foo"`
	}
	p := mustTestParser[grammar](t)
	_, err := p.ParseString("", ``)
	assert.Error(t, err)
}

func TestEOFAfterRepeat(t *testing.T) {
	type grammar struct {
		Text string `@Ident*`
	}
	p := mustTestParser[grammar](t)
	_, err := p.ParseString("", ``)
	assert.NoError(t, err)
}

func TestTrailing(t *testing.T) {
	type grammar struct {
		Text string `@Ident`
	}
	p := mustTestParser[grammar](t)
	_, err := p.ParseString("", `foo bar`)
	assert.Error(t, err)
}

// TODO: Figure out how to make table driven tests less ugly.

// func TestModifiers(t *testing.T) {
// 	nonEmptyGrammar := &struct {
// 		A string `@( ("x"? "y"? "z"?)! "b" )`
// 	}{}
// 	tests := []struct {
// 		name     string
// 		grammar  interface{}
// 		input    string
// 		expected string
// 		fail     bool
// 	}{
// 		{name: "NonMatchingOptionalNonEmpty",
// 			input:   "b",
// 			fail:    true,
// 			grammar: nonEmptyGrammar},
// 		{name: "NonEmptyMatch",
// 			input:    "x b",
// 			expected: "xb",
// 			grammar:  nonEmptyGrammar},
// 		{name: "NonEmptyMatchAll",
// 			input:    "x y z b",
// 			expected: "xyzb",
// 			grammar:  nonEmptyGrammar},
// 		{name: "NonEmptyMatchSome",
// 			input:    "x z b",
// 			expected: "xzb",
// 			grammar:  nonEmptyGrammar},
// 		{name: "MatchingOptional",
// 			input:    "a b",
// 			expected: "ab",
// 			grammar: &struct {
// 				A string `@( "a"? "b" )`
// 			}{}},
// 		{name: "NonMatchingOptionalIsSkipped",
// 			input:    "b",
// 			expected: "b",
// 			grammar: &struct {
// 				A string `@( "a"? "b" )`
// 			}{}},
// 		{name: "MatchingOneOrMore",
// 			input:    "a a a a a",
// 			expected: "aaaaa",
// 			grammar: &struct {
// 				A string `@( "a"+ )`
// 			}{}},
// 		{name: "NonMatchingOneOrMore",
// 			input: "",
// 			fail:  true,
// 			grammar: &struct {
// 				A string `@( "a"+ )`
// 			}{}},
// 		{name: "MatchingZeroOrMore",
// 			input: "aaaaaaa",
// 			fail:  true,
// 			grammar: &struct {
// 				A string `@( "a"* )`
// 			}{}},
// 		{name: "NonMatchingZeroOrMore",
// 			input: "",
// 			grammar: &struct {
// 				A string `@( "a"* )`
// 			}{}},
// 	}
// 	for _, test := range tests {
// 		// nolint: scopelint
// 		t.Run(test.name, func(t *testing.T) {
// 			p := mustTestParser(t, test.grammar)
// 			err := p.ParseString("", test.input, test.grammar)
// 			if test.fail {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 				actual := reflect.ValueOf(test.grammar).Elem().FieldByName("A").String()
// 				assert.Equal(t, test.expected, actual)
// 			}
// 		})
// 	}
// }

func TestNonEmptyMatchWithOptionalGroup(t *testing.T) {
	type term struct {
		Minus bool   `@'-'?`
		Name  string `@Ident`
	}
	type grammar struct {
		Start term `parser:"'[' (@@?"`
		End   term `parser:"     (':' @@)?)! ']'"`
	}

	p := mustTestParser[grammar](t)

	result, err := p.ParseString("", "[-x]")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{Start: term{Minus: true, Name: "x"}}, result)

	result, err = p.ParseString("", "[a:-b]")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{Start: term{Name: "a"}, End: term{Minus: true, Name: "b"}}, result)

	result, err = p.ParseString("", "[:end]")
	assert.NoError(t, err)
	assert.Equal(t, &grammar{End: term{Name: "end"}}, result)

	result, err = p.ParseString("", "[]")
	assert.Equal(t, err.Error(), `1:2: sub-expression (Term? (":" Term)?)! cannot be empty`)
}

func TestIssue60(t *testing.T) {
	type grammar struct {
		A string `@("one" | | "two")`
	}
	_, err := participle.Build[grammar]()
	assert.Error(t, err)
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
	_, err := participle.Build[Issue62Foo]()
	assert.NoError(t, err)
}

// nolint: structcheck, unused
func TestIssue71(t *testing.T) {
	type Sub struct {
		name string `@Ident`
	}
	type grammar struct {
		pattern *Sub `@@`
	}

	_, err := participle.Build[grammar]()
	assert.Error(t, err)
}

func TestAllowTrailing(t *testing.T) {
	type G struct {
		Name string `@Ident`
	}

	p, err := participle.Build[G]()
	assert.NoError(t, err)

	g, err := p.ParseString("", `hello world`)
	assert.Error(t, err)
	g, err = p.ParseString("", `hello world`, participle.AllowTrailing(true))
	assert.NoError(t, err)
	assert.Equal(t, &G{"hello"}, g)
}

func TestDisjunctionErrorReporting(t *testing.T) {
	type statement struct {
		Add    bool `  @"add"`
		Remove bool `| @"remove"`
	}
	type grammar struct {
		Statements []*statement `"{" ( @@ )* "}"`
	}
	p := mustTestParser[grammar](t)
	_, err := p.ParseString("", `{ add foo }`)
	// TODO: This should produce a more useful error. This is returned by sequence.Parse().
	assert.Equal(t, err.Error(), `1:7: unexpected token "foo" (expected "}")`)
}

func TestCustomInt(t *testing.T) {
	type MyInt int
	type G struct {
		Value MyInt `@Int`
	}

	p, err := participle.Build[G]()
	assert.NoError(t, err)

	g, err := p.ParseString("", `42`)
	assert.NoError(t, err)
	assert.Equal(t, &G{42}, g)
}

func TestBoolIfSet(t *testing.T) {
	type G struct {
		Value bool `@"true"?`
	}

	p, err := participle.Build[G]()
	assert.NoError(t, err)

	g, err := p.ParseString("", `true`)
	assert.NoError(t, err)
	assert.Equal(t, &G{true}, g)
	g, err = p.ParseString("", ``)
	assert.NoError(t, err)
	assert.Equal(t, &G{false}, g)
}

func TestCustomBoolIfSet(t *testing.T) {
	type MyBool bool
	type G struct {
		Value MyBool `@"true"?`
	}

	p, err := participle.Build[G]()
	assert.NoError(t, err)

	g, err := p.ParseString("", `true`)
	assert.NoError(t, err)
	assert.Equal(t, &G{true}, g)
	g, err = p.ParseString("", ``)
	assert.NoError(t, err)
	assert.Equal(t, &G{false}, g)
}

func TestPointerToList(t *testing.T) {
	type grammar struct {
		List *[]string `@Ident*`
	}
	p := mustTestParser[grammar](t)
	ast := &grammar{}
	ast, err := p.ParseString("", `foo bar`)
	assert.NoError(t, err)
	l := []string{"foo", "bar"}
	assert.Equal(t, &grammar{List: &l}, ast)
}

// I'm not sure if this is a problem that should be solved like this.

// func TestMatchHydratesNullFields(t *testing.T) {
// 	type grammar struct {
// 		List []string `"{" @Ident* "}"`
// 	}
// 	p := mustTestParser(t, &grammar{})
// 	ast := &grammar{}
// 	err := p.ParseString(`{}`, ast)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, ast.List)
// }

func TestNegation(t *testing.T) {
	type grammar struct {
		EverythingUntilSemicolon *[]string `@!';'* @';'`
	}
	p := mustTestParser[grammar](t)
	ast := &grammar{}
	ast, err := p.ParseString("", `hello world ;`)
	assert.NoError(t, err)
	assert.Equal(t, &[]string{"hello", "world", ";"}, ast.EverythingUntilSemicolon)

	ast, err = p.ParseString("", `hello world`)
	assert.Error(t, err)
}

func TestNegationWithPattern(t *testing.T) {
	type grammar struct {
		EverythingMoreComplex *[]string `@!(';' String)* @';' @String`
	}

	p := mustTestParser[grammar](t, participle.Unquote())
	// j, err := json.MarshalIndent(p.root, "", "  ")
	// log.Print(j)
	// log.Print(ebnf(p.root))
	ast, err := p.ParseString("", `hello world ; "some-str"`)
	assert.NoError(t, err)
	assert.Equal(t, &[]string{"hello", "world", ";", `some-str`}, ast.EverythingMoreComplex)

	ast, err = p.ParseString("", `hello ; world ; "hey"`)
	assert.NoError(t, err)
	assert.Equal(t, &[]string{"hello", ";", "world", ";", `hey`}, ast.EverythingMoreComplex)

	ast, err = p.ParseString("", `hello ; world ;`)
	assert.Error(t, err)
}

func TestNegationWithDisjunction(t *testing.T) {
	type grammar struct {
		EverythingMoreComplex *[]string `@!(';' | ',')* @(';' | ',')`
	}

	// Note: we need more lookahead since (';' String) needs some before failing to match
	p := mustTestParser[grammar](t)
	ast, err := p.ParseString("", `hello world ;`)
	assert.NoError(t, err)
	assert.Equal(t, &[]string{"hello", "world", ";"}, ast.EverythingMoreComplex)

	ast, err = p.ParseString("", `hello world , `)
	assert.NoError(t, err)
	assert.Equal(t, &[]string{"hello", "world", ","}, ast.EverythingMoreComplex)
}

func TestLookaheadGroup_Positive_SingleToken(t *testing.T) {
	type val struct {
		Str string `  @String`
		Int int    `| @Int`
	}
	type op struct {
		Op      string `@('+' | '*' (?= @Int))`
		Operand val    `@@`
	}
	type sum struct {
		Left val  `@@`
		Ops  []op `@@*`
	}
	p := mustTestParser[sum](t)

	ast, err := p.ParseString("", `"x" + "y" + 4`)
	assert.NoError(t, err)
	assert.Equal(t, &sum{Left: val{Str: `"x"`}, Ops: []op{{"+", val{Str: `"y"`}}, {"+", val{Int: 4}}}}, ast)

	ast, err = p.ParseString("", `"a" * 4 + "b"`)
	assert.NoError(t, err)
	assert.Equal(t, &sum{Left: val{Str: `"a"`}, Ops: []op{{"*", val{Int: 4}}, {"+", val{Str: `"b"`}}}}, ast)

	ast, err = p.ParseString("", `1 * 2 * 3`)
	assert.NoError(t, err)
	assert.Equal(t, &sum{Left: val{Int: 1}, Ops: []op{{"*", val{Int: 2}}, {"*", val{Int: 3}}}}, ast)

	ast, err = p.ParseString("", `"a" * "x" + "b"`)
	assert.Equal(t, err.Error(), `1:7: unexpected '"x"'`)
	ast, err = p.ParseString("", `4 * 2 + 0 * "b"`)
	assert.Equal(t, err.Error(), `1:13: unexpected '"b"'`)
}

func TestLookaheadGroup_Negative_SingleToken(t *testing.T) {
	type variable struct {
		Name string `@Ident`
	}
	type grammar struct {
		Identifiers []variable `((?! 'except'|'end') @@)*`
		Except      *variable  `('except' @@)? 'end'`
	}
	p := mustTestParser[grammar](t)

	ast, err := p.ParseString("", `one two three exception end`)
	assert.NoError(t, err)
	assert.Equal(t, []variable{{"one"}, {"two"}, {"three"}, {"exception"}}, ast.Identifiers)
	assert.Equal(t, nil, ast.Except)

	ast, err = p.ParseString("", `anything except this end`)
	assert.NoError(t, err)
	assert.Equal(t, []variable{{"anything"}}, ast.Identifiers)
	assert.Equal(t, &variable{"this"}, ast.Except)

	ast, err = p.ParseString("", `except the end`)
	assert.NoError(t, err)
	assert.Equal(t, nil, ast.Identifiers)
	assert.Equal(t, &variable{"the"}, ast.Except)

	ast, err = p.ParseString("", `no ending`)
	assert.Equal(t, err.Error(), `1:10: unexpected token "<EOF>" (expected "end")`)

	ast, err = p.ParseString("", `no end in sight`)
	assert.Equal(t, err.Error(), `1:8: unexpected token "in"`)
}

func TestLookaheadGroup_Negative_MultipleTokens(t *testing.T) {
	type grammar struct {
		Parts []string `((?! '.' '.' '.') @(Ident | '.'))*`
	}
	p := mustTestParser[grammar](t)

	ast, err := p.ParseString("", `x.y.z.`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"x", ".", "y", ".", "z", "."}, ast.Parts)

	ast, err = p.ParseString("", `..x..`)
	assert.NoError(t, err)
	assert.Equal(t, []string{".", ".", "x", ".", "."}, ast.Parts)

	ast, err = p.ParseString("", `two.. are fine`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"two", ".", ".", "are", "fine"}, ast.Parts)

	ast, err = p.ParseString("", `but this... is just wrong`)
	assert.Equal(t, err.Error(), `1:9: unexpected token "."`)
}

func TestASTTokens(t *testing.T) {
	type subject struct {
		Tokens []lexer.Token

		Word string `@Ident`
	}

	type hello struct {
		Tokens []lexer.Token

		Subject subject `"hello" @@`
	}

	p := mustTestParser[hello](t,
		participle.Elide("Whitespace"),
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{"Ident", `\w+`},
			{"Whitespace", `\s+`},
		})))
	actual, err := p.ParseString("", "hello world")
	assert.NoError(t, err)
	tokens := []lexer.Token{
		{-2, "hello", lexer.Position{Line: 1, Column: 1}},
		{-3, " ", lexer.Position{Offset: 5, Line: 1, Column: 6}},
		{-2, "world", lexer.Position{Offset: 6, Line: 1, Column: 7}},
	}
	expected := &hello{
		Tokens: tokens,
		Subject: subject{
			Tokens: tokens[1:],
			Word:   "world",
		},
	}
	assert.Equal(t, expected, actual)
}

func TestCaptureIntoToken(t *testing.T) {
	type ast struct {
		Head lexer.Token   `@Ident`
		Tail []lexer.Token `@(Ident*)`
	}

	p := mustTestParser[ast](t)
	actual, err := p.ParseString("", "hello waz baz")
	assert.NoError(t, err)
	expected := &ast{
		Head: lexer.Token{-2, "hello", lexer.Position{Line: 1, Column: 1}},
		Tail: []lexer.Token{
			{-2, "waz", lexer.Position{Offset: 6, Line: 1, Column: 7}},
			{-2, "baz", lexer.Position{Offset: 10, Line: 1, Column: 11}},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestEndPos(t *testing.T) {
	type Ident struct {
		Pos    lexer.Position
		EndPos lexer.Position
		Text   string `parser:"@Ident"`
	}

	type AST struct {
		First  *Ident `parser:"@@"`
		Second *Ident `parser:"@@"`
	}

	var (
		Lexer = lexer.Must(lexer.New(lexer.Rules{
			"Root": {
				{"Ident", `[\w:]+`, nil},
				{"Whitespace", `[\r\t ]+`, nil},
			},
		}))

		Parser = participle.MustBuild[AST](
			participle.Lexer(Lexer),
			participle.Elide("Whitespace"),
		)
	)

	mod, err := Parser.Parse("", strings.NewReader("foo bar"))
	assert.NoError(t, err)
	assert.Equal(t, 0, mod.First.Pos.Offset)
	assert.Equal(t, 3, mod.First.EndPos.Offset)
}

func TestBug(t *testing.T) {
	type A struct {
		Shared string `parser:"@'1'"`
		Diff   string `parser:"@A"`
	}
	type B struct {
		Shared string `parser:"@'1'"`
		Diff   string `parser:"@B"`
	}
	type AST struct {
		Branch string `parser:"@'branch'"`
		A      *A     `parser:"( @@"`
		B      *B     `parser:"| @@ )"`
	}
	var (
		lexer = lexer.Must(lexer.New(lexer.Rules{
			"Root": {
				{"A", `@`, nil},
				{"B", `!`, nil},
				{"Ident", `[\w:]+`, nil},
				{"Whitespace", `[\r\t ]+`, nil},
			},
		}))
		parser = participle.MustBuild[AST](
			participle.Lexer(lexer),
			participle.Elide("Whitespace"),
		)
	)
	expected := &AST{
		Branch: "branch",
		B: &B{
			Shared: "1",
			Diff:   "!",
		},
	}
	actual, err := parser.Parse("name", strings.NewReader(`branch 1!`))
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

type sliceCapture string

func (c *sliceCapture) Capture(values []string) error {
	*c = sliceCapture(strings.ToUpper(values[0]))
	return nil
}

func TestCaptureOnSliceElements(t *testing.T) { // nolint:dupl
	type capture struct {
		Single   *sliceCapture   `@Capture`
		Slice    []sliceCapture  `@Capture @Capture`
		SlicePtr []*sliceCapture `@Capture @Capture`
	}

	parser := participle.MustBuild[capture]([]participle.Option{
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{Name: "Capture", Pattern: `[a-z]{3}`},
			{Name: "Whitespace", Pattern: `[\s|\n]+`},
		})),
		participle.Elide("Whitespace"),
	}...)

	captured, err := parser.ParseString("capture_slice", `abc def ijk lmn opq`)
	assert.NoError(t, err)

	expectedSingle := sliceCapture("ABC")
	expectedSlicePtr1 := sliceCapture("LMN")
	expectedSlicePtr2 := sliceCapture("OPQ")
	expected := &capture{
		Single:   &expectedSingle,
		Slice:    []sliceCapture{"DEF", "IJK"},
		SlicePtr: []*sliceCapture{&expectedSlicePtr1, &expectedSlicePtr2},
	}

	assert.Equal(t, expected, captured)
}

type sliceParse string

func (s *sliceParse) Parse(lex *lexer.PeekingLexer) error {
	token, err := lex.Peek(0)
	if err != nil {
		return err
	}
	if len(token.Value) != 3 {
		return participle.NextMatch
	}
	_, err = lex.Next()
	if err != nil {
		return err
	}
	*s = sliceParse(strings.Repeat(token.Value, 2))
	return nil
}

func TestParseOnSliceElements(t *testing.T) { // nolint:dupl
	type parse struct {
		Single *sliceParse  `@@`
		Slice  []sliceParse `@@+`
	}

	parser := participle.MustBuild[parse]([]participle.Option{
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{Name: "Element", Pattern: `[a-z]{3}`},
			{Name: "Whitespace", Pattern: `[\s|\n]+`},
		})),
		participle.Elide("Whitespace"),
	}...)

	parsed, err := parser.ParseString("parse_slice", `abc def ijk`)
	assert.NoError(t, err)

	expectedSingle := sliceParse("abcabc")
	expected := &parse{
		Single: &expectedSingle,
		Slice:  []sliceParse{"defdef", "ijkijk"},
	}

	assert.Equal(t, expected, parsed)
}

func TestUnmarshalNetIP(t *testing.T) {
	type grammar struct {
		IP net.IP `@IP`
	}

	parser := mustTestParser[grammar](t, participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
		{"IP", `[\d.]+`},
	})))
	ast, err := parser.ParseString("", "10.2.3.4")
	assert.NoError(t, err)
	assert.Equal(t, "10.2.3.4", ast.IP.String())
}

type Address net.IP

func (a *Address) Capture(values []string) error {
	fmt.Println("does not run at all")
	*a = Address(net.ParseIP(values[0]))
	return nil
}

func TestCaptureIP(t *testing.T) {
	type grammar struct {
		IP Address `@IP`
	}

	parser := mustTestParser[grammar](t, participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
		{"IP", `[\d.]+`},
	})))
	ast, err := parser.ParseString("", "10.2.3.4")
	assert.NoError(t, err)
	assert.Equal(t, "10.2.3.4", (net.IP)(ast.IP).String())
}

func BenchmarkIssue143(b *testing.B) {
	type Disjunction struct {
		Long1 bool `parser:"  '<' '1' ' ' 'l' 'o' 'n' 'g' ' ' 'r' 'u' 'l' 'e' ' ' 't' 'o' ' ' 'f' 'o' 'r' 'm' 'a' 't' '>'"`
		Long2 bool `parser:"| '<' '2' ' ' 'l' 'o' 'n' 'g' ' ' 'r' 'u' 'l' 'e' ' ' 't' 'o' ' ' 'f' 'o' 'r' 'm' 'a' 't' '>'"`
		Long3 bool `parser:"| '<' '3' ' ' 'l' 'o' 'n' 'g' ' ' 'r' 'u' 'l' 'e' ' ' 't' 'o' ' ' 'f' 'o' 'r' 'm' 'a' 't' '>'"`
		Long4 bool `parser:"| '<' '4' ' ' 'l' 'o' 'n' 'g' ' ' 'r' 'u' 'l' 'e' ' ' 't' 'o' ' ' 'f' 'o' 'r' 'm' 'a' 't' '>'"`
		Real  bool `parser:"| '<' 'x' '>'"`
	}

	type Disjunctions struct {
		List []Disjunction `parser:"@@*"`
	}

	var disjunctionParser = participle.MustBuild[Disjunctions]()
	input := "<x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x> <x>"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := disjunctionParser.ParseString("", input); err != nil {
			panic(err)
		}
	}
}

type Boxes struct {
	Pos   lexer.Position
	Boxes Box `@Ident`
}

type Box struct {
	Pos lexer.Position
	Val string `@Ident`
}

func (b *Box) Capture(values []string) error {
	b.Val = values[0]
	return nil
}

func TestBoxedCapture(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"Ident", `[a-zA-Z](\w|\.|/|:|-)*`},
		{"whitespace", `\s+`},
	})

	parser := participle.MustBuild[Boxes](participle.Lexer(lex), participle.UseLookahead(2))
	if _, err := parser.ParseString("test", "abc::cdef.abc"); err != nil {
		t.Fatal(err)
	}
}

func TestMatchEOF(t *testing.T) {
	type testMatchNewlineOrEOF struct {
		Text []string `@Ident+ ("\n" | EOF)`
	}
	p := mustTestParser[testMatchNewlineOrEOF](t)
	_, err := p.ParseString("", "hell world")
	assert.NoError(t, err)
	_, err = p.ParseString("", "hell world\n")
	assert.NoError(t, err)
}
