package lexer

// Elide wraps a Lexer, removing tokens matching the given types.
func Elide(def Definition, tokens []rune) Definition {
	table := map[rune]bool{}
	for _, r := range tokens {
		table[r] = true
	}
	return Map(def, func(token *Token) *Token {
		if table[token.Type] {
			return nil
		}
		return token
	})
}
