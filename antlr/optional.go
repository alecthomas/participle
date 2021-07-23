package antlr

import "github.com/alecthomas/participle/v2/antlr/ast"

type OptionalChecker struct {
	ast.BaseVisitor
	optional bool
}

func NewOptionalChecker() *OptionalChecker {
	return &OptionalChecker{}
}

func (oc *OptionalChecker) RuleIsOptional(pr *ast.ParserRule) bool {
	pr.Accept(oc)
	return oc.optional
}

func (oc *OptionalChecker) VisitParserRule(pr *ast.ParserRule) {
	pr.Alt.Accept(oc)
}

func (oc *OptionalChecker) VisitAlternative(a *ast.Alternative) {
	if a.Exp == nil || a.EmptyNext {
		oc.optional = true
	} else if a.Next != nil {
		a.Next.Accept(oc)
	}
}
