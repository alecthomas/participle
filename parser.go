package parser

import (
	"reflect"
	"text/scanner"
)

// Skip used as a field type in a lexer definition tells the lexer to skip matching tokens.
type Skip int

// Build a reflect.Value from a lexer. Returns true if match.
type builder func(lex *lexer) string

type lexer struct {
	scan *scanner.Scanner
}

// NextIf the next rune matches and return true. Otherwise do not advance and return false.
func (l *lexer) NextIf(ch rune) bool {
	if l.scan.Peek() == ch {
		l.scan.Next()
		return true
	}
	return false
}

// func Parse(lexer interface{}, ast interface{}, text string) (interface{}, error) {
// 	lexMap := buildLexer(reflect.ValueOf(lexer))
// 	out := reflect.ValueOf(ast)
// 	return build()
// }

func build(lexer map[string]expression, ast reflect.Value, l *lexer) {
}

func buildLexer(lexer reflect.Value) map[string]expression {
	out := map[string]expression{}
	for i := 0; i < lexer.NumField(); i++ {
		f := lexer.Type().Field(i)
		out[f.Name] = parseTag(string(f.Tag))
	}
	return out
}
