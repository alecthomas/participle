package lexer

// Upgrade a SimpleLexer to a full Lexer with arbitrary lookahead.
func Upgrade(lexer SimpleLexer) Lexer {
	transform, _ := lexer.(Transform)
	return &lookaheadLexer{SimpleLexer: lexer, transform: transform}
}

type lookaheadLexer struct {
	SimpleLexer
	peeked    []Token
	transform Transform
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

func (l *lookaheadLexer) Transform(token Token) Token {
	if l.transform != nil {
		return l.transform.Transform(token)
	}
	return token
}
