package lexer

// PeekingLexer supports arbitrary lookahead as well as cloning.
type PeekingLexer struct {
	cursor int
	eof    Token
	tokens []Token
}

// Upgrade a Lexer to a PeekingLexer with arbitrary lookahead.
func Upgrade(lex Lexer) (*PeekingLexer, error) {
	r := &PeekingLexer{}
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

// Cursor position in tokens.
func (p *PeekingLexer) Cursor() int {
	return p.cursor
}

// Length returns the number of tokens consumed by the lexer.
func (p *PeekingLexer) Length() int {
	return len(p.tokens)
}

// Next consumes and returns the next token.
func (p *PeekingLexer) Next() (Token, error) {
	if p.cursor >= len(p.tokens) {
		return p.eof, nil
	}
	p.cursor++
	return p.tokens[p.cursor-1], nil
}

// Peek ahead at the n+1 token. ie. Peek(0) will peek at the next token.
func (p *PeekingLexer) Peek(n int) (Token, error) {
	i := p.cursor + n
	if i >= len(p.tokens) {
		return p.eof, nil
	}
	return p.tokens[i], nil
}

// Clone creates a clone of this PeekingLexer at its current token.
//
// The parent and clone are completely independent.
func (p *PeekingLexer) Clone() *PeekingLexer {
	clone := *p
	return &clone
}
