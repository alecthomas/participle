package participle

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

type stringerVisitor struct {
	bytes.Buffer
	seen map[node]bool
}

func stringer(n node) string {
	v := &stringerVisitor{seen: map[node]bool{}}
	v.visit(n, 1, false)
	return v.String()
}

func (s *stringerVisitor) visit(n node, depth int, disjunctions bool) {
	if s.seen[n] || depth <= 0 {
		fmt.Fprintf(s, "...")
		return
	}
	s.seen[n] = true

	switch n := n.(type) {
	case *disjunction:
		if disjunctions {
			fmt.Fprintf(s, "(")
		}
		for i, c := range n.nodes {
			if i > 0 {
				fmt.Fprint(s, " | ")
			}
			s.visit(c, depth, disjunctions || len(n.nodes) > 1)
		}
		if disjunctions {
			fmt.Fprintf(s, ")")
		}

	case *strct:
		s.visit(n.expr, depth, disjunctions)

	case *sequence:
		for c, i := n, 0; c != nil && depth-i > 0; c, i = c.next, i+1 {
			if c != n {
				fmt.Fprint(s, " ")
			}
			s.visit(c.node, depth-i, disjunctions)
		}

	case *parseable:
		fmt.Fprint(s, n.t.Name())

	case *capture:
		if _, ok := n.node.(*parseable); ok {
			fmt.Fprint(s, n.field.Name)
		} else {
			if n.node == nil {
				fmt.Fprintf(s, "<%s>", strings.ToLower(n.field.Name))
			} else {
				s.visit(n.node, depth, disjunctions)
			}
		}

	case *reference:
		fmt.Fprintf(s, "<%s>", strings.ToLower(n.identifier))

	case *optional:
		fmt.Fprint(s, "[ ")
		s.visit(n.node, depth, disjunctions)
		fmt.Fprint(s, " ]")
		if n.next != nil {
			fmt.Fprint(s, " ")
			s.visit(n.next, depth, disjunctions)
		}

	case *repetition:
		fmt.Fprint(s, "( ")
		s.visit(n.node, depth, disjunctions)
		fmt.Fprint(s, " )")

	case *literal:
		fmt.Fprintf(s, "%q", n.s)
		if n.t != lexer.EOF && n.s == "" {
			fmt.Fprintf(s, ":%s", n.tt)
		}
	}
}
