package participle_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alecthomas/participle/v2"
)

type leftRecursionSimple struct {
	Begin string               `  @Ident`
	More  *leftRecursionSimple `| @@ "more"`
}

func TestValidateLeftRecursion(t *testing.T) {
	_, err := participle.Build(&leftRecursionSimple{})
	require.Error(t, err)
	require.Equal(t, err.Error(), `left recursion detected on

  LeftRecursionSimple = <ident> | (LeftRecursionSimple "more") .`)
}

type leftRecursionNestedInner struct {
	Begin string               `  @Ident`
	Next  *leftRecursionNested `| @@`
}

type leftRecursionNested struct {
	Begin string                    `  @Ident`
	More  *leftRecursionNestedInner `| @@ "more"`
}

func TestValidateLeftRecursionNested(t *testing.T) {
	_, err := participle.Build(&leftRecursionNested{})
	require.Error(t, err)
	require.Equal(t, err.Error(), `left recursion detected on

  LeftRecursionNested = <ident> | (LeftRecursionNestedInner "more") .
  LeftRecursionNestedInner = <ident> | LeftRecursionNested .`)
}
