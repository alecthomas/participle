package participle_test

import (
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/require"
)

func TestBuild_Errors_Negation(t *testing.T) {
	type grammar struct {
		Whatever string `'a' | ! | 'b'`
	}
	_, err := participle.Build(&grammar{})
	require.EqualError(t, err, "Whatever: unexpected token |")
}

func TestBuild_Errors_Capture(t *testing.T) {
	type grammar struct {
		Whatever string `'a' | @ | 'b'`
	}
	_, err := participle.Build(&grammar{})
	require.EqualError(t, err, "Whatever: unexpected token |")
}

func TestBuild_Errors_UnclosedGroup(t *testing.T) {
	type grammar struct {
		Whatever string `'a' | ('b' | 'c'`
	}
	_, err := participle.Build(&grammar{})
	require.EqualError(t, err, `Whatever: expected ) but got "<EOF>"`)
}

func TestBuild_Errors_LookaheadGroup(t *testing.T) {
	type grammar struct {
		Whatever string `'a' | (?? 'what') | 'b'`
	}
	_, err := participle.Build(&grammar{})
	require.EqualError(t, err, `Whatever: expected = or ! but got "?"`)
}
