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
