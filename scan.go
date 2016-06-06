package parser

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
)

const (
	EOF rune = -1 - iota
	Skip
)

// A Scanner scans runes.
type Scanner interface {
	// Peek at the next rune. EOF will be returned if at end of stream.
	Peek() rune
	// Next consumes the next rune. EOF will be returned if at end of stream.
	Next() rune
}

// A structScanner scans over the tags of struct fields while tracking the current field.
type structScanner struct {
	s     reflect.Type
	field int
	r     *strings.Reader
}

func newStructScanner(s reflect.Type) *structScanner {
	return &structScanner{
		s: s,
		r: strings.NewReader(string(s.Field(0).Tag)),
	}
}

func (s *structScanner) NumField() int {
	return s.s.NumField()
}

func (s *structScanner) Field() reflect.StructField {
	return s.s.Field(s.field)
}

func (s *structScanner) Peek() rune {
	field := s.field
	reader := s.r
	for {
		r, _, err := reader.ReadRune()
		if err != io.EOF {
			reader.UnreadRune()
			return r
		}
		field++
		if field >= s.s.NumField() {
			return EOF
		}
		reader = strings.NewReader(string(s.s.Field(field).Tag))
	}
}

func (s *structScanner) Next() rune {
	r, _, err := s.r.ReadRune()
	if err != io.EOF {
		return r
	}
	if s.field+1 >= s.s.NumField() {
		return EOF
	}
	s.field++
	s.r = strings.NewReader(string(s.s.Field(s.field).Tag))
	return s.Next()
}

// An adapter from io.Reader to Scanner.
type readerScanner struct {
	r *bufio.Reader
}

// ByteScanner scans over a byte slice.
func ByteScanner(b []byte) Scanner {
	return ReaderScanner(bytes.NewReader(b))
}

// StringScanner scans over a string.
func StringScanner(s string) Scanner {
	return ReaderScanner(strings.NewReader(s))
}

// ReaderScanner creates a new Scanner over an io.Reader.
func ReaderScanner(r io.Reader) Scanner {
	return &readerScanner{bufio.NewReader(r)}
}

func (r *readerScanner) Peek() rune {
	rn, _, err := r.r.ReadRune()
	if err == io.EOF {
		return EOF
	}
	r.r.UnreadRune()
	return rn
}

func (r *readerScanner) Next() rune {
	rn, _, err := r.r.ReadRune()
	if err == io.EOF {
		return EOF
	}
	return rn
}
