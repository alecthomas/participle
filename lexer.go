package parser

import (
	"bytes"
	"io"
	"strings"
	"text/scanner"
)

const (
	EOF rune = -(iota + 1)
)

// EOFToken is a Token representing EOF.
var EOFToken = Token{EOF, ""}

type Position scanner.Position

type Token struct {
	Type  rune
	Value string
}

// RuneToken represents a rune as a Token.
func RuneToken(r rune) Token {
	return Token{r, string(r)}
}

func (t Token) EOF() bool {
	return t.Type == EOF
}

func (t Token) String() string {
	return t.Value
}

// A Lexer returns tokens from a source.
type Lexer interface {
	// Peek at the next token.
	Peek() Token
	// Next consumes and returns the next token.
	Next() Token
	// Position returns the current cursor position in the input.
	Position() Position

	// Symbols is the table of token types (runes) supported by this Lexer, mapped to their symbol
	// names. It is used by the parser generator to support recognition of tokens, eg. @Ident
	Symbols() map[rune]string
}

// textScannerLexer is a Lexer based on text/scanner.Scanner
type textScannerLexer struct {
	scanner scanner.Scanner
	peek    *Token
}

// Lex an io.Reader with text/scanner.Scanner.
func Lex(r io.Reader) Lexer {
	lexer := &textScannerLexer{}
	lexer.scanner.Error = func(s *scanner.Scanner, msg string) {
		panic(msg)
	}
	lexer.scanner.Init(r)
	return lexer
}

func LexBytes(b []byte) Lexer {
	return Lex(bytes.NewReader(b))
}

func LexString(s string) Lexer {
	return Lex(strings.NewReader(s))
}

func (t *textScannerLexer) Next() Token {
	if t.peek == nil {
		t.Peek()
	}
	token := t.peek
	t.peek = nil
	return *token
}

func (t *textScannerLexer) Peek() Token {
	if t.peek == nil {
		t.peek = &Token{
			Type:  t.scanner.Scan(),
			Value: t.scanner.TokenText(),
		}
	}
	return *t.peek
}

func (t *textScannerLexer) Position() Position {
	return Position(t.scanner.Pos())
}

func (t *textScannerLexer) Symbols() map[rune]string {
	return map[rune]string{
		scanner.EOF:       "EOF",
		scanner.Ident:     "Ident",
		scanner.Int:       "Int",
		scanner.Float:     "Float",
		scanner.Char:      "Char",
		scanner.String:    "String",
		scanner.RawString: "RawString",
		scanner.Comment:   "Comment",
	}
}
