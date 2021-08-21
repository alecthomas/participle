package main

import (
	"strings"
	"testing"

	"github.com/alecthomas/repr"
	"github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	program := &Program{}
	err := parser.ParseString("", sample, program)
	assert.NoError(t, err)
	repr.Println(program)
}

func BenchmarkParser(b *testing.B) {
	src := strings.Repeat(sample, 10)
	b.ReportAllocs()
	b.ReportMetric(float64(len(src)*b.N), "B/s")
	for i := 0; i < b.N; i++ {
		program := &Program{}
		_ = parser.ParseString("", src, program)
	}
}
