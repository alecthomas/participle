package participle

import (
	"fmt"
	"sort"

	"github.com/alecthomas/participle/lexer"
)

type lroot int

type lookahead struct {
	root  int
	depth int
	token lexer.Token
}

func (l lookahead) String() string {
	return fmt.Sprintf("lookahead{root: %d, depth: %d, token: %#v}", l.root, l.depth, l.token)
}

func buildLookahead(nodes ...node) (table []lookahead, err error) {
	l := &lookaheadWalker{limit: 16, seen: map[node]int{}}
	for root, node := range nodes {
		l.push(root, 0, node)
	}
	for depth := 0; depth < 16; depth++ {
		ambiguous := l.ambiguous()
		if len(ambiguous) == 0 {
			return l.collect(), nil
		}
		// Randomly step one of each ambiguous group.
		for _, group := range ambiguous {
			for _, c := range group {
				// for _, c := range ambiguous {
				// fmt.Printf("root=%d, depth=%d: %T %#v\n", c.root, c.depth, c.branch, c.token)
				l.step(c.branch, c)
			}
		}
		// }
		// fmt.Println()
	}
	return nil, fmt.Errorf("could not disambiguate lookahead up to 16 tokens ahead")
}

type lookaheadCursor struct {
	branch node // Branch leaf was stepped from.
	lookahead
}

type lookaheadGroup struct {
	depth int
	token lexer.Token
}

type lookaheadWalker struct {
	seen    map[node]int
	limit   int
	cursors []*lookaheadCursor
}

func (l *lookaheadWalker) collect() []lookahead {
	out := []lookahead{}
	for _, cursor := range l.cursors {
		out = append(out, cursor.lookahead)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].depth > out[j].depth || len(out[i].token.Value) > len(out[j].token.Value)
	})
	return out
}

// Find cursors that are still ambiguous.
func (l *lookaheadWalker) ambiguous() [][]*lookaheadCursor {
	grouped := map[lookaheadGroup][]*lookaheadCursor{}
	for _, cursor := range l.cursors {
		key := lookaheadGroup{cursor.depth, cursor.token}
		grouped[key] = append(grouped[key], cursor)
	}
	out := [][]*lookaheadCursor{}
	for _, group := range grouped {
		if len(group) > 1 {
			out = append(out, group)
		}
	}
	return out
}

func (l *lookaheadWalker) push(root, depth int, node node) {
	cursor := &lookaheadCursor{
		branch: node,
		lookahead: lookahead{
			root:  root,
			depth: depth,
			token: lexer.EOFToken,
		},
	}
	l.cursors = append(l.cursors, cursor)
	l.step(node, cursor)
}

func (l *lookaheadWalker) remove(cursor *lookaheadCursor) {
	for i, c := range l.cursors {
		if cursor == c {
			l.cursors = append(l.cursors[:i], l.cursors[i+1:]...)
		}
	}
}

func (l *lookaheadWalker) step(node node, cursor *lookaheadCursor) {
	l.seen[node]++
	if l.seen[node] > 32 {
		return
	}
	switch n := node.(type) {
	case *disjunction:
		for _, c := range n.nodes {
			l.push(cursor.root, cursor.depth, c)
		}
		l.remove(cursor)

	case *sequence:
		if n != nil {
			cursor.branch = n.next
			if !n.head {
				cursor.depth++
			}
			l.step(n.node, cursor)
		}

	case *capture:
		l.step(n.node, cursor)

	case *strct:
		l.step(n.expr, cursor)

	case *optional:
		l.step(n.node, cursor)

	case *repetition:
		l.step(n.node, cursor)

	case *parseable:

	case *literal:
		cursor.token = lexer.Token{Type: n.t, Value: n.s}

	case *reference:
		cursor.token = lexer.Token{Type: n.typ}

	default:
		panic(fmt.Sprintf("unsupported node type %T", n))
	}
}

func applyLookahead(m node, seen map[node]bool) {
	if seen[m] {
		return
	}
	seen[m] = true
	switch n := m.(type) {
	case *disjunction:
		lookahead, err := buildLookahead(n.nodes...)
		if err != nil {
			panic(Error(err.Error() + ": " + stringer(n, 1)))
		}
		n.lookahead = lookahead
		for _, c := range n.nodes {
			applyLookahead(c, seen)
		}
	case *sequence:
		for c := n; c != nil; c = c.next {
			applyLookahead(c.node, seen)
		}
	case *literal:
	case *capture:
		applyLookahead(n.node, seen)
	case *reference:
	case *strct:
		applyLookahead(n.expr, seen)
	case *optional:
		applyLookahead(n.node, seen)
	case *repetition:
		applyLookahead(n.node, seen)
	case *parseable:
	default:
		panic(fmt.Sprintf("unsupported node type %T", m))
	}
}
