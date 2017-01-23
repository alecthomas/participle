package lexer

import "io"

type mapperDef struct {
	def Definition
	f   MapFunc
}

// MapFunc transforms tokens.
//
// If nil is returned that token will be discarded.
type MapFunc func(*Token) *Token

// Map is a Lexer that applies a mapping function to a Lexer's tokens.
func Map(def Definition, f MapFunc) Definition {
	return &mapperDef{def, f}
}

func (m *mapperDef) Lex(r io.Reader) Lexer {
	return &mapper{lexer: m.def.Lex(r), f: m.f}
}

func (m *mapperDef) Symbols() map[string]rune {
	return m.def.Symbols()
}

type mapper struct {
	lexer Lexer
	f     MapFunc
	peek  *Token
}

func (m *mapper) Peek() Token {
	for m.peek == nil {
		t := m.lexer.Next()
		m.peek = m.f(&t)
	}
	return *m.peek
}

func (m *mapper) Next() Token {
	t := m.Peek()
	m.peek = nil
	return t
}
