package participle

import (
	"testing"
	"text/scanner"

	"github.com/alecthomas/assert"
)

func TestLexer(t *testing.T) {
	lexer := LexString("hello world")
	helloPos := Position{Offset: 0, Line: 1, Column: 1}
	worldPos := Position{Offset: 5, Line: 1, Column: 6}
	eofPos := Position{Offset: 11, Line: 1, Column: 12}
	assert.Equal(t, Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lexer.Peek())
	assert.Equal(t, Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lexer.Peek())
	assert.Equal(t, Token{Type: scanner.Ident, Value: "hello", Pos: helloPos}, lexer.Next())
	assert.Equal(t, Token{Type: scanner.Ident, Value: "world", Pos: worldPos}, lexer.Peek())
	assert.Equal(t, Token{Type: scanner.Ident, Value: "world", Pos: worldPos}, lexer.Next())
	assert.Equal(t, Token{Type: scanner.EOF, Value: "", Pos: eofPos}, lexer.Peek())
	assert.Equal(t, Token{Type: scanner.EOF, Value: "", Pos: eofPos}, lexer.Next())
}

func TestLexString(t *testing.T) {
	lexer := LexString(`"hello\nworld"`)
	assert.Equal(t, lexer.Next(), Token{Type: scanner.String, Value: "hello\nworld", Pos: Position{Line: 1, Column: 1}})
}

func TestLexSingleString(t *testing.T) {
	lexer := LexString(`'hello\nworld'`)
	assert.Equal(t, lexer.Next(), Token{Type: scanner.String, Value: "hello\nworld", Pos: Position{Line: 1, Column: 1}})
	lexer = LexString(`'\U00008a9e'`)
	assert.Equal(t, lexer.Next(), Token{Type: scanner.Char, Value: "\U00008a9e", Pos: Position{Line: 1, Column: 1}})
}

func TestLexBacktickString(t *testing.T) {
	lexer := LexString("`hello\\nworld`")
	assert.Equal(t, lexer.Next(), Token{Type: scanner.String, Value: "hello\\nworld", Pos: Position{Line: 1, Column: 1}})
}
