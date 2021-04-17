package participle

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// Trace the parse to "w".
func Trace(w io.Writer) Option {
	return func(p *Parser) error {
		p.trace = w
		return nil
	}
}

type trace struct {
	w      io.Writer
	indent int
	node
}

func (t *trace) Parse(ctx *parseContext, parent reflect.Value) ([]reflect.Value, error) {
	tok, _ := ctx.Peek(0)
	fmt.Fprintf(t.w, "%s%q %s\n", strings.Repeat(" ", t.indent), tok, t.node.GoString())
	return t.node.Parse(ctx, parent)
}

func injectTrace(w io.Writer, indent int, n node) node {
	out := &trace{w, indent, n}
	switch n := n.(type) {
	case *disjunction:
		for i, child := range n.nodes {
			n.nodes[i] = injectTrace(w, indent+2, child)
		}
	case *strct:
		n.expr = injectTrace(w, indent+2, n.expr)
	case *sequence:
		n.node = injectTrace(w, indent+2, n.node)
		// injectTrace(w, indent, n.next)
	case *parseable:
	case *capture:
		n.node = injectTrace(w, indent+2, n.node)
	case *reference:
	case *optional:
		n.node = injectTrace(w, indent+2, n.node)
	case *repetition:
		n.node = injectTrace(w, indent+2, n.node)
	case *negation:
		n.node = injectTrace(w, indent+2, n.node)
	case *literal:
	case *group:
		n.expr = injectTrace(w, indent+2, n.expr)
	}
	return out
}
