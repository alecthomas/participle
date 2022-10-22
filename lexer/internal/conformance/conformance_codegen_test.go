//go:build generated

package conformance_test

import (
	"testing"

	"github.com/alecthomas/participle/v2/lexer/internal/conformance"
)

// This should only be run by TestLexerConformanceGenerated.
func TestLexerConformanceGeneratedInternal(t *testing.T) {
	testLexer(t, conformance.GeneratedConformanceLexer)
}
