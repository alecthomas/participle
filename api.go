package participle

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Capture can be implemented by fields in order to transform captured tokens into field values.
type Capture interface {
	Capture(values []string) error
}

// The Parseable interface can be implemented by any element in the grammar to provide custom parsing.
type Parseable interface {
	// Parse into the receiver.
	//
	// Should return NextMatch if no tokens matched and parsing should continue.
	// Nil should be returned if parsing was successful.
	Parse(lex *lexer.PeekingLexer) error
}

// The Fuzzable interface can be implemented by any element in the grammar to provide custom fuzzing.
type Fuzzable interface {
	// Generate a valid string that can be parsed to get a value from
	// the corresponding Node.
	Fuzz(l lexer.Fuzzer) string
}
