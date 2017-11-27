package participle

import (
	"reflect"
	"text/scanner"

	"github.com/alecthomas/participle/lexer"
)

type generatorContext struct {
	lexer.Definition
	typeNodes map[reflect.Type]node
}

// Takes a type and builds a tree of nodes out of it.
func parseType(context *generatorContext, t reflect.Type) node {
	rt := t
	t = indirectType(t)
	defer decorate(t.Name())
	if n, ok := context.typeNodes[t]; ok {
		return n
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Ptr:
		t = indirectType(t.Elem())
		fallthrough

	case reflect.Struct:
		if rt.Implements(parseableType) {
			return &parseable{rt}
		}
		out := &strct{typ: t}
		context.typeNodes[t] = out
		slexer := lexStruct(t)
		defer func() {
			if msg := recover(); msg != nil {
				panic(slexer.Field().Name + ": " + msg.(string))
			}
		}()
		e := parseExpression(context, slexer)
		if !slexer.Peek().EOF() {
			panic("unexpected input " + slexer.Peek().Value)
		}
		out.expr = e
		return out
	}
	panic("expected struct type but got " + t.String())
}

func parseExpression(context *generatorContext, slexer *structLexer) node {
	out := disjunction{}
	for {
		out = append(out, parseAlternative(context, slexer))
		if slexer.Peek().Type != '|' {
			break
		}
		slexer.Next() // |
	}
	if len(out) == 1 {
		return out[0]
	}
	return out
}

func parseAlternative(context *generatorContext, slexer *structLexer) node {
	elements := sequence{}
loop:
	for {
		switch slexer.Peek().Type {
		case lexer.EOF:
			break loop
		default:
			term := parseTerm(context, slexer)
			if term == nil {
				break loop
			}
			elements = append(elements, term)
		}
	}
	if len(elements) == 1 {
		return elements[0]
	}
	return elements
}

func parseTerm(context *generatorContext, slexer *structLexer) node {
	r := slexer.Peek()
	switch r.Type {
	case '@':
		return parseCapture(context, slexer)
	case scanner.String, scanner.RawString, scanner.Char:
		return parseLiteral(context, slexer)
	case '[':
		return parseOptional(context, slexer)
	case '{':
		return parseRepetition(context, slexer)
	case '(':
		return parseGroup(context, slexer)
	case scanner.Ident:
		return parseTokenReference(context, slexer)
	case lexer.EOF:
		slexer.Next()
		return nil
	default:
		return nil
	}
}

// @<expression> captures <expression> into the current field.
func parseCapture(context *generatorContext, slexer *structLexer) node {
	slexer.Next()
	token := slexer.Peek()
	field := slexer.Field()
	if token.Type == '@' {
		slexer.Next()
		return &reference{field, parseType(context, field.Type)}
	}
	if indirectType(field.Type).Kind() == reflect.Struct && !field.Type.Implements(captureType) {
		panic("structs can only be parsed with @@ or by implementing the Capture interface")
	}
	return &reference{field, parseTerm(context, slexer)}
}

// A reference in the form <identifier> refers to a named token from the lexer.
func parseTokenReference(context *generatorContext, slexer *structLexer) node {
	token := slexer.Next()
	if token.Type != scanner.Ident {
		panic("expected identifier")
	}
	typ, ok := context.Symbols()[token.Value]
	if !ok {
		panicf("unknown token type %q", token.String())
	}
	return &tokenReference{typ, token.Value}
}

// [ <expression> ] optionally matches <expression>.
func parseOptional(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // [
	optional := &optional{parseExpression(context, slexer)}
	next := slexer.Peek()
	if next.Type != ']' {
		panic("expected ] but got " + next.String())
	}
	slexer.Next()
	return optional
}

// { <expression> } matches 0 or more repititions of <expression>
func parseRepetition(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // {
	n := &repetition{
		node: parseExpression(context, slexer),
	}
	next := slexer.Next()
	if next.Type != '}' {
		panic("expected } but got " + next.String())
	}
	return n
}

// ( <expression> ) groups a sub-expression
func parseGroup(context *generatorContext, slexer *structLexer) node {
	slexer.Next() // (
	n := parseExpression(context, slexer)
	next := slexer.Peek() // )
	if next.Type != ')' {
		panic("expected ) but got " + next.Value)
	}
	slexer.Next() // )
	return n
}

// A literal string.
//
// Note that for this to match, the tokeniser must be able to produce this string. For example,
// if the tokeniser only produces individual charactersk but the literal is "hello", or vice versa.
func parseLiteral(context *generatorContext, lex *structLexer) node { // nolint: interfacer
	token := lex.Next()
	if token.Type != scanner.String && token.Type != scanner.RawString && token.Type != scanner.Char {
		panic("expected quoted string but got " + token.String())
	}
	s := token.Value
	t := rune(-1)
	token = lex.Peek()
	if token.Value == ":" {
		lex.Next()
		token = lex.Next()
		if token.Type != scanner.Ident {
			panic("expected identifier for literal type constraint but got " + token.String())
		}
		var ok bool
		t, ok = context.Symbols()[token.Value]
		if !ok {
			panic("unknown token type " + token.String() + " in literal type constraint")
		}
	}
	return &literal{s: s, t: t}
}
