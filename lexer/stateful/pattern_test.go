package stateful

import (
	"log"
	"testing"
)

type PatternTest struct {
	pattern     string
	errexpected bool
}

func TestPatternStartingPatterns(t *testing.T) {
	var (
		tests = []PatternTest{
			{`^`, true},
			{`"`, false},
			{`\W`, false},
			{`^a`, false},
			{`^(a|[bc]|\w+|')`, false},
			{`\${`, false},
			{`[^$"\\]+`, false},
			{`\s+`, false},
			{`[-+/*%]`, false},
			{`[[:alnum:]]`, false},
			{`<<(\w+)\b`, false},
			{`Ȫ`, false},
			{`$`, true},
			{`.toto|\wsdkj`, false},
			{`[^\na]`, false},
			{`abcd`, false},
		}
	)

	for _, tst := range tests {
		cmp, err := computeRuneRanges(tst.pattern)
		if tst.errexpected && err == nil || !tst.errexpected && err != nil {
			log.Printf(`/%s/ (%v) => %v | %v`, tst.pattern, cmp.nbmatch, cmp.runes, err)
		}
	}
}
