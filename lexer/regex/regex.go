// Package regex provides a regex based lexer using a readable list of named patterns.
//
// eg.
//
//     Ident = [[:ascii:]][\w\d]*
//     Whitespace = \s+
package regex

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/participle/lexer"
)

var eolBytes = []byte("\n")

// New creates a regex lexer from a readable list of named patterns.
//
// This accepts a grammar where each line is a named regular expression in the form:
//
//     # <comment>
//     <name>  = <regexp>
//
// eg.
//
//     Ident = [[:ascii:]][\w\d]*
//     Whitespace = \s+
//
// Order is relevant. Comments may only occur at the beginning of a line. The regular
// expression will have surrounding whitespace trimmed before being parsed. Lower-case
// rules are ignored.
func New(grammar string) (lexer.Definition, error) {
	rules := []reRule{}
	symbols := map[string]rune{
		"EOF": lexer.EOF,
	}
	lines := strings.Split(grammar, "\n")
	i := 0
	for _, rule := range lines {
		rule = strings.TrimSpace(rule)
		if rule == "" || strings.HasPrefix(rule, "#") {
			continue
		}

		parts := strings.SplitN(rule, "=", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("rule should be in the form <Name> = <regex>, not %q", rule)
		}

		name := strings.TrimSpace(parts[0])
		pattern := "^(?:" + strings.TrimSpace(parts[1]) + ")"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex in rule %q: %s", name, err)
		}

		symbols[name] = lexer.EOF - 1 - rune(i)
		i++
		rules = append(rules, reRule{
			name:   name,
			ignore: unicode.IsLower(rune(name[0])),
			re:     re,
		})
	}

	return &reDefinition{
		symbols: symbols,
		rules:   rules,
	}, nil
}

type reRule struct {
	name   string
	ignore bool
	re     *regexp.Regexp
}

type reDefinition struct {
	symbols map[string]rune
	rules   []reRule
}

func (d *reDefinition) Lex(r io.Reader) (lexer.Lexer, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &reLexer{
		rules:   d.rules,
		symbols: d.symbols,
		data:    data,
		pos: lexer.Position{
			Filename: lexer.NameOfReader(r),
			Line:     1,
			Column:   1,
		},
	}, nil
}

func (d *reDefinition) Symbols() map[string]rune { return d.symbols }

type reLexer struct {
	pos     lexer.Position
	rules   []reRule
	symbols map[string]rune
	data    []byte
}

var _ lexer.Lexer = &reLexer{}

func (r *reLexer) Next() (lexer.Token, error) {
	for len(r.data) > 0 {
		var match []int
		var rule *reRule
		for _, re := range r.rules {
			match = re.re.FindIndex(r.data)
			if match != nil {
				rule = &re // nolint: scopelint
				break
			}
		}
		if rule == nil || match == nil {
			rn, _ := utf8.DecodeRune(r.data)
			return lexer.Token{}, lexer.Errorf(r.pos, "invalid token %q", rn)
		}

		span := r.data[match[0]:match[1]]
		r.data = r.data[match[1]:]

		// Update position.
		pos := r.pos
		r.pos.Offset += match[1]
		lines := bytes.Count(span, eolBytes)
		r.pos.Line += lines
		// Update column.
		if lines == 0 {
			r.pos.Column += utf8.RuneCount(span)
		} else {
			r.pos.Column = utf8.RuneCount(span[bytes.LastIndex(span, eolBytes):])
		}
		if rule.ignore {
			continue
		}
		return lexer.Token{
			Type:  r.symbols[rule.name],
			Value: string(span),
			Pos:   pos,
		}, nil
	}
	return lexer.EOFToken(r.pos), nil
}
