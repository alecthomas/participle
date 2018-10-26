package ebnf

import (
	"strings"
	"text/scanner"

	"github.com/alecthomas/participle/lexer/ebnf/internal"
)

// TODO: Add a "repeatedCharacterSet" to represent the common case of { set } ??

func makeSet(pos scanner.Position, set string) internal.Expression {
	ascii := &asciiSet{pos: pos}
	for _, rn := range set {
		if !ascii.Insert(rn) {
			return &characterSet{pos: pos, Set: set}
		}
	}
	return ascii
}

// A set of arbitrary runes represented by a string.
//
// Uses strings.ContainsRune() to check if a rune is in the set.
type characterSet struct {
	pos scanner.Position
	Set string
}

func (c *characterSet) Pos() scanner.Position {
	return c.pos
}

func (c *characterSet) Has(rn rune) bool {
	return strings.ContainsRune(c.Set, rn)
}

// A faster representation of a character set using a 256-bit-wide bitset.
type asciiSet struct {
	pos   scanner.Position
	ascii [4]uint64
}

func (a *asciiSet) Insert(rn rune) bool {
	if rn < 0 || rn > 255 {
		return false
	}
	a.ascii[rn>>6] |= (1 << uint64(rn&0x3f))
	return true
}

func (a *asciiSet) Has(rn rune) bool {
	return rn >= 0 && rn <= 255 && a.ascii[rn>>6]&(1<<uint64(rn&0x3f)) > 0
}

func (a *asciiSet) Pos() scanner.Position {
	return a.pos
}

type ebnfRange struct {
	pos        scanner.Position
	start, end rune
	exclude    internal.Expression
}

func (e *ebnfRange) Pos() scanner.Position {
	return e.pos
}

type ebnfToken struct {
	pos   scanner.Position
	runes []rune
}

func (e *ebnfToken) Pos() scanner.Position {
	return e.pos
}
