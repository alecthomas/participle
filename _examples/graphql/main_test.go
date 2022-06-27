package main

import (
	"io/ioutil"
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func BenchmarkParser(b *testing.B) {
	source, err := ioutil.ReadFile("example.graphql")
	require.NoError(b, err)
	b.ReportAllocs()
	b.ReportMetric(float64(len(source)*b.N), "B/s")
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseBytes("", source)
	}
}
