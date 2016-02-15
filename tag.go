package parser

import (
	"strconv"
	"strings"
	"text/scanner"
)

func parseTag(tag string) expression {
	scan := &scanner.Scanner{}
	scan.Init(strings.NewReader(tag))
	scan.Whitespace = 0
	e := parseExpression(scan)
	if peekNextNonSpace(scan) != scanner.EOF {
		panic("unexpected end of input")
	}
	return e
}

type expression []alternative

func parseExpression(scan *scanner.Scanner) expression {
	expression := expression{parseAlternative(scan)}
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
		expression = append(expression, parseAlternative(scan))
	}
	return expression
}

type alternative []interface{}

func parseAlternative(scan *scanner.Scanner) alternative {
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
	return alternative(elements)
}

func parseTerm(scan *scanner.Scanner) interface{} {
	switch scan.Peek() {
	case '@':
		scan.Next()
		return self{}
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

type optional expression

func parseOptional(scan *scanner.Scanner) optional {
	scan.Next() // [
	optional := optional(parseExpression(scan))
	next := scan.Next()
	if next != ']' {
		panic("expected ] but got " + string(next))
	}
	return optional
}

type repitition expression

func parseRepitition(scan *scanner.Scanner) repitition {
	scan.Next() // {
	n := repitition(parseExpression(scan))
	next := scan.Next()
	if next != '}' {
		panic("expected } but got " + string(next))
	}
	return n
}

type group expression

func parseGroup(scan *scanner.Scanner) group {
	scan.Next() // (
	n := group(parseExpression(scan))
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
	return srange{n, parseQuotedString(scan)}
}

type srange struct {
	start str
	end   str
}

type str string

func parseQuotedString(scan *scanner.Scanner) str {
	r := scan.Scan()
	if r != scanner.String && r != scanner.RawString && r != scanner.Char {
		panic("expected string but got " + string(r))
	}
	token, err := strconv.Unquote(scan.TokenText())
	if err != nil {
		panic("invalid string")
	}
	return str(token)
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
