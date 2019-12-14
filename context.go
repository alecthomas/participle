package participle

import (
	"reflect"

	"github.com/alecthomas/participle/lexer"
)

type contextFieldSet struct {
	pos        lexer.Position
	strct      reflect.Value
	field      structLexerField
	fieldValue []reflect.Value
}

// Context for a single parse.
type parseContext struct {
	*lexer.PeekingLexer
	lookahead       int
	caseInsensitive map[rune]bool
	apply           []*contextFieldSet
	allowTrailing   bool
}

func newParseContext(lex *lexer.PeekingLexer, lookahead int, caseInsensitive map[rune]bool) *parseContext {
	return &parseContext{
		PeekingLexer:    lex,
		caseInsensitive: caseInsensitive,
		lookahead:       lookahead,
	}
}

// Defer adds a function to be applied once a branch has been picked.
func (p *parseContext) Defer(pos lexer.Position, strct reflect.Value, field structLexerField, fieldValue []reflect.Value) {
	p.apply = append(p.apply, &contextFieldSet{pos, strct, field, fieldValue})
}

// Apply deferred functions.
func (p *parseContext) Apply() error {
	for _, apply := range p.apply {
		if err := setField(apply.pos, apply.strct, apply.field, apply.fieldValue); err != nil {
			return err
		}
	}
	p.apply = nil
	return nil
}

// Branch accepts the branch as the correct branch.
func (p *parseContext) Accept(branch *parseContext) {
	p.apply = append(p.apply, branch.apply...)
	p.PeekingLexer = branch.PeekingLexer
}

// Branch starts a new lookahead branch.
func (p *parseContext) Branch() *parseContext {
	branch := &parseContext{}
	*branch = *p
	branch.apply = nil
	branch.PeekingLexer = p.PeekingLexer.Clone()
	return branch
}

// Stop returns true if parsing should terminate after the given "branch" failed to match.
func (p *parseContext) Stop(branch *parseContext) bool {
	if branch.PeekingLexer.Cursor() > p.PeekingLexer.Cursor()+p.lookahead {
		p.Accept(branch)
		return true
	}
	return false
}
