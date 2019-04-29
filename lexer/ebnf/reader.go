package ebnf

import (
	"io"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
)

// A rewindable rune reader.
//
// Allows for multiple attempts to be made to read a sequence of runes.
type tokenReader struct {
	r      io.RuneReader
	cursor int
	runes  []rune
	oldPos lexer.Position
	pos    lexer.Position
}

func newTokenReader(r io.RuneReader, pos lexer.Position) *tokenReader {
	return &tokenReader{r: r, pos: pos}
}

func (r *tokenReader) Pos() lexer.Position {
	return r.pos
}

// Begin a new token attempt.
func (r *tokenReader) Begin() {
	r.runes = r.runes[r.cursor:]
	r.cursor = 0
	r.oldPos = r.pos
}

// Rewind to beginning of token attempt.
func (r *tokenReader) Rewind() {
	r.cursor = 0
	r.pos = r.oldPos
}

func (r *tokenReader) Read() (rune, error) {
	// Need to buffer?
	rn, err := r.Peek()
	if err != nil {
		return 0, err
	}
	r.pos.Offset += utf8.RuneLen(rn)
	if rn == '\n' {
		r.pos.Line++
		r.pos.Column = 1
	} else {
		r.pos.Column++
	}
	r.cursor++
	return rn, nil
}

func (r *tokenReader) Peek() (rune, error) {
	if r.cursor >= len(r.runes) {
		return r.buffer()
	}
	return r.runes[r.cursor], nil
}

// Buffer a rune without moving the cursor.
func (r *tokenReader) buffer() (rune, error) {
	rn, _, err := r.r.ReadRune()
	if err != nil {
		return 0, err
	}
	r.runes = append(r.runes, rn)
	return rn, nil
}
