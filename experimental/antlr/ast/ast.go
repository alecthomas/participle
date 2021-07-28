// Package ast contains a lexer and parser for ANTLR grammar files,
// as well as all of the resulting AST objects.
package ast

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	// Lexer is the default lexer for Antlr files.
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
	// Parser is the default parser for Antlr files.
	Parser = MustBuildParser(&AntlrFile{})
)

// MustBuildParser is a utility that creates a Participle parser with default settings.
func MustBuildParser(n Node) *participle.Parser {
	return participle.MustBuild(n,
		participle.Lexer(Lexer),
		participle.UseLookahead(2),
	)
}

// Visitor allows for walking an Antlr file's AST.
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

// BaseVisitor can be embedded in your Visitor implementation to provide no-op versions
// of methods that don't matter to your implementation.
type BaseVisitor struct{}

// VisitAntlrFile is a no-op.
func (bv *BaseVisitor) VisitAntlrFile(af *AntlrFile) {}

// VisitGrammarStmt is a no-op.
func (bv *BaseVisitor) VisitGrammarStmt(af *GrammarStmt) {}

// VisitOptionStmt is a no-op.
func (bv *BaseVisitor) VisitOptionStmt(af *OptionStmt) {}

// VisitOption is a no-op.
func (bv *BaseVisitor) VisitOption(af *Option) {}

// VisitLexerRules is a no-op.
func (bv *BaseVisitor) VisitLexerRules(lr LexerRules) {}

// VisitLexerRule is a no-op.
func (bv *BaseVisitor) VisitLexerRule(af *LexerRule) {}

// VisitParserRule is a no-op.
func (bv *BaseVisitor) VisitParserRule(pr *ParserRule) {}

// VisitParserRules is a no-op.
func (bv *BaseVisitor) VisitParserRules(pr ParserRules) {}

// VisitAlternative is a no-op.
func (bv *BaseVisitor) VisitAlternative(a *Alternative) {}

// VisitExpression is a no-op.
func (bv *BaseVisitor) VisitExpression(exp *Expression) {}

// VisitUnary is a no-op.
func (bv *BaseVisitor) VisitUnary(u *Unary) {}

// VisitPrimary is a no-op.
func (bv *BaseVisitor) VisitPrimary(pr *Primary) {}

// VisitCharRange is a no-op.
func (bv *BaseVisitor) VisitCharRange(cr *CharRange) {}

// Node is conformed to by all AST nodes.
type Node interface {
	Accept(Visitor)
}

// AntlrFile is an overall Antlr4 grammar file, whether Lexer or Parser.
type AntlrFile struct {
	Grammar *GrammarStmt `parser:" @@ "`
	Options *OptionStmt  `parser:" @@? "`
	Rules   []*Rule      `parser:" @@* "`
}

// LexRules returns all of the lexer rules present in the Antlr file.
func (af *AntlrFile) LexRules() (ret LexerRules) {
	ret = make(LexerRules, 0, len(af.Rules))
	for _, r := range af.Rules {
		if r.LexRule != nil {
			ret = append(ret, r.LexRule)
		}
	}
	return
}

// PrsRules returns all of the parser rules present in the Antlr file.
func (af *AntlrFile) PrsRules() (ret ParserRules) {
	ret = make(ParserRules, 0, len(af.Rules))
	for _, r := range af.Rules {
		if r.PrsRule != nil {
			ret = append(ret, r.PrsRule)
		}
	}
	return
}

// Accept is used for the Visitor interface.
func (af *AntlrFile) Accept(v Visitor) {
	v.VisitAntlrFile(af)
}

// Merge unifies two Antlr files by combining their contents.
// Meant to be used when the lexer and parser are defined separately.
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
		Grammar: &GrammarStmt{Name: af.Grammar.Name},
		Rules:   make([]*Rule, len(af.Rules)+len(af2.Rules)),
	}

	for i, v := range af.Rules {
		ret.Rules[i] = v
	}
	for i, v := range af2.Rules {
		ret.Rules[i+len(af.Rules)] = v
	}

	return ret, nil
}

// GrammarStmt declares the name of the Antlr grammar,
// and whether the file is lexer-only, parser-only, or combined.
type GrammarStmt struct {
	LexerOnly  bool   `parser:" @'lexer'? "`
	ParserOnly bool   `parser:" @'parser'? "`
	Name       string `parser:" 'grammar' @( UpperIdent | LowerIdent) ';' "`
}

// Accept is used for the Visitor interface.
func (gs *GrammarStmt) Accept(v Visitor) {
	v.VisitGrammarStmt(gs)
}

// OptionStmt is a declaration of options for the Antlr file.
type OptionStmt struct {
	Opts []*Option `parser:" 'options' '{' @@ '}' "`
}

// Lookup retrives the value for a particular option.
func (os *OptionStmt) Lookup(key string) string {
	for _, kv := range os.Opts {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// Accept is used for the Visitor interface.
func (os *OptionStmt) Accept(v Visitor) {
	v.VisitOptionStmt(os)
}

// Option is a key-value pair.
type Option struct {
	Key   string `parser:" @(LowerIdent|UpperIdent) '=' "`
	Value string `parser:" @(LowerIdent|UpperIdent) ';' "`
}

// Accept is used for the Visitor interface.
func (o *Option) Accept(v Visitor) {
	v.VisitOption(o)
}

// Rule represents either a lexer or parser rule.
type Rule struct {
	LexRule *LexerRule  `parser:" ( @@ "`
	PrsRule *ParserRule `parser:" | @@ ) "`
}

// ParserRules is a convenience type.
type ParserRules []*ParserRule

// Accept is used for the Visitor interface.
func (pr ParserRules) Accept(v Visitor) {
	v.VisitParserRules(pr)
}

// ParserRule represents one named parser rule in the Antlr grammar.
// Parser rules begin with a lowercase letter.
type ParserRule struct {
	Name string       `parser:" @LowerIdent ':' "`
	Alt  *Alternative `parser:" @@ ';' "`
}

// Accept is used for the Visitor interface.
func (pr *ParserRule) Accept(v Visitor) {
	v.VisitParserRule(pr)
}

// LexerRules is a convenience type.
type LexerRules []*LexerRule

// Accept is used for the Visitor interface.
func (lr LexerRules) Accept(v Visitor) {
	v.VisitLexerRules(lr)
}

// LexerRule represents one named lexer rule in the Antlr grammar.
// Lexer rules begin with an uppercase letter.
type LexerRule struct {
	Fragment bool         `parser:" @'fragment'? "`
	Name     string       `parser:" @UpperIdent ':' "`
	Alt      *Alternative `parser:" @@ "`
	Skip     bool         `parser:" ('-' '>' ( @'skip' "`
	Channel  string       `parser:" | 'channel' '(' @UpperIdent ')' ) )? ';' "`
}

// Accept is used for the Visitor interface.
func (lr *LexerRule) Accept(v Visitor) {
	v.VisitLexerRule(lr)
}

// Alternative is one of the pipe-separated possible matches for a lexer or parser rule.
// In a parser rule, an alternative may also be empty, and each alternative may be labeled.
type Alternative struct {
	Exp       *Expression  `parser:" @@? "`
	Label     *string      `parser:" ('#' (@UpperIdent|@LowerIdent) )? "`
	Next      *Alternative `parser:" (  '|' @@"`
	EmptyNext bool         `parser:" | @'|' (?=';') )? "`
}

// Accept is used for the Visitor interface.
func (a *Alternative) Accept(v Visitor) {
	v.VisitAlternative(a)
}

// Expression is a potentially labeled terminal or nonterminal in the grammar.
// Alternatives are made up of one or more Expressions.
type Expression struct {
	Label   *string     `parser:" ( @(UpperIdent|LowerIdent) "`
	LabelOp *string     `parser:" @( '=' | '+' '=' ) )? "`
	Unary   *Unary      `parser:" @@ "`
	Next    *Expression `parser:" ( @@ )? "`
}

// Accept is used for the Visitor interface.
func (exp *Expression) Accept(v Visitor) {
	v.VisitExpression(exp)
}

// Unary is an operator applied to the contents of an Expression.
// Currently the only unary operator is negation.
type Unary struct {
	Op      string   `parser:" ( @( '~' ) "`
	Unary   *Unary   `parser:"     @@   ) "`
	Primary *Primary `parser:" | @@ "`
}

// Accept is used for the Visitor interface.
func (u *Unary) Accept(v Visitor) {
	v.VisitUnary(u)
}

// Primary is a terminal (string literal, regex group, character range) within an Expression.
// It can also contain a parenthesized sub-rule with one or more Alternatives.
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

// Accept is used for the Visitor interface.
func (pr *Primary) Accept(v Visitor) {
	v.VisitPrimary(pr)
}

// CharRange is a set of characters like 'a'..'z' which equates to [a-z].
type CharRange struct {
	Start string `parser:" @String '.' '.' "`
	End   string `parser:" @String "`
}

// Accept is used for the Visitor interface.
func (cr *CharRange) Accept(v Visitor) {
	v.VisitCharRange(cr)
}
