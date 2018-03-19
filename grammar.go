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

func newGeneratorContext(lex lexer.Definition) *generatorContext {
	return &generatorContext{Definition: lex, typeNodes: map[reflect.Type]node{}}
}

// Takes a type and builds a tree of nodes out of it.
func (g *generatorContext) parseType(t reflect.Type) node {
	rt := t
	t = indirectType(t)
	defer decorate(t.Name())
	if n, ok := g.typeNodes[t]; ok {
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
		g.typeNodes[t] = out
		slexer := lexStruct(t)
		defer func() {
			if msg := recover(); msg != nil {
				panic(slexer.Field().Name + ": " + msg.(string))
			}
		}()
		e := g.parseExpression(slexer)
		if !slexer.Peek().EOF() {
			panic("unexpected input " + slexer.Peek().Value)
		}
		out.expr = e
		return out
	}
	panic("expected struct type but got " + t.String())
}

func (g *generatorContext) parseExpression(slexer *structLexer) node {
	out := &disjunction{}
	for {
		out.nodes = append(out.nodes, g.parseAlternative(slexer))
		if slexer.Peek().Type != '|' {
			break
		}
		slexer.Next() // |
	}
	if len(out.nodes) == 1 {
		return out.nodes[0]
	}
	return out
}

func (g *generatorContext) parseAlternative(slexer *structLexer) node {
	elements := sequence{}
loop:
	for {
		switch slexer.Peek().Type {
		case lexer.EOF:
			break loop
		default:
			term := g.parseTerm(slexer)
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

func (g *generatorContext) parseTerm(slexer *structLexer) node {
	r := slexer.Peek()
	switch r.Type {
	case '@':
		return g.parseCapture(slexer)
	case scanner.String, scanner.RawString, scanner.Char:
		return g.parseLiteral(slexer)
	case '[':
		return g.parseOptional(slexer)
	case '{':
		return g.parseRepetition(slexer)
	case '(':
		return g.parseGroup(slexer)
	case scanner.Ident:
		return g.parseReference(slexer)
	case lexer.EOF:
		slexer.Next()
		return nil
	default:
		return nil
	}
}

// @<expression> captures <expression> into the current field.
func (g *generatorContext) parseCapture(slexer *structLexer) node {
	slexer.Next()
	token := slexer.Peek()
	field := slexer.Field()
	if token.Type == '@' {
		slexer.Next()
		return &capture{field, g.parseType(field.Type)}
	}
	if indirectType(field.Type).Kind() == reflect.Struct && !field.Type.Implements(captureType) {
		panic("structs can only be parsed with @@ or by implementing the Capture interface")
	}
	return &capture{field, g.parseTerm(slexer)}
}

// A reference in the form <identifier> refers to a named token from the lexer.
func (g *generatorContext) parseReference(slexer *structLexer) node {
	token := slexer.Next()
	if token.Type != scanner.Ident {
		panic("expected identifier")
	}
	typ, ok := g.Symbols()[token.Value]
	if !ok {
		panicf("unknown token type %q", token.String())
	}
	return &reference{typ, token.Value}
}

// [ <expression> ] optionally matches <expression>.
func (g *generatorContext) parseOptional(slexer *structLexer) node {
	slexer.Next() // [
	optional := &optional{g.parseExpression(slexer)}
	next := slexer.Peek()
	if next.Type != ']' {
		panic("expected ] but got " + next.String())
	}
	slexer.Next()
	return optional
}

// { <expression> } matches 0 or more repititions of <expression>
func (g *generatorContext) parseRepetition(slexer *structLexer) node {
	slexer.Next() // {
	n := &repetition{
		node: g.parseExpression(slexer),
	}
	next := slexer.Next()
	if next.Type != '}' {
		panic("expected } but got " + next.String())
	}
	return n
}

// ( <expression> ) groups a sub-expression
func (g *generatorContext) parseGroup(slexer *structLexer) node {
	slexer.Next() // (
	n := g.parseExpression(slexer)
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
func (g *generatorContext) parseLiteral(lex *structLexer) node { // nolint: interfacer
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
		t, ok = g.Symbols()[token.Value]
		if !ok {
			panic("unknown token type " + token.String() + " in literal type constraint")
		}
	}
	return &literal{s: s, t: t}
}
