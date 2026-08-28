package participle

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"

	"github.com/alecthomas/participle/v2/lexer"
)

// A structLexer lexes over the tags of struct fields while tracking the current field.
type structLexer struct {
	s       reflect.Type
	field   int
	indexes [][]int
	lexer   *lexer.PeekingLexer
}

func lexStruct(s reflect.Type) (*structLexer, error) {
	indexes, err := collectFieldIndexes(s)
	if err != nil {
		return nil, err
	}
	slex := &structLexer{
		s:       s,
		indexes: indexes,
	}
	if len(slex.indexes) > 0 {
		tag := fieldLexerTag(slex.Field().StructField)
		slex.lexer, err = lexer.Upgrade(newTagLexer(s.Name(), tag))
		if err != nil {
			return nil, err
		}
	}
	return slex, nil
}

// NumField returns the number of fields in the struct associated with this structLexer.
func (s *structLexer) NumField() int {
	return len(s.indexes)
}

type structLexerField struct {
	reflect.StructField
	Index []int
}

// Field returns the field associated with the current token.
func (s *structLexer) Field() structLexerField {
	return s.GetField(s.field)
}

func (s *structLexer) GetField(field int) structLexerField {
	if field >= len(s.indexes) {
		field = len(s.indexes) - 1
	}
	return structLexerField{
		StructField: s.s.FieldByIndex(s.indexes[field]),
		Index:       s.indexes[field],
	}
}

func (s *structLexer) Peek() (*lexer.Token, error) {
	field := s.field
	lex := s.lexer
	for {
		token := lex.Peek()
		if !token.EOF() {
			token.Pos.Line = field + 1
			return token, nil
		}
		field++
		if field >= s.NumField() {
			t := lexer.EOFToken(token.Pos)
			return &t, nil
		}
		ft := s.GetField(field).StructField
		tag := fieldLexerTag(ft)
		var err error
		lex, err = lexer.Upgrade(newTagLexer(ft.Name, tag))
		if err != nil {
			return token, err
		}
	}
}

func (s *structLexer) Next() (*lexer.Token, error) {
	token := s.lexer.Next()
	if !token.EOF() {
		token.Pos.Line = s.field + 1
		return token, nil
	}
	if s.field+1 >= s.NumField() {
		t := lexer.EOFToken(token.Pos)
		return &t, nil
	}
	s.field++
	ft := s.Field().StructField
	tag := fieldLexerTag(ft)
	var err error
	s.lexer, err = lexer.Upgrade(newTagLexer(ft.Name, tag))
	if err != nil {
		return token, err
	}
	return s.Next()
}

func fieldLexerTag(field reflect.StructField) string {
	if tag, ok := field.Tag.Lookup("parser"); ok {
		return tag
	}
	return string(field.Tag)
}

// Recursively collect flattened indices for top-level fields and embedded fields.
func collectFieldIndexes(s reflect.Type) (out [][]int, err error) {
	if s.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct but got %q", s)
	}
	defer decorate(&err, s.String)
	for i := range s.NumField() {
		f := s.Field(i)
		switch {
		case f.Anonymous && f.Type.Kind() == reflect.Struct: // Embedded struct.
			children, err := collectFieldIndexes(f.Type)
			if err != nil {
				return nil, err
			}
			for _, idx := range children {
				out = append(out, append(f.Index, idx...))
			}

		case f.PkgPath != "":
			continue

		case fieldLexerTag(f) != "":
			out = append(out, f.Index)
		}
	}
	return
}

// escapeControlChars rewrites control characters as Go escape sequences
// (eg. '\n' -> "\n") so that text/scanner accepts them inside string literals.
func escapeControlChars(s string) string {
	var out strings.Builder
	for _, r := range s {
		if r < ' ' {
			q := strconv.Quote(string(r))
			out.WriteString(q[1 : len(q)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// tagLexer is a Lexer based on text/scanner.Scanner
type tagLexer struct {
	scanner  *scanner.Scanner
	filename string
	err      error
}

func newTagLexer(filename string, tag string) *tagLexer {
	// text/scanner's char literals don't support escape sequences, so convert
	// single-quoted literals containing escapes (eg. '\t', '\n') to
	// double-quoted strings before scanning.
	tag = quoteEscapedCharLiterals(tag)
	s := &scanner.Scanner{}
	s.Init(strings.NewReader(tag))
	lexer := &tagLexer{
		filename: filename,
		scanner:  s,
	}
	lexer.scanner.Error = func(_ *scanner.Scanner, msg string) {
		// This is to support single quoted strings. Hacky.
		if !strings.HasSuffix(msg, "char literal") {
			lexer.err = fmt.Errorf("%s: %s", lexer.scanner.Pos(), msg)
		}
	}
	return lexer
}

// quoteEscapedCharLiterals rewrites single-quoted literals containing escape
// sequences (eg. '\n', '\'', '\\') as double-quoted strings, which
// text/scanner handles correctly.
func quoteEscapedCharLiterals(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		switch s[i] {
		case '"':
			out.WriteByte(s[i])
			i++
			for i < len(s) && (s[i] != '"' || s[i-1] == '\\') {
				out.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				out.WriteByte(s[i])
				i++
			}
		case '`':
			out.WriteByte(s[i])
			i++
			for i < len(s) && s[i] != '`' {
				out.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				out.WriteByte(s[i])
				i++
			}
		case '\'':
			j := i + 1
			escaped := false
			for j < len(s) {
				if s[j] == '\\' {
					j += 2
					escaped = true
					continue
				}
				if s[j] < ' ' {
					j++
					escaped = true
					continue
				}
				if s[j] == '\'' {
					break
				}
				j++
			}
			if j >= len(s) {
				out.WriteByte(s[i])
				i++
				continue
			}
			if escaped {
				out.WriteByte('"')
				out.WriteString(escapeControlChars(s[i+1 : j]))
				out.WriteByte('"')
			} else {
				out.WriteString(s[i : j+1])
			}
			i = j + 1
		default:
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

func (t *tagLexer) Next() (lexer.Token, error) {
	typ := t.scanner.Scan()
	text := t.scanner.TokenText()
	pos := lexer.Position(t.scanner.Position)
	pos.Filename = t.filename
	if t.err != nil {
		return lexer.Token{}, t.err
	}
	return textScannerTransform(lexer.Token{
		Type:  lexer.TokenType(typ),
		Value: text,
		Pos:   pos,
	})
}

func textScannerTransform(token lexer.Token) (lexer.Token, error) {
	// Unquote strings.
	switch token.Type {
	case scanner.Char:
		// FIXME(alec): This is pretty hacky...we convert a single quoted char into a double
		// quoted string in order to support single quoted strings.
		token.Value = fmt.Sprintf("\"%s\"", token.Value[1:len(token.Value)-1])
		fallthrough
	case scanner.String:
		s, err := strconv.Unquote(token.Value)
		if err != nil {
			return lexer.Token{}, Errorf(token.Pos, "%s: %q", err.Error(), token.Value)
		}
		token.Value = s
		if token.Type == scanner.Char && utf8.RuneCountInString(s) > 1 {
			token.Type = scanner.String
		}
	case scanner.RawString:
		token.Value = token.Value[1 : len(token.Value)-1]

	default:
	}
	return token, nil
}
