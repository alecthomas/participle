package antlr

import (
	"reflect"

	"github.com/alecthomas/participle/v2/antlr/ast"
)

type CanInvert struct {
	ast.BaseVisitor
	ruleMap map[string]*ast.LexerRule
	result  bool
}

func NewCanInvert(rm map[string]*ast.LexerRule) *CanInvert {
	return &CanInvert{
		ruleMap: rm,
	}
}

func (ci *CanInvert) Check(a *ast.Alternative) bool {
	return ci.visit(a)
}

func (ci *CanInvert) visit(node interface {
	Accept(ast.Visitor)
}) bool {
	if node == nil || reflect.ValueOf(node) == reflect.Zero(reflect.TypeOf(node)) {
		return true
	}
	ci.result = false
	node.Accept(ci)
	return ci.result
}

func (ci *CanInvert) VisitAlternative(a *ast.Alternative) {
	ci.result = ci.visit(a.Exp) && ci.visit(a.Next)
}

func (ci *CanInvert) VisitExpression(exp *ast.Expression) {
	ci.result = ci.visit(exp.Unary) && exp.Next == nil
}

func (ci *CanInvert) VisitUnary(u *ast.Unary) {
	ci.result = u.Op != "~" && ci.visit(u.Unary) && ci.visit(u.Primary)
}

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
