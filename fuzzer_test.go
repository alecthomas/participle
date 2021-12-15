package participle_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/repr"
)

func doFuzzTest(grammar interface{}, t *testing.T) {
	parser := participle.MustBuild(grammar)

	rand.Seed(0)

	for i := 0; i < 5; i++ {
		start := time.Now()
		println("start fuzz")
		data := parser.Fuzz(lexer.DefaultDefinition.(lexer.Fuzzer))
		println("fuzz", (start.Sub(time.Now()).String()))

		err := parser.ParseString("test", data, grammar)
		if err != nil {
			t.Fatalf("error parsing (%s): %s", repr.String(data), err)
		}

		println("parse", (start.Sub(time.Now()).String()))
	}
}

func TestFuzz_LookAhead(t *testing.T) {
	type val struct {
		Str string `  @String`
		Int int    `| @Int`
	}
	type op struct {
		Op      string `@('+' | '*' (?= @Int))`
		Operand val    `@@`
	}
	type sum struct {
		Left val  `@@`
		Ops  []op `@@*`
	}

	doFuzzTest(&sum{}, t)
}

func TestFuzz_Disjunction(t *testing.T) {
	type grammar struct {
		Whatever string `'a' | @String | 'b'`
	}

	doFuzzTest(&grammar{}, t)
}
