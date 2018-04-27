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

func stringer(n node) string {
	v := &stringerVisitor{seen: map[node]bool{}}
	v.visit(n)
	return v.String()
}

func (s *stringerVisitor) visit(n node) {
	if s.seen[n] {
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
			s.visit(c)
		}

	case *strct:
		s.visit(n.expr)

	case *sequence:
		s.visit(n.node)
		if n.next != nil {
			fmt.Fprint(s, " ")
			s.visit(n.next)
		}

	case *capture:
		s.visit(n.node)

	case *reference:
		fmt.Fprintf(s, "%s", n.identifier)

	case *optional:
		fmt.Fprint(s, "[ ")
		s.visit(n.node)
		fmt.Fprint(s, " ]")

	case *repetition:
		fmt.Fprint(s, "( ")
		s.visit(n.node)
		fmt.Fprint(s, " )")

	case *literal:
		fmt.Fprintf(s, "%q", n.s)
		if n.t != lexer.EOF {
			fmt.Fprintf(s, ":%s", n.tt)
		}
	}
}
