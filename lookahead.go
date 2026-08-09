package participle

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Lookahead computes, for a production in the grammar, the mapping from each
// starting token to the disjunction alternatives that can begin with it. It
// is useful for diagnostics and for driving recovery/autocompletion.
//
// node must be a value of a type that appears as a production in the grammar
// (a struct parsed via @@, a union registered with Union, or a custom type
// registered with ParseTypeWith). Pass an instance of the type, e.g.
// parser.Lookahead(&Stmt{}).
//
// The returned map keys are token descriptors:
//   - named token references like "<ident>"
//   - literals like `"if"` or `"if":Keyword` when type-constrained
//
// Values are the 0-based indices of the disjunction alternatives that can
// begin with that token. An alternative contributes multiple keys if it has
// multiple possible starting tokens (e.g. a group).
func (p *Parser[G]) Lookahead(example any) (map[string][]int, error) {
	n, err := p.resolveProduction(example)
	if err != nil {
		return nil, err
	}
	disj, ok := unwrapDisjunction(n)
	if !ok {
		return nil, fmt.Errorf("%T does not have alternatives to look ahead on", example)
	}
	out := map[string][]int{}
	for i, alt := range disj.nodes {
		seen := map[node]bool{}
		matches := startSet(alt, seen, p.caseInsensitiveTokens)
		for _, key := range startSetDescriptors(alt, seen, p.caseInsensitiveTokens, p.typeNodes) {
			out[key] = append(out[key], i)
		}
		_ = matches
	}
	return out, nil
}

// StartTokens returns the set of tokens that can begin a match of the given
// production, as deduplicated descriptor strings (see Lookahead).
func (p *Parser[G]) StartTokens(example any) ([]string, error) {
	n, err := p.resolveProduction(example)
	if err != nil {
		return nil, err
	}
	seen := map[node]bool{}
	descs := startSetDescriptors(n, seen, p.caseInsensitiveTokens, p.typeNodes)
	sort.Strings(descs)
	return descs, nil
}

// resolveProduction resolves an example value to its grammar node.
func (p *Parser[G]) resolveProduction(example any) (node, error) {
	t := indirectType(reflect.TypeOf(example))
	n, ok := p.typeNodes[t]
	if !ok {
		return nil, fmt.Errorf("parser does not contain a production of type %s", t)
	}
	return n, nil
}

// unwrapDisjunction peels a single-element strct/union down to its
// disjunction, returning the disjunction node and true.
func unwrapDisjunction(n node) (*disjunction, bool) {
	switch n := n.(type) {
	case *disjunction:
		return n, true
	case *strct:
		return unwrapDisjunction(n.expr)
	}
	return nil, false
}

// startSetDescriptors walks a node and emits a descriptor for each distinct
// starting token. It mirrors startSet but produces string descriptors instead
// of predicates. typeNodes maps Go types to their nodes so strct/union
// references resolve once more.
func startSetDescriptors(n node, seen map[node]bool, ci map[lexer.TokenType]bool, typeNodes map[reflect.Type]node) []string {
	if seen[n] {
		return nil
	}
	seen[n] = true
	defer delete(seen, n)

	switch n := n.(type) {
	case *literal:
		d := fmt.Sprintf("%q", n.s)
		if n.t != lexer.EOF && n.tt != "" {
			d += ":" + n.tt
		}
		return []string{d}

	case *reference:
		return []string{"<" + strings.ToLower(n.identifier) + ">"}

	case *strct:
		return startSetDescriptors(n.expr, seen, ci, typeNodes)

	case *disjunction:
		var out []string
		for _, alt := range n.nodes {
			out = append(out, startSetDescriptors(alt, seen, ci, typeNodes)...)
		}
		return out

	case *sequence:
		out := startSetDescriptors(n.node, seen, ci, typeNodes)
		if nullable(n.node, seen) {
			out = append(out, startSetDescriptors(n.next, seen, ci, typeNodes)...)
		}
		return out

	case *capture:
		return startSetDescriptors(n.node, seen, ci, typeNodes)

	case *group:
		return startSetDescriptors(n.expr, seen, ci, typeNodes)

	case *union:
		var out []string
		for _, member := range n.disjunction.nodes {
			out = append(out, startSetDescriptors(member, seen, ci, typeNodes)...)
		}
		return out

	case *custom, *parseable, *negation, *lookaheadGroup:
		return nil
	}
	panic(fmt.Sprintf("unhandled node type %T", n))
}
