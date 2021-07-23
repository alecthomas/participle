package antlr

import "github.com/alecthomas/participle/v2/antlr/ast"

type AltCounter struct {
	ast.BaseVisitor
	count int
}

func NewAltCounter() *AltCounter {
	return &AltCounter{}
}

func (ac *AltCounter) VisitAlternative(a *ast.Alternative) {
	ac.count++
	if a.Next != nil {
		a.Next.Accept(ac)
	}
}

func (ac *AltCounter) CountAlts(a *ast.Alternative) int {
	ac.VisitAlternative(a)
	return ac.count
}
