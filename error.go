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
	// Unadorned message.
	Message() string
	// Position error occurred.
	Position() lexer.Position
}

// UnexpectedTokenError is returned by Parse when an unexpected token is encountered.
//
// This is useful for composing parsers in order to detect when a sub-parser has terminated.
type UnexpectedTokenError struct {
	Unexpected lexer.Token
	Expected   string
}

func (u UnexpectedTokenError) Error() string {
	return lexer.FormatError(u.Unexpected.Pos, u.Message())
}

func (u UnexpectedTokenError) Message() string { // nolint: golint
	var expected string
	if u.Expected != "" {
		expected = fmt.Sprintf(" (expected %s)", u.Expected)
	}
	return fmt.Sprintf("unexpected token %q%s", u.Unexpected.Value, expected)
}
func (u UnexpectedTokenError) Position() lexer.Position { return u.Unexpected.Pos } // nolint: golint

type parseError struct {
	Msg string
	Pos lexer.Position
}

func (p *parseError) Error() string            { return lexer.FormatError(p.Pos, p.Msg) }
func (p *parseError) Message() string          { return p.Msg }
func (p *parseError) Position() lexer.Position { return p.Pos }

// AnnotateError wraps an existing error with a position.
//
// If the existing error is a lexer.Error or participle.Error it will be returned unmodified.
func AnnotateError(pos lexer.Position, err error) error {
	if perr, ok := err.(Error); ok {
		return perr
	}
	return &parseError{Msg: err.Error(), Pos: pos}
}

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) error {
	return &parseError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

// Wrapf attempts to wrap an existing participle.Error in a new message.
func Wrapf(err error, format string, args ...interface{}) error {
	if perr, ok := err.(Error); ok {
		return Errorf(perr.Position(), "%s: %s", fmt.Sprintf(format, args...), perr.Message())
	}
	return fmt.Errorf("%s: %s", fmt.Sprintf(format, args...), err.Error())
}
