/// If any rules or lexer codegen logic are updated, then
/// regenerate the lexers by running `refresh-codegen`
package internal

import "github.com/alecthomas/participle/v2/lexer"

var ARules = lexer.Rules{
	"Root": {
		{Name: "A", Pattern: `a`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
		{Name: "whitespace", Pattern: `\s+`, Action: nil},
	},
}

var BasicRules = lexer.Rules{
	"Root": []lexer.Rule{
		{Name: "String", Pattern: `"(\\"|[^"])*"`, Action: nil},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`, Action: nil},
		{Name: "Ident", Pattern: `[a-zA-Z_]\w*`, Action: nil},
		{Name: "Punct", Pattern: `[!-/:-@[-` + "`" + `{-~]+`, Action: nil},
		{Name: "EOL", Pattern: `\n`, Action: nil},
		{Name: "Comment", Pattern: `(?i)rem[^\n]*\n`, Action: nil},
		{Name: "Whitespace", Pattern: `[ \t]+`, Action: nil},
	},
}

var HeredocRules = lexer.Rules{
	"Root": {
		{Name: "Heredoc", Pattern: `<<(\w+\b)`, Action: lexer.Push("Heredoc")},
		lexer.Include("Common"),
	},
	"Heredoc": {
		{Name: "End", Pattern: `\b\1\b`, Action: lexer.Pop()},
		lexer.Include("Common"),
	},
	"Common": {
		{Name: "whitespace", Pattern: `\s+`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
	},
}

var HeredocWithWhitespaceRules = lexer.Rules{
	"Root": {
		{Name: "Heredoc", Pattern: `<<(\w+\b)`, Action: lexer.Push("Heredoc")},
		lexer.Include("Common"),
	},
	"Heredoc": {
		{Name: "End", Pattern: `\b\1\b`, Action: lexer.Pop()},
		lexer.Include("Common"),
	},
	"Common": {
		{Name: "Whitespace", Pattern: `\s+`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
	},
}

var InterpolatedRules = lexer.Rules{
	"Root": {
		{Name: `String`, Pattern: `"`, Action: lexer.Push("String")},
	},
	"String": {
		{Name: "Escaped", Pattern: `\\.`, Action: nil},
		{Name: "StringEnd", Pattern: `"`, Action: lexer.Pop()},
		{Name: "Expr", Pattern: `\${`, Action: lexer.Push("Expr")},
		{Name: "Char", Pattern: `[^$"\\]+`, Action: nil},
	},
	"Expr": {
		lexer.Include("Root"),
		{Name: `whitespace`, Pattern: `\s+`, Action: nil},
		{Name: `Oper`, Pattern: `[-+/*%]`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
		{Name: "ExprEnd", Pattern: `}`, Action: lexer.Pop()},
	},
}

var InterpolatedWithWhitespaceRules = lexer.Rules{
	"Root": {
		{Name: `String`, Pattern: `"`, Action: lexer.Push("String")},
	},
	"String": {
		{Name: "Escaped", Pattern: `\\.`, Action: nil},
		{Name: "StringEnd", Pattern: `"`, Action: lexer.Pop()},
		{Name: "Expr", Pattern: `\${`, Action: lexer.Push("Expr")},
		{Name: "Char", Pattern: `[^$"\\]+`, Action: nil},
	},
	"Expr": {
		lexer.Include("Root"),
		{Name: `Whitespace`, Pattern: `\s+`, Action: nil},
		{Name: `Oper`, Pattern: `[-+/*%]`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
		{Name: "ExprEnd", Pattern: `}`, Action: lexer.Pop()},
	},
}

var ReferenceRules = lexer.Rules{
	"Root": {
		{Name: "Ident", Pattern: `\w+`, Action: lexer.Push("Reference")},
		{Name: "whitespace", Pattern: `\s+`, Action: nil},
	},
	"Reference": {
		{Name: "Dot", Pattern: `\.`, Action: nil},
		{Name: "Ident", Pattern: `\w+`, Action: nil},
		lexer.Return(),
	},
}
