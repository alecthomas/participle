package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/participle/v2/_examples/csv/csv"
	"github.com/stretchr/testify/require"
)

func TestExe(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.csv")
	require.NoError(t, err)

	for _, m := range matches {
		t.Run(m, func(t *testing.T) {
			r, err := os.Open(m)
			require.NoError(t, err)

			res := &csv.CsvFile{}

			err = Parser.Parse("m", r, res)
			require.NoError(t, err)

			err = r.Close()
			require.NoError(t, err)
		})
	}
}
