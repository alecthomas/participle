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

// Definition provides the parser with metadata for a lexer.
type Definition interface {
	// Lex an io.Reader.
	Lex(io.Reader) Lexer
	// Symbols returns a map of symbolic names to the corresponding pseudo-runes for those symbols.
	// This is the same approach as used by text/scanner. For example, "EOF" might have the rune
	// value of -1, "Ident" might be -2, and so on.
	Symbols() map[string]rune
}

// A SimpleLexer returns tokens from a source.
//
// Errors are reported via panic, with the panic value being an instance of Error.
type SimpleLexer interface {
	// Next consumes and returns the next token.
	Next() Token
}

// A Lexer returns tokens from a source and allows peeking.
//
// Errors are reported via panic, with the panic value being an instance of Error.
type Lexer interface {
	SimpleLexer
	// Peek at the next token.
	Peek(n int) Token
}

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

// ConsumeAll reads all tokens from a Lexer.
func ConsumeAll(lexer Lexer) (tokens []Token, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = msg.(error)
		}
	}()
	for {
		token := lexer.Next()
		tokens = append(tokens, token)
		if token.Type == EOF {
			return
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

// EOF returns true if this Token is an EOF token.
func (t Token) EOF() bool {
	return t.Type == EOF
}

func (t Token) String() string {
	return t.Value
}
