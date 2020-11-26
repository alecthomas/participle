package participle_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2"
)

func TestIssue3Example1(t *testing.T) {
	type LAT1Decl struct {
		SourceFilename string `  "source_filename" "=" @String`
		DataLayout     string `| "target" "datalayout" "=" @String`
		TargetTriple   string `| "target" "triple" "=" @String`
	}

	type LAT1Module struct {
		Decls []*LAT1Decl `{ @@ }`
	}

	g := &LAT1Module{}
	p := mustTestParser(t, g, participle.UseLookahead(5), participle.Unquote())
	err := p.ParseString("", `
		source_filename = "foo.c"
		target datalayout = "bar"
		target triple = "baz"
	`, g)
	require.NoError(t, err)
	require.Equal(t,
		&LAT1Module{
			Decls: []*LAT1Decl{
				{SourceFilename: "foo.c"},
				{DataLayout: "bar"},
				{TargetTriple: "baz"},
			},
		}, g)
}

type LAT2Config struct {
	Entries []*LAT2Entry `@@ { @@ }`
}

type LAT2Entry struct {
	Attribute *LAT2Attribute `@@`
	Group     *LAT2Group     `| @@`
}

type LAT2Attribute struct {
	Key   string `@Ident "="`
	Value string `@String`
}

type LAT2Group struct {
	Name    string       `@Ident "{"`
	Entries []*LAT2Entry `@@ { @@ } "}"`
}

func TestIssue3Example2(t *testing.T) {
	g := &LAT2Config{}
	p := mustTestParser(t, g, participle.UseLookahead(2), participle.Unquote())
	err := p.ParseString("", `
		key = "value"
		block {
			inner_key = "inner_value"
		}
	`, g)
	require.NoError(t, err)
	require.Equal(t,
		&LAT2Config{
			Entries: []*LAT2Entry{
				{Attribute: &LAT2Attribute{Key: "key", Value: "value"}},
				{
					Group: &LAT2Group{
						Name: "block",
						Entries: []*LAT2Entry{
							{Attribute: &LAT2Attribute{Key: "inner_key", Value: "inner_value"}},
						},
					},
				},
			},
		},
		g,
	)
}

type LAT3Grammar struct {
	Expenses []*LAT3Expense `{ @@ }`
}

type LAT3Expense struct {
	Name   string     `@Ident "paid"`
	Amount *LAT3Value `@@ { Ident } "."`
}

type LAT3Value struct {
	Float   float64 `  "$" @Float`
	Integer int     `| "$" @Int`
}

func TestIssue11(t *testing.T) {
	g := &LAT3Grammar{}
	p := mustTestParser(t, g, participle.UseLookahead(5))
	err := p.ParseString("", `
		A paid $30.80 for snacks.
		B paid $70 for housecleaning.
		C paid $63.50 for utilities.
	`, g)
	require.NoError(t, err)
	require.Equal(t,
		g,
		&LAT3Grammar{
			Expenses: []*LAT3Expense{
				{Name: "A", Amount: &LAT3Value{Float: 30.8}},
				{Name: "B", Amount: &LAT3Value{Integer: 70}},
				{Name: "C", Amount: &LAT3Value{Float: 63.5}},
			},
		},
	)
}

func TestLookaheadOptional(t *testing.T) {
	type grammar struct {
		Key   string `[ @Ident "=" ]`
		Value string `@Ident`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))
	actual := &grammar{}
	err := p.ParseString("", `value`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Value: "value"}, actual)
	err = p.ParseString("", `key = value`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Key: "key", Value: "value"}, actual)
}

func TestLookaheadOptionalNoTail(t *testing.T) {
	type grammar struct {
		Key   string `@Ident`
		Value string `[ "=" @Int ]`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))
	actual := &grammar{}
	err := p.ParseString("", `key`, actual)
	require.NoError(t, err)
}

func TestLookaheadDisjunction(t *testing.T) {
	type grammar struct {
		B string `  "hello" @Ident "world"`
		C string `| "hello" "world" @Ident`
		A string `| "hello" @Ident`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))

	g := &grammar{}
	err := p.ParseString("", `hello moo`, g)
	require.NoError(t, err)
	require.Equal(t, &grammar{A: "moo"}, g)

	err = p.ParseString("", `hello moo world`, g)
	require.NoError(t, err)
	require.Equal(t, &grammar{B: "moo"}, g)
}

func TestLookaheadNestedDisjunctions(t *testing.T) {
	g := &struct {
		A string `  "hello" ( "foo" @Ident | "bar" "waz" @Ident)`
		B string `| "hello" @"world"`
	}{}
	p := mustTestParser(t, g, participle.UseLookahead(5))

	err := p.ParseString("", `hello foo FOO`, g)
	require.NoError(t, err)
	require.Equal(t, g.A, "FOO")

	err = p.ParseString("", `hello world`, g)
	require.NoError(t, err)
	require.Equal(t, g.B, "world")
}

func TestLookaheadTerm(t *testing.T) {
	g := &struct {
		A string `  @Ident`
		B struct {
			A string `@String`
		} `| @@`
		C struct {
			A string `@String`
			B string `"…" @String`
		} `| @@`
		D struct {
			A string `"[" @Ident "]"`
		} `| @@`
		E struct {
			A string `"(" @Ident ")"`
		} `| @@`
	}{}
	mustTestParser(t, g, participle.UseLookahead(5))
}

// Term holds the different possible terms
type issue28Term struct {
	KV   *issue28KV ` @@ `
	Text *string    `| @String `
}

// KV represents a json kv
type issue28KV struct {
	Key   *issue28Key   `@@`
	Value *issue28Value `@@`
}

// Key holds the possible key types for a kv
type issue28Key struct {
	Ident *string `@Ident ":"`
	Str   *string `| @String ":"`
}

// Value holds the possible values for a kv
type issue28Value struct {
	Bool  *bool    `(@"true" | "false")`
	Str   *string  `| @String`
	Ident *string  `| @Ident`
	Int   *int64   `| @Int`
	Float *float64 `| @Float`
}

func TestIssue28(t *testing.T) {
	p := mustTestParser(t, &issue28Term{}, participle.UseLookahead(5), participle.Unquote())

	actual := &issue28Term{}
	err := p.ParseString("", `"key": "value"`, actual)
	require.NoError(t, err)
	key := "key"
	value := "value"
	expected := &issue28Term{
		KV: &issue28KV{
			Key: &issue28Key{
				Str: &key,
			},
			Value: &issue28Value{
				Str: &value,
			},
		},
	}
	require.Equal(t, expected, actual)

	err = p.ParseString("", `"some text string"`, actual)
	require.NoError(t, err)
	text := "some text string"
	expected = &issue28Term{
		Text: &text,
	}
	require.Equal(t, expected, actual)
}

// This test used to fail because the lookahead table only tracks (root, depth, token) for each root. In this case there
// are two roots that have the same second token (0, 1, "=") and (2, 1, "="). As (depth, token) is the uniqueness
// constraint, this never disambiguates.
//
// To solve this, each ambiguous group will need to track the history of tokens.
//
// eg.
//
// 		0.	groups = [
//   			{history: [">"] roots: [0, 1]},
// 				{history: ["<"], roots: [2, 3]},
//     		]
//      1.	groups = [
//      		{history: [">", "="], roots: [0]},
//         		{history: [">"], roots: [1]},
//         		{history: ["<", "="], roots: [2]},
//         		{history: ["<"], roots: [3]},
//           ]
func TestLookaheadWithConvergingTokens(t *testing.T) {
	type grammar struct {
		Left string   `@Ident`
		Op   string   `[ @( ">" "=" | ">" | "<" "=" | "<" )`
		Next *grammar `  @@ ]`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))
	actual := &grammar{}
	err := p.ParseString("", "a >= b", actual)
	require.NoError(t, err)
}

// type leftRecursionType struct {
// 	Type     string                 `  @("int" | "float" | "string")`
// 	Function *leftRecursionFuncType `| @@`
// }

// type leftRecursionFuncType struct {
// 	Return   *leftRecursionType   `@@`
// 	Function string               `@Ident`
// 	Args     []*leftRecursionType `"(" @@ { "," @@ } ")"`
// }

// func TestLeftRecursion(t *testing.T) {
// 	p := mustTestParser(t, &leftRecursionType{}, participle.UseLookahead(5))
// 	actual := &leftRecursionType{}
// 	err := p.ParseString(`int f()`, actual)
// 	require.NoError(t, err)
// 	require.Equal(t, &leftRecursionType{
// 		Function: &leftRecursionFuncType{
// 			Return:   &leftRecursionType{Type: "int"},
// 			Function: "f",
// 		},
// 	}, actual)
// }

func TestIssue27(t *testing.T) {
	type grammar struct {
		Number int    `  @(["-"] Int)`
		String string `| @String`
	}
	p := mustTestParser(t, &grammar{})
	actual := &grammar{}

	err := p.ParseString("", `- 100`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Number: -100}, actual)

	err = p.ParseString("", `100`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Number: 100}, actual)
}

func TestLookaheadDisambiguateByType(t *testing.T) {
	type grammar struct {
		Int   int     `  @(["-"] Int)`
		Float float64 `| @(["-"] Float)`
	}

	p := mustTestParser(t, &grammar{}, participle.UseLookahead(5))
	actual := &grammar{}

	err := p.ParseString("", `- 100`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Int: -100}, actual)

	err = p.ParseString("", `- 100.5`, actual)
	require.NoError(t, err)
	require.Equal(t, &grammar{Float: -100.5}, actual)
}

func TestShowNearestError(t *testing.T) {
	type grammar struct {
		A string `  @"a" @"b" @"c"`
		B string `| @"a" @"z"`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(10))
	actual := &grammar{}
	err := p.ParseString("", `a b d`, actual)
	require.EqualError(t, err, `1:5: unexpected token "d" (expected "c")`)
}

func TestRewindDisjunction(t *testing.T) {
	type grammar struct {
		Function string `  @Ident "(" ")"`
		Ident    string `| @Ident`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(2))
	ast := &grammar{}
	err := p.ParseString("", `name`, ast)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ident: "name"}, ast)
}

func TestRewindOptional(t *testing.T) {
	type grammar struct {
		Var string `  [ "int" "int" ] @Ident`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(3))
	ast := &grammar{}

	err := p.ParseString("", `one`, ast)
	require.NoError(t, err)
	require.Equal(t, &grammar{Var: "one"}, ast)

	err = p.ParseString("", `int int one`, ast)
	require.NoError(t, err)
	require.Equal(t, &grammar{Var: "one"}, ast)
}

func TestRewindRepetition(t *testing.T) {
	type grammar struct {
		Ints  []string `{ @"int" }`
		Ident string   `@Ident`
	}
	p := mustTestParser(t, &grammar{}, participle.UseLookahead(3))
	ast := &grammar{}

	err := p.ParseString("", `int int one`, ast)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ints: []string{"int", "int"}, Ident: "one"}, ast)

	err = p.ParseString("", `int int one`, ast)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ints: []string{"int", "int"}, Ident: "one"}, ast)
}
