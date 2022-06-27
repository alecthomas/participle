package main

import (
	"strings"
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	program, err := parser.ParseString("", sample)
	require.NoError(t, err)
	repr.Println(program)
}

func BenchmarkParser(b *testing.B) {
	src := strings.Repeat(sample, 10)
	b.ReportAllocs()
	b.ReportMetric(float64(len(src)*b.N), "B/s")
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseString("", src)
	}
}
