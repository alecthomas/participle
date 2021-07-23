package antlr

import (
	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
)

type AntlrVisitor struct {
	ast.BaseVisitor

	Structs           []*gen.Struct
	Literals          []string
	ChildRuleCounters map[string]int
	OptionalRules     map[string]bool
	LexerTokens       map[string]struct{}
}

func NewAntlrVisitor() *AntlrVisitor {
	return &AntlrVisitor{
		ChildRuleCounters: map[string]int{},
		OptionalRules:     map[string]bool{},
		LexerTokens:       map[string]struct{}{},
	}
}

func (sv *AntlrVisitor) VisitAntlrFile(af *ast.AntlrFile) {
	for _, pr := range af.PrsRules {
		sv.OptionalRules[pr.Name] = NewOptionalChecker().RuleIsOptional(pr)
	}
	for _, pr := range af.PrsRules {
		pr.Accept(sv)
	}
}

func (sv *AntlrVisitor) VisitParserRule(pr *ast.ParserRule) {
	v := NewStructVisitor(sv.OptionalRules, sv.LexerTokens)
	v.ComputeStruct(pr)
	sv.Structs = append(sv.Structs, v.Result)
	sv.Literals = append(sv.Literals, v.literals...)
	for k, v := range v.ChildRuleCounters {
		sv.ChildRuleCounters[k] += v
	}
}
