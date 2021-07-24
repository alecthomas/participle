package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2/_examples/capnproto/capnproto"
)

func TestExe(t *testing.T) {
	r, err := os.Open("testdata/test.capnp")
	require.NoError(t, err)

	res := &capnproto.Document{}

	err = Parser.Parse("test.capnp", r, res)
	require.NoError(t, err)
}
