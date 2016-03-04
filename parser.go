package parser

import (
	"bytes"
	"reflect"
)

const (
	EOF rune = -1 - iota
)

type Lexer interface {
	Scan() rune
	TokenText() string
}

type parseNode interface {
	Parse(scan Scanner) []reflect.Value
}

func parseType(t reflect.Type) strct {
	switch t.Kind() {
	case reflect.Struct:
		scan := newStructScanner(t)
		e := parseExpression(scan)
		if peekNextNonSpace(scan) != EOF {
			panic("unexpected input: " + string(scan.Peek()))
		}
		return strct{typ: t, expr: e}
	case reflect.Ptr:
		return parseType(t.Elem())
	}
	panic("unsupported struct type " + t.String())
}

type strct struct {
	typ  reflect.Type
	expr expression
}

func (s *strct) Parse(scan Scanner) []reflect.Value {
	fields := s.expr.Parse(scan)
	if len(fields) != s.typ.NumField() {
		panic("number of parsed fields does not match number of fields in struct " + s.typ.String())
	}
	out := reflect.New(s.typ).Elem()
	for i := 0; i < s.typ.NumField(); i++ {
		out.Field(i).Set(fields[i])
	}
	return []reflect.Value{out}
}

type expression []alternative

func (e expression) Parse(scan Scanner) []reflect.Value {
	for _, a := range e {
		if out := a.Parse(scan); out != nil {
			return out
		}
	}
	return nil
}

func parseExpression(scan *structScanner) expression {
	out := expression{}
	for {
		out = append(out, parseAlternative(scan))
		if peekNextNonSpace(scan) != '|' {
			break
		}
		scan.Next()
	}
	return out
}

type alternative []parseNode

func (a alternative) Parse(scan Scanner) (out []reflect.Value) {
	for i, n := range a {
		fragment := n.Parse(scan)
		if fragment == nil {
			if i == 0 {
				return nil
			}
			panic("unexpected")
		}
		out = append(out, fragment...)
	}
	return out
}

func parseAlternative(scan *structScanner) alternative {
	elements := []parseNode{}
loop:
	for {
		switch scan.Peek() {
		case EOF:
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

func (d dot) Parse(scan Scanner) []reflect.Value {
	return oneValue(scan.Next())
}

// Expand a struct field.
type self struct {
	field reflect.StructField
	strct strct
}

func (s self) Parse(scan Scanner) []reflect.Value {
	return s.strct.Parse(scan)
}

type reference struct {
	field reflect.StructField
	expr  parseNode
}

func (r reference) Parse(scan Scanner) []reflect.Value {
	out := ""
	for _, v := range r.expr.Parse(scan) {
		out += v.String()
	}
	return oneValue(out)
}

func isIdentifierStart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_'
}

func parseTerm(scan *structScanner) parseNode {
	r := scan.Peek()
	switch r {
	case '.':
		scan.Next()
		return dot{}
	case '@':
		scan.Next()
		r := scan.Peek()
		f := scan.Field()
		if r == '@' {
			scan.Next()
			return self{f, parseType(scan.Field().Type)}
		}
		return reference{f, parseTerm(scan)}
	case '"':
		return parseQuotedStringOrRange(scan)
	case '[':
		return parseOptional(scan)
	case '{':
		return parseRepitition(scan)
	case '(':
		return parseGroup(scan)
	case EOF:
		return nil
	default:
		if isIdentifierStart(r) {
			return parseIdentifier(scan)
		}
		return nil
	}
}

type identifier string

func (i identifier) Parse(scan Scanner) []reflect.Value {
	panic("not implemented")
}

func parseIdentifier(scan *structScanner) identifier {
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

func (o optional) Parse(scan Scanner) []reflect.Value {
	panic("not implemented")
}

func parseOptional(scan *structScanner) optional {
	scan.Next() // [
	optional := optional(parseExpression(scan))
	next := scan.Next()
	if next != ']' {
		panic("expected ] but got " + string(next))
	}
	return optional
}

type repitition expression

func (r repitition) Parse(scan Scanner) []reflect.Value {
	panic("not implemented")
}

func parseRepitition(scan *structScanner) repitition {
	scan.Next() // {
	n := repitition(parseExpression(scan))
	next := scan.Next()
	if next != '}' {
		panic("expected } but got " + string(next))
	}
	return n
}

type group expression

func (g group) Parse(scan Scanner) []reflect.Value {
	return ((expression)(g)).Parse(scan)
}

func parseGroup(scan *structScanner) group {
	scan.Next() // (
	n := group(parseExpression(scan))
	next := scan.Next()
	if next != ')' {
		panic("expected ) but got " + string(next))
	}
	return n
}

func parseQuotedStringOrRange(scan *structScanner) parseNode {
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

func (s srange) Parse(scan Scanner) []reflect.Value {
	panic("not implemented")
}

type str string

func (s str) Parse(scan Scanner) []reflect.Value {
	out := ""
	for i, r := range s {
		if scan.Peek() != r {
			if i == 0 {
				return nil
			}
			panic("expected '" + s + "'")
		}
		out += string(scan.Next())
	}
	return oneValue(out)
}

func parseQuotedString(scan Scanner) str {
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
		case EOF:
			panic("unexpected EOF")
		default:
			out.WriteRune(scan.Next())
		}
	}
	return str(out.String())
}

func peekNextNonSpace(scan Scanner) rune {
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
