package lexer

import "fmt"

// This file exists to break circular imports. The types and functions in here
// mirror those in the participle package.

type errorInterface interface {
	error
	Message() string
	Position() Position
}

// Error represents an error while lexing.
//
// It complies with the participle.Error interface.
type lexerError struct {
	Msg string
	Pos Position
}

var _ errorInterface = &lexerError{}

// Creates a new Error at the given position.
func errorf(pos Position, format string, args ...interface{}) *lexerError {
	return &lexerError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

func (e *lexerError) Message() string    { return e.Msg } // nolint: golint
func (e *lexerError) Position() Position { return e.Pos } // nolint: golint

// Error formats the error with FormatError.
func (e *lexerError) Error() string { return formatError(e.Pos, e.Msg) }

// An error in the form "[<filename>:][<line>:<pos>:] <message>"
func formatError(pos Position, message string) string {
	msg := ""
	if pos.Filename != "" {
		msg += pos.Filename + ":"
	}
	if pos.Line != 0 || pos.Column != 0 {
		msg += fmt.Sprintf("%d:%d:", pos.Line, pos.Column)
	}
	if msg != "" {
		msg += " " + message
	} else {
		msg = message
	}
	return msg
}

type wrappingLexerError struct {
	err error
	lexerError
}

var _ errorInterface = &wrappingLexerError{}

func (w *wrappingLexerError) Unwrap() error { return w.err }

func wrapf(pos Position, err error, format string, args ...interface{}) error {
	var msg string
	if perr, ok := err.(errorInterface); ok {
		pos = perr.Position()
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), perr.Message())
	} else {
		msg = fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), err.Error())
	}
	return &wrappingLexerError{err: err, lexerError: lexerError{Msg: msg, Pos: pos}}
}
