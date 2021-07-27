package antlr

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
	"github.com/alecthomas/repr"
)

var reUnicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// LexerVisitor visits an Antlr grammar AST to build Participle lexer rules.
type LexerVisitor struct {
	ast.BaseVisitor

	RuleMap       map[string]*ast.LexerRule
	computedRules map[*ast.LexerRule]gen.LexerRule
	processing    strStack
	negated       boolStack
	alts          int

	collapse  bool
	debug     bool
	recursion int

	result gen.LexerRule
}

// NewLexerVisitor returns a ready LexerVisitor.
func NewLexerVisitor(rm map[string]*ast.LexerRule) *LexerVisitor {
	return &LexerVisitor{
		RuleMap:       rm,
		computedRules: map[*ast.LexerRule]gen.LexerRule{},
	}
}

// Visit builds information a Participle lexer rule.
func (lv *LexerVisitor) Visit(a ast.Node) gen.LexerRule {
	lv.printf("LexerVisitor Start: %T", a)
	lv.result = gen.LexerRule{}
	a.Accept(lv)
	lv.printf("LexerVisitor Done: %T = %s", a, repr.String(lv.result))
	return lv.result
}

// VisitLexerRule computes a full Participle lexer rule from an Antlr lexer rule.
// Note that recursive Antlr lexer rules are not supported and will cause a panic.
func (lv *LexerVisitor) VisitLexerRule(lr *ast.LexerRule) {
	if comp, exist := lv.computedRules[lr]; exist {
		lv.result = comp
		return
	}

	// TODO: Can't handle recursive lexer rules at the moment.
	if lv.processing.contains(lr.Name) {
		panic(fmt.Sprintf("lexer rule %s is recursive, this is currently not supported", lr.Name))
	}

	lv.processing.push(lr.Name)
	defer lv.processing.pop()
	ret := lv.Visit(lr.Alt)
	lv.computedRules[lr] = ret
	lv.result = ret
}

// VisitAlternative generates lexing information from one Antlr rule alternate.
func (lv *LexerVisitor) VisitAlternative(a *ast.Alternative) {
	// TODO: Handle the label
	exp := lv.Visit(a.Exp)
	if a.Next != nil {
		lv.alts++
		next := lv.Visit(a.Next)
		lv.alts--

		format := "%s|%s"
		switch {
		case lv.collapse:
			format = "%s%s"
		case lv.alts == 0:
			format = "(%s|%s)"
		}

		lv.result = gen.LexerRule{
			Content:    fmt.Sprintf(format, exp.Content, next.Content),
			NotLiteral: true,
			Length:     max(exp.Length, next.Length),
		}
	} else {
		lv.result = exp
	}
}

// VisitExpression generates lexing information from one Antlr rule expression.
func (lv *LexerVisitor) VisitExpression(exp *ast.Expression) {
	// TODO: Handle the label & op
	ret := lv.Visit(exp.Unary)
	if exp.Next != nil {
		ret = ret.Plus(lv.Visit(exp.Next))
	}
	lv.result = ret
}

// VisitUnary generates lexing information from one Antlr unary operator.
func (lv *LexerVisitor) VisitUnary(u *ast.Unary) {
	if u.Unary != nil {
		lv.negated.push(u.Op == "~")
		defer func() {
			lv.negated.pop()
		}()
		lv.Visit(u.Unary)
	} else {
		lv.Visit(u.Primary)
	}
}

// VisitPrimary generates lexing information from one Antlr rule terminal or sub-rule.
func (lv *LexerVisitor) VisitPrimary(pr *ast.Primary) {
	sf := pr.Arity + ifStr(pr.NonGreedy, "?")
	suffix := gen.LexerRule{
		Content:    sf,
		NotLiteral: sf != "",
		Length:     0,
	}
	var ret gen.LexerRule
	switch {
	case pr.Range != nil:
		ret = lv.Visit(pr.Range)
	case pr.Str != nil:
		ret = gen.LiteralLexerRule(strToRegex(*pr.Str))
		if lv.negated.safePeek() {
			if regexCharLen(ret.Content) == 1 {
				ret = gen.LexerRule{
					Content:    fmt.Sprintf("[^%s]", ret.Content),
					NotLiteral: true,
					Length:     ret.Length,
				}
			} else {
				panic(fmt.Sprintf("negating multi-character strings is not supported: %s", ret.Content))
			}
		}
	case pr.Ident != nil:
		v := NewLexerVisitor(lv.RuleMap)
		v.recursion = lv.recursion + 1
		v.collapse = lv.collapse
		v.processing = lv.processing
		v.debug = lv.debug
		ret = v.Visit(lv.RuleMap[*pr.Ident])
		if suffix.Content != "" {
			ret.Content = withParens(ret.Content)
		}
	case pr.Group != nil:
		txt := fixUnicodeEscapes(*pr.Group)
		l := len(txt) - 2
		if lv.negated.safePeek() {
			txt = "[^" + txt[1:]
		}
		ret = gen.LexerRule{
			Content:    txt,
			NotLiteral: true,
			Length:     l,
		}
	case pr.Any:
		ret = gen.LexerRule{
			Content:    ".",
			NotLiteral: true,
			Length:     1,
		}
	case pr.Sub != nil:

		invert := lv.negated.safePeek()
		canInvert := invert && NewCanInvert(lv.RuleMap).Check(pr.Sub)

		v := NewLexerVisitor(lv.RuleMap)
		v.collapse = lv.collapse || (invert && canInvert)
		v.recursion = lv.recursion + 1
		v.processing = lv.processing
		v.debug = lv.debug

		ret = v.Visit(pr.Sub)
		if invert && !canInvert {
			panic(fmt.Sprintf("subexpression cannot be inverted: %s", ret.Content))
		}

		switch {
		case lv.collapse:
		case invert:
			ret.Content = "[^" + ret.Content + "]"
		default:
			ret.Content = withParens(ret.Content)
		}
	}
	lv.result = ret.Plus(suffix)
}

// VisitCharRange generates lexing information from an Antlr character range.
func (lv *LexerVisitor) VisitCharRange(cr *ast.CharRange) {
	start, end := stripQuotes(fixUnicodeEscapes(cr.Start)), stripQuotes(fixUnicodeEscapes(cr.End))
	lv.result = gen.LexerRule{
		Content:    fmt.Sprintf("[%s%s-%s]", ifStr(lv.negated.safePeek(), "^"), start, end),
		NotLiteral: true,
		Length:     1,
	}
}

func (lv *LexerVisitor) printf(format string, args ...interface{}) {
	if lv.debug {
		log.Printf(strings.Repeat("    ", lv.recursion)+format, args...)
	}
}
