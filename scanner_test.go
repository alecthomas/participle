package parser

import (
	"bytes"

	"github.com/alecthomas/assert"

	"testing"
)

func TestRawScanner(t *testing.T) {
	r := bytes.NewReader([]byte("a … z"))
	actual := ScanAll(RawScanner(r))
	expected := []rune{'a', ' ', '…', ' ', 'z'}
	assert.Equal(t, expected, actual)
}

func TestSkipWhitespaceScanner(t *testing.T) {
	r := bytes.NewReader([]byte("a … z"))
	actual := ScanAll(SkipWhitespaceScanner(RawScanner(r)))
	expected := []rune{'a', '…', 'z'}
	assert.Equal(t, expected, actual)
}
