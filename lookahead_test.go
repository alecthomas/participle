package participle

import (
	"testing"
)

func TestComputeLookahead(t *testing.T) {
	type grammar struct {
		a string `  "hello" @"world"`
		b string `| "hello" @"there"`
	}
	p := mustTestParser(t, grammar{})
	l := newLookahead()
	l.merge(p.root)
	reprLookahed(l, "")
}
