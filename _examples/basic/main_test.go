package main

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	src := `5  REM inputting the argument
10  PRINT "Factorial of:"
20  INPUT A
30  LET B = 1
35  REM beginning of the loop
40  IF A <= 1 THEN 80
50  LET B = B * A
60  LET A = A - 1
70  GOTO 40
75  REM prints the result
80  PRINT B
`
	_, err := Parse(strings.NewReader(src))
	assert.NoError(t, err)
}
