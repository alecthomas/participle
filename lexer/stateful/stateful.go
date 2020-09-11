// Package stateful defines a nested stateful lexer.
//
// This lexer is based heavily on the approach used by Chroma (and Pygments).
//
// The lexer is a state machine defined by a map of rules keyed by state. Each rule
// is a named regex and optional operation to apply when the rule matches.
//
// As a convenience, any Rule starting with a lowercase letter will be elided from output.
//
// Lexing starts in the "Root" group. Each rule is matched in order, with the first
// successful match producing a lexeme. If the matching rule has an associated Action
// it will be executed. The name of each non-root rule is prefixed with the name
// of its group to yield the token identifier used during matching.
//
// A state change can be introduced with the Action `Push(state)`. `Pop()` will
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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var (
	eolBytes       = []byte("\n")
	backrefReplace = regexp.MustCompile(`(\\+)(\d)`)
)

// A Rule matching input and possibly changing state.
type Rule struct {
	Name    string
	Pattern string
	Action  Action
}

// Rules grouped by name.
type Rules map[string][]Rule

// compiledRule is a Rule with its pattern compiled.
type compiledRule struct {
	Rule
	ignore bool
	RE     *regexp.Regexp
}

// compiledRules grouped by name.
type compiledRules map[string][]compiledRule

// A Action is applied when a rule matches.
type Action interface {
	// Actions are responsible for validating the match. ie. if they consumed any input.
	applyAction(lexer *Lexer, groups []string) error
}

// RulesAction is an optional interface that Actions can implement.
//
// It is applied during rule construction to mutate the rule map.
type RulesAction interface {
	applyRules(state string, rule int, rules compiledRules) error
}

// ActionFunc is a function that is also a Action.
type ActionFunc func(*Lexer, []string) error

func (m ActionFunc) applyAction(lexer *Lexer, groups []string) error { return m(lexer, groups) } // nolint: golint

// Pop to the previous state.
func Pop() Action {
	return ActionFunc(func(lexer *Lexer, groups []string) error {
		if groups[0] == "" {
			return errors.New("did not consume any input")
		}
		lexer.stack = lexer.stack[:len(lexer.stack)-1]
		return nil
	})
}

var returnToParent = Rule{"popIfEmpty", "", nil}

// Return to the parent state.
//
// Useful as the last rule in a sub-state.
func Return() Rule { return returnToParent }

// Push to the given state.
//
// The target state will then be the set of rules used for matching
// until another Push or Pop is encountered.
func Push(state string) Action {
	return ActionFunc(func(lexer *Lexer, groups []string) error {
		if groups[0] == "" {
			return errors.New("did not consume any input")
		}
		lexer.stack = append(lexer.stack, lexerState{name: state, groups: groups})
		return nil
	})
}

type include struct{ state string }

func (i include) applyAction(lexer *Lexer, groups []string) error { panic("should not be called") }

func (i include) applyRules(state string, rule int, rules compiledRules) error {
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
	return Rule{Action: include{state}}
}

// Definition is the lexer.Definition.
type Definition struct {
	rules   compiledRules
	symbols map[string]rune
	// Map of key->*regexp.Regexp
	backrefCache sync.Map
}

// NewSimple creates a new stateful lexer with a single "Root" state.
func NewSimple(rules []Rule) (*Definition, error) {
	return New(Rules{"Root": rules})
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
			var match = backrefReplace.FindStringSubmatch(rule.Pattern)
			if match == nil || len(match[1])%2 == 0 {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("%s.%d: %s", key, i, err)
				}
			}
			compiled[key] = append(compiled[key], compiledRule{
				Rule:   rule,
				ignore: len(rule.Name) > 0 && unicode.IsLower(rune(rule.Name[0])),
				RE:     re,
			})
		}
	}
restart:
	for state, rules := range compiled {
		for i, rule := range rules {
			if action, ok := rule.Action.(RulesAction); ok {
				if err := action.applyRules(state, i, compiled); err != nil {
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
	duplicates := map[string]compiledRule{}
	rn := lexer.EOF - 1
	for _, key := range keys {
		for i, rule := range compiled[key] {
			if dup, ok := duplicates[rule.Name]; ok && rule.Pattern != dup.Pattern {
				panic(fmt.Sprintf("duplicate key %q with different patterns %q != %q", rule.Name, rule.Pattern, dup.Pattern))
			}
			duplicates[rule.Name] = rule
			compiled[key][i] = rule
			symbols[rule.Name] = rn
			rn--
		}
	}
	return &Definition{
		rules:   compiled,
		symbols: symbols,
	}, nil
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
next:
	for len(l.data) > 0 {
		var (
			rule  *compiledRule
			match []int
		)
		for _, candidate := range rules {
			// Special case "Return()".
			if candidate.Rule == returnToParent {
				l.stack = l.stack[:len(l.stack)-1]
				parent = l.stack[len(l.stack)-1]
				rules = l.def.rules[parent.name]
				continue next
			}
			re, err := l.getPattern(candidate)
			if err != nil {
				return lexer.Token{}, participle.Wrapf(l.pos, err, "rule %q", candidate.Name)
			}
			match = re.FindSubmatchIndex(l.data)
			if match != nil {
				rule = &candidate // nolint: scopelint
				break
			}
		}
		if match == nil || rule == nil {
			sample := ""
			if len(l.data) < 16 {
				sample = string(l.data)
			} else {
				sample = string(l.data[:16]) + "..."
			}
			return lexer.Token{}, participle.Errorf(l.pos, "no lexer rules in state %q matched input text %q", parent.name, sample)
		}

		if rule.Action != nil {
			groups := make([]string, 0, len(match)/2)
			for i := 0; i < len(match); i += 2 {
				groups = append(groups, string(l.data[match[i]:match[i+1]]))
			}
			if err := rule.Action.applyAction(l, groups); err != nil {
				return lexer.Token{}, participle.Errorf(l.pos, "rule %q: %s", rule.Name, err)
			}
		} else if match[0] == match[1] {
			return lexer.Token{}, participle.Errorf(l.pos, "rule %q did not match any input", rule.Name)
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
		if rule.ignore {
			parent = l.stack[len(l.stack)-1]
			rules = l.def.rules[parent.name]
			continue
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
	parent := l.stack[len(l.stack)-1]
	key := candidate.Pattern + "\000" + strings.Join(parent.groups, "\000")
	cached, ok := l.def.backrefCache.Load(key)
	if ok {
		return cached.(*regexp.Regexp), nil
	}

	var (
		re  *regexp.Regexp
		err error
	)
	pattern := backrefReplace.ReplaceAllStringFunc(candidate.Pattern, func(s string) string {
		var rematch = backrefReplace.FindStringSubmatch(s)
		n, nerr := strconv.ParseInt(rematch[2], 10, 64)
		if nerr != nil {
			err = nerr
			return s
		}
		if len(parent.groups) == 0 || int(n) >= len(parent.groups) {
			err = fmt.Errorf("invalid group %d from parent with %d groups", n, len(parent.groups))
			return s
		}
		// concatenate the leading \\\\ which are already escaped to the quoted match.
		return rematch[1][:len(rematch[1])-1] + regexp.QuoteMeta(parent.groups[n])
	})
	if err == nil {
		re, err = regexp.Compile("^(?:" + pattern + ")")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid backref expansion: %q: %s", pattern, err)
	}
	l.def.backrefCache.Store(key, re)
	return re, nil
}
