package participle

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// recoverNode is a registered error-recovery point: a production whose
// starting tokens act as synchronization points during parsing.
type recoverNode struct {
	match func(lexer.Token) bool
}

type recoveryDef struct {
	typ reflect.Type
}

// RecoverTo enables fault-tolerant parsing for the given production types.
//
// Each type must correspond to a production in the grammar (a struct type
// parsed via @@, a union type registered with Union, or a custom type
// registered with ParseTypeWith). When parsing fails anywhere in the input,
// the parser scans forward from the error position for the first token that
// could begin one of the registered productions, skips the malformed region,
// and resumes parsing from that synchronization point.
//
// The partial AST up to the failure is retained; the recovered production's
// value is not populated, only its starting token is synchronized on. If no
// synchronization token is found before EOF the original error is returned
// unchanged.
//
// Recovery never crosses structural literals — tokens that appear verbatim in
// the grammar but are not synchronization points, such as the closing
// delimiters of enclosing productions. A structural literal that is itself a
// synchronization point is used directly.
//
// Example:
//
//	type Statements struct {
//		Stmts []*Stmt `@@*`
//	}
//
//	parser := participle.MustBuild[Statements](
//		participle.RecoverTo(&Stmt{}),
//	)
func RecoverTo(nodes ...any) Option {
	return func(p *parserOptions) error {
		for _, node := range nodes {
			p.recoveryDefs = append(p.recoveryDefs, recoveryDef{typ: reflect.TypeOf(node)})
		}
		return nil
	}
}

// resolveRecovery resolves registered recovery definitions against the
// constructed node graph, computing synchronization predicates for each. It
// also collects the set of structural literals (tokens that appear verbatim
// in the grammar) so recovery does not cross them. Must be called after the
// root production has been parsed and case insensitivity has been configured.
func (p *parserOptions) resolveRecovery(rootNode node) error {
	p.recovery = make([]recoverNode, 0, len(p.recoveryDefs))
	for _, def := range p.recoveryDefs {
		t := indirectType(def.typ)
		n, ok := p.typeNodes[t]
		if !ok {
			return fmt.Errorf("RecoverTo: parser does not contain a production of type %s", def.typ)
		}
		seen := map[node]bool{}
		matches := startSet(n, seen, p.caseInsensitiveTokens)
		p.recovery = append(p.recovery, recoverNode{
			match: func(t lexer.Token) bool { return startSetAny(matches, t) },
		})
	}
	p.structuralLiterals = map[string]bool{}
	seen := map[node]bool{}
	err := visit(rootNode, func(n node, next func() error) error {
		if lit, ok := n.(*literal); ok {
			p.structuralLiterals[lit.s] = true
		}
		if seen[n] {
			return nil
		}
		seen[n] = true
		return next()
	})
	return err
}

// startSet computes the set of synchronization predicates for the tokens that
// can begin a match of n. The result is a slice of token predicates; a token
// can begin n if any predicate matches it. Nodes whose first-token set cannot
// be statically determined (custom, parseable, negation, lookahead) contribute
// nothing, so a production rooted at such a node is never a synchronization
// point.
func startSet(n node, seen map[node]bool, ci map[lexer.TokenType]bool) []func(lexer.Token) bool {
	if seen[n] {
		return nil
	}
	seen[n] = true
	defer delete(seen, n)

	switch n := n.(type) {
	case *literal:
		return []func(lexer.Token) bool{
			func(t lexer.Token) bool {
				equal := t.Value == n.s
				if ci[t.Type] && n.s != "" {
					equal = strings.EqualFold(t.Value, n.s)
				}
				return (n.t == lexer.EOF || n.t == t.Type) && equal
			},
		}

	case *reference:
		return []func(lexer.Token) bool{
			func(t lexer.Token) bool { return t.Type == n.typ },
		}

	case *strct:
		return startSet(n.expr, seen, ci)

	case *disjunction:
		var out []func(lexer.Token) bool
		for _, alt := range n.nodes {
			out = append(out, startSet(alt, seen, ci)...)
		}
		return out

	case *sequence:
		out := startSet(n.node, seen, ci)
		if nullable(n.node, seen) {
			out = append(out, startSet(n.next, seen, ci)...)
		}
		return out

	case *capture:
		return startSet(n.node, seen, ci)

	case *group:
		// A group can always begin its inner expression, regardless of mode.
		return startSet(n.expr, seen, ci)

	case *union:
		var out []func(lexer.Token) bool
		for _, member := range n.disjunction.nodes {
			out = append(out, startSet(member, seen, ci)...)
		}
		return out

	case *custom, *parseable, *negation, *lookaheadGroup:
		return nil
	}
	panic(fmt.Sprintf("unhandled node type %T", n))
}

// startSetAny reports whether any predicate in matches accepts t.
func startSetAny(matches []func(lexer.Token) bool, t lexer.Token) bool {
	for _, match := range matches {
		if match(t) {
			return true
		}
	}
	return false
}

// nullable reports whether n can match the empty input. Used to propagate
// first-token sets through sequences with nullable heads.
func nullable(n node, seen map[node]bool) bool {
	if n == nil || seen[n] {
		return false
	}
	seen[n] = true
	defer delete(seen, n)

	switch n := n.(type) {
	case *literal, *reference, *custom, *parseable, *negation, *lookaheadGroup:
		return false

	case *strct:
		return nullable(n.expr, seen)

	case *disjunction:
		for _, alt := range n.nodes {
			if nullable(alt, seen) {
				return true
			}
		}
		return false

	case *sequence:
		if !nullable(n.node, seen) {
			return false
		}
		return n.next == nil || nullable(n.next, seen)

	case *capture:
		return nullable(n.node, seen)

	case *group:
		switch n.mode {
		case groupMatchZeroOrOne, groupMatchZeroOrMore:
			return true
		case groupMatchOnce, groupMatchOneOrMore, groupMatchNonEmpty:
			return nullable(n.expr, seen)
		}
		return false

	case *union:
		for _, member := range n.disjunction.nodes {
			if nullable(member, seen) {
				return true
			}
		}
		return false
	}
	panic(fmt.Sprintf("unhandled node type %T", n))
}

// maybeRecover attempts to recover from a parse failure by scanning forward
// for the first token that can begin a registered recovery production.
//
// scan is the context to scan (the caller's speculative branch for repetition
// loops). failed is the token at the scan cursor that failed to parse; it is
// skipped before searching.
//
// consume selects the resume semantics:
//   - true: the synchronization token is consumed, guaranteeing forward
//     progress even if the surrounding grammar cannot parse it; used when the
//     caller will re-attempt parsing from after the sync token.
//   - false: the synchronization token is not consumed, so the caller can
//     resume parsing from it; the caller's next iteration is expected to
//     consume it.
//
// Recovery never crosses structural literals (tokens that appear verbatim in
// the grammar but are not synchronization points), such as the closing
// delimiters of enclosing productions.
//
// A recovery is only attempted if it can make forward progress beyond the
// last recovery point, which prevents non-terminating recovery loops.
//
// On success error state is cleared and true is returned.
func (p *parseContext) maybeRecover(scan *parseContext, failed *lexer.Token, consume bool) bool {
	if len(p.recovery) == 0 {
		return false
	}
	start := scan.RawCursor()
	if !consume && start <= p.lastRecovery {
		return false
	}
	if failed != nil && !failed.EOF() {
		if p.structural[failed.Value] && !p.isSyncToken(*failed) {
			return false
		}
		if p.isSyncToken(*failed) {
			if consume {
				scan.Next()
			}
			p.deepestError = nil
			p.deepestErrorDepth = 0
			p.lastRecovery = scan.RawCursor()
			return true
		}
		scan.Next() // Skip the token that failed to parse.
	} else {
		scan.Next()
	}
	for {
		sync := scan.MakeCheckpoint()
		tok := scan.Next()
		if tok.EOF() {
			return false
		}
		for _, rn := range p.recovery {
			if rn.match(*tok) {
				if !consume {
					// Restore the cursor to the synchronization token so the
					// caller can resume parsing from it.
					scan.LoadCheckpoint(sync)
				}
				p.deepestError = nil
				p.deepestErrorDepth = 0
				p.lastRecovery = scan.RawCursor()
				return true
			}
		}
		if p.structural[tok.Value] {
			return false
		}
	}
}

// isSyncToken reports whether t can begin any registered recovery production.
func (p *parseContext) isSyncToken(t lexer.Token) bool {
	for _, rn := range p.recovery {
		if rn.match(t) {
			return true
		}
	}
	return false
}
