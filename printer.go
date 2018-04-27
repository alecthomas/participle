package participle

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

func dumpNode(v node) string {
	seen := map[node]bool{}
	return nodePrinter(seen, v)
}

func nodePrinter(seen map[node]bool, v node) string {
	if seen[v] {
		return "<>"
	}
	seen[v] = true
	switch n := v.(type) {
	case *disjunction:
		out := []string{}
		for _, n := range n.nodes {
			out = append(out, nodePrinter(seen, n))
		}
		return strings.Join(out, "|")

	case *strct:
		return fmt.Sprintf("strct(type=%s, expr=%s)", n.typ, nodePrinter(seen, n.expr))

	case *sequence:
		out := []string{}
		for c := n; c != nil; c = c.next {
			out = append(out, nodePrinter(seen, c.node))
		}
		return fmt.Sprintf("(%s)", strings.Join(out, " "))

	case *capture:
		return fmt.Sprintf("@(field=%s, node=%s)", n.field.Name, nodePrinter(seen, n.node))

	case *reference:
		return fmt.Sprintf("%s", n.identifier)

	case *optional:
		return fmt.Sprintf("[%s]", nodePrinter(seen, n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", nodePrinter(seen, n.node))

	case *literal:
		if n.t == lexer.EOF {
			return fmt.Sprintf("%q", n.s)
		}
		return fmt.Sprintf("%q:%s", n.s, n.tt)

	default:
		panicf("unsupported type %T", v)
		return ""
	}
}
