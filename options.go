package participle

import (
	"fmt"
	"reflect"

	"github.com/alecthomas/participle/v2/lexer"
)

// An Option to modify the behaviour of the Parser.
type Option func(p *Parser) error

// Lexer is an Option that sets the lexer to use with the given grammar.
func Lexer(def lexer.Definition) Option {
	return func(p *Parser) error {
		p.lex = def
		return nil
	}
}

// UseLookahead allows branch lookahead up to "n" tokens.
//
// If parsing cannot be disambiguated before "n" tokens of lookahead, parsing will fail.
//
// Note that increasing lookahead has a minor performance impact, but also
// reduces the accuracy of error reporting.
//
// If "n" is negative, this will be treated as "infinite lookahead".
// This _will_ impact performance, but can be useful for parsing ambiguous
// grammars.
func UseLookahead(n int) Option {
	return func(p *Parser) error {
		p.useLookahead = n
		return nil
	}
}

// CaseInsensitive allows the specified token types to be matched case-insensitively.
//
// Note that the lexer itself will also have to be case-insensitive; this option
// just controls whether literals in the grammar are matched case insensitively.
func CaseInsensitive(tokens ...string) Option {
	return func(p *Parser) error {
		for _, token := range tokens {
			p.caseInsensitive[token] = true
		}
		return nil
	}
}

func UseCustom(parseFn interface{}) Option {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	peekingLexerType := reflect.TypeOf((*lexer.PeekingLexer)(nil))
	return func(p *Parser) error {
		parseFnVal := reflect.ValueOf(parseFn)
		parseFnType := parseFnVal.Type()
		if parseFnType.Kind() != reflect.Func {
			return fmt.Errorf("production parser must be a function (got %s)", parseFnType)
		}
		if parseFnType.NumIn() != 1 || parseFnType.In(0) != reflect.TypeOf((*lexer.PeekingLexer)(nil)) {
			return fmt.Errorf("production parser must take a single parameter of type %s", peekingLexerType)
		}
		if parseFnType.NumOut() != 2 {
			return fmt.Errorf("production parser must return exactly two values: the parsed production, and an error")
		}
		if parseFnType.Out(0).Kind() != reflect.Interface {
			return fmt.Errorf("production parser's first return must be an interface type")
		}
		if parseFnType.Out(1) != errorType {
			return fmt.Errorf("production parser's second return must be %s", errorType)
		}
		prodType := parseFnType.Out(0)
		p.customDefs = append(p.customDefs, customDef{prodType, parseFnVal})
		return nil
	}
}

func UseUnion[T any](members ...T) Option {
	return func(p *Parser) error {
		unionType := reflect.TypeOf((*T)(nil)).Elem()
		if unionType.Kind() != reflect.Interface {
			return fmt.Errorf("union type must be an interface (got %s)", unionType)
		}
		memberTypes := make([]reflect.Type, 0, len(members))
		for _, m := range members {
			memberTypes = append(memberTypes, reflect.TypeOf(m))
		}
		p.unionDefs = append(p.unionDefs, unionDef{unionType, memberTypes})
		return nil
	}
}

// ParseOption modifies how an individual parse is applied.
type ParseOption func(p *parseContext)

// AllowTrailing tokens without erroring.
//
// That is, do not error if a full parse completes but additional tokens remain.
func AllowTrailing(ok bool) ParseOption {
	return func(p *parseContext) {
		p.allowTrailing = ok
	}
}
