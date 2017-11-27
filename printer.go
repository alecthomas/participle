package participle

import (
	"fmt"
	"reflect"
	"strings"
)

func dumpNode(v node) string {
	seen := map[reflect.Value]bool{}
	return nodePrinter(seen, v)
}

func nodePrinter(seen map[reflect.Value]bool, v node) string {
	if seen[reflect.ValueOf(v)] {
		return "<>"
	}
	seen[reflect.ValueOf(v)] = true
	switch n := v.(type) {
	case disjunction:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return strings.Join(out, "|")

	case *strct:
		return fmt.Sprintf("strct(type=%s, expr=%s)", n.typ, nodePrinter(seen, n.expr))

	case sequence:
		out := []string{}
		for _, n := range n {
			out = append(out, nodePrinter(seen, n))
		}
		return fmt.Sprintf("(%s)", strings.Join(out, " "))

	case *reference:
		return fmt.Sprintf("@(field=%s, node=%s)", n.field.Name, nodePrinter(seen, n.node))

	case *tokenReference:
		return fmt.Sprintf("token(%q)", n.identifier)

	case *optional:
		return fmt.Sprintf("[%s]", nodePrinter(seen, n.node))

	case *repetition:
		return fmt.Sprintf("{ %s }", nodePrinter(seen, n.node))

	case *literal:
		return n.String()

	}
	return "?"
}
