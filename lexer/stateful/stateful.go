// Package stateful defines a nested stateful lexer.
//
// This lexer is based heavily on the approach used by Chroma (and Pygments).
//
// The lexer is a state machine defined by a map of rules keyed by state. Each rule
// is a named regex and optional operation to apply when the rule matches.
//
// Lexing starts in the "Root" group. Each rule is matched in order, with the first
// successful match producing a lexeme. If the matching rule has an associated Mutator
// it will be executed. The name of each rule is prefixed with the name of its group
// to yield the token identifier used during matching.
//
// A state change can be introduced with the Mutator `Push(state)`. `Pop()` will
// return to the previous state.
//
// To reuse rules from another state, use `Include(state)`.
//
// As a special case, regexes containing backrefs in the form \N (where N is a digit)
// will match the corresponding capture group from the immediate parent group. This
// can be used to parse, among other things, heredocs.
//
// See the example and tests in this package for details.
package stateful

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var (
	eolBytes       = []byte("\n")
	backrefReplace = regexp.MustCompile(`\\(\d)`)
)

// A Rule matching input and possibly changing state.
type Rule struct {
	Name    string
	Pattern string
	Mutator Mutator
}

// Rules grouped by name.
type Rules map[string][]Rule

// compiledRule is a Rule with its pattern compiled.
type compiledRule struct {
	Rule
	RE *regexp.Regexp
}

// compiledRules grouped by name.
type compiledRules map[string][]compiledRule

// A Mutator mutates the state of the Lexer
type Mutator interface {
	mutateLexer(lexer *Lexer, groups []string) error
}

// RulesMutator is an optional interface that Mutators can implement.
//
// It is applied during rule construction to mutate the rule map.
type RulesMutator interface {
	mutateRules(state string, rule int, rules compiledRules) error
}

// MutatorFunc is a function that is also a Mutator.
type MutatorFunc func(*Lexer, []string) error

func (m MutatorFunc) mutateLexer(lexer *Lexer, groups []string) error { return m(lexer, groups) } // nolint: golint

// Pop to the previous state.
func Pop() Mutator {
	return MutatorFunc(func(lexer *Lexer, groups []string) error {
		lexer.stack = lexer.stack[:len(lexer.stack)-1]
		return nil
	})
}

// Push to the given state.
//
// The target state will then be the set of rules used for matching
// until another Push or Pop is encountered.
func Push(state string) Mutator {
	return MutatorFunc(func(lexer *Lexer, groups []string) error {
		lexer.stack = append(lexer.stack, lexerState{name: state, groups: groups})
		return nil
	})
}

type include struct{ state string }

func (i include) mutateLexer(lexer *Lexer, groups []string) error { panic("should not be called") }

func (i include) mutateRules(state string, rule int, rules compiledRules) error {
	includedRules, ok := rules[i.state]
	if !ok {
		return fmt.Errorf("invalid include state %q", i.state)
	}
	clone := make([]compiledRule, len(includedRules))
	copy(clone, includedRules)
	rules[state] = append(rules[state][:rule], append(clone, rules[state][rule+1:]...)...)
	return nil
}

// Include rules from another state in this one.
func Include(state string) Rule {
	return Rule{Mutator: include{state}}
}

// Definition is the lexer.Definition.
type Definition struct {
	rules   compiledRules
	symbols map[string]rune
}

// New constructs a new stateful lexer from rules.
func New(rules Rules) (*Definition, error) {
	compiled := compiledRules{}
	for key, set := range rules {
		for i, rule := range set {
			pattern := "^(?:" + rule.Pattern + ")"
			var (
				re  *regexp.Regexp
				err error
			)
			if backrefReplace.FindString(rule.Pattern) == "" {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("%s.%d: %s", key, i, err)
				}
			}
			compiled[key] = append(compiled[key], compiledRule{
				Rule: rule,
				RE:   re,
			})
		}
	}
restart:
	for state, rules := range compiled {
		for i, rule := range rules {
			if mutator, ok := rule.Mutator.(RulesMutator); ok {
				if err := mutator.mutateRules(state, i, compiled); err != nil {
					return nil, fmt.Errorf("%s.%d: %s", state, i, err)
				}
				goto restart
			}
		}
	}
	keys := make([]string, 0, len(compiled))
	for key := range compiled {
		keys = append(keys, key)
	}
	symbols := map[string]rune{
		"EOF": lexer.EOF,
	}
	sort.Strings(keys)
	rn := lexer.EOF - 1
	for _, key := range keys {
		for i, rule := range compiled[key] {
			rule.Name = key + rule.Name
			compiled[key][i] = rule
			if _, ok := symbols[rule.Name]; ok {
				panic("duplicate key " + rule.Name)
			}
			symbols[rule.Name] = rn
			rn--
		}
	}
	return &Definition{rules: compiled, symbols: symbols}, nil
}

func (d *Definition) Lex(r io.Reader) (lexer.Lexer, error) { // nolint: golint
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &Lexer{
		def:   d,
		data:  data,
		stack: []lexerState{{name: "Root"}},
		pos: lexer.Position{
			Filename: lexer.NameOfReader(r),
			Line:     1,
			Column:   1,
		},
	}, nil
}

func (d *Definition) Symbols() map[string]rune { // nolint: golint
	return d.symbols
}

type lexerState struct {
	name   string
	groups []string
}

// Lexer implementation.
type Lexer struct {
	stack []lexerState
	def   *Definition
	data  []byte
	pos   lexer.Position
}

func (l *Lexer) Next() (lexer.Token, error) { // nolint: golint
	parent := l.stack[len(l.stack)-1]
	rules := l.def.rules[parent.name]
	for len(l.data) > 0 {
		var (
			rule  *compiledRule
			match []int
		)
		for _, candidate := range rules {
			re, err := l.getPattern(candidate)
			if err != nil {
				return lexer.Token{}, participle.Wrapf(l.pos, err, "rule %q", candidate.Name)
			}
			match = re.FindSubmatchIndex(l.data)
			if match != nil {
				rule = &candidate // nolint: scopelint
				if match[0] == 0 && match[1] == 0 {
					return lexer.Token{}, fmt.Errorf("rule %q matched, but did not consume any input", rule.Name)
				}
				break
			}
		}
		if match == nil || rule == nil {
			return lexer.Token{}, participle.Errorf(l.pos, "no match")
		}

		if rule.Mutator != nil {
			groups := make([]string, 0, len(match)/2)
			for i := 0; i < len(match); i += 2 {
				groups = append(groups, string(l.data[match[i]:match[i+1]]))
			}
			if err := rule.Mutator.mutateLexer(l, groups); err != nil {
				return lexer.Token{}, participle.Errorf(l.pos, "rule %q mutator returned an error", rule.Name)
			}
		}

		span := l.data[match[0]:match[1]]
		l.data = l.data[match[1]:]
		// l.groups = groups

		// Update position.
		pos := l.pos
		l.pos.Offset += match[1]
		lines := bytes.Count(span, eolBytes)
		l.pos.Line += lines
		// Update column.
		if lines == 0 {
			l.pos.Column += utf8.RuneCount(span)
		} else {
			l.pos.Column = utf8.RuneCount(span[bytes.LastIndex(span, eolBytes):])
		}
		return lexer.Token{
			Type:  l.def.symbols[rule.Name],
			Value: string(span),
			Pos:   pos,
		}, nil
	}
	return lexer.EOFToken(l.pos), nil
}

func (l *Lexer) getPattern(candidate compiledRule) (*regexp.Regexp, error) {
	if candidate.RE != nil {
		return candidate.RE, nil
	}

	// We don't have a compiled RE. This means there are back-references
	// that need to be substituted first.
	// TODO: Cache?
	parent := l.stack[len(l.stack)-1]
	var (
		re  *regexp.Regexp
		err error
	)
	pattern := backrefReplace.ReplaceAllStringFunc(candidate.Pattern, func(s string) string {
		n, nerr := strconv.ParseInt(s[1:], 10, 64)
		if nerr != nil {
			err = nerr
			return s
		}
		if len(parent.groups) == 0 || int(n) >= len(parent.groups) {
			err = fmt.Errorf("invalid group %d from parent with %d groups", n, len(parent.groups))
			return s
		}
		return regexp.QuoteMeta(parent.groups[n])
	})
	if err == nil {
		re, err = regexp.Compile("^(?:" + pattern + ")")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid backref expansion: %q: %s", pattern, err)
	}
	return re, nil
}
