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

func TestBuild_Colon_OK(t *testing.T) {
	type grammar struct {
		TokenTypeTest bool   `  'TokenTypeTest'  :   Ident`
		DoubleCapture string `| 'DoubleCapture' ":" @Ident`
		SinglePresent bool   `| 'SinglePresent' ':'  Ident`
		SingleCapture string `| 'SingleCapture' ':' @Ident`
	}
	parser, err := participle.Build(&grammar{})
	require.NoError(t, err)
	require.Equal(t, `Grammar = "TokenTypeTest"`+
		` | ("DoubleCapture" ":" <ident>)`+
		` | ("SinglePresent" ":" <ident>)`+
		` | ("SingleCapture" ":" <ident>) .`, parser.String())
}

func TestBuild_Colon_MissingTokenType(t *testing.T) {
	type grammar struct {
		Key string `'name' : @Ident`
	}
	_, err := participle.Build(&grammar{})
	require.EqualError(t, err, `Key: expected identifier for literal type constraint but got "@"`)
}
