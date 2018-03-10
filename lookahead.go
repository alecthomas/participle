package participle

import (
	"fmt"
	"reflect"

	"github.com/alecthomas/participle/lexer"
)

type lookahead struct {
	nodes     []node
	lookahead map[lexer.Token]*lookahead
}

func newLookahead() *lookahead {
	return &lookahead{lookahead: map[lexer.Token]*lookahead{}}
}

func (l *lookahead) match(n node) bool {
	for _, nn := range l.nodes {
		if nn == n {
			return true
		}
	}
	return false
}

func (l *lookahead) merge(n node) *lookahead {
	switch n := n.(type) {
	case *strct:
		l.nodes = append(l.nodes, n)
		return l.merge(n.expr)

	case *disjunction:
		l.nodes = append(l.nodes, n)
		for _, c := range n.nodes {
			l.merge(c)
		}
		return l

	case sequence:
		l.nodes = append(l.nodes, n)
		next := l
		for _, c := range n {
			next = next.merge(c)
		}
		return next

	case *literal:
		tok := lexer.Token{Type: n.t, Value: n.s}
		c, ok := l.lookahead[tok]
		if !ok {
			c = newLookahead()
			l.lookahead[tok] = c
		}
		c.nodes = append(c.nodes, n)
		return c

	case *reference:
		l.nodes = append(l.nodes, n)
		return l.merge(n.node)

	case *tokenReference:
		l.nodes = append(l.nodes, n)
		return l

	case *optional:
		l.nodes = append(l.nodes, n)
		return l.merge(n.node)

	default:
		panic("unsupported node type " + reflect.TypeOf(n).String())
	}
}

func reprLookahed(l *lookahead, indent string) {
	fmt.Printf("%snodes: ", indent)
	for i, n := range l.nodes {
		if i != 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%T", n)
	}
	fmt.Println()
	for t, c := range l.lookahead {
		fmt.Printf("%s(%q:%d)\n", indent, t.Value, t.Type)
		reprLookahed(c, indent+"  ")
	}
}
