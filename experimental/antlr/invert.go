package antlr

import (
	"reflect"

	"github.com/alecthomas/participle/v2/experimental/antlr/ast"
)

// CanInvert is an Antlr grammar AST visitor.
// Its purpose is to check if an expression in a lexer rule
// can be inverted.  For example ~('a'|'b') can invert to [^ab]
// but more complex inversions like ~('a'('b'|'c')) are not supported.
type CanInvert struct {
	ast.BaseVisitor
	ruleMap map[string]*ast.LexerRule
	result  bool
}

// NewCanInvert returns a ready CanInvert instance.
func NewCanInvert(rm map[string]*ast.LexerRule) *CanInvert {
	return &CanInvert{
		ruleMap: rm,
	}
}

// Check returns if an Alternative can be inverted.
func (ci *CanInvert) Check(a *ast.Alternative) bool {
	return ci.visit(a)
}

func (ci *CanInvert) visit(node ast.Node) bool {
	if node == nil || reflect.ValueOf(node) == reflect.Zero(reflect.TypeOf(node)) {
		return true
	}
	ci.result = false
	node.Accept(ci)
	return ci.result
}

// VisitAlternative implements the ast.Visitor interface.
func (ci *CanInvert) VisitAlternative(a *ast.Alternative) {
	ci.result = ci.visit(a.Exp) && ci.visit(a.Next)
}

// VisitExpression implements the ast.Visitor interface.
func (ci *CanInvert) VisitExpression(exp *ast.Expression) {
	ci.result = ci.visit(exp.Unary) && exp.Next == nil
}

// VisitUnary implements the ast.Visitor interface.
func (ci *CanInvert) VisitUnary(u *ast.Unary) {
	ci.result = u.Op != "~" && ci.visit(u.Unary) && ci.visit(u.Primary)
}

// VisitPrimary implements the ast.Visitor interface.
func (ci *CanInvert) VisitPrimary(pr *ast.Primary) {
	switch {
	case pr.Str != nil:
		ci.result = antlrLiteralLen(stripQuotes(*pr.Str)) == 1
	case pr.Ident != nil:
		rule, exist := ci.ruleMap[*pr.Ident]
		ci.result = exist && ci.visit(rule.Alt)
	case pr.Sub != nil:
		ci.result = ci.visit(pr.Sub)
	default:
		ci.result = false
	}
}
