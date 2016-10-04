package participle

import (
	"reflect"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// A structLexer lexes over the tags of struct fields while tracking the current field.
type structLexer struct {
	s     reflect.Type
	field int
	lexer lexer.Lexer
	r     *strings.Reader
}

func lexStruct(s reflect.Type) *structLexer {
	return &structLexer{
		s:     s,
		lexer: lexer.LexString(fieldLexerTag(s.Field(0))),
	}
}

// NumField returns the number of fields in the struct associated with this structLexer.
func (s *structLexer) NumField() int {
	return s.s.NumField()
}

// Field returns the field associated with the current token.
func (s *structLexer) Field() reflect.StructField {
	return s.s.Field(s.field)
}

func (s *structLexer) Peek() lexer.Token {
	field := s.field
	lex := s.lexer
	for {
		token := lex.Peek()
		if !token.EOF() {
			token.Pos.Line = field + 1
			return token
		}
		field++
		if field >= s.s.NumField() {
			return lexer.EOFToken
		}
		tag := fieldLexerTag(s.s.Field(field))
		lex = lexer.LexString(tag)
	}
}

func (s *structLexer) Next() lexer.Token {
	token := s.lexer.Next()
	if !token.EOF() {
		token.Pos.Line = s.field + 1
		return token
	}
	if s.field+1 >= s.s.NumField() {
		return lexer.EOFToken
	}
	s.field++
	tag := fieldLexerTag(s.s.Field(s.field))
	s.lexer = lexer.LexString(tag)
	return s.Next()
}

func fieldLexerTag(field reflect.StructField) string {
	if tag := field.Tag.Get("parser"); tag != "" {
		return tag
	}
	return string(field.Tag)
}
