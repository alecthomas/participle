package lexer

import "fmt"

// Error represents an error while parsing.
//
// It complies with the participle.Error interface.
type lexerError struct {
	Msg string
	Pos Position
}

// Errorf creats a new Error at the given position.
func errorf(pos Position, format string, args ...interface{}) *lexerError {
	return &lexerError{Msg: fmt.Sprintf(format, args...), Pos: pos}
}

func (e *lexerError) Message() string    { return e.Msg } // nolint: golint
func (e *lexerError) Position() Position { return e.Pos } // nolint: golint

// Error formats the error with FormatError.
func (e *lexerError) Error() string { return formatError(e.Pos, e.Msg) }

// FormatError formats an error in the form "[<filename>:][<line>:<pos>:] <message>"
//
// This exists in the lexer to avoid circular imports.
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
