package parser

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"text/scanner"
)

const (
	EOF rune = -(iota + 1)
)

var (
	// EOFToken is a Token representing EOF.
	EOFToken = Token{EOF, "<<EOF>>"}

	// DefaultLexerDefinition defines properties for the default lexer.
	DefaultLexerDefinition LexerDefinition = &defaultLexerDefinition{}
)

// Position of a token.
type Position scanner.Position

// A Token returned by a Lexer.
type Token struct {
	// Type of token.
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

type LexerDefinition interface {
	Lex(io.Reader) Lexer
	Symbols() map[string]rune
}

type defaultLexerDefinition struct{}

func (d *defaultLexerDefinition) Lex(r io.Reader) Lexer {
	return Lex(r)
}

func (d *defaultLexerDefinition) Symbols() map[string]rune {
	return map[string]rune{
		"EOF":       scanner.EOF,
		"Ident":     scanner.Ident,
		"Int":       scanner.Int,
		"Float":     scanner.Float,
		"Char":      scanner.Char,
		"String":    scanner.String,
		"RawString": scanner.RawString,
		"Comment":   scanner.Comment,
	}
}

// A Lexer returns tokens from a source.
type Lexer interface {
	// Peek at the next token.
	Peek() Token
	// Next consumes and returns the next token.
	Next() Token
	// Position returns the current cursor position in the input.
	Position() Position
}

// textScannerLexer is a Lexer based on text/scanner.Scanner
type textScannerLexer struct {
	scanner scanner.Scanner
	peek    *Token
	pos     scanner.Position
}

// Lex an io.Reader with text/scanner.Scanner.
//
// Note that this differs from text/scanner.Scanner in that string tokens will be unquoted.
func Lex(r io.Reader) Lexer {
	lexer := &textScannerLexer{
		pos: scanner.Position{Column: 1, Line: 1},
	}
	lexer.scanner.Error = func(s *scanner.Scanner, msg string) {
		panic(msg)
	}
	lexer.scanner.Init(r)
	return lexer
}

// LexString returns a new default lexer over bytes.
func LexBytes(b []byte) Lexer {
	return Lex(bytes.NewReader(b))
}

// LexString returns a new default lexer over a string.
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
		t.pos = t.scanner.Pos()
		t.peek = &Token{
			Type:  t.scanner.Scan(),
			Value: t.scanner.TokenText(),
		}
		// Unquote strings.
		if t.peek.Type == scanner.String || t.peek.Type == scanner.RawString {
			s, err := strconv.Unquote(t.peek.Value)
			if err != nil {
				panic(err.Error())
			}
			t.peek.Value = s
		}
	}
	return *t.peek
}

func (t *textScannerLexer) Position() Position {
	return Position(t.pos)
}
