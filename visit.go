package participle

type visitorFunc func(n node, next func() error) error

func visit(n node, visitor visitorFunc) error {
	return _visit(map[node]bool{}, n, visitor)
}

func _visit(seen map[node]bool, n node, visitor visitorFunc) error {
	if seen[n] {
		return nil
	}
	seen[n] = true
	return visitor(n, func() error {
		switch n := n.(type) {
		case *strct:
			return _visit(seen, n.expr, visitor)

		case *disjunction:
			for _, c := range n.nodes {
				if err := _visit(seen, c, visitor); err != nil {
					return err
				}
			}

		case *sequence:
			for c := n; c != nil; c = c.next {
				if err := _visit(seen, c.node, visitor); err != nil {
					return err
				}
			}

		case *parseable:

		case *capture:
			return _visit(seen, n.node, visitor)

		case *reference:

		case *negation:
			return _visit(seen, n.node, visitor)

		case *literal:

		case *group:
			return _visit(seen, n.expr, visitor)

		default:
			panic("unsupported")

		}
		return nil
	})
}
