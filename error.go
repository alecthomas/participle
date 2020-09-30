package participle

import (
	"fmt"

	"github.com/alecthomas/participle/lexer"
)

// Error represents an error while parsing.
//
// The format of an Error is in the form "[<filename>:][<line>:<pos>:] <message>".
//
// The error will contain positional information if available.
type Error interface {
	error
	// Unadorned message.
	Message() string
	// Closest position to error location.
	Position() lexer.Position
}

// FormatError formats an error in the form "[<filename>:][<line>:<pos>:] <message>"
func FormatError(err Error) string {
	msg := ""
	pos := err.Position()
	if pos.Filename != "" {
		msg += pos.Filename + ":"
	}
	if pos.Line != 0 || pos.Column != 0 {
		msg += fmt.Sprintf("%d:%d:", pos.Line, pos.Column)
	}
	if msg != "" {
		msg += " " + err.Message()
	} else {
		msg = err.Message()
	}
	return msg
}

// UnexpectedTokenError is returned by Parse when an unexpected token is encountered.
//
// This is useful for composing parsers in order to detect when a sub-parser has terminated.
type UnexpectedTokenError struct {
	Unexpected lexer.Token
	Expected   string
}

func (u UnexpectedTokenError) Error() string { return FormatError(u) }

func (u UnexpectedTokenError) Message() string { // nolint: golint
	var expected string
	if u.Expected != "" {
		expected = fmt.Sprintf(" (expected %s)", u.Expected)
	}
	return fmt.Sprintf("unexpected token %q%s", u.Unexpected, expected)
}
func (u UnexpectedTokenError) Position() lexer.Position { return u.Unexpected.Pos } // nolint: golint

type parseError struct {
	Msg string
	Pos lexer.Position
}

func (p *parseError) Error() string            { return FormatError(p) }
func (p *parseError) Message() string          { return p.Msg }
func (p *parseError) Position() lexer.Position { return p.Pos }

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) Error {
	return &parseError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

// Wrapf attempts to wrap an existing Error in a new message.
//
// If "err" is a participle.Error, its positional information will be uesd.
func Wrapf(pos lexer.Position, err error, format string, args ...interface{}) Error {
	if perr, ok := err.(Error); ok {
		return Errorf(perr.Position(), "%s: %s", fmt.Sprintf(format, args...), perr.Message())
	}
	return Errorf(pos, "%s: %s", fmt.Sprintf(format, args...), err.Error())
}
