package participle

import (
	"fmt"

	"github.com/alecthomas/participle/lexer"
)

// Error represents an error while parsing.
//
// The error will contain positional information if available.
type Error interface {
	error
	Position() lexer.Position
}

// UnexpectedTokenError is returned by Parse when an unexpected token is encountered.
//
// This is useful for composing parsers in order to detect when a sub-parser has terminated.
type UnexpectedTokenError struct{ lexer.Token }

func (u UnexpectedTokenError) Error() string {
	return lexer.FormatError(u.Pos, fmt.Sprintf("unexpected token %q", u.Value))
}

func (u UnexpectedTokenError) Position() lexer.Position { return u.Pos } // nolint: golint

type parseError struct {
	Message string
	Pos     lexer.Position
}

func (p *parseError) Position() lexer.Position { return p.Pos }

// AnnotateError wraps an existing error with a position.
//
// If the existing error is a lexer.Error or participle.Error it will be returned unmodified.
func AnnotateError(pos lexer.Position, err error) error {
	if perr, ok := err.(Error); ok {
		return perr
	}
	return &parseError{Message: err.Error(), Pos: pos}
}

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) error {
	return &parseError{Message: fmt.Sprintf(format, args...), Pos: pos}
}

func (p *parseError) Error() string {
	return lexer.FormatError(p.Pos, p.Message)
}
