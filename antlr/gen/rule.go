package gen

type LexerRule struct {
	Name       string
	Content    string
	NotLiteral bool
	Length     int
}

type LexerRules []LexerRule

func LiteralLexerRule(s string) LexerRule {
	return LexerRule{Content: s, Length: len(s)}
}

func (lr LexerRule) Plus(lr2 LexerRule) LexerRule {
	return LexerRule{
		Content:    lr.Content + lr2.Content,
		NotLiteral: lr.NotLiteral || lr2.NotLiteral,
		Length:     lr.Length + lr2.Length,
	}
}
