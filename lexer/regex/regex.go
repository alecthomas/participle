// Package regex provides a regex based lexer using a readable list of named patterns.
//
// eg.
//
//     Ident = [[:ascii:]][\w\d]*
//     Whitespace = \s+
package regex

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/stateful"
)

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
	rules := []stateful.Rule{}
	lines := strings.Split(grammar, "\n")
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
		rules = append(rules, stateful.Rule{
			Name:    name,
			Pattern: pattern,
		})
	}

	return stateful.New(stateful.Rules{"Root": rules})
}
