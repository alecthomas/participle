package lexer

import "fmt"

// Error represents an error while parsing.
type Error struct {
	Message string
	Pos     Position
}

// Panic throws a parse error.
func Panic(pos Position, message string) {
	panic(&Error{Message: message, Pos: pos})
}

// Panicf throws an *Error while parsing.
func Panicf(pos Position, format string, args ...interface{}) {
	panic(Errorf(pos, format, args...))
}

// Errorf creats a new Error at the given position.
func Errorf(pos Position, format string, args ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(format, args...),
		Pos:     pos,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Pos.Filename, e.Pos.Line, e.Pos.Column, e.Message)
}
