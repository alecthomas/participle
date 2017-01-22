package lexer

import (
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
)

var eolBytes = []byte("\n")

type regexpDefinition struct {
	re      *regexp.Regexp
	symbols map[string]rune
}

// Regexp creates a lexer definition from a regular expression.
//
// Each named sub-expression in the regular expression matches a token.
//
// eg.
//
//     	def, err := Regexp(`(?P<Ident>[a-z]+)|(?P<Whitespace>\s+)|(?P<Number>\d+)`)
func Regexp(pattern string) (Definition, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	symbols := map[string]rune{
		"EOF": EOF,
	}
	for i, sym := range re.SubexpNames()[1:] {
		symbols[sym] = EOF - 1 - rune(i)
	}
	return &regexpDefinition{re: re, symbols: symbols}, nil
}

func (rd *regexpDefinition) Lex(r io.Reader) Lexer {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		// TODO: Make Lex also return an error.
		panic(err)
	}
	return &regexpLexer{
		pos: Position{
			Filename: NameOfReader(r),
			Line:     1,
			Column:   1,
		},
		b:  b,
		re: rd.re,
	}
}

func (rd *regexpDefinition) Symbols() map[string]rune {
	return rd.symbols
}

type regexpLexer struct {
	pos     Position
	b       []byte
	re      *regexp.Regexp
	symbols map[string]rune
	peek    *Token
}

func (r *regexpLexer) Peek() Token {
	if r.peek != nil {
		return *r.peek
	}
	if len(r.b) == 0 {
		eof := EOFToken
		eof.Pos = r.pos
		return eof
	}
	matches := r.re.FindSubmatchIndex(r.b)
	if matches == nil || matches[0] != 0 {
		Panic(r.pos, "invalid token")
	}
	match := r.b[:matches[1]]
	token := Token{
		Pos:   r.pos,
		Value: string(match),
	}
	for i := 2; i < len(matches); i += 2 {
		if matches[i] != -1 {
			token.Type = EOF - rune(i/2)
		}
	}
	// Update lexer state.
	r.pos.Offset += matches[1]
	lines := bytes.Count(match, eolBytes)
	r.pos.Line += lines
	// Update column.
	if lines == 0 {
		r.pos.Column += len(match)
	} else {
		r.pos.Column = len(match) - bytes.LastIndex(match, eolBytes)
	}
	r.b = r.b[matches[1]:]
	r.peek = &token
	return token
}

func (r *regexpLexer) Next() Token {
	token := r.Peek()
	r.peek = nil
	return token
}
