package lexer

import "strconv"

// Unquote applies strconv.Unquote() to tokens of the given types.
//
// Tokens of type "String" will be unquoted if no other types are provided.
func Unquote(def Definition, types ...string) Definition {
	if len(types) == 0 {
		types = []string{"String"}
	}
	sym := def.Symbols()
	table := map[rune]bool{}
	for _, r := range types {
		table[sym[r]] = true
	}
	return Map(def, func(t *Token) *Token {
		if table[t.Type] {
			var err error
			t.Value, err = strconv.Unquote(t.Value)
			if err != nil {
				Panicf(t.Pos, "invalid quoted string: %s", err.Error())
			}
		}
		return t
	})
}
