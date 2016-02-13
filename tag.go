package parser

import (
	"bytes"
	"strings"
	"text/scanner"
)

func parseTag(tag string) *expression {
	scan := &scanner.Scanner{}
	scan.Init(strings.NewReader(tag))
	scan.Whitespace = 0
	scan.Mode = 0
	e := parseExpression(scan)
	if peekNextNonSpace(scan) != scanner.EOF {
		panic("unexpected end of input")
	}
	return e
}

type expression struct {
	Alternatives []*alternative
}

func parseExpression(scan *scanner.Scanner) *expression {
	alternatives := []*alternative{parseAlternative(scan)}
outer:
	for {
	inner:
		for {
			switch scan.Peek() {
			case '|':
				scan.Next()
				break inner
			case ' ', '\t':
				scan.Next()
				continue inner
			case scanner.EOF:
				break outer
			default:
				break outer
			}
		}
		alternatives = append(alternatives, parseAlternative(scan))
	}
	return &expression{alternatives}
}

type alternative struct {
	Terms []interface{}
}

func parseAlternative(scan *scanner.Scanner) *alternative {
	elements := []interface{}{}
loop:
	for {
		switch scan.Peek() {
		case scanner.EOF:
			break loop
		case ' ', '\t':
			scan.Next()
			continue loop
		default:
			term := parseTerm(scan)
			if term == nil {
				break loop
			}
			elements = append(elements, term)
		}
	}
	return &alternative{elements}
}

func parseTerm(scan *scanner.Scanner) interface{} {
	switch scan.Peek() {
	case '@':
		scan.Next()
		return &self{}
	case '"':
		return parseQuotedStringOrRange(scan)
	case '[':
		return parseOptional(scan)
	case '{':
		return parseRepitition(scan)
	case '(':
		return parseGroup(scan)
	case scanner.EOF:
		return nil
	default:
		return nil
	}
}

type optional struct {
	Expression *expression
}

func parseOptional(scan *scanner.Scanner) *optional {
	scan.Next() // [
	optional := &optional{parseExpression(scan)}
	next := scan.Next()
	if next != ']' {
		panic("expected ] but got " + string(next))
	}
	return optional
}

type repitition struct {
	Expression *expression
}

func parseRepitition(scan *scanner.Scanner) *repitition {
	scan.Next() // {
	n := &repitition{parseExpression(scan)}
	next := scan.Next()
	if next != '}' {
		panic("expected } but got " + string(next))
	}
	return n
}

type group struct {
	Expression *expression
}

func parseGroup(scan *scanner.Scanner) *group {
	scan.Next() // (
	n := &group{parseExpression(scan)}
	next := scan.Next()
	if next != ')' {
		panic("expected ) but got " + string(next))
	}
	return n
}

type self struct{}

func parseQuotedStringOrRange(scan *scanner.Scanner) interface{} {
	n := parseQuotedString(scan)
	if peekNextNonSpace(scan) != '…' {
		return n
	}
	scan.Next()
	if peekNextNonSpace(scan) != '"' {
		panic("expected ending quoted string for range but got " + string(scan.Peek()))
	}
	return &stringRange{n, parseQuotedString(scan)}
}

type stringRange struct {
	Start *quotedString
	End   *quotedString
}

type quotedString struct {
	Text string
}

func parseQuotedString(scan *scanner.Scanner) *quotedString {
	scan.Next() // "
	out := bytes.Buffer{}
loop:
	for {
		switch scan.Peek() {
		case '\\':
			scan.Next()
			out.WriteRune(scan.Next())
		case '"':
			scan.Next()
			break loop
		case scanner.EOF:
			panic("unexpected EOF")
		default:
			out.WriteRune(scan.Next())
		}
	}
	return &quotedString{out.String()}
}

func peekNextNonSpace(scan *scanner.Scanner) rune {
	for {
		switch scan.Peek() {
		case ' ', '\t':
			scan.Next()
			continue
		default:
			return scan.Peek()
		}
	}
}
