package antlr

import "github.com/alecthomas/participle/v2/antlr/ast"

// OptionalChecker visits an Antlr parser rule to see if it contains
// an empty alternative, which marks the entire rule as optional.
// See https://github.com/antlr/antlr4/blob/master/doc/parser-rules.md#parser-rules
type OptionalChecker struct {
	ast.BaseVisitor
	optional bool
}

// RuleIsOptional returns if the Antlr parser is optional.
func (oc *OptionalChecker) RuleIsOptional(pr *ast.ParserRule) bool {
	pr.Accept(oc)
	return oc.optional
}

// VisitParserRule implements the ast.Visitor interface.
func (oc *OptionalChecker) VisitParserRule(pr *ast.ParserRule) {
	pr.Alt.Accept(oc)
}

// VisitParserRule implements the ast.Visitor interface.
func (oc *OptionalChecker) VisitAlternative(a *ast.Alternative) {
	if a.Exp == nil || a.EmptyNext {
		oc.optional = true
	} else if a.Next != nil {
		a.Next.Accept(oc)
	}
}
