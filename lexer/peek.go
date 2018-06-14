package lexer

// Upgrade a Lexer to a PeekingLexer with arbitrary lookahead.
func Upgrade(lexer Lexer) PeekingLexer {
	if peeking, ok := lexer.(PeekingLexer); ok {
		return peeking
	}
	transform, _ := lexer.(Transform)
	return &lookaheadLexer{Lexer: lexer, transform: transform}
}

type lookaheadLexer struct {
	Lexer
	peeked    []Token
	transform Transform
}

func (l *lookaheadLexer) Peek(n int) Token {
	for len(l.peeked) <= n {
		t := l.Lexer.Next()
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
	return l.Lexer.Next()
}

func (l *lookaheadLexer) Transform(token Token) Token {
	if l.transform != nil {
		return l.transform.Transform(token)
	}
	return token
}
