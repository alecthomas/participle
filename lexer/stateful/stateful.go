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
// See the example and tests in this package for details.
package stateful

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var eolBytes = []byte("\n")

// A Rule matching input and possibly changing state.
type Rule struct {
	Name    string
	Pattern string
	Mutator Mutator
}

// Rules grouped by name.
type Rules map[string][]Rule

// CompiledRule is a Rule with its pattern compiled.
type CompiledRule struct {
	Rule
	RE *regexp.Regexp
}

// CompiledRules grouped by name.
type CompiledRules map[string][]CompiledRule

// A Mutator mutates the state of the Lexer
type Mutator interface {
	MutateLexer(lexer *Lexer) error
}

// RulesMutator is an optional interface that Mutators can implement.
//
// It is applied during rule construction to mutate the rule map.
type RulesMutator interface {
	MutateRules(state string, rule int, rules CompiledRules) error
}

// MutatorFunc is a function that is also a Mutator.
type MutatorFunc func(*Lexer) error

func (m MutatorFunc) MutateLexer(lexer *Lexer) error { return m(lexer) } // nolint: golint

// Pop to the previous state.
func Pop() Mutator {
	return MutatorFunc(func(lexer *Lexer) error {
		lexer.stack = lexer.stack[:len(lexer.stack)-1]
		return nil
	})
}

// Push to the given state.
//
// The target state will then be the set of rules used for matching
// until another Push or Pop is encountered.
func Push(state string) Mutator {
	return MutatorFunc(func(lexer *Lexer) error {
		lexer.stack = append(lexer.stack, state)
		return nil
	})
}

type include struct{ state string }

func (i include) MutateLexer(lexer *Lexer) error { panic("should not be called") }

func (i include) MutateRules(state string, rule int, rules CompiledRules) error {
	includedRules, ok := rules[i.state]
	if !ok {
		return fmt.Errorf("invalid include state %q", i.state)
	}
	clone := make([]CompiledRule, len(includedRules))
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
	rules   CompiledRules
	symbols map[string]rune
}

// New constructs a new stateful lexer from rules.
func New(rules Rules) (*Definition, error) {
	compiled := CompiledRules{}
	for key, set := range rules {
		for i, rule := range set {
			pattern := "^(?:" + rule.Pattern + ")"
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("%s.%d: %s", key, i, err)
			}
			compiled[key] = append(compiled[key], CompiledRule{rule, re})
		}
	}
restart:
	for state, rules := range compiled {
		for i, rule := range rules {
			if mutator, ok := rule.Mutator.(RulesMutator); ok {
				if err := mutator.MutateRules(state, i, compiled); err != nil {
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
		stack: []string{"Root"},
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

// Lexer implementation.
type Lexer struct {
	stack []string
	def   *Definition
	data  []byte
	pos   lexer.Position
}

func (l *Lexer) Next() (lexer.Token, error) { // nolint: golint
	ruleKey := l.stack[len(l.stack)-1]
	rules := l.def.rules[ruleKey]
	for len(l.data) > 0 {
		var match []int
		var rule *CompiledRule
		for _, re := range rules {
			match = re.RE.FindIndex(l.data)
			if match != nil {
				rule = &re // nolint: scopelint
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
			if err := rule.Mutator.MutateLexer(l); err != nil {
				return lexer.Token{}, participle.Errorf(l.pos, "rule %q mutator returned an error", rule.Name)
			}
		}

		span := l.data[match[0]:match[1]]
		l.data = l.data[match[1]:]

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
