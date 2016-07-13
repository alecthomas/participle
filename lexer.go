package participle

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"
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

// LexerDefinition provides the parser with metadata for a lexer.
type LexerDefinition interface {
	// Lex an io.Reader.
	Lex(io.Reader) Lexer
	// Symbols returns a map of symbolic names to the corresponding pseudo-runes for those symbols.
	// This is the same approach as used by text/scanner. For example, "EOF" might have the rune
	// value of -1, "Ident" might be -2, and so on.
	Symbols() map[string]rune
}

// A Lexer returns tokens from a source.
type Lexer interface {
	// Peek at the next token.
	Peek() Token
	// Next consumes and returns the next token.
	Next() Token
	// Position returns the position of the last token consumed.
	Position() Position
}

type defaultLexerDefinition struct{}

func (d *defaultLexerDefinition) Lex(r io.Reader) Lexer {
	return Lex(r)
}

func (d *defaultLexerDefinition) Symbols() map[string]rune {
	return map[string]rune{
		"EOF":       scanner.EOF,
		"Char":      scanner.Char,
		"Ident":     scanner.Ident,
		"Int":       scanner.Int,
		"Float":     scanner.Float,
		"String":    scanner.String,
		"RawString": scanner.RawString,
		"Comment":   scanner.Comment,
	}
}

// textScannerLexer is a Lexer based on text/scanner.Scanner
type textScannerLexer struct {
	scanner  scanner.Scanner
	peek     *Token
	pos      scanner.Position
	filename string
}

type namedReader interface {
	Name() string
}

// Lex an io.Reader with text/scanner.Scanner.
//
// Note that this differs from text/scanner.Scanner in that string tokens will be unquoted.
func Lex(r io.Reader) Lexer {
	lexer := &textScannerLexer{
		pos:      scanner.Position{Column: 1, Line: 1},
		filename: "<source>",
	}
	if n, ok := r.(namedReader); ok {
		lexer.filename = n.Name()
	}
	lexer.scanner.Init(r)
	lexer.scanner.Error = func(s *scanner.Scanner, msg string) {
		// This is to support single quoted strings. Hacky.
		if msg != "illegal char literal" {
			panic(msg)
		}
	}
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
	if t.peek != nil {
		return *t.peek
	}
	t.pos = t.scanner.Pos()
	t.peek = &Token{
		Type:  t.scanner.Scan(),
		Value: t.scanner.TokenText(),
	}
	// Unquote strings.
	switch t.peek.Type {
	case scanner.Char:
		// FIXME(alec): This is pretty hacky...we convert a single quoted char into a double
		// quoted string in order to support single quoted strings.
		t.peek.Value = fmt.Sprintf("\"%s\"", t.peek.Value[1:len(t.peek.Value)-1])
		fallthrough
	case scanner.String:
		s, err := strconv.Unquote(t.peek.Value)
		if err != nil {
			panic(err.Error())
		}
		t.peek.Value = s
		if t.peek.Type == scanner.Char && utf8.RuneCountInString(s) > 1 {
			t.peek.Type = scanner.String
		}
	case scanner.RawString:
		t.peek.Value = t.peek.Value[1 : len(t.peek.Value)-1]
	}
	return *t.peek
}

func (t *textScannerLexer) Position() Position {
	t.pos.Filename = t.filename
	return Position(t.pos)
}
