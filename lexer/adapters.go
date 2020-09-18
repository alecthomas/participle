package lexer

import (
	"bytes"
	"io"
	"strings"
)

type legacy struct {
	legacy interface {
		Lex(io.Reader) (Lexer, error)
		Symbols() map[string]rune
	}
}

type namedReader struct {
	io.Reader
	name string
}

func (n *namedReader) Name() string { return n.name }

func (l legacy) LexReader(filename string, r io.Reader) (Lexer, error) {
	return l.legacy.Lex(namedReader{r, filename})
}
func (l legacy) LexString(filename string, s string) (Lexer, error) {
	return l.legacy.Lex(namedReader{strings.NewReader(s), filename})
}
func (l legacy) LexBytes(filename string, b []byte) (Lexer, error) {
	return l.legacy.Lex(namedReader{bytes.NewReader(b), filename})
}
func (l legacy) Symbols() map[string]rune { return l.legacy.Symbols() }

// Legacy is a shim for Participle v0 lexer definitions.
func Legacy(def interface {
	Lex(io.Reader) (Lexer, error)
	Symbols() map[string]rune
}) Definition {
	return legacy{def}
}

// Simple upgrades a lexer that only implements LexReader() by using
// strings/bytes.NewReader().
func Simple(def interface {
	Symbols() map[string]rune
	LexReader(string, io.Reader) (Lexer, error)
}) Definition {
	return simple{def}
}

type simplei interface {
	Symbols() map[string]rune
	LexReader(string, io.Reader) (Lexer, error)
}

type simple struct{ simplei }

func (s simple) LexString(filename string, str string) (Lexer, error) {
	return s.LexReader(filename, strings.NewReader(str))
}
func (s simple) LexBytes(filename string, b []byte) (Lexer, error) {
	return s.LexReader(filename, bytes.NewReader(b))
}
