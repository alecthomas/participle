package lexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

var (
	backrefReplace = regexp.MustCompile(`(\\+)(\d)`)
)

// A Rule matching input and possibly changing state.
type Rule struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Action  Action `json:"action"`
}

var _ json.Marshaler = &Rule{}
var _ json.Unmarshaler = &Rule{}

type jsonRule struct {
	Name    string          `json:"name,omitempty"`
	Pattern string          `json:"pattern,omitempty"`
	Action  json.RawMessage `json:"action,omitempty"`
}

func (r *Rule) UnmarshalJSON(data []byte) error {
	jrule := jsonRule{}
	err := json.Unmarshal(data, &jrule)
	if err != nil {
		return err
	}
	r.Name = jrule.Name
	r.Pattern = jrule.Pattern
	jaction := struct {
		Kind string `json:"kind"`
	}{}
	if jrule.Action == nil {
		return nil
	}
	err = json.Unmarshal(jrule.Action, &jaction)
	if err != nil {
		return fmt.Errorf("lexer: could not unmarshal action %q: %w", string(jrule.Action), err)
	}
	var action Action
	switch jaction.Kind {
	case "push":
		actual := ActionPush{}
		if err := json.Unmarshal(jrule.Action, &actual); err != nil {
			return err
		}
		action = actual
	case "pop":
		actual := ActionPop{}
		if err := json.Unmarshal(jrule.Action, &actual); err != nil {
			return err
		}
		action = actual
	case "include":
		actual := include{}
		if err := json.Unmarshal(jrule.Action, &actual); err != nil {
			return err
		}
		action = actual
	case "":
	default:
		return fmt.Errorf("lexer: unknown action %q", jaction.Kind)
	}
	r.Action = action
	return nil
}

func (r *Rule) MarshalJSON() ([]byte, error) {
	jrule := jsonRule{
		Name:    r.Name,
		Pattern: r.Pattern,
	}
	if r.Action != nil {
		actionData, err := json.Marshal(r.Action)
		if err != nil {
			return nil, fmt.Errorf("lexer: failed to map action: %w", err)
		}
		jaction := map[string]any{}
		err = json.Unmarshal(actionData, &jaction)
		if err != nil {
			return nil, fmt.Errorf("lexer: failed to map action: %w", err)
		}
		switch r.Action.(type) {
		case nil:
		case ActionPop:
			jaction["kind"] = "pop"
		case ActionPush:
			jaction["kind"] = "push"
		case include:
			jaction["kind"] = "include"
		default:
			return nil, fmt.Errorf("lexer: unsupported action %T", r.Action)
		}
		actionJSON, err := json.Marshal(jaction)
		if err != nil {
			return nil, err
		}
		jrule.Action = actionJSON
	}
	return json.Marshal(&jrule)
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

// capture is the text matched by a single regex capture group.
//
// A group that did not participate in the match at all is distinct from one
// that matched the empty string: a backreference to the former cannot match,
// while a backreference to the latter matches the empty string.
type capture struct {
	text    string
	present bool
}

// A Action is applied when a rule matches.
type Action interface {
	// Actions are responsible for validating the match. ie. if they consumed any input.
	applyAction(lexer *StatefulLexer, groups []capture) error
}

// RulesAction is an optional interface that Actions can implement.
//
// It is applied during rule construction to mutate the rule map.
type RulesAction interface {
	applyRules(state string, rule int, rules compiledRules) error
}

type validatingRule interface {
	validate(rules Rules) error
}

// ActionPop pops to the previous state when the Rule matches.
type ActionPop struct{}

func (p ActionPop) applyAction(lexer *StatefulLexer, groups []capture) error {
	if groups[0].text == "" {
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
type ActionPush struct {
	State string `json:"state"`
}

func (p ActionPush) applyAction(lexer *StatefulLexer, groups []capture) error {
	if groups[0].text == "" {
		return errors.New("did not consume any input")
	}
	lexer.stack = append(lexer.stack, lexerState{name: p.State, groups: groups})
	return nil
}

func (p ActionPush) validate(rules Rules) error {
	if _, ok := rules[p.State]; !ok {
		return fmt.Errorf("lexer: push to unknown state %q", p.State)
	}
	return nil
}

// Push to the given state.
//
// The target state will then be the set of rules used for matching
// until another Push or Pop is encountered.
func Push(state string) Action {
	return ActionPush{state}
}

type include struct {
	State string `json:"state"`
}

func (i include) applyAction(_ *StatefulLexer, _ []capture) error {
	panic("should not be called")
}

func (i include) applyRules(state string, rule int, rules compiledRules) error {
	includedRules, ok := rules[i.State]
	if !ok {
		return fmt.Errorf("lexer: invalid include state %q", i.State)
	}
	clone := make([]compiledRule, len(includedRules))
	copy(clone, includedRules)
	rules[state] = append(rules[state][:rule], append(clone, rules[state][rule+1:]...)...) //nolint:makezero // intentional: clone is appended after fixed-size head
	return nil
}

// Include rules from another state in this one.
func Include(state string) Rule {
	return Rule{Action: include{state}}
}

// StatefulDefinition is the lexer.Definition.
type StatefulDefinition struct {
	rules   compiledRules
	symbols map[string]TokenType
	// Map of key->*regexp.Regexp
	backrefCache sync.Map
	matchLongest bool
}

// MustStateful creates a new stateful lexer and panics if it is incorrect.
func MustStateful(rules Rules) *StatefulDefinition {
	def, err := New(rules)
	if err != nil {
		panic(err)
	}
	return def
}

// New constructs a new stateful lexer from rules.
func New(rules Rules) (*StatefulDefinition, error) {
	compiled := compiledRules{}
	for key, set := range rules {
		for i, rule := range set {
			if validate, ok := rule.Action.(validatingRule); ok {
				if err := validate.validate(rules); err != nil {
					return nil, fmt.Errorf("lexer: invalid action for rule %q: %w", rule.Name, err)
				}
			}
			pattern := "^(?:" + rule.Pattern + ")"
			var (
				re  *regexp.Regexp
				err error
			)
			var match = backrefReplace.FindStringSubmatch(rule.Pattern)
			if match == nil || len(match[1])%2 == 0 {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("lexer: %s.%d: %w", key, i, err)
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
					return nil, fmt.Errorf("lexer: %s.%d: %w", state, i, err)
				}
				goto restart
			}
		}
	}
	keys := make([]string, 0, len(compiled))
	for key := range compiled {
		keys = append(keys, key)
	}
	symbols := map[string]TokenType{
		"EOF": EOF,
	}
	sort.Strings(keys)
	duplicates := map[string]compiledRule{}
	rn := EOF - 1
	for _, key := range keys {
		for i, rule := range compiled[key] {
			if dup, ok := duplicates[rule.Name]; ok && rule.Pattern != dup.Pattern {
				panic(fmt.Sprintf("lexer: duplicate key %q with different patterns %q != %q", rule.Name, rule.Pattern, dup.Pattern))
			}
			duplicates[rule.Name] = rule
			compiled[key][i] = rule
			symbols[rule.Name] = rn
			rn--
		}
	}
	d := &StatefulDefinition{
		rules:   compiled,
		symbols: symbols,
	}
	return d, nil
}

func (d *StatefulDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.rules)
}

// Rules returns the user-provided Rules used to construct the lexer.
func (d *StatefulDefinition) Rules() Rules {
	out := Rules{}
	for state, rules := range d.rules {
		for _, rule := range rules {
			out[state] = append(out[state], rule.Rule)
		}
	}
	return out
}

// LexString is a fast-path implementation for lexing strings.
func (d *StatefulDefinition) LexString(filename string, s string) (Lexer, error) {
	return &StatefulLexer{
		def:   d,
		data:  s,
		stack: []lexerState{{name: "Root"}},
		pos: Position{
			Filename: filename,
			Line:     1,
			Column:   1,
		},
	}, nil
}

func (d *StatefulDefinition) Lex(filename string, r io.Reader) (Lexer, error) {
	w := &strings.Builder{}
	_, err := io.Copy(w, r)
	if err != nil {
		return nil, err
	}
	return d.LexString(filename, w.String())
}

func (d *StatefulDefinition) Symbols() map[string]TokenType {
	return d.symbols
}

// lexerState stored when switching states in the lexer.
type lexerState struct {
	name   string
	groups []capture
}

// StatefulLexer implementation.
type StatefulLexer struct {
	stack []lexerState
	def   *StatefulDefinition
	data  string
	pos   Position
}

func (l *StatefulLexer) Next() (Token, error) {
	parent := l.stack[len(l.stack)-1]
	rules := l.def.rules[parent.name]
next:
	for len(l.data) > 0 {
		var (
			rule  *compiledRule
			m     []int
			match []int
		)
		for i, candidate := range rules {
			// Special case "Return()".
			if candidate.Rule == ReturnRule {
				l.stack = l.stack[:len(l.stack)-1]
				parent = l.stack[len(l.stack)-1]
				rules = l.def.rules[parent.name]
				continue next
			}
			re, err := l.getPattern(candidate)
			if err != nil {
				return Token{}, errorf(l.pos, "lexer: rule %q: %s", candidate.Name, err)
			}
			m = re.FindStringSubmatchIndex(l.data)
			if m != nil && (match == nil || m[1] > match[1]) {
				match = m
				rule = &rules[i]
				if !l.def.matchLongest {
					break
				}
			}
		}
		if match == nil || rule == nil {
			sample := []rune(l.data)
			if len(sample) > 16 {
				sample = append(sample[:16], []rune("...")...)
			}
			return Token{}, errorf(l.pos, "lexer: invalid input text %q", string(sample))
		}

		if rule.Action != nil {
			// A capture group that did not participate in the match has an
			// offset of -1. Keep its slot, so that groups[n] is always group n
			// for backreferences to resolve against, and record that it was
			// absent so that a backreference to it cannot match.
			groups := make([]capture, len(match)/2)
			for i := 0; i < len(match); i += 2 {
				if match[i] >= 0 {
					groups[i/2] = capture{text: l.data[match[i]:match[i+1]], present: true}
				}
			}
			if err := rule.Action.applyAction(l, groups); err != nil {
				return Token{}, errorf(l.pos, "lexer: rule %q: %s", rule.Name, err)
			}
		} else if match[0] == match[1] {
			return Token{}, errorf(l.pos, "lexer: rule %q did not match any input", rule.Name)
		}

		span := l.data[match[0]:match[1]]
		l.data = l.data[match[1]:]
		// l.groups = groups

		// Update position.
		pos := l.pos
		l.pos.Advance(span)
		if rule.ignore {
			parent = l.stack[len(l.stack)-1]
			rules = l.def.rules[parent.name]
			continue
		}
		return Token{
			Type:  l.def.symbols[rule.Name],
			Value: span,
			Pos:   pos,
		}, nil
	}
	return EOFToken(l.pos), nil
}

func (l *StatefulLexer) getPattern(candidate compiledRule) (*regexp.Regexp, error) {
	if candidate.RE != nil {
		return candidate.RE, nil
	}
	// We don't have a compiled RE. This means there are back-references
	// that need to be substituted first.
	return backrefRegex(&l.def.backrefCache, candidate.Pattern, l.stack[len(l.stack)-1].groups)
}

// BackrefRegex returns a compiled regular expression with backreferences replaced by groups.
func BackrefRegex(backrefCache *sync.Map, input string, groups []string) (*regexp.Regexp, error) {
	captures := make([]capture, len(groups))
	for i, group := range groups {
		captures[i] = capture{text: group, present: true}
	}
	return backrefRegex(backrefCache, input, captures)
}

// neverMatch is a zero-width expression that can never match, because a
// position cannot be both a word boundary and not a word boundary. It stands in
// for a backreference to a capture group that did not participate in the match,
// so that the expression around it still composes: "\1" cannot match, while
// "\1?" remains optional.
const neverMatch = `(?:\b\B)`

// backrefRegex returns a compiled regular expression with backreferences
// replaced by the parent's captures.
func backrefRegex(backrefCache *sync.Map, input string, groups []capture) (*regexp.Regexp, error) {
	key := backrefCacheKey(input, groups)
	cached, ok := backrefCache.Load(key)
	if ok {
		return cached.(*regexp.Regexp), nil
	}

	var (
		re  *regexp.Regexp
		err error
	)
	inClass := characterClassSpans(input)
	expanded := strings.Builder{}
	last := 0
	for _, match := range backrefReplace.FindAllStringSubmatchIndex(input, -1) {
		start, end := match[0], match[1]
		slashes := input[match[2]:match[3]]
		expanded.WriteString(input[last:start])
		last = end

		n, nerr := strconv.ParseInt(input[match[4]:match[5]], 10, 64)
		if nerr != nil {
			err = nerr
			expanded.WriteString(input[start:end])
			continue
		}
		if len(groups) == 0 || int(n) >= len(groups) {
			err = fmt.Errorf("invalid group %d from parent with %d groups", n, len(groups))
			expanded.WriteString(input[start:end])
			continue
		}
		// concatenate the leading \\\\ which are already escaped to the quoted match.
		expanded.WriteString(slashes[:len(slashes)-1])
		expanded.WriteString(expandBackref(groups[n], inClass[start]))
	}
	expanded.WriteString(input[last:])
	pattern := expanded.String()
	if err == nil {
		re, err = regexp.Compile("^(?:" + pattern + ")")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid backref expansion: %q: %w", pattern, err)
	}
	backrefCache.Store(key, re)
	return re, nil
}

// expandBackref returns the text a backreference expands to, which depends on
// where it sits: a character class holds members, everywhere else holds a
// subexpression.
func expandBackref(group capture, inCharacterClass bool) string {
	if inCharacterClass {
		// Grouping syntax is not grouping syntax inside a class. Wrapping here
		// would add "(", "?", ":" and ")" to the set, so "[\1]" capturing "a"
		// would also match a parenthesis. An absent capture contributes no
		// members at all, which leaves the rest of the class to decide, or
		// fails to compile when there is no rest -- the same as the empty
		// capture this expansion has always produced there.
		if !group.present {
			return ""
		}
		return regexp.QuoteMeta(group.text)
	}
	if !group.present {
		return neverMatch
	}
	// A single group, so that a quantified or alternated backreference still
	// parses when the capture is empty.
	return "(?:" + regexp.QuoteMeta(group.text) + ")"
}

// characterClassSpans reports for each byte of input whether it lies inside a
// character class. RE2 has no nested classes, so one flag is enough; what it
// does have is escapes, \Q...\E literal runs, POSIX names such as [:alpha:]
// whose brackets neither open nor close a class, and a "]" that is the first
// member of a class, which is a literal rather than the delimiter: "[]a]" is
// the two-member class RE2 accepts, not an empty class it rejects.
func characterClassSpans(input string) []bool {
	inClass := make([]bool, len(input))
	open, literal := false, false
	// Index of the "[" that opened the current class, so the first member can
	// be recognised. -1 while no class is open.
	start := -1
	for i := 0; i < len(input); i++ {
		inClass[i] = open
		switch {
		case literal:
			if input[i] == '\\' && i+1 < len(input) && input[i+1] == 'E' {
				literal = false
				i++
				inClass[i] = open
			}
		case input[i] == '\\':
			if i+1 < len(input) {
				literal = input[i+1] == 'Q'
				i++
				inClass[i] = open
			}
		case input[i] == '[':
			if !open {
				open, start = true, i
			} else if end := posixClassEnd(input, i); end >= 0 {
				for ; i <= end; i++ {
					inClass[i] = true
				}
				i--
			}
		case input[i] == ']':
			if !firstClassMember(input, start, i) {
				open, start = false, -1
			}
		}
	}
	return inClass
}

// firstClassMember reports whether the byte at i is the first member of the
// class opened at start, where a "]" stands for itself instead of closing.
// That is the position right after "[", or right after "[^".
func firstClassMember(input string, start, i int) bool {
	if start < 0 {
		return false
	}
	if i == start+1 {
		return true
	}
	return i == start+2 && input[start+1] == '^'
}

// posixClassEnd returns the index of the "]" closing a POSIX name such as
// [:alpha:] opening at open, or -1 when that is not what starts there.
func posixClassEnd(input string, open int) int {
	if open+1 >= len(input) || input[open+1] != ':' {
		return -1
	}
	end := strings.Index(input[open+2:], ":]")
	if end < 0 {
		return -1
	}
	return open + 2 + end + 1
}

// backrefCacheKey builds a key identifying a pattern together with the parent
// captures it is expanded against. Lengths are included because the captures
// themselves may contain the separator.
func backrefCacheKey(input string, groups []capture) string {
	size := len(input) + 8
	for _, group := range groups {
		size += len(group.text) + 8
	}
	key := strings.Builder{}
	key.Grow(size)
	key.WriteString(strconv.Itoa(len(input)))
	key.WriteString(":")
	key.WriteString(input)
	for _, group := range groups {
		key.WriteString("\000")
		if !group.present {
			continue
		}
		key.WriteString(strconv.Itoa(len(group.text)))
		key.WriteString(":")
		key.WriteString(group.text)
	}
	return key.String()
}
