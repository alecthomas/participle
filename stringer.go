package participle

import (
	"bytes"
	"fmt"

	"github.com/alecthomas/participle/lexer"
)

type stringerVisitor struct {
	bytes.Buffer
	seen map[node]bool
}

func stringer(n node, depth int) string {
	v := &stringerVisitor{seen: map[node]bool{}}
	v.visit(n, depth)
	return v.String()
}

func (s *stringerVisitor) visit(n node, depth int) {
	if s.seen[n] || depth <= 0 {
		fmt.Fprintf(s, "...")
		return
	}
	s.seen[n] = true

	switch n := n.(type) {
	case *disjunction:
		for i, c := range n.nodes {
			if i > 0 {
				fmt.Fprint(s, " | ")
			}
			s.visit(c, depth)
		}

	case *strct:
		s.visit(n.expr, depth)

	case *sequence:
		for c, i := n, 0; c != nil && depth-i > 0; c, i = c.next, i+1 {
			if c != n {
				fmt.Fprint(s, " ")
			}
			s.visit(c.node, depth-i)
		}

	case *parseable:
		fmt.Fprint(s, n.t.Name())

	case *capture:
		if _, ok := n.node.(*parseable); ok {
			fmt.Fprint(s, n.field.Name)
		} else {
			fmt.Fprintf(s, "%s<", n.field.Name)
			s.visit(n.node, depth)
			fmt.Fprint(s, ">")
		}

	case *reference:
		fmt.Fprintf(s, "%s", n.identifier)

	case *optional:
		fmt.Fprint(s, "[ ")
		s.visit(n.node, depth)
		fmt.Fprint(s, " ]")

	case *repetition:
		fmt.Fprint(s, "( ")
		s.visit(n.node, depth)
		fmt.Fprint(s, " )")

	case *literal:
		fmt.Fprintf(s, "%q", n.s)
		if n.t != lexer.EOF {
			fmt.Fprintf(s, ":%s", n.tt)
		}
	}
}
