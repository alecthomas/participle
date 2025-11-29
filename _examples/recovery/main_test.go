package main

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/alecthomas/participle/v2"
)

func TestRecoveryExample(t *testing.T) {
	// Valid input
	t.Run("ValidInput", func(t *testing.T) {
		input := `let x = 42; let y = 100;`
		ast, err := parser.ParseString("test", input,
			participle.Recover(participle.SkipPast(";")),
		)
		assert.NoError(t, err)
		assert.NotZero(t, ast)
		assert.Equal(t, 2, len(ast.Statements))
	})

	// Input with error - recovery enabled
	t.Run("ErrorWithRecovery", func(t *testing.T) {
		input := `let x = 42; let y = ; let z = 100;`
		ast, err := parser.ParseString("test", input,
			participle.Recover(participle.SkipPast(";")),
		)
		// Should have errors but also parsed everything
		var recErr *participle.RecoveryError
		assert.True(t, errors.As(err, &recErr))
		assert.Equal(t, 1, len(recErr.Errors))
		assert.NotZero(t, ast)
		assert.Equal(t, 3, len(ast.Statements))
	})

	// Input with error - recovery disabled
	t.Run("ErrorWithoutRecovery", func(t *testing.T) {
		input := `let x = 42; let y = ; let z = 100;`
		ast, err := parser.ParseString("test", input)
		// Should error and not be a RecoveryError
		assert.Error(t, err)
		var recErr *participle.RecoveryError
		assert.False(t, errors.As(err, &recErr))
		// Partial AST is still returned
		assert.NotZero(t, ast)
	})
}
