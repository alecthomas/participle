package participle_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2"
)

// Aliases are non-capturing grammar fragments: they match literal and named
// token patterns but cannot bind values — captures (@ and @@) always belong
// to a struct field of the surrounding production, so aliases reject them at
// build time.

func TestAliasLiteralPattern(t *testing.T) {
	type grammar struct {
		V string `<comma> @Ident`
	}
	parser, err := participle.Build[grammar](
		participle.Alias("comma", `":"`),
	)
	assert.NoError(t, err)

	// <comma> matches ":", then the field captures the Ident.
	actual, err := parser.ParseString("", ":hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", actual.V)
}

func TestAliasReusedAcrossFields(t *testing.T) {
	type grammar struct {
		A string `@Ident <delim> @Ident`
		B string `<delim> @Ident`
	}
	parser, err := participle.Build[grammar](
		participle.Alias("delim", `"|"`),
	)
	assert.NoError(t, err)

	actual, err := parser.ParseString("", "x | y | z")
	assert.NoError(t, err)
	// Non-capturing <delim> is matched but not part of the captured value.
	assert.Equal(t, "xy", actual.A)
	assert.Equal(t, "z", actual.B)
}

func TestAliasMultipleAliases(t *testing.T) {
	type grammar struct {
		V string `<open> @Ident <close>`
	}
	parser, err := participle.Build[grammar](
		participle.Alias("open", `"("`),
		participle.Alias("close", `")"`),
	)
	assert.NoError(t, err)

	actual, err := parser.ParseString("", "(name)")
	assert.NoError(t, err)
	assert.Equal(t, "name", actual.V)
}

func TestAliasModifierRepetition(t *testing.T) {
	type grammar struct {
		Parts []string `<comma>? @Ident`
	}
	_ = grammar{}
	type g2 struct {
		Names string `(<comma>)? @Ident`
	}
	parser, err := participle.Build[g2](
		participle.Alias("comma", `","`),
	)
	assert.NoError(t, err)

	actual, err := parser.ParseString("", ",hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", actual.Names)
}

func TestAliasMustNotCapture(t *testing.T) {
	type grammar struct {
		X string `<bad>`
	}
	_, err := participle.Build[grammar](
		participle.Alias("bad", `@Ident`),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot capture")
}

func TestAliasMustNotCaptureStruct(t *testing.T) {
	type grammar struct {
		Items []string `<list>`
	}
	_, err := participle.Build[grammar](
		participle.Alias("list", `@@+`),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot capture")
}

func TestAliasUnknownReference(t *testing.T) {
	type grammar struct {
		X string `<nope>`
	}
	_, err := participle.Build[grammar]()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alias")
}

func TestAliasDuplicate(t *testing.T) {
	type grammar struct {
		X string `<a>`
	}
	_, err := participle.Build[grammar](
		participle.Alias("a", `"x"`),
		participle.Alias("a", `"y"`),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestAliasEmptyName(t *testing.T) {
	type grammar struct {
		X string `<a>`
	}
	_, err := participle.Build[grammar](
		participle.Alias("", `"x"`),
	)
	assert.Error(t, err)
}

func TestAliasUnknownReferenceInGrammarToo(t *testing.T) {
	// An undeclared alias used in a tag surfaces as an error.
	type grammar struct {
		X string `<undeclared>`
	}
	type grammar2 struct {
		X string `<declared>`
	}
	_, err := participle.Build[grammar2](
		participle.Alias("other", `"x"`),
	)
	assert.Error(t, err)
	_ = grammar{}
}
