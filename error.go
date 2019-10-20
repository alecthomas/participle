package participle

import (
	"fmt"

	"github.com/alecthomas/participle/lexer"
)

// Error represents an error while parsing.
//
// The error will contain positional information if available.
type Error struct {
	Message string
	Pos     lexer.Position
}

// autoError attempts to determine position from an existing error.
func autoError(err error) error {
	switch err := err.(type) {
	case *lexer.Error:
		return (*Error)(err)

	case *Error:
		return err

	case nil:
		return nil

	default:
		return &Error{Message: err.Error()}
	}
}

// AnnotateError wraps an existing error with a position.
//
// If the existing error is a lexer.Error or participle.Error it will be returned unmodified.
func AnnotateError(pos lexer.Position, err error) error {
	switch err := err.(type) {
	case *lexer.Error:
		return (*Error)(err)

	case *Error:
		return err

	case nil:
		return nil

	default:
		return &Error{Message: err.Error(), Pos: pos}
	}
}

// Errorf creats a new Error at the given position.
func Errorf(pos lexer.Position, format string, args ...interface{}) error {
	return &Error{Message: fmt.Sprintf(format, args...), Pos: pos}
}

// Error formats an error in the form "[<filename>:][<line>:<pos>:] <message>"
func (e *Error) Error() string {
	msg := ""
	if e.Pos.Filename != "" {
		msg += e.Pos.Filename + ":"
	}
	if e.Pos.Line != 0 || e.Pos.Column != 0 {
		msg += fmt.Sprintf("%d:%d:", e.Pos.Line, e.Pos.Column)
	}
	if msg != "" {
		msg += " " + e.Message
	} else {
		msg = e.Message
	}
	return msg
}
