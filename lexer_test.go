package parser

import (
	"testing"
	"text/scanner"

	"github.com/alecthomas/assert"
)

func TestLexer(t *testing.T) {
	lexer := LexString("hello world")
	assert.Equal(t, lexer.Peek(), Token{scanner.Ident, "hello"})
	assert.Equal(t, lexer.Peek(), Token{scanner.Ident, "hello"})
	assert.Equal(t, lexer.Next(), Token{scanner.Ident, "hello"})
	assert.Equal(t, lexer.Peek(), Token{scanner.Ident, "world"})
	assert.Equal(t, lexer.Next(), Token{scanner.Ident, "world"})
	assert.Equal(t, lexer.Peek(), Token{scanner.EOF, ""})
	assert.Equal(t, lexer.Next(), Token{scanner.EOF, ""})
}
