package lexer

// Elide wraps a Lexer, removing tokens matching the given types.
func Elide(def Definition, types ...string) Definition {
	sym := def.Symbols()
	table := map[rune]bool{}
	for _, r := range types {
		table[sym[r]] = true
	}
	return Map(def, func(token *Token) *Token {
		if table[token.Type] {
			return nil
		}
		return token
	})
}
