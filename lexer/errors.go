package lexer

import "fmt"

// Error represents an error while parsing.
type Error struct {
	Msg string
	Pos Position
}

// Errorf creats a new Error at the given position.
func Errorf(pos Position, format string, args ...interface{}) *Error {
	return &Error{
		Msg: fmt.Sprintf(format, args...),
		Pos: pos,
	}
}

func (e *Error) Message() string    { return e.Msg } // nolint: golint
func (e *Error) Position() Position { return e.Pos } // nolint: golint

// Error complies with the error interface and reports the position of an error.
func (e *Error) Error() string {
	return FormatError(e.Pos, e.Msg)
}

// FormatError formats an error in the form "[<filename>:][<line>:<pos>:] <message>"
func FormatError(pos Position, message string) string {
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
