package stateful

import (
	"log"
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternStartingPatterns(t *testing.T) {
	type PatternTest struct {
		pattern     string
		errexpected bool
	}

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
			{`(?i)abcd`, false},
		}
	)

	for _, tst := range tests {
		cmp, err := computeRuneRanges(tst.pattern)
		if !tst.errexpected {
			assert.NoError(t, err)
		}
		if tst.errexpected && err == nil || !tst.errexpected && err != nil {
			assert.Error(t, err)
			log.Printf(`/%s/ (%v) => %v | %v`, tst.pattern, cmp.nbmatch, cmp.runes, err)
		}
	}
	var e, _ = syntax.Parse(`(?i)abcd`, syntax.Perl)
	log.Print(e.Rune)
}
