package participle

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
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
	at         node
}

func (u UnexpectedTokenError) Error() string { return FormatError(u) }

func (u UnexpectedTokenError) Message() string { // nolint: golint
	var expected string
	if u.at != nil {
		expected = fmt.Sprintf(" (expected %s)", u.at)
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

type wrappingParseError struct {
	err error
	parseError
}

func (w *wrappingParseError) Unwrap() error { return w.err }

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) Error {
	return &parseError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

// Wrapf attempts to wrap an existing error in a new message.
//
// If "err" is a participle.Error, its positional information will be uesd.
//
// The returned error implements the Unwrap() method supported by the errors package.
func Wrapf(pos lexer.Position, err error, format string, args ...interface{}) Error {
	var msg string
	if perr, ok := err.(Error); ok {
		pos = perr.Position()
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), perr.Message())
	} else {
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), err.Error())
	}
	return &wrappingParseError{err: err, parseError: parseError{Msg: msg, Pos: pos}}
}

// AnnotateError wraps an existing error with a position.
//
// If the existing error is a lexer.Error or participle.Error it will be returned unmodified.
func AnnotateError(pos lexer.Position, err error) error {
	if perr, ok := err.(Error); ok {
		return perr
	}
	return &wrappingParseError{err: err, parseError: parseError{Msg: err.Error(), Pos: pos}}
}
