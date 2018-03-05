package lexer

// Upgrade a SimpleLexer to a Lexer with arbitrary lookahead.
func Upgrade(simple SimpleLexer) Lexer {
	return &lookaheadLexer{SimpleLexer: simple}
}

type lookaheadLexer struct {
	SimpleLexer
	peeked []Token
}

func (l *lookaheadLexer) Peek(n int) Token {
	for len(l.peeked) <= n {
		t := l.SimpleLexer.Next()
		if t.EOF() {
			return t
		}
		l.peeked = append(l.peeked, t)
	}

	return l.peeked[n]
}

func (l *lookaheadLexer) Next() Token {
	if len(l.peeked) > 0 {
		t := l.peeked[0]
		l.peeked = l.peeked[1:]
		return t
	}
	return l.SimpleLexer.Next()
}
