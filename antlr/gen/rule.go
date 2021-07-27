package gen

// LexerRule is the result of walking the Antlr grammar AST
// with the intention of building a Participle lexer rule.
// It is essentially a regex expression with additional metadata.
type LexerRule struct {
	Name       string
	Content    string
	NotLiteral bool
	Length     int
}

// LexerRules is a convenience type.
type LexerRules []LexerRule

// LiteralLexerRule is a utility factory for LexerRules that are literal text,
// with no groups, alternatives or ranges.
func LiteralLexerRule(s string) LexerRule {
	return LexerRule{Content: s, Length: len(s)}
}

// Plus appends two LexerRules intelligently.
func (lr LexerRule) Plus(lr2 LexerRule) LexerRule {
	return LexerRule{
		Content:    lr.Content + lr2.Content,
		NotLiteral: lr.NotLiteral || lr2.NotLiteral,
		Length:     lr.Length + lr2.Length,
	}
}
