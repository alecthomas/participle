package ast

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	Lexer = lexer.MustStateful(lexer.Rules{
		"Root": {
			{"comment", `//[^\n]*`, nil},
			{"comment2", `/\\*`, lexer.Push("BlockComment")},
			{"String", `'(\\\\|\\'|[^'])*'`, nil},
			{"Group", `\[(\\]|[^\]])*\]`, nil},
			{"UpperIdent", `[A-Z][a-zA-Z_]*\w*`, nil},
			{"LowerIdent", `[a-z][a-zA-Z_]*\w*`, nil},
			{"Punct", `[-[!@#$%^&*()+_={}\|:;"'<,>.?/~]|]`, nil},
			{"whitespace", `[ \t\r\n]+`, nil},
		},
		"BlockComment": {
			{"end", "\\*/", lexer.Pop()},
			{"any", "([^*]|\\*[^/])+", nil},
		},
	})
	Parser = MustBuildParser(&AntlrFile{})
)

func MustBuildParser(v interface{}) *participle.Parser {
	return participle.MustBuild(v,
		participle.Lexer(Lexer),
		participle.UseLookahead(2),
	)
}

type Visitor interface {
	VisitAntlrFile(*AntlrFile)
	VisitGrammarStmt(*GrammarStmt)
	VisitOptionStmt(*OptionStmt)
	VisitOption(*Option)
	VisitParserRules(ParserRules)
	VisitParserRule(*ParserRule)
	VisitLexerRules(LexerRules)
	VisitLexerRule(*LexerRule)
	VisitAlternative(*Alternative)
	VisitExpression(*Expression)
	VisitUnary(*Unary)
	VisitPrimary(*Primary)
	VisitCharRange(*CharRange)
}

type BaseVisitor struct{}

func (bv *BaseVisitor) VisitAntlrFile(af *AntlrFile)     {}
func (bv *BaseVisitor) VisitGrammarStmt(af *GrammarStmt) {}
func (bv *BaseVisitor) VisitOptionStmt(af *OptionStmt)   {}
func (bv *BaseVisitor) VisitOption(af *Option)           {}
func (bv *BaseVisitor) VisitLexerRules(lr LexerRules)    {}
func (bv *BaseVisitor) VisitLexerRule(af *LexerRule)     {}
func (bv *BaseVisitor) VisitParserRule(pr *ParserRule)   {}
func (bv *BaseVisitor) VisitParserRules(pr ParserRules)  {}
func (bv *BaseVisitor) VisitAlternative(a *Alternative)  {}
func (bv *BaseVisitor) VisitExpression(exp *Expression)  {}
func (bv *BaseVisitor) VisitUnary(u *Unary)              {}
func (bv *BaseVisitor) VisitPrimary(pr *Primary)         {}
func (bv *BaseVisitor) VisitCharRange(cr *CharRange)     {}

type AntlrFile struct {
	Grammar *GrammarStmt `parser:" @@ "`
	Options *OptionStmt  `parser:" @@? "`
	Rules   []*Rule      `parser:" @@* "`

	LexRules LexerRules
	PrsRules ParserRules
}

// HACK: Fix this.
func (af *AntlrFile) SplitRules() {
	for _, r := range af.Rules {
		if r.LexRule != nil {
			af.LexRules = append(af.LexRules, r.LexRule)
		} else {
			af.PrsRules = append(af.PrsRules, r.PrsRule)
		}
	}
}

func (af *AntlrFile) Accept(v Visitor) {
	v.VisitAntlrFile(af)
}

func (af *AntlrFile) Merge(af2 *AntlrFile) (*AntlrFile, error) {
	if af == nil {
		return af2, nil
	}
	if af2.Grammar.ParserOnly {
		return af2.Merge(af)
	}
	if vocab := af.Options.Lookup("tokenVocab"); vocab != af2.Grammar.Name {
		return nil, fmt.Errorf("parser %s expected lexer %s but found %s", af.Grammar.Name, vocab, af2.Grammar.Name)
	}
	ret := &AntlrFile{
		Grammar:  &GrammarStmt{Name: af.Grammar.Name},
		LexRules: make(LexerRules, len(af.LexRules)+len(af2.LexRules)),
		PrsRules: make(ParserRules, len(af.PrsRules)+len(af2.PrsRules)),
	}

	for i, v := range af.LexRules {
		ret.LexRules[i] = v
	}
	for i, v := range af2.LexRules {
		ret.LexRules[i+len(af.LexRules)] = v
	}

	for i, v := range af.PrsRules {
		ret.PrsRules[i] = v
	}
	for i, v := range af2.PrsRules {
		ret.PrsRules[i+len(af.PrsRules)] = v
	}

	return ret, nil
}

type GrammarStmt struct {
	LexerOnly  bool   `parser:" @'lexer'? "`
	ParserOnly bool   `parser:" @'parser'? "`
	Name       string `parser:" 'grammar' @( UpperIdent | LowerIdent) ';' "`
}

func (gs *GrammarStmt) Accept(v Visitor) {
	v.VisitGrammarStmt(gs)
}

type OptionStmt struct {
	Opts []*Option `parser:" 'options' '{' @@ '}' "`
}

func (os *OptionStmt) Lookup(key string) string {
	for _, kv := range os.Opts {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

func (os *OptionStmt) Accept(v Visitor) {
	v.VisitOptionStmt(os)
}

type Option struct {
	Key   string `parser:" @(LowerIdent|UpperIdent) '=' "`
	Value string `parser:" @(LowerIdent|UpperIdent) ';' "`
}

func (o *Option) Accept(v Visitor) {
	v.VisitOption(o)
}

type Rule struct {
	LexRule *LexerRule  `parser:" ( @@ "`
	PrsRule *ParserRule `parser:" | @@ ) "`
}

type ParserRules []*ParserRule

func (pr ParserRules) Accept(v Visitor) {
	v.VisitParserRules(pr)
}

type ParserRule struct {
	Name string       `parser:" @LowerIdent ':' "`
	Alt  *Alternative `parser:" @@ ';' "`
}

func (pr *ParserRule) Accept(v Visitor) {
	v.VisitParserRule(pr)
}

type LexerRules []*LexerRule

func (lr LexerRules) Accept(v Visitor) {
	v.VisitLexerRules(lr)
}

type LexerRule struct {
	Fragment bool         `parser:" @'fragment'? "`
	Name     string       `parser:" @UpperIdent ':' "`
	Alt      *Alternative `parser:" @@ "`
	Skip     bool         `parser:" ('-' '>' ( @'skip' "`
	Channel  string       `parser:" | 'channel' '(' @UpperIdent ')' ) )? ';' "`
}

func (lr *LexerRule) Accept(v Visitor) {
	v.VisitLexerRule(lr)
}

type Alternative struct {
	Exp       *Expression  `parser:" @@? "`
	Label     *string      `parser:" ('#' (@UpperIdent|@LowerIdent) )? "`
	Next      *Alternative `parser:" (  '|' @@"`
	EmptyNext bool         `parser:" | @'|' (?=';') )? "`
}

func (a *Alternative) Accept(v Visitor) {
	v.VisitAlternative(a)
}

type Expression struct {
	Label   *string     `parser:" ( @(UpperIdent|LowerIdent) "`
	LabelOp *string     `parser:" @( '=' | '+' '=' ) )? "`
	Unary   *Unary      `parser:" @@ "`
	Next    *Expression `parser:" ( @@ )? "`
}

func (exp *Expression) Accept(v Visitor) {
	v.VisitExpression(exp)
}

type Unary struct {
	Op      string   `parser:" ( @( '~' ) "`
	Unary   *Unary   `parser:"     @@   ) "`
	Primary *Primary `parser:" | @@ "`
}

func (u *Unary) Accept(v Visitor) {
	v.VisitUnary(u)
}

type Primary struct {
	Range     *CharRange   `parser:" ( @@ "`
	Str       *string      `parser:" | @String "`
	Ident     *string      `parser:" | @(UpperIdent|LowerIdent) "`
	Group     *string      `parser:" | @Group "`
	Any       bool         `parser:" | @'.' "`
	Sub       *Alternative `parser:" | '(' @@ ')' ) "`
	Arity     string       `parser:" ( @('+' | '*' | '?') "`
	NonGreedy bool         `parser:"   @'?'? )? "`
}

func (pr *Primary) Accept(v Visitor) {
	v.VisitPrimary(pr)
}

type CharRange struct {
	Start string `parser:" @String '.' '.' "`
	End   string `parser:" @String "`
}

func (cr *CharRange) Accept(v Visitor) {
	v.VisitCharRange(cr)
}
