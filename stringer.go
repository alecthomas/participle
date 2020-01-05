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

func stringern(n node, depth int) string {
	v := &stringerVisitor{seen: map[node]bool{}}
	v.visit(n, depth)
	return v.String()
}

func stringer(n node) string {
	return stringern(n, 1)
}

func (s *stringerVisitor) visit(n node, depth int) { // nolint: gocognit
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
		c := n
		for i := 0; c != nil && depth-i > 0; c, i = c.next, i+1 {
			if c != n {
				fmt.Fprint(s, " ")
			}
			s.visit(c.node, depth-i)
		}

	case *parseable:
		fmt.Fprintf(s, "<%s>", strings.ToLower(n.t.Name()))

	case *capture:
		if _, ok := n.node.(*parseable); ok {
			fmt.Fprintf(s, "<%s>", strings.ToLower(n.field.Name))
		} else {
			if n.node == nil {
				fmt.Fprintf(s, "<%s>", strings.ToLower(n.field.Name))
			} else {
				s.visit(n.node, depth)
			}
		}

	case *reference:
		fmt.Fprintf(s, "<%s>", strings.ToLower(n.identifier))

	case *optional:
		composite := compositeNode(map[node]bool{}, n)
		if composite {
			fmt.Fprint(s, "(")
		}
		s.visit(n.node, depth)
		if composite {
			fmt.Fprint(s, ")")
		}
		fmt.Fprint(s, "?")

	case *repetition:
		composite := compositeNode(map[node]bool{}, n)
		if composite {
			fmt.Fprint(s, "(")
		}
		s.visit(n.node, depth)
		if composite {
			fmt.Fprint(s, ")")
		}
		fmt.Fprint(s, "*")

	case *literal:
		fmt.Fprintf(s, "%q", n.s)
		if n.t != lexer.EOF && n.s == "" {
			fmt.Fprintf(s, ":%s", n.tt)
		}

	case *group:
		composite := (n.mode != groupMatchOnce) && compositeNode(map[node]bool{}, n)

		if composite {
			fmt.Fprint(s, "(")
		}
		if child, ok := n.expr.(*group); ok && child.mode == groupMatchOnce {
			s.visit(child.expr, depth)
		} else if child, ok := n.expr.(*capture); ok {
			if grandchild, ok := child.node.(*group); ok && grandchild.mode == groupMatchOnce {
				s.visit(grandchild.expr, depth)
			} else {
				s.visit(n.expr, depth)
			}
		} else {
			s.visit(n.expr, depth)
		}
		if composite {
			fmt.Fprint(s, ")")
		}
		switch n.mode {
		case groupMatchNonEmpty:
			fmt.Fprintf(s, "!")
		case groupMatchZeroOrOne:
			fmt.Fprintf(s, "?")
		case groupMatchZeroOrMore:
			fmt.Fprintf(s, "*")
		case groupMatchOneOrMore:
			fmt.Fprintf(s, "+")
		}

	default:
		panic("unsupported")
	}
}

func compositeNode(seen map[node]bool, n node) bool {
	if n == nil || seen[n] {
		return false
	}
	seen[n] = true

	switch n := n.(type) {
	case *sequence:
		return n.next != nil

	case *disjunction:
		for _, c := range n.nodes {
			if compositeNode(seen, c) {
				return true
			}
		}
		return false

	case *reference, *literal, *parseable:
		return false

	case *strct:
		return compositeNode(seen, n.expr)

	case *capture:
		return compositeNode(seen, n.node)

	case *optional:
		return compositeNode(seen, n.node)

	case *repetition:
		return compositeNode(seen, n.node)

	case *group:
		return compositeNode(seen, n.expr)

	default:
		panic("unsupported")
	}
}
