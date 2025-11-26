// Package main demonstrates error recovery in participle.
//
// Error recovery allows the parser to continue parsing after encountering errors,
// collecting multiple errors and producing a partial AST. This is particularly
// useful for IDE integration, linters, and providing comprehensive error messages.
//
// This example shows how to parse a simple programming language with deliberate
// syntax errors and recover from them using various recovery strategies.
package main

import (
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Grammar for a simple statement-based language
type Program struct {
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Pos           lexer.Position
	Recovered     bool           // Set to true if this statement was recovered
	RecoveredSpan lexer.Position // Position where recovery started

	VarDecl  *VarDecl  `parser:"  @@"`
	FuncCall *FuncCall `parser:"| @@"`
}

type VarDecl struct {
	Keyword string `parser:"@\"let\""`
	Name    string `parser:"@Ident"`
	Eq      string `parser:"@\"=\""`
	Value   *Expr  `parser:"@@"`
	Semi    string `parser:"@\";\""`
}

type FuncCall struct {
	Name string  `parser:"@Ident"`
	Args []*Expr `parser:"\"(\" (@@ (\",\" @@)*)? \")\""`
	Semi string  `parser:"@\";\""`
}

type Expr struct {
	Number *int    `parser:"  @Number"`
	String *string `parser:"| @String"`
	Ident  string  `parser:"| @Ident"`
}

var (
	simpleLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"String", `"[^"]*"`},
		{"Number", `\d+`},
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`},
		{"Punct", `[;=(),]`},
	})

	parser = participle.MustBuild[Program](
		participle.Lexer(simpleLexer),
	)
)

func main() {
	fmt.Println("=== Example 1: Valid input ===")
	runExample(`
let x = 42;
let y = 100;
print(x);
`)

	fmt.Println("\n=== Example 2: Input with errors (no recovery) ===")
	runExampleNoRecovery(`
let x = 42;
let y = ;
let z = 100;
`)

	fmt.Println("\n=== Example 3: Input with errors (with recovery) ===")
	runExample(`
let x = 42;
let y = ;
let z = 100;
`)

	fmt.Println("\n=== Example 4: Multiple errors with recovery ===")
	runExample(`
let x = 42;
let = 100;
let y = ;
print(a);
let z = 50;
`)
}

func runExample(input string) {
	fmt.Println("Input:", input)

	// Parse with error recovery enabled
	// SkipPast skips tokens until a sync token is found and consumes it.
	// This allows recovery to the next statement after encountering an error.
	ast, err := parser.ParseString("example.lang", input,
		participle.Recover(
			participle.SkipPast(";"),
		),
	)

	printResult(ast, err)
}

func runExampleNoRecovery(input string) {
	fmt.Println("Input:", input)

	// Parse WITHOUT recovery - stops at first error
	ast, err := parser.ParseString("example.lang", input)

	printResult(ast, err)
}

func printResult(ast *Program, err error) {
	// Print what we were able to parse
	if ast != nil {
		fmt.Printf("Parsed %d statements:\n", len(ast.Statements))
		for i, stmt := range ast.Statements {
			recoveredInfo := ""
			if stmt.Recovered {
				recoveredInfo = fmt.Sprintf(" [RECOVERED at %v]", stmt.RecoveredSpan)
			}

			if stmt.VarDecl != nil {
				value := "?"
				if stmt.VarDecl.Value != nil {
					if stmt.VarDecl.Value.Number != nil {
						value = fmt.Sprintf("%d", *stmt.VarDecl.Value.Number)
					} else if stmt.VarDecl.Value.String != nil {
						value = *stmt.VarDecl.Value.String
					} else if stmt.VarDecl.Value.Ident != "" {
						value = stmt.VarDecl.Value.Ident
					}
				}
				name := stmt.VarDecl.Name
				if name == "" {
					name = "<missing>"
				}
				fmt.Printf("  %d. VarDecl: let %s = %s%s\n", i+1, name, value, recoveredInfo)
			} else if stmt.FuncCall != nil {
				fmt.Printf("  %d. FuncCall: %s(...)%s\n", i+1, stmt.FuncCall.Name, recoveredInfo)
			}
		}
	} else {
		fmt.Println("No AST produced")
	}

	// Handle errors
	if err != nil {
		fmt.Println("Errors:")
		var recErr *participle.RecoveryError
		if errors.As(err, &recErr) {
			// Multiple errors were recovered
			for i, e := range recErr.Errors {
				fmt.Printf("  %d. %v\n", i+1, e)
			}
		} else {
			// Single error
			fmt.Printf("  - %v\n", err)
		}
	} else {
		fmt.Println("No errors!")
	}
}
