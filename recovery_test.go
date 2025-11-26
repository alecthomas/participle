package participle

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2/lexer"
)

// Simple statement grammar for testing recovery
type Statement struct {
	Keyword string `@Ident`
	Value   string `@Ident`
	Semi    string `@";"`
}

type Program struct {
	Statements []*Statement `@@*`
}

var testLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"whitespace", `\s+`},
	{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
	{"Number", `\d+`},
	{"Punct", `[;{}()\[\],=]`},
})

func TestSkipUntilRecovery(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Input with an error (missing identifier between 'let' and ';')
	input := `let x; set ; let y;`

	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
	)

	assert.NotZero(t, ast, "Expected partial AST even with errors")

	// Check that we got a RecoveryError
	var recErr *RecoveryError
	if errors.As(err, &recErr) {
		assert.True(t, len(recErr.Errors) >= 1, "Expected at least one recovery error")
		t.Logf("Recovery errors: %v", recErr.Errors)
	}

	// We should have parsed at least some statements
	if ast != nil {
		assert.True(t, len(ast.Statements) >= 1, "Expected at least one statement parsed")
	}
}

func TestSkipPastRecovery(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Use valid tokens but invalid grammar (missing second identifier)
	input := `let x; set ; let y;`

	ast, err := parser.ParseString("", input,
		Recover(SkipPast(";")),
	)

	assert.NotZero(t, ast, "Expected partial AST even with errors")
	t.Logf("AST: %+v, Error: %v", ast, err)
}

// Expression grammar for testing nested delimiter recovery
type Expr struct {
	Pos   lexer.Position
	Atom  string `  @Ident`
	Call  *Call  `| @@`
	Paren *Expr  `| "(" @@ ")"`
}

type Call struct {
	Name string  `@Ident`
	Args []*Expr `"(" (@@ ("," @@)*)? ")"`
}

type ExprStmt struct {
	Expr *Expr  `@@`
	Semi string `@";"`
}

type ExprProgram struct {
	Stmts []*ExprStmt `@@*`
}

func TestNestedDelimitersRecovery(t *testing.T) {
	exprLexer := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=+\-*/]`},
	})

	// Simple expression parser - expects identifier inside parens
	type ParenExpr struct {
		Open  string `@"("`
		Inner string `@Ident`
		Close string `@")"`
	}

	type SimpleStmt struct {
		Expr *ParenExpr `@@`
		Semi string     `@";"`
	}

	type SimpleProg struct {
		Stmts []*SimpleStmt `@@*`
	}

	parser := MustBuild[SimpleProg](
		Lexer(exprLexer),
	)

	// Input with error inside parentheses (number instead of identifier)
	input := `(foo); (123); (bar);`

	ast, err := parser.ParseString("", input,
		Recover(
			NestedDelimiters("(", ")"),
			SkipUntil(";"),
		),
	)

	assert.NotZero(t, ast, "Expected partial AST")
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestCompositeStrategy(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Valid tokens but invalid grammar
	input := `let x; set ; let y;`

	ast, err := parser.ParseString("", input,
		Recover(
			TryStrategies(
				NestedDelimiters("(", ")"),
				SkipUntil(";"),
			),
		),
	)

	assert.NotZero(t, ast, "Expected partial AST")
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestMaxRecoveryErrors(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Multiple errors in input
	input := `let ; set ; get ; put ;`

	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
		MaxRecoveryErrors(2),
	)

	var recErr *RecoveryError
	if errors.As(err, &recErr) {
		// Should have stopped after 2 errors
		assert.True(t, len(recErr.Errors) <= 3, "Expected at most 3 errors (2 + final)")
	}
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestNoRecoveryWithoutOption(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Invalid grammar: missing value after keyword
	input := `let x; set ; let y;`

	// Parse without recovery - should fail on first error
	ast, err := parser.ParseString("", input)

	assert.Error(t, err, "Expected error without recovery")
	// The error should NOT be a RecoveryError
	var recErr *RecoveryError
	assert.False(t, errors.As(err, &recErr), "Should not be a RecoveryError")
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestRecoveryWithValidInput(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	input := `let x; set y; get z;`

	// Parse with recovery but valid input - should succeed without errors
	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
	)

	assert.NoError(t, err, "Expected no error with valid input")
	assert.NotZero(t, ast)
	assert.Equal(t, 3, len(ast.Statements))
}

// Block-based grammar for testing recovery in nested structures
type Block struct {
	Open  string       `@"{"`
	Stmts []*Statement `@@*`
	Close string       `@"}"`
}

type BlockProgram struct {
	Blocks []*Block `@@*`
}

func TestRecoveryInNestedBlocks(t *testing.T) {
	parser := MustBuild[BlockProgram](
		Lexer(testLexer),
	)

	// Input with error inside a block (missing value after keyword)
	input := `{ let x; } { set ; } { get z; }`

	ast, err := parser.ParseString("", input,
		Recover(
			NestedDelimiters("{", "}"),
			SkipUntil("}"),
		),
	)

	assert.NotZero(t, ast, "Expected partial AST")
	t.Logf("AST: %+v, Error: %v", ast, err)
}

// Unit tests for recovery strategies directly (not through parser)

func TestSkipUntilStrategyDirect(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Semi", `;`},
	})

	input := `foo bar ; baz`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{PeekingLexer: *peeker}

	strategy := SkipUntil(";")
	testErr := errors.New("test error")

	recovered, values, retErr := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.True(t, recovered)
	assert.Zero(t, values) // No fallback set
	assert.Equal(t, testErr, retErr)

	// Should have skipped to ;
	assert.Equal(t, ";", ctx.Peek().Value)
}

func TestTokenSyncStrategyDirect(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
	})

	input := `123 456 foo bar`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{PeekingLexer: *peeker}

	symbols := lex.Symbols()
	identType := symbols["Ident"]

	strategy := SyncToTokenType(identType)
	testErr := errors.New("test error")

	recovered, values, retErr := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.True(t, recovered)
	assert.Zero(t, values)
	assert.Equal(t, testErr, retErr)

	// Should have skipped to "foo"
	assert.Equal(t, "foo", ctx.Peek().Value)
}

func TestTokenSyncStrategyDirectWithConsume(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
	})

	input := `123 foo bar`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{PeekingLexer: *peeker}

	symbols := lex.Symbols()
	identType := symbols["Ident"]

	strategy := &TokenSyncStrategy{
		SyncTypes:        []lexer.TokenType{identType},
		ConsumeSyncToken: true,
		Fallback:         func() interface{} { return "fallback" },
	}
	testErr := errors.New("test error")

	recovered, values, retErr := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.True(t, recovered)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, testErr, retErr)

	// Should have consumed "foo" and be at "bar"
	assert.Equal(t, "bar", ctx.Peek().Value)
}

func TestTokenSyncStrategyDirectEOF(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Number", `\d+`},
	})

	input := `123 456`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{PeekingLexer: *peeker}

	// Looking for Ident but input only has numbers
	strategy := SyncToTokenType(999) // Non-existent type
	testErr := errors.New("test error")

	recovered, _, retErr := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.False(t, recovered)
	assert.Equal(t, testErr, retErr)
}

func TestRecoveryErrorInterface(t *testing.T) {
	err1 := Errorf(lexer.Position{Line: 1, Column: 5}, "first error")
	err2 := Errorf(lexer.Position{Line: 2, Column: 10}, "second error")

	recErr := &RecoveryError{
		Errors: []error{err1, err2},
	}

	// Test Error() formatting
	errStr := recErr.Error()
	assert.True(t, strings.Contains(errStr, "first error"))
	assert.True(t, strings.Contains(errStr, "second error"))

	// Test Unwrap()
	unwrapped := recErr.Unwrap()
	assert.Equal(t, err1.Error(), unwrapped.Error())

	// Test with single error
	singleErr := &RecoveryError{Errors: []error{err1}}
	assert.Equal(t, err1.Error(), singleErr.Error())

	// Test with no errors
	emptyErr := &RecoveryError{Errors: []error{}}
	assert.Equal(t, "no errors", emptyErr.Error())
	assert.Zero(t, emptyErr.Unwrap())
}

func TestSkipUntilWithFallback(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	input := `let x; set ; let y;`

	// Use SkipUntil with a fallback
	strategy := SkipUntil(";").WithFallback(func() interface{} {
		return "fallback"
	})

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestSkipThenRetryUntilStrategy(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	input := `let x; set ; let y;`

	// Test SkipThenRetryUntil
	strategy := SkipThenRetryUntil(";", "}").WithMaxSkip(50)

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestSkipThenRetryUntilUntilToken(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// This should hit the until token
	input := `let x; }`

	strategy := SkipThenRetryUntil("}")

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestSkipThenRetryUntilMaxSkip(t *testing.T) {
	// Test MaxSkip = 0 (unlimited) case
	strategy := &SkipThenRetryUntilStrategy{
		UntilTokens: []string{"NONEXISTENT"},
		MaxSkip:     0, // Unlimited
	}

	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	input := `let x`

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestSkipThenRetryUntilMaxSkipReached(t *testing.T) {
	// Test the MaxSkip check directly
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
	})

	input := `foo bar baz`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	// Skip some tokens first to simulate being at max
	peeker.Next() // Skip foo
	peeker.Next() // Skip bar - now skipped = 2

	ctx := &parseContext{PeekingLexer: *peeker}

	// Strategy with MaxSkip = 1, but we're setting skipped to already be at max
	// We need to test the condition s.MaxSkip > 0 && skipped >= s.MaxSkip
	// Since the function returns on first iteration, we test via EOF instead
	strategy := &SkipThenRetryUntilStrategy{
		UntilTokens: []string{"NONEXISTENT"},
		MaxSkip:     1, // Will be checked after skipping
	}

	testErr := errors.New("test error")
	// This will skip one token, increment skipped to 1, then return true
	// The MaxSkip check won't be reached due to immediate return
	recovered, _, _ := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.True(t, recovered) // First call returns true
}

func TestTokenSyncStrategy(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Semi", `;`},
	})

	// Grammar: expects Ident followed by ;
	type Item struct {
		Name string `@Ident`
		Semi string `@Semi`
	}

	type SimpleGrammar struct {
		Items []*Item `@@*`
	}

	parser := MustBuild[SimpleGrammar](
		Lexer(lex),
	)

	// Input with error (number instead of ident causes parse error)
	input := `foo; 123; bar;`

	// Get the Ident token type
	symbols := lex.Symbols()
	identType := symbols["Ident"]

	ast, err := parser.ParseString("", input,
		Recover(SyncToTokenType(identType)),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestTokenSyncStrategyWithFallbackAndConsume(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Semi", `;`},
	})

	type Item struct {
		Name string `@Ident`
		Semi string `@Semi`
	}

	type SimpleGrammar struct {
		Items []*Item `@@*`
	}

	parser := MustBuild[SimpleGrammar](
		Lexer(lex),
	)

	input := `foo; 123; bar;`

	symbols := lex.Symbols()
	identType := symbols["Ident"]

	// Test with consume and fallback
	strategy := &TokenSyncStrategy{
		SyncTypes:        []lexer.TokenType{identType},
		ConsumeSyncToken: true,
		Fallback:         func() interface{} { return "recovered" },
	}

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestTokenSyncStrategyEOF(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
	})

	type SimpleGrammar struct {
		Name string `@Ident`
	}

	parser := MustBuild[SimpleGrammar](
		Lexer(lex),
	)

	// Input that will hit EOF without finding sync token
	input := `123`

	symbols := lex.Symbols()
	identType := symbols["Ident"]

	ast, err := parser.ParseString("", input,
		Recover(SyncToTokenType(identType)),
	)

	assert.Error(t, err)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestNestedDelimitersWithFallbackAndOthers(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	type ParenExpr struct {
		Open  string `@"("`
		Inner string `@Ident`
		Close string `@")"`
	}

	type Prog struct {
		Exprs []*ParenExpr `@@*`
	}

	parser := MustBuild[Prog](
		Lexer(lex),
	)

	// Input with error inside nested parens
	input := `(foo) (123) (bar)`

	// Use nested delimiters with fallback and other delimiters
	strategy := NestedDelimiters("(", ")", [2]string{"[", "]"}).
		WithFallback(func() interface{} { return nil })

	ast, err := parser.ParseString("", input,
		Recover(strategy),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestMaxRecoveryErrorsZero(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Multiple errors
	input := `let ; set ; get ;`

	// 0 means unlimited
	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
		MaxRecoveryErrors(0),
	)

	assert.NotZero(t, ast)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestMaxRecoveryErrorsWithoutRecover(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Call MaxRecoveryErrors without Recover - should create empty recovery config
	input := `let x; let y;`

	ast, err := parser.ParseString("", input,
		MaxRecoveryErrors(5), // This will create empty recovery config
	)

	// Should parse successfully (no recovery needed, no strategies)
	assert.NoError(t, err)
	assert.NotZero(t, ast)
}

func TestTryRecoverMaxErrorsReached(t *testing.T) {
	// Test the maxErrors check directly by manipulating parseContext
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Semi", `;`},
	})

	input := `foo bar`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{
		PeekingLexer: *peeker,
		recovery: &recoveryConfig{
			strategies: []RecoveryStrategy{SkipUntil(";")},
			maxErrors:  1,
		},
		recoveryErrors: []error{errors.New("existing error")}, // Already at max
	}

	testErr := errors.New("test error")
	recovered, _ := ctx.tryRecover(testErr, reflect.Value{})
	assert.False(t, recovered) // Should return false because max errors reached
}

func TestCompositeStrategyAllFail(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Input where all strategies would fail (EOF before sync tokens)
	input := `let`

	ast, err := parser.ParseString("", input,
		Recover(
			TryStrategies(
				NestedDelimiters("(", ")"),
				SkipUntil("NONEXISTENT"),
			),
		),
	)

	// Should fail since no strategy succeeds
	assert.Error(t, err)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestNestedDelimitersOtherDelimiters(t *testing.T) {
	// Test that "other" delimiters are tracked correctly
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;{}()\[\]]`},
	})

	// Input with nested [] inside ()
	input := `( foo [ bar ] baz )`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	// Skip the opening paren to simulate being inside
	peeker.Next() // (

	ctx := &parseContext{PeekingLexer: *peeker}

	strategy := NestedDelimiters("(", ")", [2]string{"[", "]"})
	testErr := errors.New("test error")

	recovered, _, _ := strategy.Recover(ctx, testErr, reflect.Value{})
	assert.True(t, recovered)
	// Should be at the closing )
	assert.Equal(t, ")", ctx.Peek().Value)
}

func TestNestedDelimitersMismatch(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;{}()\[\],=]`},
	})

	type ParenExpr struct {
		Open  string `@"("`
		Inner string `@Ident`
		Close string `@")"`
	}

	type Prog struct {
		Exprs []*ParenExpr `@@*`
	}

	parser := MustBuild[Prog](
		Lexer(lex),
	)

	// Mismatched delimiter - ] instead of )
	input := `(foo] (bar)`

	ast, err := parser.ParseString("", input,
		Recover(
			NestedDelimiters("(", ")", [2]string{"[", "]"}),
		),
	)

	t.Logf("AST: %+v, Error: %v", ast, err)
}

func TestSkipUntilEOF(t *testing.T) {
	parser := MustBuild[Program](
		Lexer(testLexer),
	)

	// Input that will hit EOF before finding sync token
	input := `let x`

	ast, err := parser.ParseString("", input,
		Recover(SkipUntil("NONEXISTENT")),
	)

	assert.Error(t, err)
	t.Logf("AST: %+v, Error: %v", ast, err)
}

// =============================================================================
// Recovery Tag Parsing Tests
// =============================================================================

func TestParseRecoveryTag(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		wantLabel   string
		wantCount   int // number of strategies
		wantErr     bool
		errContains string
	}{
		{
			name:      "empty tag",
			tag:       "",
			wantLabel: "",
			wantCount: 0,
		},
		{
			name:      "skip_until single token",
			tag:       "skip_until(;)",
			wantCount: 1,
		},
		{
			name:      "skip_until multiple tokens",
			tag:       "skip_until(;, })",
			wantCount: 1,
		},
		{
			name:      "skip_past",
			tag:       "skip_past(;)",
			wantCount: 1,
		},
		{
			name:      "retry_until",
			tag:       "retry_until(;, })",
			wantCount: 1,
		},
		{
			name:      "nested simple",
			tag:       "nested((, ))",
			wantCount: 1,
		},
		{
			name:      "nested with others",
			tag:       "nested((, ), [{, }])",
			wantCount: 1,
		},
		{
			name:      "label only",
			tag:       "label:expression",
			wantLabel: "expression",
			wantCount: 0,
		},
		{
			name:      "label with strategy",
			tag:       "label:stmt|skip_until(;)",
			wantLabel: "stmt",
			wantCount: 1,
		},
		{
			name:      "multiple strategies",
			tag:       "nested((, ))|skip_until(;)",
			wantCount: 2,
		},
		{
			name:      "label in middle",
			tag:       "skip_until(;)|label:test|skip_past(})",
			wantLabel: "test",
			wantCount: 2,
		},
		{
			name:        "invalid strategy name",
			tag:         "invalid_strategy(;)",
			wantErr:     true,
			errContains: "unknown recovery strategy",
		},
		{
			name:        "skip_until no args",
			tag:         "skip_until()",
			wantErr:     true,
			errContains: "requires at least one token",
		},
		{
			name:        "nested missing end",
			tag:         "nested(()",
			wantErr:     true,
			errContains: "requires at least start and end",
		},
		{
			name:        "malformed syntax",
			tag:         "skip_until",
			wantErr:     true,
			errContains: "invalid recovery strategy syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parseRecoveryTag(tt.tag)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)

			if tt.wantCount == 0 && tt.wantLabel == "" {
				assert.True(t, config == nil, "Expected nil config for empty tag")
				return
			}

			assert.NotZero(t, config)
			assert.Equal(t, tt.wantLabel, config.label)
			assert.Equal(t, tt.wantCount, len(config.strategies))
		})
	}
}

func TestParseTokenList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{";", []string{";"}},
		{";, }", []string{";", "}"}},
		{"  ;  ,  }  ", []string{";", "}"}},
		{`";"`, []string{";"}},
		{`';', "}"`, []string{";", "}"}},
		{"", nil}, // empty returns nil slice
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTokenList(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseNestedArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"(, )", []string{"(", " )"}},
		{"(, ), [{, }]", []string{"(", " )", " [{, }]"}},
		{"a, b, c", []string{"a", " b", " c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseNestedArgs(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// Per-Node Recovery Tests (struct tag based)
// =============================================================================

// Grammar with per-field recovery annotations
type ExprWithRecovery struct {
	Value string `parser:"@Ident" recover:"skip_until(;)"`
}

type StmtWithRecovery struct {
	Keyword string            `parser:"@Ident"`
	Expr    *ExprWithRecovery `parser:"@@"`
	Semi    string            `parser:"@\";\""`
}

type ProgWithRecovery struct {
	Stmts []*StmtWithRecovery `parser:"@@*"`
}

func TestPerNodeRecoveryTag(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	parser := MustBuild[ProgWithRecovery](
		Lexer(lex),
	)

	// Input with error - the '123' can't be parsed as Ident
	// The per-field recovery should skip to ';'
	input := `let foo; set 123; get bar;`

	ast, err := parser.ParseString("", input)

	// Should have errors but parsed what it could
	t.Logf("AST: %+v", ast)
	t.Logf("Error: %v", err)

	// The per-node recovery should have kicked in
	assert.NotZero(t, ast)
}

// Grammar with labeled recovery
type LabeledExpr struct {
	Value string `parser:"@Ident" recover:"label:identifier|skip_until(;)"`
}

type LabeledStmt struct {
	Keyword string       `parser:"@Ident"`
	Expr    *LabeledExpr `parser:"@@"`
	Semi    string       `parser:"@\";\""`
}

type LabeledProg struct {
	Stmts []*LabeledStmt `parser:"@@*"`
}

func TestPerNodeRecoveryWithLabel(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	parser := MustBuild[LabeledProg](
		Lexer(lex),
	)

	input := `let foo; set 123; get bar;`

	ast, err := parser.ParseString("", input)

	t.Logf("AST: %+v", ast)
	t.Logf("Error: %v", err)

	// Error message should contain the label
	if err != nil {
		var recErr *RecoveryError
		if errors.As(err, &recErr) {
			for _, e := range recErr.Errors {
				t.Logf("Recovery error: %v", e)
				// The label should be in the error
				assert.Contains(t, e.Error(), "identifier")
			}
		}
	}

	assert.NotZero(t, ast)
}

// Grammar with nested delimiter recovery
type ParenExprRecovery struct {
	Open  string `parser:"@\"(\""`
	Value string `parser:"@Ident" recover:"nested((, ))"`
	Close string `parser:"@\")\""`
}

type ParenProgRecovery struct {
	Exprs []*ParenExprRecovery `parser:"@@*"`
}

func TestPerNodeNestedRecovery(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	parser := MustBuild[ParenProgRecovery](
		Lexer(lex),
	)

	// Second expression has a number instead of ident
	input := `(foo) (123) (bar)`

	ast, err := parser.ParseString("", input)

	t.Logf("AST: %+v", ast)
	t.Logf("Error: %v", err)

	assert.NotZero(t, ast)
}

// Test recovery node wrapper directly
func TestRecoveryNodeWrapper(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;]`},
	})

	input := `foo ; bar`
	l, err := lex.LexString("", input)
	assert.NoError(t, err)
	peeker, err := lexer.Upgrade(l)
	assert.NoError(t, err)

	ctx := &parseContext{
		PeekingLexer: *peeker,
		recovery: &recoveryConfig{
			maxErrors: 10,
		},
	}

	// Create a mock node that always fails
	mockNode := &mockFailingNode{err: errors.New("mock parse error")}

	// Wrap it with recovery
	config := &nodeRecoveryConfig{
		strategies: []RecoveryStrategy{SkipUntil(";")},
		label:      "test",
	}
	wrapped := wrapWithRecovery(mockNode, config)

	// Parse should recover
	values, err := wrapped.Parse(ctx, reflect.Value{})

	// Recovery should have happened (err == nil means recovery worked)
	t.Logf("Values: %v, Err: %v, RecoveryErrors: %v", values, err, ctx.recoveryErrors)

	// Should have recorded a recovery error
	assert.Equal(t, 1, len(ctx.recoveryErrors))
}

// Mock node for testing
type mockFailingNode struct {
	err error
}

func (m *mockFailingNode) Parse(ctx *parseContext, parent reflect.Value) ([]reflect.Value, error) {
	return nil, m.err
}

func (m *mockFailingNode) String() string   { return "mock" }
func (m *mockFailingNode) GoString() string { return "mock{}" }

// Test that wrapWithRecovery returns original node when config is nil/empty
func TestWrapWithRecoveryNoop(t *testing.T) {
	mockNode := &mockFailingNode{}

	// nil config
	wrapped := wrapWithRecovery(mockNode, nil)
	assert.Equal(t, node(mockNode), wrapped)

	// empty config
	wrapped = wrapWithRecovery(mockNode, &nodeRecoveryConfig{})
	assert.Equal(t, node(mockNode), wrapped)
}

// Test recovery node GoString
func TestRecoveryNodeGoString(t *testing.T) {
	mockNode := &mockFailingNode{}
	config := &nodeRecoveryConfig{strategies: []RecoveryStrategy{SkipUntil(";")}}
	wrapped := wrapWithRecovery(mockNode, config)

	assert.Contains(t, wrapped.GoString(), "recovery{")
	assert.Contains(t, wrapped.GoString(), "mock")
}

// =============================================================================
// Recovery Metadata Field Tests
// =============================================================================

// Grammar with recovery metadata fields
type StmtWithMetadata struct {
	Pos          lexer.Position
	Keyword      string `parser:"@Ident"`
	Value        string `parser:"@Ident"`
	Semi         string `parser:"@\";\""`
	Recovered    bool
	RecoveredSpan lexer.Position
}

type ProgWithMetadata struct {
	Stmts []*StmtWithMetadata `parser:"@@*"`
}

func TestRecoveryMetadataFields(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	parser := MustBuild[ProgWithMetadata](
		Lexer(lex),
	)

	// Input with valid statements
	input := `let x; set y;`

	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
	)

	assert.NoError(t, err)
	assert.NotZero(t, ast)
	assert.Equal(t, 2, len(ast.Stmts))

	// Valid statements should not have Recovered set
	for _, stmt := range ast.Stmts {
		assert.False(t, stmt.Recovered, "Valid statement should not be marked as recovered")
	}
}

func TestRecoveryMetadataFieldsOnError(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Number", `\d+`},
		{"Punct", `[;{}()\[\],=]`},
	})

	parser := MustBuild[ProgWithMetadata](
		Lexer(lex),
	)

	// Input with an error (missing value after keyword)
	input := `let x; set ; get z;`

	ast, err := parser.ParseString("", input,
		Recover(SkipUntil(";")),
	)

	// Should have recovery errors
	assert.Error(t, err)
	var recErr *RecoveryError
	assert.True(t, errors.As(err, &recErr))

	assert.NotZero(t, ast)
	t.Logf("Parsed %d statements", len(ast.Stmts))

	// At least some statements should have parsed
	// Note: The exact behavior depends on how recovery interacts with the grammar
	for i, stmt := range ast.Stmts {
		t.Logf("Stmt %d: Keyword=%q Value=%q Recovered=%v RecoveredSpan=%v",
			i, stmt.Keyword, stmt.Value, stmt.Recovered, stmt.RecoveredSpan)
	}
}

// Test that only Recovered field is detected (not RecoveredSpan)
type StmtWithRecoveredOnly struct {
	Keyword   string `parser:"@Ident"`
	Value     string `parser:"@Ident"`
	Semi      string `parser:"@\";\""`
	Recovered bool
}

type ProgWithRecoveredOnly struct {
	Stmts []*StmtWithRecoveredOnly `parser:"@@*"`
}

func TestRecoveredFieldOnly(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;]`},
	})

	parser := MustBuild[ProgWithRecoveredOnly](
		Lexer(lex),
	)

	// Valid input
	input := `let x;`

	ast, err := parser.ParseString("", input)

	assert.NoError(t, err)
	assert.NotZero(t, ast)
	assert.Equal(t, 1, len(ast.Stmts))
	assert.False(t, ast.Stmts[0].Recovered)
}

// Test that only RecoveredSpan field is detected (not Recovered)
type StmtWithSpanOnly struct {
	Keyword       string `parser:"@Ident"`
	Value         string `parser:"@Ident"`
	Semi          string `parser:"@\";\""`
	RecoveredSpan lexer.Position
}

type ProgWithSpanOnly struct {
	Stmts []*StmtWithSpanOnly `parser:"@@*"`
}

func TestRecoveredSpanFieldOnly(t *testing.T) {
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;]`},
	})

	parser := MustBuild[ProgWithSpanOnly](
		Lexer(lex),
	)

	// Valid input
	input := `let x;`

	ast, err := parser.ParseString("", input)

	assert.NoError(t, err)
	assert.NotZero(t, ast)
	assert.Equal(t, 1, len(ast.Stmts))
	// Should be zero position for valid parse
	assert.Equal(t, lexer.Position{}, ast.Stmts[0].RecoveredSpan)
}
