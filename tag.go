package parser

import (
	"bytes"
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
	return elements
}

type dot struct{}

type self struct{}

type reference alternative

func isIdentifierStart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_'
}

func parseTerm(scan *scanner.Scanner) interface{} {
	r := scan.Peek()
	switch r {
	case '.':
		scan.Next()
		return dot{}
	case '@':
		scan.Next()
		r := scan.Peek()
		if r == '@' {
			scan.Next()
			return self{}
		}
		return reference{parseTerm(scan)}
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
		if isIdentifierStart(r) {
			return parseIdentifier(scan)
		}
		return nil
	}
}

type identifier string

func parseIdentifier(scan *scanner.Scanner) identifier {
	out := bytes.Buffer{}
	out.WriteRune(scan.Next())
	for {
		r := scan.Peek()
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r >= '0' && r <= '9' {
			out.WriteRune(scan.Next())
		} else {
			break
		}
	}
	return identifier(out.String())
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

// "a" … "b"
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
