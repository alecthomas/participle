package participle

import (
	"reflect"
	"strings"
)

// A structLexer lexes over the tags of struct fields while tracking the current field.
type structLexer struct {
	s     reflect.Type
	field int
	lexer Lexer
	r     *strings.Reader
}

func lexStruct(s reflect.Type) *structLexer {
	return &structLexer{
		s:     s,
		lexer: LexString(fieldLexerTag(s.Field(0))),
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

func (s *structLexer) Peek() Token {
	field := s.field
	lexer := s.lexer
	for {
		token := lexer.Peek()
		if !token.EOF() {
			return token
		}
		field++
		if field >= s.s.NumField() {
			return EOFToken
		}
		tag := fieldLexerTag(s.s.Field(field))
		lexer = LexString(tag)
	}
}

func (s *structLexer) Next() Token {
	token := s.lexer.Next()
	if !token.EOF() {
		return token
	}
	if s.field+1 >= s.s.NumField() {
		return EOFToken
	}
	s.field++
	tag := fieldLexerTag(s.s.Field(s.field))
	s.lexer = LexString(tag)
	return s.Next()
}

func (s *structLexer) Position() Position {
	pos := s.lexer.Position()
	pos.Line = s.field + 1
	return pos
}

func fieldLexerTag(field reflect.StructField) string {
	if tag := field.Tag.Get("parser"); tag != "" {
		return tag
	}
	return string(field.Tag)
}
