// Package indenter contains a Lexer that adds support for indentation
// based grammars to an underlying Lexer.
package indenter

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

var indentRe = regexp.MustCompile(`\n\s+`)

// New adds support for indentation based grammars to an underlying Lexer.
//
// The underlying lexer must emit the token type "Whitespace". This lexer
// will transform indentation in the whitespace into either "Indent" or
// "Dedent" tokens.
func New(def lexer.Definition) (lexer.Definition, error) {
	symbols := def.Symbols()
	key, ok := symbols["Whitespace"]
	if !ok {
		return nil, fmt.Errorf("underlying lexer does not provide an \"Whitespace\" token")
	}
	symbolsCopy := make(map[string]lexer.TokenType, len(symbols)+2)
	min := lexer.EOF
	for name, typ := range symbols {
		if typ < min {
			min = typ
		}
		symbolsCopy[name] = typ
	}
	symbolsCopy["Indent"] = min - 1
	symbolsCopy["Dedent"] = min - 2
	return &definition{
		super:   def,
		key:     key,
		indent:  symbolsCopy["Indent"],
		dedent:  symbolsCopy["Dedent"],
		symbols: symbolsCopy,
	}, nil
}

type definition struct {
	super   lexer.Definition
	key     lexer.TokenType
	indent  lexer.TokenType
	dedent  lexer.TokenType
	symbols map[string]lexer.TokenType
}

func (d *definition) Symbols() map[string]lexer.TokenType {
	return d.symbols
}

func (d *definition) Lex(filename string, r io.Reader) (lexer.Lexer, error) {
	super, err := d.super.Lex(filename, r)
	if err != nil {
		return nil, err
	}
	return &indentLexer{def: d, super: super}, nil
}

type indentLexer struct {
	def    *definition
	super  lexer.Lexer
	stack  []string
	indent string
	buffer []lexer.Token
}

func (l *indentLexer) Next() (lexer.Token, error) {
	if len(l.buffer) > 0 {
		token := l.buffer[len(l.buffer)-1]
		l.buffer = l.buffer[:len(l.buffer)-1]
		return token, nil
	}
	token, err := l.super.Next()
	if err != nil {
		return token, err
	}
	// Not a Whitespace token, return it as-is.
	if token.Type != l.def.key {
		return token, nil
	}
	fmt.Printf("%#v\n", indentRe.FindAllStringIndex(token.Value, -1))
	// Have whitespace
	if len(token.Value) > len(l.indent) {
		l.stack = append(l.stack, l.indent)
		l.indent = token.Value
		token.Value = ""
		token.Type = l.def.indent
		return token, nil
	} else if len(token.Value) < len(l.indent) {
		l.indent = l.stack[len(l.stack)-1]
		l.stack = l.stack[:len(l.stack)-1]
		for len(l.stack) > 0 && strings.HasPrefix(l.stack[len(l.stack)-1], token.Value) {
			l.buffer = append(l.buffer, lexer.Token{
				Type:  l.def.dedent,
				Value: l.indent,
				Pos:   token.Pos,
			})
			l.indent = l.stack[len(l.stack)-1]
			l.stack = l.stack[:len(l.stack)-1]
		}
		token.Value = ""
		token.Type = l.def.Symbols()["Dedent"]
		return token, nil
	}
	// No indent or dedent, skip the token.
	return l.Next()
}
