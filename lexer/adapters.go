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

func (l legacy) LexReader(r io.Reader) (Lexer, error) { return l.legacy.Lex(r) }
func (l legacy) LexString(s string) (Lexer, error)    { return l.legacy.Lex(strings.NewReader(s)) }
func (l legacy) LexBytes(b []byte) (Lexer, error)     { return l.legacy.Lex(bytes.NewReader(b)) }
func (l legacy) Symbols() map[string]rune             { return l.legacy.Symbols() }

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
	LexReader(io.Reader) (Lexer, error)
}) Definition {
	return simple{def}
}

type simplei interface {
	Symbols() map[string]rune
	LexReader(io.Reader) (Lexer, error)
}

type simple struct{ simplei }

func (s simple) LexString(str string) (Lexer, error) { return s.LexReader(strings.NewReader(str)) }
func (s simple) LexBytes(b []byte) (Lexer, error)    { return s.LexReader(bytes.NewReader(b)) }
