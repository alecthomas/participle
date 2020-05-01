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
	// Closest token to error location.
	Token() lexer.Token
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
	return fmt.Sprintf("unexpected token %q%s", u.Unexpected, expected)
}
func (u UnexpectedTokenError) Token() lexer.Token { return u.Unexpected } // nolint: golint

type parseError struct {
	Msg string
	Tok lexer.Token
}

func (p *parseError) Error() string      { return lexer.FormatError(p.Tok.Pos, p.Msg) }
func (p *parseError) Message() string    { return p.Msg }
func (p *parseError) Token() lexer.Token { return p.Tok }

// AnnotateError wraps an existing error with a position.
//
// If the existing error is a lexer.Error or participle.Error it will be returned unmodified.
func AnnotateError(pos lexer.Position, err error) error {
	if perr, ok := err.(Error); ok {
		return perr
	}
	return &parseError{Msg: err.Error(), Tok: lexer.Token{Pos: pos}}
}

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) error {
	return &parseError{Msg: fmt.Sprintf(format, args...), Tok: lexer.Token{Pos: pos}}
}

// ErrorWithTokenf creats a new Error with the given token as context.
func ErrorWithTokenf(tok lexer.Token, format string, args ...interface{}) error {
	return &parseError{Msg: fmt.Sprintf(format, args...), Tok: tok}
}

// Wrapf attempts to wrap an existing participle.Error in a new message.
func Wrapf(pos lexer.Position, err error, format string, args ...interface{}) error {
	if perr, ok := err.(Error); ok {
		return Errorf(perr.Token().Pos, "%s: %s", fmt.Sprintf(format, args...), perr.Message())
	}
	return Errorf(pos, "%s: %s", fmt.Sprintf(format, args...), err.Error())
}
