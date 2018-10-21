package ebnf

// Option for configuring the EBNF lexer.
type Option func(*ebnfLexerDefinition)

func Elide(tokens ...string) Option {
	return func(l *ebnfLexerDefinition) {
		for _, t := range tokens {
			l.elide[t] = true
		}
	}
}
