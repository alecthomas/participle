package lexer

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

// TextScannerLexer is a lexer that uses the text/scanner module.
var TextScannerLexer Definition = &defaultDefinition{}

type defaultDefinition struct{}

func (d *defaultDefinition) Lex(r io.Reader) Lexer {
	return Lex(r)
}

var runeList = map[string]rune{
	"EOF":       scanner.EOF,
	"Char":      scanner.Char,
	"Ident":     scanner.Ident,
	"Int":       scanner.Int,
	"Float":     scanner.Float,
	"String":    scanner.String,
	"RawString": scanner.RawString,
	"Comment":   scanner.Comment,
}

func (d *defaultDefinition) Symbols() map[string]rune {
	return runeList
}

// CommentScannerLexer is a lexer that uses the text/scanner with comment lexing enabled.
var CommentScannerLexer Definition = &commentDefinition{}

type commentDefinition struct{}

func (d *commentDefinition) Lex(r io.Reader) Lexer {
	return LexComments(r)
}

func (d *commentDefinition) Symbols() map[string]rune {
	return runeList
}

// textScannerLexer is a Lexer based on text/scanner.Scanner
type textScannerLexer struct {
	scanner  scanner.Scanner
	peek     *Token
	filename string
}

// Lex an io.Reader with text/scanner.Scanner.
//
// Note that this differs from text/scanner.Scanner in that string tokens will be unquoted.
func Lex(r io.Reader) Lexer {
	var s scanner.Scanner
	s.Init(r)
	s.Error = func(s *scanner.Scanner, msg string) {
		// This is to support single quoted strings. Hacky.
		if msg != "illegal char literal" {
			Panic(Position(s.Pos()), msg)
		}
	}
	return LexScanner(r, s)
}

// LexComments returns a new a lexer using the text/scanner with comment lexing enabled.
func LexComments(r io.Reader) Lexer {
	var s scanner.Scanner
	s.Init(r)

	//Remove the SkipComments flag so that Go style comments ( /**/ and // ) are lexed
	s.Mode &^= scanner.SkipComments

	s.Error = func(s *scanner.Scanner, msg string) {
		// This is to support single quoted strings. Hacky.
		if msg != "illegal char literal" {
			Panic(Position(s.Pos()), msg)
		}
	}
	return LexScanner(r, s)
}

// LexScanner returns a new default lexer with custom text/scanner.Scanner.
func LexScanner(r io.Reader, s scanner.Scanner) Lexer {
	lexer := &textScannerLexer{
		scanner:  s,
		filename: NameOfReader(r),
	}
	return lexer
}

// LexBytes returns a new default lexer over bytes.
func LexBytes(b []byte) Lexer {
	return Lex(bytes.NewReader(b))
}

// LexString returns a new default lexer over a string.
func LexString(s string) Lexer {
	return Lex(strings.NewReader(s))
}

func (t *textScannerLexer) Next() Token {
	if t.peek == nil {
		t.Peek()
	}
	token := t.peek
	t.peek = nil
	return *token
}

func (t *textScannerLexer) Peek() Token {
	if t.peek != nil {
		return *t.peek
	}
	typ := t.scanner.Scan()
	text := t.scanner.TokenText()
	pos := Position(t.scanner.Position)
	pos.Filename = t.filename
	t.peek = &Token{
		Type:  typ,
		Value: text,
		Pos:   pos,
	}
	t.peek.Pos.Filename = t.filename
	// Unquote strings.
	switch t.peek.Type {
	case scanner.Char:
		// FIXME(alec): This is pretty hacky...we convert a single quoted char into a double
		// quoted string in order to support single quoted strings.
		t.peek.Value = fmt.Sprintf("\"%s\"", t.peek.Value[1:len(t.peek.Value)-1])
		fallthrough
	case scanner.String:
		s, err := strconv.Unquote(t.peek.Value)
		if err != nil {
			Panic(t.peek.Pos, err.Error())
		}
		t.peek.Value = s
		if t.peek.Type == scanner.Char && utf8.RuneCountInString(s) > 1 {
			t.peek.Type = scanner.String
		}
	case scanner.RawString:
		t.peek.Value = t.peek.Value[1 : len(t.peek.Value)-1]
	}
	return *t.peek
}
