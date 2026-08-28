// Command indentation demonstrates how to parse indentation-based
// (Python-like) grammars with Participle by supplying a custom lexer that
// turns indentation into first-class INDENT/DEDENT tokens.
//
// The lexer below is a minimal Definition: it reads all input up-front,
// measures the leading whitespace of each non-blank line, and emits
// INDENT/DEDENT tokens whenever the indentation depth changes. The grammar
// then refers to those tokens just like any other named token.
package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Block is the top-level grammar: a list of statements.
type Block struct {
	Statements []*Statement `@@*`
}

// Statement is either an "if" block, or a simple command.
type Statement struct {
	If   *If      `@@`
	Line *Command `| @@`
}

// If is a conditional block. Its condition and body live in the struct's own
// grammar; the exported field consumes the trailing DEDENT without capturing
// it (unexported fields, including "_", are ignored by participle).
type If struct {
	Condition  string       `"if" @Ident ":" EOL INDENT`
	Statements []*Statement `@@*`
	BlockEnd   struct{}     `DEDENT`
}

// Command is a simple statement: one identifier followed by zero or more
// identifier arguments, terminated by a newline.
type Command struct {
	Name    string   `@Ident`
	Args    []string `@Ident*`
	LineEnd struct{} `EOL`
}

// Token types for the custom lexer. They must not collide with lexer.EOF
// (-1) or with each other; they only need to match the symbols exposed via
// Symbols() below.
const (
	tokIdent  = lexer.TokenType(-2)
	tokEOL    = lexer.TokenType(-3)
	tokIndent = lexer.TokenType(-4)
	tokDedent = lexer.TokenType(-5)
)

var tokenRe = regexp.MustCompile(`\w+|:`)

type indentationDefinition struct{}

func (indentationDefinition) Symbols() map[string]lexer.TokenType {
	return map[string]lexer.TokenType{
		"EOF":    lexer.EOF,
		"Ident":  tokIdent,
		"EOL":    tokEOL,
		"INDENT": tokIndent,
		"DEDENT": tokDedent,
	}
}

func (indentationDefinition) Lex(filename string, r io.Reader) (lexer.Lexer, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return lexIndentation(filename, string(data))
}

type indentationLexer struct {
	filename string
	tokens   []lexer.Token
	index    int
}

// lexIndentation converts input into a flat token stream, inserting
// INDENT/DEDENT tokens between lines based on their leading whitespace.
func lexIndentation(filename, input string) (*indentationLexer, error) {
	l := &indentationLexer{filename: filename}
	indents := []int{0}
	for lineNo, raw := range strings.Split(input, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		for indent < indents[len(indents)-1] {
			indents = indents[:len(indents)-1]
			l.emit(tokDedent, "", lineNo+1, indent)
		}
		if indent > indents[len(indents)-1] {
			indents = append(indents, indent)
			l.emit(tokIndent, "", lineNo+1, indent)
		}

		rest := line[indent:]
		col := 0
		for rest != "" {
			m := tokenRe.FindStringIndex(rest)
			if m == nil {
				break
			}
			tok := rest[m[0]:m[1]]
			rest = rest[m[1]:]
			col += m[0]
			if tok == ":" {
				l.emit(lexer.TokenType(':'), ":", lineNo+1, indent+col)
			} else {
				l.emit(tokIdent, tok, lineNo+1, indent+col)
			}
			col += len(tok)
		}
		l.emit(tokEOL, "\n", lineNo+1, len(line))
	}
	return l, nil
}

func (l *indentationLexer) emit(t lexer.TokenType, value string, line, column int) {
	l.tokens = append(l.tokens, lexer.Token{
		Type:  t,
		Value: value,
		Pos:   lexer.Position{Filename: l.filename, Line: line, Column: column},
	})
}

func (l *indentationLexer) Next() (lexer.Token, error) {
	if l.index >= len(l.tokens) {
		return lexer.Token{Type: lexer.EOF}, nil
	}
	tok := l.tokens[l.index]
	l.index++
	return tok, nil
}

func main() {
	parser, err := participle.Build[Block](participle.Lexer(indentationDefinition{}))
	if err != nil {
		panic(err)
	}
	ast, err := parser.ParseString("", `if x:
    print hello
    if y:
        print deep
    print done
print outside`)
	if err != nil {
		panic(err)
	}
	fmt.Printf("parsed %d statement(s)\n", len(ast.Statements))
	dump(ast, 0)
}

func dump(n any, depth int) {
	switch v := n.(type) {
	case *Block:
		for _, s := range v.Statements {
			dump(s, depth)
		}
	case *Statement:
		if v.If != nil {
			fmt.Printf("%*sif:\n", depth*4, "")
			dump(v.If, depth+1)
		}
		if v.Line != nil {
			fmt.Printf("%*s%s %s\n", depth*4, "", v.Line.Name, strings.Join(v.Line.Args, " "))
		}
	case *If:
		for _, s := range v.Statements {
			dump(s, depth)
		}
	}
}
