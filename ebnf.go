package participle

import (
	"fmt"
	"strings"
)

// String returns the EBNF for the grammar.
//
// Productions are always upper case. Lexer tokens are always lower case.
func (p *Parser) String() string {
	seen := map[node]bool{}
	outp := []*ebnfp{}
	ebnf(true, p.root, seen, nil, &outp)
	out := []string{}
	for _, p := range outp {
		out = append(out, fmt.Sprintf("%s = %s .", p.name, p.out))
	}
	return strings.Join(out, "\n")
}

type ebnfp struct {
	name string
	out  string
}

func ebnf(root bool, n node, seen map[node]bool, p *ebnfp, outp *[]*ebnfp) {
	switch n := n.(type) {
	case *disjunction:
		if !root {
			p.out += "("
		}
		for i, next := range n.nodes {
			if i > 0 {
				p.out += " | "
			}
			ebnf(false, next, seen, p, outp)
		}
		if !root {
			p.out += ")"
		}
		return

	case *strct:
		name := strings.ToUpper(n.typ.Name()[:1]) + n.typ.Name()[1:]
		if p != nil {
			p.out += name
		}
		if seen[n] {
			return
		}
		seen[n] = true
		p = &ebnfp{name: name}
		*outp = append(*outp, p)
		ebnf(true, n.expr, seen, p, outp)
		return

	case *sequence:
		group := n.next != nil && !root
		if group {
			p.out += "("
		}
		for n != nil {
			ebnf(false, n.node, seen, p, outp)
			n = n.next
			if n != nil {
				p.out += " "
			}
		}
		if group {
			p.out += ")"
		}
		return

	case *parseable:
		p.out += n.t.Name()

	case *capture:
		ebnf(false, n.node, seen, p, outp)

	case *reference:
		p.out += "<" + strings.ToLower(n.identifier) + ">"

	case *optional:
		ebnf(false, n.node, seen, p, outp)
		p.out += "?"

	case *repetition:
		ebnf(false, n.node, seen, p, outp)
		p.out += "*"

	case *negation:
		p.out += "!"
		ebnf(false, n.node, seen, p, outp)
		return

	case *literal:
		p.out += fmt.Sprintf("%q", n.s)

	case *group:
		if child, ok := n.expr.(*group); ok && child.mode == groupMatchOnce {
			ebnf(false, child.expr, seen, p, outp)
		} else if child, ok := n.expr.(*capture); ok {
			if grandchild, ok := child.node.(*group); ok && grandchild.mode == groupMatchOnce {
				ebnf(false, grandchild.expr, seen, p, outp)
			} else {
				ebnf(false, n.expr, seen, p, outp)
			}
		} else {
			ebnf(false, n.expr, seen, p, outp)
		}
		switch n.mode {
		case groupMatchNonEmpty:
			p.out += "!"
		case groupMatchZeroOrOne:
			p.out += "?"
		case groupMatchZeroOrMore:
			p.out += "*"
		case groupMatchOneOrMore:
			p.out += "+"
		}
		return

	default:
		panic(fmt.Sprintf("unsupported node type %T", n))
	}
}
