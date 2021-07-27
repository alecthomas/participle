package antlr

import (
	"log"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
	"github.com/alecthomas/repr"
)

// StructVisitor visits an Antlr grammar AST to build Participle
// parse objects, or more accurately, proto-structs.
//
// It works by repeatedly accumulating information and then
// coalescing it into a struct field.  It attempts to create
// proto-structs that emulate hand-rolled Participle grammars.
type StructVisitor struct {
	ast.BaseVisitor

	ChildRuleCounters map[string]int
	LexerTokens       map[string]struct{}
	Result            *gen.Struct

	optionalRules map[string]bool

	accum       gen.StructFields
	lastCapture int
	lastPipe    int
	lastClose   int
	depth       int
	breadth     int
	alts        int
	lbls        strStack
	nots        boolStack
	arities     strStack
	literals    []string
	stringables []string

	captureTopLevel bool
	recursion       int
	logLabel        string
	debug           bool
}

// NewStructVisitor returns a ready StructVisitor.
func NewStructVisitor(optRls map[string]bool, lexToks map[string]struct{}) *StructVisitor {
	return &StructVisitor{
		Result:            &gen.Struct{},
		ChildRuleCounters: map[string]int{},
		LexerTokens:       lexToks,

		optionalRules:   optRls,
		lastCapture:     -1,
		lastPipe:        -1,
		captureTopLevel: true,
	}
}

// ComputeStruct returns a Participle proto-struct made by processing one Antlr parser rule.
func (sv *StructVisitor) ComputeStruct(pr *ast.ParserRule) *gen.Struct {
	sv.Visit(pr)
	return sv.Result
}

// Visit walks Antlr AST and builds a proto-struct.
// It is suitable for use at any point of the AST.
// When it finishes walking, any trailing information in the accumulator
// is appropriately merged into an existing or new struct field.
func (sv *StructVisitor) Visit(a ast.Node) {
	sv.visit(a)
	if sv.lastCapture > -1 {
		sv.printf("Completed a VISIT, squashing remaining to index %d: %s", sv.lastCapture, repr.String(sv.accum))
		sv.Result.AddFields(sv.accum.SquashToIndex(sv.lastCapture))
		sv.accum = sv.accum[0:0]
	} else if len(sv.Result.Fields) > 0 {
		sv.printf("Completed a VISIT, appending trailing to last field: %s", repr.String(sv.accum))
		sf := sv.Result.Fields[len(sv.Result.Fields)-1]
		sf = append(gen.StructFields{sf}, sv.accum...).SquashToIndex(0)
		sv.Result.Fields[len(sv.Result.Fields)-1] = sf
	} else {
		sv.printf("Completed a VISIT, squashing trailing: %s", repr.String(sv.accum))
		sv.Result.AddFields(sv.accum.SquashToIndex(-1))
		sv.accum = sv.accum[0:0]
	}
}

func (sv *StructVisitor) visit(a interface {
	Accept(ast.Visitor)
}) {
	if a == nil || reflect.ValueOf(a) == reflect.Zero(reflect.TypeOf(a)) {
		return
	}
	a.Accept(sv)
}

// VisitParserRule builds a proto-struct from a parser rule.
func (sv *StructVisitor) VisitParserRule(pr *ast.ParserRule) {
	// If the rule is just multiple literals, rewrite the AST to group them and thereby match them all in one field.
	v := sv.doSubVisit(pr.Alt)
	if !v.didCapture() && len(v.literals) > 0 {
		lbl := saySymbols(strings.Join(v.literals, "_"))
		pr.Alt = &ast.Alternative{
			Exp: &ast.Expression{
				Label: &lbl,
				Unary: &ast.Unary{
					Primary: &ast.Primary{
						Sub: pr.Alt,
					},
				},
			},
		}
	}
	sv.Result.Name = toCamel(pr.Name)
	sv.visit(pr.Alt)
}

// VisitParserRule builds field information from an Antlr rule alternative.
func (sv *StructVisitor) VisitAlternative(a *ast.Alternative) {
	// A parser rule alternate can be empty, which marks the entire rule as optional.
	// Such an alternate doesn't impact the internals of this rule, so ignore the node and/or rewrite the tree to remove it.
	if a.Exp == nil {
		if a.Next == nil {
			return
		}
		a = a.Next
	}
	// If there are only two top-level alternates, and at least one of them captures, don't force-capture.
	if sv.captureTopLevel && sv.depth == 0 && sv.alts == 0 {
		alts := new(AltCounter).CountAlts(a)
		if alts == 2 && (sv.doSubVisit(a.Exp, "check-capture").didCapture() || sv.doSubVisit(a.Exp.Next, "check-capture-2").didCapture()) {
			sv.captureTopLevel = false
			defer func() {
				sv.captureTopLevel = true
			}()
		}
	}
	// For top-level alternates that contain no capture, capture the entire alternate, wrapping in a group if need be.
	if sv.captureTopLevel && sv.depth == 0 {
		v := sv.doSubVisit(a.Exp, "check-capture-3")
		if !v.didCapture() && a.Exp.Label == nil {
			if a.Exp.Next != nil {
				a.Exp = &ast.Expression{
					Unary: &ast.Unary{
						Primary: &ast.Primary{
							Sub: &ast.Alternative{
								Exp: a.Exp,
							},
						},
					},
				}
			}
			lbl, op := saySymbols(strings.Join(v.literals, "_")), "="
			a.Exp.Label, a.Exp.LabelOp = &lbl, &op
		}
	}
	sv.visit(a.Exp)
	if a.Next != nil {
		sv.alts++
		sv.accum = append(sv.accum, &gen.StructField{Tag: "|"})
		sv.lastPipe = len(sv.accum) - 1
		sv.visit(a.Next)
		sv.alts--
	}
}

// VisitParserRule builds field information from an Antlr rule expression.
func (sv *StructVisitor) VisitExpression(exp *ast.Expression) {
	sv.lbls.push(ifStrPtr(exp.Label))
	sv.visit(exp.Unary)
	sv.lbls.pop()
	if exp.Next != nil {
		sv.breadth++
		sv.visit(exp.Next)
		sv.breadth--
	}
}

// VisitParserRule builds field information from an Antlr rule unary operator.
func (sv *StructVisitor) VisitUnary(u *ast.Unary) {
	if u.Unary != nil {
		sv.nots.push(u.Op == "~")
		defer func() {
			sv.nots.pop()
		}()
		sv.visit(u.Unary)
	} else {
		sv.visit(u.Primary)
	}
}

// VisitParserRule builds field information from an Antlr rule terminal or sub-rule.
func (sv *StructVisitor) VisitPrimary(pr *ast.Primary) {
	if pr == nil {
		return
	}
	not := sv.nots.safePeek()
	arity := pr.Arity
	suffix := func() string { return arity + ifStr(pr.NonGreedy, "?") }
	switch {
	case pr.Str != nil:
		sv.literals = append(sv.literals, stripQuotes(*pr.Str))
		sv.stringables = append(sv.stringables, stripQuotes(*pr.Str))
		lbl := sv.lbls.peek()
		if lbl == "" && not {
			lbl = saySymbols(stripQuotes(*pr.Str))
		}
		if lbl != "" {
			if sv.lastCapture > -1 {
				sv.doCapture()
			}
			sv.lastCapture = len(sv.accum)
		}
		sv.accum = append(sv.accum, &gen.StructField{
			Name:    toCamel(ifStr(lbl != "" && not, "not_") + lbl),
			RawType: ifStr(lbl != "", strTern(not, "*string", "bool")),
			Tag:     ifStr(lbl != "", "@") + ifStr(not, "!") + *pr.Str + suffix(),
		})
	case pr.Ident != nil:
		if *pr.Ident == "EOF" {
			sv.accum = append(sv.accum, &gen.StructField{
				Tag: "EOF",
			})
		} else {
			sv.stringables = append(sv.stringables, *pr.Ident)
			if isLexerRule(*pr.Ident) {
				sv.LexerTokens[*pr.Ident] = struct{}{}
				if sv.lastCapture > -1 {
					sv.doCapture()
				}
				multi := strIn(orStr(sv.arities.safePeek(), pr.Arity), "*", "+")
				sf := &gen.StructField{
					Name:    toCamel(orStr(sv.lbls.peek(), ifStr(not, "not_")+*pr.Ident)),
					RawType: ifStr(multi, "[]") + "*string",
					Tag:     "@" + ifStr(not, "!") + *pr.Ident + suffix(),
				}
				sv.lastCapture = len(sv.accum)
				sv.accum = append(sv.accum, sf)
			} else {
				if not {
					panic("inverting a recursive type capture is not supported")
				}
				if sv.optionalRules[*pr.Ident] {
					switch arity {
					case "":
						arity = "?"
					case "+":
						arity = "*"
					}
				}
				multi := strIn(orStr(sv.arities.safePeek(), pr.Arity), "*", "+")
				sf := &gen.StructField{
					Name:    toCamel(orStr(sv.lbls.peek(), *pr.Ident)),
					RawType: ifStr(multi, "[]") + "*" + toCamel(*pr.Ident),
					Tag:     "@@" + suffix(),
				}
				if sv.lastCapture > -1 {
					if sv.accum[sv.lastCapture].CanMerge(sf) && sv.lastCapture >= sv.lastPipe {
						sf.RawType = "[]" + strings.TrimPrefix(sf.RawType, "[]")
					} else {
						sv.doCapture()
					}
				}
				sv.lastCapture = len(sv.accum)
				sv.accum = append(sv.accum, sf)
				sv.ChildRuleCounters[*pr.Ident]++
			}
		}
	case pr.Sub != nil:
		v := sv.doSubVisit(pr.Sub, "subexp")
		countCap := gen.NewCaptureCounter().Visit(v.Result)
		if countCap > 1 && strIn(pr.Arity, "*", "+") {
			if not {
				panic("inverting a recursive type capture is not supported")
			}
			if sv.lastCapture > -1 {
				sv.doCapture()
			}
			sv.lastCapture = -1
			name := toCamel(orStr(sv.lbls.peek(), strings.Join(v.stringables, "_")))
			v.Result.Name = name
			sv.accum = append(sv.accum, &gen.StructField{
				Name:    name,
				SubType: v.Result,
				Tag:     "@@" + suffix(),
			})
			sv.Result.AddFields(sv.accum.SquashToIndex(len(sv.accum) - 1))
			sv.accum = sv.accum[0:0]
		} else {

			sv.arities.push(pr.Arity)

			lbl := sv.lbls.peek()
			if lbl != "" {
				if sv.lastCapture > -1 {
					sv.doCapture()
				}
				sv.lastCapture = len(sv.accum)
			}
			sv.accum = append(sv.accum, &gen.StructField{
				Name:    toCamel(lbl),
				RawType: ifStr(lbl != "", strTern(countCap > 1 || new(AltCounter).CountAlts(pr.Sub) > 1, "*string", "bool")),
				Tag:     ifStr(lbl != "", "@") + ifStr(not, "!") + "(",
			})

			sv.depth++
			sv.visit(pr.Sub)
			sv.depth--

			sv.accum = append(sv.accum, &gen.StructField{
				Tag: ")" + suffix(),
			})
			sv.lastClose = len(sv.accum)

			sv.arities.pop()
		}
	}
}

func (sv *StructVisitor) doCapture() {
	splitAt := max(sv.lastCapture, sv.lastPipe-1, sv.lastClose-1)
	field := sv.accum[:splitAt+1].SquashToIndex(sv.lastCapture)
	sv.Result.AddFields(field)
	sv.accum = sv.accum[splitAt+1:]
	sv.lastPipe = -1
	sv.lastClose = -1
}

func (sv *StructVisitor) doStubField() {
	sv.Result.AddFields(sv.accum.SquashToIndex(-1))
	sv.accum = sv.accum[0:0]
	sv.lastPipe = -1
	sv.lastClose = -1
}

func (sv *StructVisitor) doSubVisit(node interface {
	Accept(ast.Visitor)
}, label ...string) *StructVisitor {
	v := NewStructVisitor(sv.optionalRules, sv.LexerTokens)
	v.captureTopLevel = false
	v.recursion = sv.recursion + 1
	v.logLabel = sv.logLabel
	if len(label) > 0 {
		v.logLabel = sv.logLabel + ifStr(sv.logLabel != "", "/") + label[0]
	}
	v.debug = sv.debug
	v.Visit(node)
	return v
}

func (sv *StructVisitor) didCapture() bool {
	return sv.Result.Fields.AreCapturing()
}

func (sv *StructVisitor) printf(s string, a ...interface{}) {
	if sv.debug {
		log.Printf(strings.Repeat("    ", sv.recursion)+ifStr(sv.logLabel != "", sv.logLabel+": ")+s, a...)
	}
}
