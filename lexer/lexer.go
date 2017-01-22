// Package lexer defines interfaces and implementations used by Participle to perform lexing.
//
// The primary interfaces are LexerDefinition and Lexer. There are two implementations of these
// interfaces:
//
// NewTextScannerLexer() is based on text/scanner. This is the fastest, but least flexible, in that
// tokens are restricted to those supported by that package. It can scan about 5M tokens/second on a
// late 2013 15" MacBook Pro.
//
// The second lexer provided accepts a lexical grammar in EBNF. Each capitalised production is a
// lexical token supported by the resulting Lexer. This is very flexible, but a bit slower, scanning
// around 730K tokens/second on the same machine, though it is currently completely unoptimised.
// This could/should be converted to a table-based lexer.
//
// Lexer implementations must use Panic/Panicf to report errors.
package lexer

import (
	"fmt"
	"io"
)

const (
	// EOF represents an end of file.
	EOF rune = -(iota + 1)
)

var (
	// EOFToken is a Token representing EOF.
	EOFToken = Token{Type: EOF, Value: "<<EOF>>"}

	// DefaultDefinition defines properties for the default lexer.
	DefaultDefinition Definition = &defaultDefinition{}
)

type namedReader interface {
	Name() string
}

// NameOfReader attempts to retrieve the filename of a reader.
func NameOfReader(r io.Reader) string {
	if nr, ok := r.(namedReader); ok {
		return nr.Name()
	}
	return ""
}

// Must takes the result of a Definition constructor call and returns the definition, but panics if
// it errors
//
// eg.
//
// 		lex = lexer.Must(lexer.Build(`Symbol = "symbol" .`))
func Must(def Definition, err error) Definition {
	if err != nil {
		panic(err)
	}
	return def
}

// ReadAll returns all tokens from a Lexer.
func ReadAll(lexer Lexer) []Token {
	out := []Token{}
	for {
		token := lexer.Next()
		out = append(out, token)
		if token.Type == EOF {
			return out
		}
	}
}

// Position of a token.
type Position struct {
	Filename string
	Offset   int
	Line     int
	Column   int
}

func (p Position) String() string {
	filename := p.Filename
	if filename == "" {
		filename = "<source>"
	}
	return fmt.Sprintf("%s:%d:%d", filename, p.Line, p.Column)
}

// A Token returned by a Lexer.
type Token struct {
	// Type of token. This is the value keyed by symbol as returned by Definition.Symbols().
	Type  rune
	Value string
	Pos   Position
}

// RuneToken represents a rune as a Token.
func RuneToken(r rune) Token {
	return Token{Type: r, Value: string(r)}
}

func (t Token) EOF() bool {
	return t.Type == EOF
}

func (t Token) String() string {
	return t.Value
}

// Definition provides the parser with metadata for a lexer.
type Definition interface {
	// Lex an io.Reader.
	Lex(io.Reader) Lexer
	// Symbols returns a map of symbolic names to the corresponding pseudo-runes for those symbols.
	// This is the same approach as used by text/scanner. For example, "EOF" might have the rune
	// value of -1, "Ident" might be -2, and so on.
	Symbols() map[string]rune
}

// A Lexer returns tokens from a source.
//
// Errors are reported via panic, with the panic value being an instance of Error.
type Lexer interface {
	// Peek at the next token.
	Peek() Token
	// Next consumes and returns the next token.
	Next() Token
}
