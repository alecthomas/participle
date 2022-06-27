package participle_test

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/participle/v2"
)

func TestIssue3Example1(t *testing.T) {
	type LAT1Decl struct {
		SourceFilename string `  "source_filename" "=" @String`
		DataLayout     string `| "target" "datalayout" "=" @String`
		TargetTriple   string `| "target" "triple" "=" @String`
	}

	type LAT1Module struct {
		Decls []*LAT1Decl `@@*`
	}

	p := mustTestParser[LAT1Module](t, participle.UseLookahead(5), participle.Unquote())
	g, err := p.ParseString("", `
		source_filename = "foo.c"
		target datalayout = "bar"
		target triple = "baz"
	`)
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
	Entries []*LAT2Entry `@@+`
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
	Entries []*LAT2Entry `@@+ "}"`
}

func TestIssue3Example2(t *testing.T) {
	p := mustTestParser[LAT2Config](t, participle.UseLookahead(2), participle.Unquote())
	g, err := p.ParseString("", `
		key = "value"
		block {
			inner_key = "inner_value"
		}
	`)
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
	Amount *LAT3Value `@@ Ident* "."`
}

type LAT3Value struct {
	Float   float64 `  "$" @Float`
	Integer int     `| "$" @Int`
}

func TestIssue11(t *testing.T) {
	p := mustTestParser[LAT3Grammar](t, participle.UseLookahead(5))
	g, err := p.ParseString("", `
		A paid $30.80 for snacks.
		B paid $70 for housecleaning.
		C paid $63.50 for utilities.
	`)
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
	p := mustTestParser[grammar](t, participle.UseLookahead(5))
	actual, err := p.ParseString("", `value`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Value: "value"}, actual)
	actual, err = p.ParseString("", `key = value`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Key: "key", Value: "value"}, actual)
}

func TestLookaheadOptionalNoTail(t *testing.T) {
	type grammar struct {
		Key   string `@Ident`
		Value string `[ "=" @Int ]`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(5))
	_, err := p.ParseString("", `key`)
	require.NoError(t, err)
}

func TestLookaheadDisjunction(t *testing.T) {
	type grammar struct {
		B string `  "hello" @Ident "world"`
		C string `| "hello" "world" @Ident`
		A string `| "hello" @Ident`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(5))

	g, err := p.ParseString("", `hello moo`)
	require.NoError(t, err)
	require.Equal(t, &grammar{A: "moo"}, g)

	g, err = p.ParseString("", `hello moo world`)
	require.NoError(t, err)
	require.Equal(t, &grammar{B: "moo"}, g)
}

func TestLookaheadNestedDisjunctions(t *testing.T) {
	type grammar struct {
		A string `  "hello" ( "foo" @Ident | "bar" "waz" @Ident)`
		B string `| "hello" @"world"`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(5))

	g, err := p.ParseString("", `hello foo FOO`)
	require.NoError(t, err)
	require.Equal(t, g.A, "FOO")

	g, err = p.ParseString("", `hello world`)
	require.NoError(t, err)
	require.Equal(t, g.B, "world")
}

func TestLookaheadTerm(t *testing.T) {
	type grammar struct {
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
	}
	mustTestParser[grammar](t, participle.UseLookahead(5))
}

func TestIssue28(t *testing.T) {
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

	// KV represents a json kv
	type issue28KV struct {
		Key   *issue28Key   `@@`
		Value *issue28Value `@@`
	}

	// Term holds the different possible terms
	type issue28Term struct {
		KV   *issue28KV ` @@ `
		Text *string    `| @String `
	}

	p := mustTestParser[issue28Term](t, participle.UseLookahead(5), participle.Unquote())

	actual, err := p.ParseString("", `"key": "value"`)
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

	actual, err = p.ParseString("", `"some text string"`)
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
	p := mustTestParser[grammar](t, participle.UseLookahead(5))
	_, err := p.ParseString("", "a >= b")
	require.NoError(t, err)
}

func TestIssue27(t *testing.T) {
	type grammar struct {
		Number int    `  @(["-"] Int)`
		String string `| @String`
	}
	p := mustTestParser[grammar](t)
	actual, err := p.ParseString("", `- 100`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Number: -100}, actual)

	actual, err = p.ParseString("", `100`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Number: 100}, actual)
}

func TestLookaheadDisambiguateByType(t *testing.T) {
	type grammar struct {
		Int   int     `  @(["-"] Int)`
		Float float64 `| @(["-"] Float)`
	}

	p := mustTestParser[grammar](t, participle.UseLookahead(5))

	actual, err := p.ParseString("", `- 100`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Int: -100}, actual)

	actual, err = p.ParseString("", `- 100.5`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Float: -100.5}, actual)
}

func TestShowNearestError(t *testing.T) {
	type grammar struct {
		A string `  @"a" @"b" @"c"`
		B string `| @"a" @"z"`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(10))
	_, err := p.ParseString("", `a b d`)
	require.EqualError(t, err, `1:5: unexpected token "d" (expected "c")`)
}

func TestRewindDisjunction(t *testing.T) {
	type grammar struct {
		Function string `  @Ident "(" ")"`
		Ident    string `| @Ident`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(2))
	ast, err := p.ParseString("", `name`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ident: "name"}, ast)
}

func TestRewindOptional(t *testing.T) {
	type grammar struct {
		Var string `  [ "int" "int" ] @Ident`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(3))

	ast, err := p.ParseString("", `one`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Var: "one"}, ast)

	ast, err = p.ParseString("", `int int one`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Var: "one"}, ast)
}

func TestRewindRepetition(t *testing.T) {
	type grammar struct {
		Ints  []string `(@"int")*`
		Ident string   `@Ident`
	}
	p := mustTestParser[grammar](t, participle.UseLookahead(3))

	ast, err := p.ParseString("", `int int one`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ints: []string{"int", "int"}, Ident: "one"}, ast)

	ast, err = p.ParseString("", `int int one`)
	require.NoError(t, err)
	require.Equal(t, &grammar{Ints: []string{"int", "int"}, Ident: "one"}, ast)
}
