package main

import (
	"github.com/alecthomas/repr"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// https://www.it.uu.se/katalog/aleji304/CompilersProject/uc.html
//
// program         ::= topdec_list
// topdec_list     ::= /empty/ | topdec topdec_list
// topdec          ::= vardec ";"
//                  | funtype ident "(" formals ")" funbody
// vardec          ::= scalardec | arraydec
// scalardec       ::= typename ident
// arraydec        ::= typename ident "[" intconst "]"
// typename        ::= "int" | "char"
// funtype         ::= typename | "void"
// funbody         ::= "{" locals stmts "}" | ";"
// formals         ::= "void" | formal_list
// formal_list     ::= formaldec | formaldec "," formal_list
// formaldec       ::= scalardec | typename ident "[" "]"
// locals          ::= /empty/ | vardec ";" locals
// stmts           ::= /empty/ | stmt stmts
// stmt            ::= expr ";"
//                  | "return" expr ";" | "return" ";"
//                  | "while" condition stmt
//                  | "if" condition stmt else_part
//                  | "{" stmts "}"
//                  | ";"
// else_part       ::= /empty/ | "else" stmt
// condition       ::= "(" expr ")"
// expr            ::= intconst
//                  | ident | ident "[" expr "]"
//                  | unop expr
//                  | expr binop expr
//                  | ident "(" actuals ")"
//                  | "(" expr ")"
// unop            ::= "-" | "!"
// binop           ::= "+" | "-" | "*" | "/"
//                  | "<" | ">" | "<=" | ">=" | "!=" | "=="
//                  | "&&"
//                  | "="
// actuals         ::= /empty/ | expr_list
// expr_list       ::= expr | expr "," expr_list

type Program struct {
	Pos lexer.Position

	TopDec []*TopDec `@@*`
}

type TopDec struct {
	Pos lexer.Position

	FunDec *FunDec `  @@`
	VarDec *VarDec `| @@ ";"`
}

type VarDec struct {
	Pos lexer.Position

	ArrayDec  *ArrayDec  `  @@`
	ScalarDec *ScalarDec `| @@`
}

type ScalarDec struct {
	Pos lexer.Position

	Type string `@Type`
	Name string `@Ident`
}

type ArrayDec struct {
	Pos  lexer.Position
	Type string `@Type`
	Name string `@Ident`
	Size int    `"[" @Int "]"`
}

type ReturnStmt struct {
	Pos lexer.Position

	Result *Expr `"return" @@?`
}

type WhileStmt struct {
	Pos lexer.Position

	Condition *Expr `"while" "(" @@ ")"`
	Body      *Stmt `@@`
}

type IfStmt struct {
	Pos lexer.Position

	Condition *Expr `"if" "(" @@ ")"`
	Body      *Stmt `@@`
	Else      *Stmt `("else" @@)?`
}

type Stmts struct {
	Pos lexer.Position

	Stmts []*Stmt `@@*`
}

type Stmt struct {
	Pos lexer.Position

	IfStmt     *IfStmt     `  @@`
	ReturnStmt *ReturnStmt `| @@`
	WhileStmt  *WhileStmt  `| @@`
	Block      *Stmts      `| "{" @@ "}"`
	Expr       *Expr       `| @@`
	Empty      bool        `| @";"`
}

type FunBody struct {
	Pos lexer.Position

	Locals []*VarDec `(@@ ";")*`
	Stmts  *Stmts    `@@`
}

type FunDec struct {
	Pos lexer.Position

	ReturnType string       `@(Type | "void")`
	Name       string       `@Ident`
	Parameters []*Parameter `"(" ((@@ ("," @@)*) | "void") ")"`
	FunBody    *FunBody     `(";" | "{" @@ "}")`
}

type Parameter struct {
	Pos lexer.Position

	Array  *ArrayParameter `  @@`
	Scalar *ScalarDec      `| @@`
}

type ArrayParameter struct {
	Pos lexer.Position

	Type  string `@Type`
	Ident string `@Ident "[" "]"`
}

type Expr struct {
	Pos lexer.Position

	Assignment *Assignment `@@`
}

type Assignment struct {
	Pos lexer.Position

	Equality *Equality `@@`
	Op       string    `( @"="`
	Next     *Equality `  @@ )?`
}

type Equality struct {
	Pos lexer.Position

	Comparison *Comparison `@@`
	Op         string      `[ @( "!" "=" | "=" "=" )`
	Next       *Equality   `  @@ ]`
}

type Comparison struct {
	Pos lexer.Position

	Addition *Addition   `@@`
	Op       string      `[ @( ">" "=" | ">" | "<" "=" | "<" )`
	Next     *Comparison `  @@ ]`
}

type Addition struct {
	Pos lexer.Position

	Multiplication *Multiplication `@@`
	Op             string          `[ @( "-" | "+" )`
	Next           *Addition       `  @@ ]`
}

type Multiplication struct {
	Pos lexer.Position

	Unary *Unary          `@@`
	Op    string          `[ @( "/" | "*" )`
	Next  *Multiplication `  @@ ]`
}

type Unary struct {
	Pos lexer.Position

	Op      string   `  ( @( "!" | "-" )`
	Unary   *Unary   `    @@ )`
	Primary *Primary `| @@`
}

type Primary struct {
	Pos lexer.Position

	Number        *int        `  @Int`
	ArrayIndex    *ArrayIndex `| @@`
	CallFunc      *CallFunc   `| @@`
	Ident         string      `| @Ident`
	SubExpression *Expr       `| "(" @@ ")" `
}

type ArrayIndex struct {
	Pos lexer.Position

	Ident string  `@Ident`
	Index []*Expr `("[" @@ "]")+`
}

type CallFunc struct {
	Pos lexer.Position

	Ident string  `@Ident`
	Index []*Expr `"(" (@@ ("," @@)*)? ")"`
}

var (
	lex = lexer.MustSimple([]lexer.SimpleRule{
		{"comment", `//.*|/\*.*?\*/`},
		{"whitespace", `\s+`},

		{"Type", `\b(int|char)\b`},
		{"Ident", `\b([a-zA-Z_][a-zA-Z0-9_]*)\b`},
		{"Punct", `[-,()*/+%{};&!=:<>]|\[|\]`},
		{"Int", `\d+`},
	})
	parser = participle.MustBuild[Program](
		participle.Lexer(lex),
		participle.UseLookahead(2))
)

const sample = `
/* This is an example uC program. */
void putint(int i);

int fac(int n)
{
    if (n < 2)
        return n;
    return n * fac(n - 1);
}

int sum(int n, int a[])
{
    int i;
    int s;

    i = 0;
    s = 0;
    while (i <= n) {
        s = s + a[i];
        i = i + 1;
    }
    return s;
}

int main(void)
{
    int a[2];

    a[0] = fac(5);
    a[1] = 27;
    putint(sum(2, a)); // prints 147
    return 0;
}
`

func main() {
	ast, err := parser.ParseString("", sample)
	repr.Println(ast)
	if err != nil {
		panic(err)
	}
}
