package participle_test

import (
	"sort"
	"testing"

	"github.com/alecthomas/assert/v2"
)

type lookaheadStmt struct {
	If   *lookaheadIf   `@@`
	Loop *lookaheadLoop `| @@`
	Call *lookaheadCall `| @@`
}

type lookaheadIf struct {
	Keyword string `"if" @Ident`
}

type lookaheadLoop struct {
	Keyword string `"loop" @Ident`
}

type lookaheadCall struct {
	Keyword string `"call" @Ident`
}

func TestLookahead(t *testing.T) {
	parser := mustTestParser[lookaheadStmt](t)

	routing, err := parser.Lookahead(&lookaheadStmt{})
	assert.NoError(t, err)

	assert.Equal(t, []int{0}, routing[`"if"`])
	assert.Equal(t, []int{1}, routing[`"loop"`])
	assert.Equal(t, []int{2}, routing[`"call"`])
}

func TestLookaheadGroupStart(t *testing.T) {
	type grammar struct {
		A string `@("foo" | "bar")`
	}
	parser := mustTestParser[grammar](t)

	// A single-sequence production has no top-level disjunction to route on,
	// so Lookahead reports that explicitly.
	_, err := parser.Lookahead(&grammar{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have alternatives")

	// The start-token set, however, reflects both group members.
	tokens, err := parser.StartTokens(&grammar{})
	assert.NoError(t, err)
	sort.Strings(tokens)
	assert.Equal(t, []string{`"bar"`, `"foo"`}, tokens)
}

func TestLookaheadNoDisjunction(t *testing.T) {
	type scalars struct {
		A string `@Ident`
	}
	parser := mustTestParser[scalars](t)

	_, err := parser.Lookahead(&scalars{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have alternatives")

	tokens, err := parser.StartTokens(&scalars{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"<ident>"}, tokens)
}

func TestLookaheadUnknownType(t *testing.T) {
	parser := mustTestParser[lookaheadStmt](t)

	type unknown struct {
		X string `@Ident`
	}
	_, err := parser.Lookahead(&unknown{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a production of type")
}

func TestStartTokens(t *testing.T) {
	parser := mustTestParser[lookaheadStmt](t)

	tokens, err := parser.StartTokens(&lookaheadStmt{})
	assert.NoError(t, err)

	sort.Strings(tokens)
	assert.Equal(t, []string{`"call"`, `"if"`, `"loop"`}, tokens)
}

func TestStartTokensNested(t *testing.T) {
	parser := mustTestParser[lookaheadStmt](t)

	tokens, err := parser.StartTokens(&lookaheadCall{})
	assert.NoError(t, err)
	assert.Equal(t, []string{`"call"`}, tokens)
}
