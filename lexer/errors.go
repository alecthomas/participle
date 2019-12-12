package lexer

import "fmt"

// Error represents an error while parsing.
type Error struct {
	Message string
	Pos     Position
}

// Errorf creats a new Error at the given position.
func Errorf(pos Position, format string, args ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(format, args...),
		Pos:     pos,
	}
}

func (e *Error) Position() Position { return e.Pos } // nolint: golint

// Error complies with the error interface and reports the position of an error.
func (e *Error) Error() string {
	return FormatError(e.Pos, e.Message)
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
