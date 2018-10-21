package ebnf

// Option for configuring the EBNF lexer.
type Option func(*ebnfLexerDefinition)

// Elide is a n Option to remove the matching tokens from the output stream.
//
// This is useful for things like whitespace, comments, etc.
func Elide(tokens ...string) Option {
	return func(l *ebnfLexerDefinition) {
		for _, t := range tokens {
			l.elide[t] = true
		}
	}
}
