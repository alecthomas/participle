package participle

import "github.com/alecthomas/participle/lexer"

// An Option to modify the behaviour of the Parser.
type Option func(p *Parser) error

// Lexer is an Option that sets the lexer to use with the given grammar.
func Lexer(def lexer.Definition) Option {
	return func(p *Parser) error {
		p.lex = def
		return nil
	}
}

// UseLookahead builds lookahead tables for disambiguating branches.
//
// NOTE: This is an experimental feature.
func UseLookahead() Option {
	return func(p *Parser) error {
		p.useLookahead = true
		return nil
	}
}

// CaseInsensitive allows the specified token types to be matched case-insensitively.
func CaseInsensitive(tokens ...string) Option {
	return func(p *Parser) error {
		for _, token := range tokens {
			p.caseInsensitive[token] = true
		}
		return nil
	}
}
