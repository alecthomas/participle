package ebnf

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/lexer"

	"github.com/stretchr/testify/require"
)

func TestTokenReader(t *testing.T) {
	r := strings.NewReader("hello world")
	tr := newTokenReader(r, lexer.Position{Column: 1, Line: 1})
	tr.Begin()
	for _, ch := range "hello" {
		rn, err := tr.Peek()
		require.NoError(t, err)
		require.Equal(t, ch, rn)
		rn, err = tr.Read()
		require.NoError(t, err)
		require.Equal(t, ch, rn)
	}
	tr.Rewind()
	for _, ch := range "hello" {
		rn, err := tr.Peek()
		require.NoError(t, err)
		require.Equal(t, ch, rn)
		rn, err = tr.Read()
		require.NoError(t, err)
		require.Equal(t, ch, rn)
	}

	rn, err := tr.Peek()
	require.NoError(t, err)
	require.Equal(t, ' ', rn)
	tr.Begin()
	rn, err = tr.Read()
	require.NoError(t, err)
	require.Equal(t, ' ', rn)
}
