package participle

import (
	"reflect"

	"github.com/alecthomas/participle/v2/lexer"
)

// RecoveryStrategy defines a strategy for recovering from parse errors.
//
// Error recovery allows the parser to continue parsing after encountering an error,
// collecting multiple errors and producing a partial AST. This is inspired by
// Chumsky's recovery system in Rust and classic compiler panic-mode recovery.
//
// There is no silver bullet strategy for error recovery. By definition, if the input
// to a parser is invalid then the parser can only make educated guesses as to the
// meaning of the input. Different recovery strategies will work better for different
// languages, and for different patterns within those languages.
type RecoveryStrategy interface {
	// Recover attempts to recover from a parse error.
	//
	// Parameters:
	//   - ctx: The parse context (positioned after the failed parse attempt)
	//   - err: The error that triggered recovery
	//   - parent: The parent value being parsed into
	//
	// Returns:
	//   - recovered: true if recovery was successful
	//   - values: any values recovered (may be nil/fallback for skip strategies)
	//   - newErr: the error to report (may be modified/wrapped)
	Recover(ctx *parseContext, err error, parent reflect.Value) (recovered bool, values []reflect.Value, newErr error)
}

// recoveryConfig holds recovery configuration for a parse context.
type recoveryConfig struct {
	strategies []RecoveryStrategy
	errors     []error
	maxErrors  int
}

// RecoveryError wraps multiple errors that occurred during parsing with recovery.
type RecoveryError struct {
	Errors []error
}

func (r *RecoveryError) Error() string {
	if len(r.Errors) == 0 {
		return "no errors"
	}
	if len(r.Errors) == 1 {
		return r.Errors[0].Error()
	}
	msg := r.Errors[0].Error()
	for i := 1; i < len(r.Errors); i++ {
		msg += "\n" + r.Errors[i].Error()
	}
	return msg
}

// Unwrap returns the first error for compatibility with errors.Is/As.
func (r *RecoveryError) Unwrap() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return r.Errors[0]
}

// SkipUntilStrategy skips tokens until one of the synchronization tokens is found.
//
// This is the classic "panic mode" recovery strategy from compiler theory.
// It's simple but effective for languages with clear statement terminators
// (like semicolons) or block delimiters.
//
// Example usage:
//
//	parser.ParseString("", input, participle.Recover(SkipUntil(";", "}", ")")))
type SkipUntilStrategy struct {
	// Tokens to synchronize on (the parser will stop before these tokens)
	SyncTokens []string
	// If true, consume the sync token; if false, leave it for the next parse
	ConsumeSyncToken bool
	// Fallback returns a fallback value when recovery succeeds.
	// If nil, an empty/zero value is used.
	Fallback func() interface{}
}

// SkipUntil creates a recovery strategy that skips tokens until a sync token is found.
//
// The sync tokens are typically statement terminators (";"), block delimiters ("}", ")"),
// or keywords that start new constructs ("if", "while", "func", etc.).
func SkipUntil(tokens ...string) *SkipUntilStrategy {
	return &SkipUntilStrategy{
		SyncTokens:       tokens,
		ConsumeSyncToken: false,
	}
}

// SkipPast creates a recovery strategy that skips tokens until a sync token is found,
// then consumes the sync token.
func SkipPast(tokens ...string) *SkipUntilStrategy {
	return &SkipUntilStrategy{
		SyncTokens:       tokens,
		ConsumeSyncToken: true,
	}
}

// WithFallback sets a fallback value generator for the skip strategy.
func (s *SkipUntilStrategy) WithFallback(f func() interface{}) *SkipUntilStrategy {
	s.Fallback = f
	return s
}

func (s *SkipUntilStrategy) Recover(ctx *parseContext, err error, parent reflect.Value) (bool, []reflect.Value, error) {
	syncSet := make(map[string]bool)
	for _, t := range s.SyncTokens {
		syncSet[t] = true
	}

	// Skip tokens until we find a sync token or EOF
	for {
		token := ctx.Peek()
		if token.EOF() {
			return false, nil, err
		}
		if syncSet[token.Value] {
			if s.ConsumeSyncToken {
				ctx.Next()
			}
			// Recovery successful
			var values []reflect.Value
			if s.Fallback != nil {
				values = []reflect.Value{reflect.ValueOf(s.Fallback())}
			}
			return true, values, err
		}
		ctx.Next()
	}
}

// SkipThenRetryUntilStrategy skips tokens and retries parsing until successful
// or a termination condition is met.
//
// This is more sophisticated than SkipUntil - it repeatedly:
// 1. Skips one token
// 2. Tries to parse again
// 3. If parsing succeeds without new errors, returns success
// 4. If parsing fails, repeats from step 1
//
// This continues until a termination token is found or EOF is reached.
type SkipThenRetryUntilStrategy struct {
	// Tokens that terminate the recovery attempt (stop trying)
	UntilTokens []string
	// Maximum tokens to skip before giving up (0 = unlimited)
	MaxSkip int
}

// SkipThenRetryUntil creates a strategy that skips tokens and retries parsing.
func SkipThenRetryUntil(untilTokens ...string) *SkipThenRetryUntilStrategy {
	return &SkipThenRetryUntilStrategy{
		UntilTokens: untilTokens,
		MaxSkip:     100, // Reasonable default to prevent infinite loops
	}
}

// WithMaxSkip sets the maximum number of tokens to skip.
func (s *SkipThenRetryUntilStrategy) WithMaxSkip(max int) *SkipThenRetryUntilStrategy {
	s.MaxSkip = max
	return s
}

func (s *SkipThenRetryUntilStrategy) Recover(ctx *parseContext, err error, parent reflect.Value) (bool, []reflect.Value, error) {
	untilSet := make(map[string]bool)
	for _, t := range s.UntilTokens {
		untilSet[t] = true
	}

	// Check if we're at a terminating token or EOF
	token := ctx.Peek()
	if token.EOF() || untilSet[token.Value] {
		return false, nil, err
	}

	// Skip one token and signal that the caller should retry parsing.
	// The caller (parseContext) will call this strategy again if parsing
	// fails again, allowing incremental recovery.
	ctx.Next()
	return true, nil, err
}

// NestedDelimitersStrategy recovers by finding balanced delimiters.
//
// This is particularly useful for recovering from errors inside parenthesized
// expressions, function arguments, array indices, etc. It respects nesting,
// so it will correctly handle nested brackets.
//
// Example: If parsing `foo(bar(1, 2, err!@#), baz)` fails on `err!@#`,
// this strategy can skip to the closing `)` of `bar(...)` while respecting
// the nested parentheses.
type NestedDelimitersStrategy struct {
	// Start delimiter (e.g., "(", "[", "{")
	Start string
	// End delimiter (e.g., ")", "]", "}")
	End string
	// Additional delimiter pairs to respect for nesting
	Others [][2]string
	// Fallback returns a fallback value when recovery succeeds.
	Fallback func() interface{}
}

// NestedDelimiters creates a strategy that skips to balanced delimiters.
func NestedDelimiters(start, end string, others ...[2]string) *NestedDelimitersStrategy {
	return &NestedDelimitersStrategy{
		Start:  start,
		End:    end,
		Others: others,
	}
}

// WithFallback sets a fallback value generator for the nested delimiters strategy.
func (n *NestedDelimitersStrategy) WithFallback(f func() interface{}) *NestedDelimitersStrategy {
	n.Fallback = f
	return n
}

func (n *NestedDelimitersStrategy) Recover(ctx *parseContext, err error, parent reflect.Value) (bool, []reflect.Value, error) {
	// Build delimiter maps
	openers := map[string]string{n.Start: n.End}
	closers := map[string]bool{n.End: true}
	for _, pair := range n.Others {
		openers[pair[0]] = pair[1]
		closers[pair[1]] = true
	}

	// Track nesting depth for each delimiter type
	depths := make(map[string]int)

	// We start inside the delimited region, so we're looking for the closing delimiter
	// at depth 0 (or the matching closer for our opener)
	targetClose := n.End
	depth := 1 // We're inside one level of our target delimiters

	for {
		token := ctx.Peek()
		if token.EOF() {
			return false, nil, err
		}

		// Check if this opens a nested delimiter
		if closer, isOpener := openers[token.Value]; isOpener {
			if token.Value == n.Start {
				depth++
			} else {
				depths[closer]++
			}
		}

		// Check if this closes a delimiter
		if closers[token.Value] {
			if token.Value == targetClose {
				depth--
				if depth == 0 {
					// Found our balanced closer - don't consume it
					var values []reflect.Value
					if n.Fallback != nil {
						values = []reflect.Value{reflect.ValueOf(n.Fallback())}
					}
					return true, values, err
				}
			} else if depths[token.Value] > 0 {
				depths[token.Value]--
			} else {
				// Mismatched closer - this is an error, but we can try to continue
				// by treating it as the end of our recovery region
				return false, nil, err
			}
		}

		ctx.Next()
	}
}

// TokenSyncStrategy synchronizes on specific token types rather than values.
//
// This is useful when you want to recover to any identifier, any string literal,
// or other token categories defined by your lexer.
type TokenSyncStrategy struct {
	// Token types to synchronize on (use lexer symbol names)
	SyncTypes []lexer.TokenType
	// If true, consume the sync token
	ConsumeSyncToken bool
	// Fallback value generator
	Fallback func() interface{}
}

// SyncToTokenType creates a strategy that syncs on token types.
func SyncToTokenType(types ...lexer.TokenType) *TokenSyncStrategy {
	return &TokenSyncStrategy{
		SyncTypes:        types,
		ConsumeSyncToken: false,
	}
}

func (t *TokenSyncStrategy) Recover(ctx *parseContext, err error, parent reflect.Value) (bool, []reflect.Value, error) {
	syncSet := make(map[lexer.TokenType]bool)
	for _, tt := range t.SyncTypes {
		syncSet[tt] = true
	}

	for {
		token := ctx.Peek()
		if token.EOF() {
			return false, nil, err
		}
		if syncSet[token.Type] {
			if t.ConsumeSyncToken {
				ctx.Next()
			}
			var values []reflect.Value
			if t.Fallback != nil {
				values = []reflect.Value{reflect.ValueOf(t.Fallback())}
			}
			return true, values, err
		}
		ctx.Next()
	}
}

// CompositeStrategy tries multiple strategies in order until one succeeds.
type CompositeStrategy struct {
	Strategies []RecoveryStrategy
}

// TryStrategies creates a composite strategy that tries each strategy in order.
func TryStrategies(strategies ...RecoveryStrategy) *CompositeStrategy {
	return &CompositeStrategy{Strategies: strategies}
}

func (c *CompositeStrategy) Recover(ctx *parseContext, err error, parent reflect.Value) (bool, []reflect.Value, error) {
	checkpoint := ctx.saveCheckpoint()

	for _, strategy := range c.Strategies {
		recovered, values, newErr := strategy.Recover(ctx, err, parent)
		if recovered {
			return true, values, newErr
		}
		// Reset cursor for next strategy attempt
		ctx.restoreCheckpoint(checkpoint)
	}
	return false, nil, err
}

// Helper functions for checkpoint-based recovery

// saveCheckpoint saves the current lexer position for potential restoration.
func (p *parseContext) saveCheckpoint() lexer.Checkpoint {
	return p.PeekingLexer.MakeCheckpoint()
}

// restoreCheckpoint restores the lexer to a previously saved position.
func (p *parseContext) restoreCheckpoint(cp lexer.Checkpoint) {
	p.PeekingLexer.LoadCheckpoint(cp)
}
