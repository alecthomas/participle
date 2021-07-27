package antlr

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/alecthomas/participle/v2/antlr/ast"
	"github.com/alecthomas/participle/v2/antlr/gen"
)

// ParticipleFromAntlr produces Go source code from an Antlr grammar AST.
// The code includes a Participle lexer and parse objects.
func ParticipleFromAntlr(antlr *ast.AntlrFile, w io.Writer) error {

	rulebody, parseObjs, root, err := compute(antlr)
	if err != nil {
		return err
	}

	// Load template
	tmpl, err := template.New("tmpl").Parse(outputTemplate)
	if err != nil {
		return err
	}

	// Render template
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, map[string]interface{}{
		"Grammar":      strings.ToLower(antlr.Grammar.Name),
		"Rules":        rulebody,
		"ParseObjs":    parseObjs,
		"RootParseObj": fmt.Sprintf("&%s{},", toCamel(root.Name)),
	}); err != nil {
		return err
	}

	// Format result
	b := buf.Bytes()
	frm, err := format.Source(b)
	if err != nil {
		log.Print(err)
		frm = b
	}

	_, err = w.Write(frm)
	return err
}

func compute(antlr *ast.AntlrFile) (lexRulesStr, parseObjs string, root *ast.ParserRule, err error) {
	// Compute each lexer rule
	lexRules := antlr.LexRules()
	rm := map[string]*ast.LexerRule{}
	for _, lr := range lexRules {
		rm[lr.Name] = lr
	}
	lv := NewLexerVisitor(rm)
	var lexResults []gen.LexerRule
	for _, r := range lexRules {
		if r.Fragment {
			continue
		}
		name := r.Name
		if r.Skip || r.Channel != "" {
			name = strings.ToLower(name)
		}

		res := lv.Visit(r)
		res.Name = name

		lexResults = append(lexResults, res)
	}

	// Compute the parser structs
	sv := NewAntlrVisitor()
	sv.VisitAntlrFile(antlr)
	structs := gen.NewTypeExtractor(sv.Structs).Extract()
	objs := make([]string, len(structs))
	for i, v := range structs {
		new(gen.FieldRenamer).VisitStruct(v)
		objs[i] = new(gen.Printer).Visit(v)
	}
	parseObjs = strings.Join(objs, "\n")

	// Add any lexer rules needed based on undeclared lexer rules used in the parsing rules -_-
	for l := range sv.LexerTokens {
		var found bool
		for _, rule := range lv.RuleMap {
			if l == rule.Name {
				found = true
				break
			}
		}
		if !found {
			lexResults = append(lexResults, gen.LexerRule{
				Name:    l,
				Content: escapeRegexMeta(l),
				Length:  len(l),
			})
		}
	}

	// Add any lexer rules needed based on literals in the parsing rules
	literals := dedupAndSort(sv.Literals)
	for _, lit := range literals {
		var found bool
		for _, rule := range lv.RuleMap {
			if lit == lv.Visit(rule).Content {
				found = true
				break
			}
		}
		if !found {
			lexResults = append(lexResults, gen.LexerRule{
				Name:    fmt.Sprintf("XXX__LITERAL_%s", toCamel(saySymbols(lit))),
				Content: escapeRegexMeta(lit),
				Length:  len(lit),
			})
		}
	}

	// sort.SliceStable(lexResults, func(i, j int) bool {
	// 	l, r := lexResults[i], lexResults[j]
	// 	switch {
	// 	case !l.NotLiteral && r.NotLiteral:
	// 		return true
	// 	case l.NotLiteral && !r.NotLiteral:
	// 		return false
	// 	default:
	// 		return l.Length < r.Length
	// 	}
	// })

	for _, lr := range lexResults {
		// lexRules += fmt.Sprintf(`{"%s", %s, nil}, // Length: %d`, lr.Name, "`"+lr.Content+"`", lr.Length) + "\n"
		lexRulesStr += fmt.Sprintf(`{"%s", %s, nil},`, lr.Name, "`"+lr.Content+"`") + "\n"
	}

	// Determine the root parser struct
	for _, v := range antlr.PrsRules() {
		if sv.ChildRuleCounters[v.Name] == 0 {
			root = v
			break
		}
	}
	if root == nil {
		err = fmt.Errorf("could not locate root")
	}

	return
}

func dedupAndSort(s []string) []string {
	dedup := map[string]struct{}{}
	for _, el := range s {
		dedup[el] = struct{}{}
	}
	sorted := make([]string, 0, len(dedup))
	for key := range dedup {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	return sorted
}
