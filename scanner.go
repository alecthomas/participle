package parser

import (
	"bufio"
	"strings"

	"io"
)

const EOF rune = -1

type Scanner interface {
	Next() rune
	Peek() rune
}

// ScanAll consumes all runes from a scanner and returns them as a slice.
func ScanAll(scanner Scanner) []rune {
	out := []rune{}
	for {
		r := scanner.Next()
		if r == EOF {
			return out
		}
		out = append(out, r)
	}
}

// SkipWhitespaceScanner wraps an existing Scanner, but skips all whitespace.
func SkipWhitespaceScanner(scanner Scanner) Scanner {
	return &skipWhitespaceScanner{s: scanner, peek: -2}
}

type skipWhitespaceScanner struct {
	s    Scanner
	peek rune
}

func (s *skipWhitespaceScanner) Next() rune {
	n := s.Peek()
	s.peek = -2
	return n
}

func (s *skipWhitespaceScanner) Peek() rune {
	if s.peek != -2 {
		return s.peek
	}
	for {
		if r := s.s.Next(); r == EOF {
			return r
		} else if !strings.ContainsRune(" \t\n\r", r) {
			s.peek = r
			return r
		}
	}
}

// RawScanner scans an io.Reader.
func RawScanner(r io.Reader) Scanner {
	return &rawScanner{r: bufio.NewReader(r), peek: -2}
}

type rawScanner struct {
	r    *bufio.Reader
	peek rune
}

func (r *rawScanner) Next() rune {
	n := r.Peek()
	r.peek = -2
	return n
}

func (r *rawScanner) Peek() rune {
	if r.peek != -2 {
		return r.peek
	}
	rn, _, err := r.r.ReadRune()
	if err == io.EOF {
		return EOF
	} else if err != nil {
		panic(err)
	}
	r.peek = rn
	return r.peek
}
