package main

import (
	"strings"
	"testing"
)

func BenchmarkParser(b *testing.B) {
	src := strings.Repeat(sample, 10)
	b.ReportAllocs()
	b.ReportMetric(float64(len(src)*b.N), "B/s")
	for i := 0; i < b.N; i++ {
		program := &Program{}
		_ = parser.ParseString("", src, program)
	}
}
