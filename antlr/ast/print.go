package ast

import (
	"fmt"
	"strings"
)

var _ Visitor = (*Printer)(nil)

type Printer struct {
	result string
}

func NewPrinter() *Printer {
	return &Printer{}
}

func (prn *Printer) Visit(n Node) string {
	prn.result = ""
	n.Accept(prn)
	return prn.result
}

func (prn *Printer) VisitAntlrFile(af *AntlrFile) {
	prn.result = prn.Visit(af.Grammar) + "\n\n" +
		ifStr(af.Options != nil, prn.Visit(af.Options)+"\n\n") +
		prn.Visit(af.LexRules()) +
		prn.Visit(af.PrsRules())
}

func (prn *Printer) VisitGrammarStmt(gs *GrammarStmt) {
	prn.result = fmt.Sprintf(
		"%s%sgrammar %s;",
		ifStr(gs.LexerOnly, "lexer "),
		ifStr(gs.ParserOnly, "parser "),
		gs.Name,
	)
}

func (prn *Printer) VisitOptionStmt(os *OptionStmt) {
	if os == nil {
		return
	}
	res := make([]string, len(os.Opts))
	for i, v := range os.Opts {
		res[i] = prn.Visit(v)
	}
	prn.result = fmt.Sprintf("options { %s }", strings.Join(res, ""))
}

func (prn *Printer) VisitOption(o *Option) {
	prn.result = fmt.Sprintf("%s=%s;", o.Key, o.Value)
}

func (prn *Printer) VisitParserRules(pr ParserRules) {
	rules := "// None"
	if len(pr) > 0 {
		res := make([]string, len(pr))
		for i, v := range pr {
			res[i] = prn.Visit(v)
		}
		rules = strings.Join(res, "\n")
	}
	prn.result = "// Parser Rules\n\n" + rules + "\n\n"
}

func (prn *Printer) VisitParserRule(pr *ParserRule) {
	prn.result = pr.Name + ": " + prn.Visit(pr.Alt) + ";"
}

func (prn *Printer) VisitLexerRules(lr LexerRules) {
	rules := "// None"
	if len(lr) > 0 {
		res := make([]string, len(lr))
		for i, v := range lr {
			res[i] = prn.Visit(v)
		}
		rules = strings.Join(res, "\n")
	}
	prn.result = "// Lexer Rules\n\n" + rules + "\n\n"
}

func (prn *Printer) VisitLexerRule(lr *LexerRule) {
	base := ifStr(lr.Fragment, "fragment ") + lr.Name + ": " + prn.Visit(lr.Alt)
	switch {
	case lr.Skip:
		base += " -> skip"
	case lr.Channel != "":
		base += " -> channel(" + lr.Channel + ")"
	}
	prn.result = base + ";"
}

func (prn *Printer) VisitAlternative(a *Alternative) {
	if a == nil {
		return
	}
	prn.result = prn.Visit(a.Exp) +
		ifStrPtr(a.Label, " #", "") +
		ifStr(a.Exp != nil && a.Next != nil, " ") +
		ifStr(a.Next != nil, "| "+prn.Visit(a.Next))
}

func (prn *Printer) VisitExpression(exp *Expression) {
	if exp == nil {
		return
	}
	str := ifStrPtr(exp.Label) + ifStrPtr(exp.LabelOp) + prn.Visit(exp.Unary)
	if exp.Next != nil {
		str += " " + prn.Visit(exp.Next)
	}
	prn.result = str
}

func (prn *Printer) VisitUnary(u *Unary) {
	if u == nil {
		return
	}
	prn.result = u.Op + prn.Visit(u.Unary) + prn.Visit(u.Primary)
}

func (prn *Printer) VisitPrimary(pr *Primary) {
	if pr == nil {
		return
	}
	prn.result = prn.Visit(pr.Range) +
		ifStrPtr(pr.Str) +
		ifStrPtr(pr.Ident) +
		ifStr(pr.Any, ".") +
		ifStrPtr(pr.Group) +
		ifStr(pr.Sub != nil, "("+prn.Visit(pr.Sub)+")") +
		pr.Arity +
		ifStr(pr.NonGreedy, "?")
}

func (prn *Printer) VisitCharRange(cr *CharRange) {
	if cr == nil {
		return
	}
	prn.result = fmt.Sprintf("%s..%s", cr.Start, cr.End)
}

func ifStr(b bool, s string) string {
	if b {
		return s
	}
	return ""
}

func ifStrPtr(sp *string, quotes ...string) string {
	if sp != nil {
		esc := *sp
		if len(quotes) > 1 {
			return quotes[0] + esc + quotes[1]
		}
		if len(quotes) > 0 {
			return quotes[0] + esc + quotes[0]
		}
		return esc
	}
	return ""
}
