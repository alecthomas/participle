// Package antlr provides visitors for converting an Antlr grammar AST
// into a Participle lexer and parser.
package antlr

import (
	"github.com/alecthomas/participle/v2/experimental/antlr/ast"
	"github.com/alecthomas/participle/v2/experimental/antlr/gen"
)

// AntlrVisitor recursively builds Participle proto-structs
// out of an Antlr grammar AST.
type AntlrVisitor struct {
	ast.BaseVisitor

	Structs           []*gen.Struct
	Literals          []string
	ChildRuleCounters map[string]int
	OptionalRules     map[string]bool
	LexerTokens       map[string]struct{}
}

// NewAntlrVisitor returns a prepared AntlrVisitor.
func NewAntlrVisitor() *AntlrVisitor {
	return &AntlrVisitor{
		ChildRuleCounters: map[string]int{},
		OptionalRules:     map[string]bool{},
		LexerTokens:       map[string]struct{}{},
	}
}

// VisitAntlrFile implements the ast.Visitor interface.
func (sv *AntlrVisitor) VisitAntlrFile(af *ast.AntlrFile) {
	rules := af.PrsRules()
	for _, pr := range rules {
		sv.OptionalRules[pr.Name] = new(OptionalChecker).RuleIsOptional(pr)
	}
	for _, pr := range rules {
		pr.Accept(sv)
	}
}

// VisitParserRule implements the ast.Visitor interface.
// It uses a StructVisitor to generate a Participle proto-struct
// corresponding to an Antlr parser rule.
func (sv *AntlrVisitor) VisitParserRule(pr *ast.ParserRule) {
	v := NewStructVisitor(sv.OptionalRules, sv.LexerTokens)
	v.ComputeStruct(pr)
	sv.Structs = append(sv.Structs, v.Result)
	sv.Literals = append(sv.Literals, v.literals...)
	for k, v := range v.ChildRuleCounters {
		sv.ChildRuleCounters[k] += v
	}
}
