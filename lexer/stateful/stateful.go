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

// compiledRuleGroup is a series of rules who have been grouped together to form a lexer state.
// They are analyzed so that for a given input, the rule group may only test the most likely
// regexps that may match the input.
type compiledRuleGroup struct {
	rules []compiledRule

	// These two variables contain information about which rules may be tested on a given rune.
	// The runemap holds a map associating a rune to an array of bool the size of the rule count,
	// that indicate whether the rule should be checked or not.
	// The fallbackMap holds a single []bool array that holds true for the rules that are too
	// big to put in the runeMap (like '.') and that are tested all the time.
	runeMap     map[rune][]bool
	fallbackMap []bool
}

func (group *compiledRuleGroup) process() error {
	var (
		rules    = group.rules
		nbrules  = len(rules)
		computed = make([]*computedRuneRange, nbrules)
		fallback = make([]bool, nbrules)
		runemap  = make(map[rune][]bool)
	)
	group.runeMap = runemap
	group.fallbackMap = fallback

	// Analyze the patterns to populate runeMap and fallbackMap
	for i, rule := range rules {
		if rule.RE == nil || rule.Pattern == "" {
			fallback[i] = true
			continue // ignore those
		}
		comp, err := computeRuneRanges(rule.Pattern)
		if err != nil {
			return err // FIXME better error handling ?
		}
		computed[i] = comp
		if comp.size >= charclassSizeLimit {
			// this one is too big to have their own map
			fallback[i] = true
			continue
		}
	}

	// Now process the rules and add them to their respective buckets.
	for idrule, comp := range computed {
		if comp == nil || fallback[idrule] {
			continue
		}
		var runes = comp.runes
		for i, l := 0, len(runes); i < l; i += 2 {
			for r, end := runes[i], runes[i+1]; r <= end; r++ {
				var (
					bools []bool
					ok    bool
				)
				if bools, ok = runemap[r]; !ok {
					bools = make([]bool, nbrules)
					runemap[r] = bools
					// when creating a new bucket, add all the fallback rules onto it if they could potentially
					for j, isFallback := range fallback {
						if isFallback {
							var comp = computed[j]
							// add all the fallback rules that are either not yet defined (or "")
							// or those that we know will never match this rune.
							bools[j] = comp == nil || !runeIsInRange(r, comp)
						}
					}
				}

				bools[idrule] = true // add it onto the bucket.
			}
		}
	}

	return nil
}

func runeIsInRange(r rune, comp *computedRuneRange) bool {
	if comp == nil {
		return true // all the unknown rules are added.
	}
	var rng = comp.runes
	for i, l := 0, len(rng); i < l; i += 2 {
		if r >= rng[i] && r <= rng[i+1] {
			return true
		}
	}
	return false
}

// Tries to match to input. Returns nil, nil if no match was found.
func (group *compiledRuleGroup) tryMatch(l *Lexer) (*compiledRule, []int, error) {
	// Our goal is to find the indices of the rules to test
	var (
		indices []bool
		ok      bool
		rules   = group.rules
		re      *regexp.Regexp
		err     error
	)

	// first, get the first rune
	r, _ := utf8.DecodeRune(l.data)
	if r == utf8.RuneError {
		return nil, nil, fmt.Errorf(`invalid utf-8 input sequence`) // FIXME this should be an error, how do we report it ?
	}

	if indices, ok = group.runeMap[r]; !ok {
		indices = group.fallbackMap // so we use the fallback to test the rest.
	}

	for i, test := range indices {
		if !test {
			continue
		}

		var rule = &rules[i]

		// FIXME: what about one char rules ?

		if rule.Rule == ReturnRule {
			return rule, []int{}, nil
		}

		if re, err = l.getPattern(rule); err != nil {
			return rule, nil, err // FIXME: forward it as-is ?
		}

		var match = re.FindSubmatchIndex(l.data)
		if match != nil {
			// found it !
			return rule, match, nil
		}
	}

	return nil, nil, nil // we found diddily squat.
}

// compiledRules grouped by name.
type compiledRules map[string]*compiledRuleGroup

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

// ActionPop pops to the previous state when the Rule matches.
type ActionPop struct{}

func (p ActionPop) applyAction(lexer *Lexer, groups []string) error {
	if groups[0] == "" {
		return errors.New("did not consume any input")
	}
	lexer.stack = lexer.stack[:len(lexer.stack)-1]
	return nil
}

// Pop to the previous state.
func Pop() Action {
	return ActionPop{}
}

// ReturnRule signals the lexer to return immediately.
var ReturnRule = Rule{"returnToParent", "", nil}

// Return to the parent state.
//
// Useful as the last rule in a sub-state.
func Return() Rule { return ReturnRule }

// ActionPush pushes the current state and switches to "State" when the Rule matches.
type ActionPush struct{ State string }

func (p ActionPush) applyAction(lexer *Lexer, groups []string) error {
	if groups[0] == "" {
		return errors.New("did not consume any input")
	}
	lexer.stack = append(lexer.stack, lexerState{name: p.State, groups: groups})
	return nil
}

// Push to the given state.
//
// The target state will then be the set of rules used for matching
// until another Push or Pop is encountered.
func Push(state string) Action {
	return ActionPush{state}
}

type include struct{ state string }

func (i include) applyAction(lexer *Lexer, groups []string) error { panic("should not be called") }

func (i include) applyRules(state string, rule int, groups compiledRules) error {
	includedRules, ok := groups[i.state]
	if !ok {
		return fmt.Errorf("invalid include state %q", i.state)
	}
	clone := make([]compiledRule, len(includedRules.rules))
	copy(clone, includedRules.rules)
	groups[state].rules = append(groups[state].rules[:rule], append(clone, groups[state].rules[rule+1:]...)...)
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

// MustSimple creates a new lexer definition based on a single state described by `rules`.
// panics if the rules trigger an error
func MustSimple(rules []Rule) *Definition {
	def, err := NewSimple(rules)
	if err != nil {
		panic(err)
	}
	return def
}

// NewSimple creates a new stateful lexer with a single "Root" state.
func NewSimple(rules []Rule) (*Definition, error) {
	return New(Rules{"Root": rules})
}

// Must creates a new stateful lexer and panics if it is incorrect.
func Must(rules Rules) *Definition {
	def, err := New(rules)
	if err != nil {
		panic(err)
	}
	return def
}

// New constructs a new stateful lexer from rules.
func New(rules Rules) (*Definition, error) {
	compiled := compiledRules{}
	for key, set := range rules {
		if _, ok := compiled[key]; !ok {
			compiled[key] = &compiledRuleGroup{}
		}

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

			compiled[key].rules = append(compiled[key].rules, compiledRule{
				Rule:   rule,
				ignore: len(rule.Name) > 0 && unicode.IsLower(rune(rule.Name[0])),
				RE:     re,
			})
		}
	}
restart:
	for state, group := range compiled {
		for i, rule := range group.rules {
			if action, ok := rule.Action.(RulesAction); ok {
				if err := action.applyRules(state, i, compiled); err != nil {
					return nil, fmt.Errorf("%s.%d: %s", state, i, err)
				}
				goto restart
			}
		}
	}

	// lookup the keys of the rules
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
		for i, rule := range compiled[key].rules {
			if dup, ok := duplicates[rule.Name]; ok && rule.Pattern != dup.Pattern {
				panic(fmt.Sprintf("duplicate key %q with different patterns %q != %q", rule.Name, rule.Pattern, dup.Pattern))
			}
			duplicates[rule.Name] = rule
			compiled[key].rules[i] = rule
			symbols[rule.Name] = rn
			rn--
		}
	}

	// Now that everything was processed, process the rules
	for _, group := range compiled {
		if err := group.process(); err != nil {
			return nil, err
		}
	}

	return &Definition{
		rules:   compiled,
		symbols: symbols,
	}, nil
}

// Rules returns the user-provided Rules used to construct the lexer.
func (d *Definition) Rules() Rules {
	out := Rules{}
	for state, rules := range d.rules {
		for _, rule := range rules.rules {
			out[state] = append(out[state], rule.Rule)
		}
	}
	return out
}

func (d *Definition) LexBytes(filename string, data []byte) (lexer.Lexer, error) { // nolint: golint
	return &Lexer{
		def:   d,
		data:  data,
		stack: []lexerState{{name: "Root"}},
		pos: lexer.Position{
			Filename: filename,
			Line:     1,
			Column:   1,
		},
	}, nil
}

type zeroCopyWriter struct{ b []byte }

func (z *zeroCopyWriter) Write(p []byte) (n int, err error) {
	z.b = p
	return len(p), nil
}

func (d *Definition) Lex(filename string, r io.Reader) (lexer.Lexer, error) { // nolint: golint
	var (
		data []byte
		err  error
	)
	switch r := r.(type) {
	case *bytes.Reader:
		w := &zeroCopyWriter{}
		_, err = r.WriteTo(w)
		data = w.b

	case *bytes.Buffer:
		data = r.Bytes()

	default:
		data, err = ioutil.ReadAll(r)
	}
	if err != nil {
		return nil, err
	}
	return d.LexBytes(filename, data)
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
			err   error
		)

		if rule, match, err = rules.tryMatch(l); err != nil {
			if rule != nil {
				return lexer.Token{}, participle.Wrapf(l.pos, err, `rule %q`, rule.Name)
			}
			return lexer.Token{}, err // FIXME
		}

		if rule != nil && rule.Rule == ReturnRule {
			l.stack = l.stack[:len(l.stack)-1]
			parent = l.stack[len(l.stack)-1]
			rules = l.def.rules[parent.name]
			continue
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

// getPattern returns a compiled regexp for a given rule.
// Its purpose is to compile and cache on the fly patterns corresponding
// to backreferences this those are not compiled at the beginning.
func (l *Lexer) getPattern(candidate *compiledRule) (*regexp.Regexp, error) {
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
