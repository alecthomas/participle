package participle

import "github.com/alecthomas/participle/lexer"

// An Option to modify the behaviour of the Parser.
type Option func(p *Parser) error

// Mapper function for mutating tokens before being applied to the AST.
type Mapper func(token lexer.Token) lexer.Token

// Map is an Option that configures the Parser to apply a mapping function to each Token from the lexer.
//
// This can be useful to eg. upper-case all tokens of a certain type, or dequote strings.
func Map(mappers ...Mapper) Option {
	return func(p *Parser) error {
		for _, mapper := range mappers {
			next := p.mapper
			p.mapper = func(token lexer.Token) lexer.Token {
				return mapper(next(token))
			}
		}
		return nil
	}
}

// ClearMappers is an Option that resets all existing (including default) mappers.
func ClearMappers() Option {
	return func(p *Parser) error {
		p.mapper = nil
		return nil
	}
}

// Lexer is an Option that sets the lexer to use with the given grammar.
func Lexer(def lexer.Definition) Option {
	return func(p *Parser) error {
		p.lex = def
		return nil
	}
}

// UseLookahead builds lookahead tables for disambiguating branches.
//
// NOTE: This is an incomplete, slightly buggy feature, but can be useful already.
func UseLookahead() Option {
	return func(p *Parser) error {
		p.useLookahead = true
		return nil
	}
}
