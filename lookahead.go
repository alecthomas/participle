package participle

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"sort"

	"github.com/alecthomas/participle/lexer"
)

type lroot int

type lookahead struct {
	root   int
	tokens []lexer.Token
}

func (l lookahead) String() string {
	return fmt.Sprintf("lookahead{root: %d, token: %#v}", l.root, l.tokens)
}

func (l *lookahead) hash() uint64 {
	w := fnv.New64a()
	for _, t := range l.tokens {
		fmt.Fprintf(w, "%d:%s\n", t.Type, t.Value)
	}
	return w.Sum64()
}

func buildLookahead(nodes ...node) (table []lookahead, err error) {
	l := &lookaheadWalker{limit: 16, seen: map[node]int{}}
	for root, node := range nodes {
		if node != nil {
			l.push(root, node, nil)
		}
	}
	depth := 0
	for ; depth < 16; depth++ {
		ambiguous := l.ambiguous()
		if len(ambiguous) == 0 {
			return l.collect(), nil
		}
		stepped := false
		for _, group := range ambiguous {
			for _, c := range group {
				// fmt.Printf("root=%d, depth=%d: %T %#v\n", c.root, c.depth, c.branch, c.token)
				if l.step(c.branch, c) {
					stepped = true
				}
			}
			// fmt.Println()
		}
		if !stepped {
			break
		}
	}
	// TODO: We should never fail to build lookahead.
	return nil, fmt.Errorf("possible left recursion: could not disambiguate after %d tokens of lookahead", depth)
}

type lookaheadCursor struct {
	branch node // Branch leaf was stepped from.
	lookahead
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
		n := len(out[i].tokens)
		m := len(out[j].tokens)
		if n > m {
			return true
		}
		return (n == m && len(out[i].tokens[n-1].Value) > len(out[j].tokens[m-1].Value)) || out[i].root < out[j].root
	})
	return out
}

// Find cursors that are still ambiguous.
func (l *lookaheadWalker) ambiguous() [][]*lookaheadCursor {
	grouped := map[uint64][]*lookaheadCursor{}
	for _, cursor := range l.cursors {
		key := cursor.hash()
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

func (l *lookaheadWalker) push(root int, node node, tokens []lexer.Token) {
	cursor := &lookaheadCursor{
		branch: node,
		lookahead: lookahead{
			root:   root,
			tokens: append([]lexer.Token{}, tokens...),
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

// Returns true if a step occurred or false if the cursor has already terminated.
func (l *lookaheadWalker) step(node node, cursor *lookaheadCursor) bool {
	l.seen[node]++
	if cursor.branch == nil || l.seen[node] > 32 {
		return false
	}
	switch n := node.(type) {
	case *disjunction:
		for _, c := range n.nodes {
			l.push(cursor.root, c, cursor.tokens)
		}
		l.remove(cursor)

	case *sequence:
		if n != nil {
			l.step(n.node, cursor)
			cursor.branch = n.next
		}

	case *capture:
		l.step(n.node, cursor)

	case *strct:
		l.step(n.expr, cursor)

	case *optional:
		l.push(cursor.root, n.node, cursor.tokens)
		if n.next != nil {
			l.push(cursor.root, n.next, cursor.tokens)
		}
		l.remove(cursor)

	case *repetition:
		l.push(cursor.root, n.node, cursor.tokens)
		if n.next != nil {
			l.push(cursor.root, n.next, cursor.tokens)
		}
		l.remove(cursor)

	case *parseable:

	case *literal:
		cursor.tokens = append(cursor.tokens, lexer.Token{Type: n.t, Value: n.s})
		cursor.branch = nil
		return true

	case *reference:
		cursor.tokens = append(cursor.tokens, lexer.Token{Type: n.typ})
		cursor.branch = nil

	default:
		panic(fmt.Sprintf("unsupported node type %T", n))
	}

	return true
}

func applyLookahead(m node, seen map[node]bool) {
	if seen[m] {
		return
	}
	seen[m] = true
	switch n := m.(type) {
	case *disjunction:
		lookahead, err := buildLookahead(n.nodes...)
		if err == nil {
			n.lookahead = lookahead
		} else {
			panic(Error(err.Error() + ": " + stringer(n)))
		}
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
		lookahead, err := buildLookahead(n.node, n.next)
		if err == nil {
			n.lookahead = lookahead
		} else {
			panic(Error(err.Error() + ": " + stringer(n)))
		}
		applyLookahead(n.node, seen)
		if n.next != nil {
			applyLookahead(n.next, seen)
		}
	case *repetition:
		lookahead, err := buildLookahead(n.node, n.next)
		if err == nil {
			n.lookahead = lookahead
		} else {
			panic(Error(err.Error() + ": " + stringer(n)))
		}
		applyLookahead(n.node, seen)
		if n.next != nil {
			applyLookahead(n.next, seen)
		}
	case *parseable:
	default:
		panic(fmt.Sprintf("unsupported node type %T", m))
	}
}

type lookaheadTable []lookahead

// Select node to use.
//
// Will return -2 if lookahead table is missing, -1 for no match, or index of selected node.
func (l lookaheadTable) Select(lex lexer.PeekingLexer, parent reflect.Value) (selected int) {
	if l == nil {
		return -2
	}
next:
	for _, look := range l {
		for depth, lt := range look.tokens {
			t := lex.Peek(depth)
			if !((lt.Value == "" || lt.Value == t.Value) && (lt.Type == lexer.EOF || lt.Type == t.Type)) {
				continue next
			}
		}
		return look.root
	}
	return -1
}
