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
	if s.seen[n] || depth == 0 {
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
			s.visit(c, depth-1)
		}

	case *strct:
		s.visit(n.expr, depth-1)

	case *sequence:
		s.visit(n.node, depth-1)
		if n.next != nil {
			fmt.Fprint(s, " ")
			s.visit(n.next, depth-1)
		}

	case *capture:
		s.visit(n.node, depth-1)

	case *reference:
		fmt.Fprintf(s, "%s", n.identifier)

	case *optional:
		fmt.Fprint(s, "[ ")
		s.visit(n.node, depth-1)
		fmt.Fprint(s, " ]")

	case *repetition:
		fmt.Fprint(s, "( ")
		s.visit(n.node, depth-1)
		fmt.Fprint(s, " )")

	case *literal:
		fmt.Fprintf(s, "%q", n.s)
		if n.t != lexer.EOF {
			fmt.Fprintf(s, ":%s", n.tt)
		}
	}
}
