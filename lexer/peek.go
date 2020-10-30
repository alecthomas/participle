package lexer

// PeekingLexer supports arbitrary lookahead as well as cloning.
type PeekingLexer struct {
	cursor int
	eof    Token
	tokens []Token
	elide  map[rune]bool
}

var _ Lexer = &PeekingLexer{}

// Upgrade a Lexer to a PeekingLexer with arbitrary lookahead.
//
// "elide" is a slice of token types to elide from processing.
func Upgrade(lex Lexer, elide ...rune) (*PeekingLexer, error) {
	r := &PeekingLexer{
		elide: make(map[rune]bool, len(elide)),
	}
	for _, rn := range elide {
		r.elide[rn] = true
	}
	for {
		t, err := lex.Next()
		if err != nil {
			return r, err
		}
		if t.EOF() {
			r.eof = t
			break
		}
		r.tokens = append(r.tokens, t)
	}
	return r, nil
}

// Range returns the slice of tokens between the two cursor points.
func (p *PeekingLexer) Range(start, end int) []Token {
	return p.tokens[start:end]
}

// Cursor position in tokens (includes elided tokens).
func (p *PeekingLexer) Cursor() int {
	return p.cursor
}

// Next consumes and returns the next token.
func (p *PeekingLexer) Next() (Token, error) {
	for p.cursor < len(p.tokens) {
		t := p.tokens[p.cursor]
		p.cursor++
		if p.elide[t.Type] {
			continue
		}
		return p.tokens[p.cursor-1], nil
	}
	return p.eof, nil
}

// Peek ahead at the n+1 token. ie. Peek(0) will peek at the next token.
func (p *PeekingLexer) Peek(n int) (Token, error) {
	for i := p.cursor; i < len(p.tokens); i++ {
		t := p.tokens[i]
		if p.elide[t.Type] {
			continue
		}
		if n == 0 {
			return t, nil
		}
		n--
	}
	return p.eof, nil
}

// PeekRaw ahead at the raw token n+1. ie. PeekRaw(0) will peek at the next token.
//
// Unlike Peek, this will include elided tokens.
func (p *PeekingLexer) PeekRaw(n int) (Token, error) {
	for i := p.cursor; i < len(p.tokens); i++ {
		t := p.tokens[i]
		if n == 0 {
			return t, nil
		}
		n--
	}
	return p.eof, nil
}

// Clone creates a clone of this PeekingLexer at its current token.
//
// The parent and clone are completely independent.
func (p *PeekingLexer) Clone() *PeekingLexer {
	clone := *p
	return &clone
}
