package antlr

import "github.com/alecthomas/participle/v2/antlr/ast"

// AltCounter counts the Alternatives in a portion of Antlr grammar AST.
// It does not recurse.
type AltCounter struct {
	ast.BaseVisitor
	count int
}

// VisitAlternative implements the Visitor interface.
func (ac *AltCounter) VisitAlternative(a *ast.Alternative) {
	ac.count++
	if a.Next != nil {
		a.Next.Accept(ac)
	}
}

// CountAlts returns the number of Alternatives in a portion of Antlr grammar AST.
func (ac *AltCounter) CountAlts(a *ast.Alternative) int {
	ac.VisitAlternative(a)
	return ac.count
}
