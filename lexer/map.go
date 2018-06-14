package lexer

import (
	"io"
	"strconv"
	"strings"
)

// MapperFunc is a convenience function implementing the Transform interface.
type MapperFunc func(Token) Token

func (m MapperFunc) Transform(t Token) Token { return m(t) }

// Map is a Lexer that applies a mapping function to a Lexer's tokens.
func Map(def Definition, f MapperFunc) Definition {
	next, _ := def.(Transform)
	return &mapperDef{def: def, f: f, next: next}
}

type mapperDef struct {
	def  Definition
	f    MapperFunc
	next Transform
}

func (m *mapperDef) Lex(r io.Reader) Lexer {
	return &mapper{Lexer: m.def.Lex(r), f: m.f, next: m.next}
}

func (m *mapperDef) Symbols() map[string]rune {
	return m.def.Symbols()
}

type mapper struct {
	Lexer
	f    MapperFunc
	next Transform
}

func (m *mapper) Transform(t Token) Token {
	if m.next != nil {
		t = m.next.Transform(t)
	}
	return m.f.Transform(t)
}

// MakeSymbolTable is a useful helper function for Definition decorator types.
func MakeSymbolTable(def Definition, types ...string) map[rune]bool {
	sym := def.Symbols()
	table := map[rune]bool{}
	for _, r := range types {
		table[sym[r]] = true
	}
	return table
}

// Unquote applies strconv.Unquote() to tokens of the given types.
//
// Tokens of type "String" will be unquoted if no other types are provided.
func Unquote(def Definition, types ...string) Definition {
	if len(types) == 0 {
		types = []string{"String"}
	}
	table := MakeSymbolTable(def, types...)
	return Map(def, func(t Token) Token {
		if table[t.Type] {
			value, err := unquote(t.Value)
			if err != nil {
				Panicf(t.Pos, "invalid quoted string %q: %s", t.Value, err.Error())
			}
			t.Value = value
		}
		return t
	})
}

func unquote(s string) (string, error) {
	quote := s[0]
	s = s[1 : len(s)-1]
	out := ""
	for s != "" {
		value, _, tail, err := strconv.UnquoteChar(s, quote)
		if err != nil {
			return "", err
		}
		s = tail
		out += string(value)
	}
	return out, nil
}

// Upper case all tokens of the given type. Useful for case normalisation.
func Upper(def Definition, types ...string) Definition {
	table := MakeSymbolTable(def, types...)
	return Map(def, func(token Token) Token {
		if table[token.Type] {
			token.Value = strings.ToUpper(token.Value)
		}
		return token
	})

}
