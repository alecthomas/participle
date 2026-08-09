package participle_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2"
)

// recoverStmt is a statement list with fault-tolerant recovery: a malformed
// statement is skipped and parsing resumes at the next statement keyword.
type recoverStmt struct {
	Stmts []*stmt `@@*`
}

type stmt struct {
	If   *ifStmt   `@@`
	Loop *loopStmt `| @@`
	Call *callStmt `| @@`
}

type ifStmt struct {
	Keyword string   `"if" @Ident "{"`
	Body    []*stmt  `@@*`
	Close   struct{} `"}"`
}

type loopStmt struct {
	Keyword string   `"loop" @Ident "{"`
	Body    []*stmt  `@@*`
	Close   struct{} `"}"`
}

type callStmt struct {
	Call string `"call" @Ident`
}

type recovExpr interface{ recovExpr() }

type numExpr struct {
	N string `@Int`
}

func (*numExpr) recovExpr() {}

type addExpr struct {
	L *numExpr  `@@`
	R recovExpr `"+" @@`
}

func (*addExpr) recovExpr() {}

type unionGrammar struct {
	Exprs []recovExpr `@@*`
}

func recoverParser(t *testing.T) *participle.Parser[recoverStmt] {
	t.Helper()
	parser, err := participle.Build[recoverStmt](
		participle.RecoverTo(&stmt{}),
	)
	assert.NoError(t, err)
	return parser
}

func callNames(stmts []*stmt) []string {
	var names []string
	for _, s := range stmts {
		switch {
		case s.If != nil:
			names = append(names, "if")
		case s.Loop != nil:
			names = append(names, "loop")
		case s.Call != nil:
			names = append(names, "call:"+s.Call.Call)
		}
	}
	return names
}

func TestRecoverToSkipsMalformedStatement(t *testing.T) {
	parser := recoverParser(t)

	// "broken" cannot begin any statement alternative and must be skipped;
	// parsing resumes at "call done".
	actual, err := parser.ParseString("", "call one broken call done")
	assert.NoError(t, err)
	assert.Equal(t, []string{"call:one", "call:done"}, callNames(actual.Stmts))
}

func TestRecoverToResumesAfterBrace(t *testing.T) {
	parser := recoverParser(t)

	// A malformed statement inside a block should skip to the next statement
	// keyword, even within the same block, without crossing the block's
	// closing brace.
	actual, err := parser.ParseString("", "if a { call one broken call two } call after")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(actual.Stmts))
	assert.NotEqual(t, (*ifStmt)(nil), actual.Stmts[0].If)
	ifStmt := actual.Stmts[0].If
	assert.Equal(t, []string{"call:one", "call:two"}, callNames(ifStmt.Body))
	assert.Equal(t, "after", actual.Stmts[1].Call.Call)
}

func TestRecoverToNoSyncTokenReturnsOriginalError(t *testing.T) {
	parser := recoverParser(t)

	// "broken" has no following statement keyword, so no synchronization
	// point exists and the original error must be returned.
	_, err := parser.ParseString("", "call broken @@")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected token")
}

func TestRecoverToWithoutOptionIsUnchanged(t *testing.T) {
	parser, err := participle.Build[recoverStmt]()
	assert.NoError(t, err)

	_, err = parser.ParseString("", "call one broken call done")
	assert.Error(t, err)
}

func TestRecoverToSkipsMultipleMalformed(t *testing.T) {
	parser := recoverParser(t)

	actual, err := parser.ParseString("", "broken1 broken2 call a call b")
	assert.NoError(t, err)
	assert.Equal(t, []string{"call:a", "call:b"}, callNames(actual.Stmts))
}

func TestRecoverToUseLookahead(t *testing.T) {
	parser, err := participle.Build[recoverStmt](
		participle.UseLookahead(10),
		participle.RecoverTo(&stmt{}),
	)
	assert.NoError(t, err)

	actual, err := parser.ParseString("", "call one broken call two")
	assert.NoError(t, err)
	assert.Equal(t, []string{"call:one", "call:two"}, callNames(actual.Stmts))
}

func TestRecoverToCaseInsensitiveLiteral(t *testing.T) {
	type ciStmt struct {
		Call string `"CALL" @Ident`
	}
	type ciGrammar struct {
		Stmts []*ciStmt `@@*`
	}
	parser, err := participle.Build[ciGrammar](
		participle.CaseInsensitive("Ident"),
		participle.RecoverTo(&ciStmt{}),
	)
	assert.NoError(t, err)

	// The "CALL" literal is matched case-insensitively, so "call ok" is a
	// valid statement; "broken" is skipped via recovery.
	actual, err := parser.ParseString("", "broken call ok")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actual.Stmts))
	assert.Equal(t, "ok", actual.Stmts[0].Call)
}

func TestRecoverToUnknownTypeIsBuildError(t *testing.T) {
	type unknown struct {
		X string `@Ident`
	}
	_, err := participle.Build[recoverStmt](
		participle.RecoverTo(&unknown{}),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a production of type")
}

func TestRecoverToUnion(t *testing.T) {
	parser, err := participle.Build[unionGrammar](
		participle.Union[recovExpr](&addExpr{}, &numExpr{}),
		participle.RecoverTo(&numExpr{}),
	)
	assert.NoError(t, err)

	// "1" parses, "broken" is not a valid expression and is skipped, "2 + 3"
	// is recovered from the "2".
	actual, err := parser.ParseString("", "1 broken 2 + 3")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actual.Exprs))
}

func TestRecoverToOneOrMore(t *testing.T) {
	type oneOrMore struct {
		Stmts []*stmt `@@+`
	}
	parser, err := participle.Build[oneOrMore](
		participle.RecoverTo(&stmt{}),
	)
	assert.NoError(t, err)

	actual, err := parser.ParseString("", "call one broken call two")
	assert.NoError(t, err)
	assert.Equal(t, []string{"call:one", "call:two"}, callNames(actual.Stmts))
}

func TestRecoverToStructLevelSync(t *testing.T) {
	parser := recoverParser(t)

	// A broken "if" statement (missing opening brace) whose failure position
	// lands on a synchronization token ("call"). The partial "if" AST is
	// retained (best effort) and "call done" is parsed as a new statement.
	actual, err := parser.ParseString("", "if broken call done")
	assert.NoError(t, err)
	assert.Equal(t, []string{"if", "call:done"}, callNames(actual.Stmts))
}

// FuzzRecoverTo ensures recovery never panics or hangs on arbitrary input,
// even with malformed structures and adversarial token runs.
func FuzzRecoverTo(f *testing.F) {
	parser, err := participle.Build[recoverStmt](
		participle.RecoverTo(&stmt{}),
	)
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range []string{
		"",
		"call one",
		"call one broken call two",
		"if a { broken } call after",
		"} { [ {",
		"if broken call done",
		"loop a { call x } loop b { broken }",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		//nolint:errcheck // any error is an acceptable fuzz outcome.
		_, _ = parser.ParseString("", in)
	})
}
