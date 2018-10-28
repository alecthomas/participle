package ebnf

import (
	"strings"
	"text/scanner"

	"github.com/alecthomas/participle/lexer/ebnf/internal"
)

// TODO: Add a "repeatedrangeSet" to represent the common case of { set } ??

func makeSet(pos scanner.Position, set *rangeSet) internal.Expression {
	if set.include[0] < 0 || set.include[1] > 255 {
		return set
	}
	ascii := &asciiSet{pos: pos}
	for rn := set.include[0]; rn <= set.include[1]; rn++ {
		ascii.Insert(rn)
	}
	for _, exclude := range set.exclude {
		for rn := exclude[0]; rn <= exclude[1]; rn++ {
			ascii.Unset(rn)
		}
	}
	return ascii
}

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

// A set of arbitrary runes represented by a string.
//
// Uses strings.ContainsRune() to check if a rune is in the set.
type rangeSet struct {
	pos     scanner.Position
	include [2]rune
	exclude [][2]rune
}

func (c *rangeSet) Pos() scanner.Position {
	return c.pos
}

func (c *rangeSet) Has(rn rune) bool {
	if rn < c.include[0] || rn > c.include[1] {
		return false
	}
	for _, exclude := range c.exclude {
		if rn >= exclude[0] && rn <= exclude[1] {
			return false
		}
	}
	return true
}

// A faster representation of a character set using a 256-bit-wide bitset.
type asciiSet struct {
	pos   scanner.Position
	ascii [4]uint64
}

func (a *asciiSet) Unset(rn rune) bool {
	if rn < 0 || rn > 255 {
		return false
	}
	a.ascii[rn>>6] &= ^(1 << uint64(rn&0x3f))
	return true
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

type ebnfToken struct {
	pos   scanner.Position
	runes []rune
}

func (e *ebnfToken) Pos() scanner.Position {
	return e.pos
}
