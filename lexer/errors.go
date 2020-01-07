package lexer

import "fmt"

// Error represents an error while parsing.
type Error struct {
	Msg string
	Tok Token
}

// Errorf creats a new Error at the given position.
func Errorf(pos Position, format string, args ...interface{}) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...), Tok: Token{Pos: pos}}
}

// ErrorWithTokenf creats a new Error with the given token as context.
func ErrorWithTokenf(tok Token, format string, args ...interface{}) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...), Tok: tok}
}

func (e *Error) Message() string { return e.Msg } // nolint: golint
func (e *Error) Token() Token    { return e.Tok } // nolint: golint

// Error complies with the error interface and reports the position of an error.
func (e *Error) Error() string {
	return FormatError(e.Tok.Pos, e.Msg)
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
