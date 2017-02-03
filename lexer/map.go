package lexer

import (
	"io"
	"strconv"
)

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

func MakeSymbolTable(def Definition, types ...string) map[rune]bool {
	sym := def.Symbols()
	table := map[rune]bool{}
	for _, r := range types {
		table[sym[r]] = true
	}
	return table
}

// Elide wraps a Lexer, removing tokens matching the given types.
func Elide(def Definition, types ...string) Definition {
	table := MakeSymbolTable(def, types...)
	return Map(def, func(token *Token) *Token {
		if table[token.Type] {
			return nil
		}
		return token
	})
}

// Unquote applies strconv.Unquote() to tokens of the given types.
//
// Tokens of type "String" will be unquoted if no other types are provided.
func Unquote(def Definition, types ...string) Definition {
	if len(types) == 0 {
		types = []string{"String"}
	}
	table := MakeSymbolTable(def, types...)
	return Map(def, func(t *Token) *Token {
		if table[t.Type] {
			value, err := strconv.Unquote(t.Value)
			if err != nil {
				Panicf(t.Pos, "invalid quoted string %q: %s", t.Value, err.Error())
			}
			t.Value = value
		}
		return t
	})
}
