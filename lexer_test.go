package participle

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

func TestLexString(t *testing.T) {
	lexer := LexString(`"hello\nworld"`)
	assert.Equal(t, lexer.Next(), Token{scanner.String, "hello\nworld"})
}

func TestLexSingleString(t *testing.T) {
	lexer := LexString(`'hello\nworld'`)
	assert.Equal(t, lexer.Next(), Token{scanner.String, "hello\nworld"})
	lexer = LexString(`'\U00008a9e'`)
	assert.Equal(t, lexer.Next(), Token{scanner.Char, "\U00008a9e"})
}

func TestLexBacktickString(t *testing.T) {
	lexer := LexString("`hello\\nworld`")
	assert.Equal(t, lexer.Next(), Token{scanner.String, "hello\\nworld"})
}
