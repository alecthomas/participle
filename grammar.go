package participle

import (
	"fmt"
	"reflect"
	"text/scanner"

	"github.com/alecthomas/participle/lexer"
)

type generatorContext struct {
	lexer.Definition
	typeNodes    map[reflect.Type]node
	symbolsToIDs map[rune]string
}

func newGeneratorContext(lex lexer.Definition) *generatorContext {
	return &generatorContext{
		Definition:   lex,
		typeNodes:    map[reflect.Type]node{},
		symbolsToIDs: lexer.SymbolsByRune(lex),
	}
}

// Takes a type and builds a tree of nodes out of it.
func (g *generatorContext) parseType(t reflect.Type) (_ node, returnedError error) {
	rt := t
	t = indirectType(t)
	if n, ok := g.typeNodes[t]; ok {
		return n, nil
	}
	if rt.Implements(parseableType) {
		return &parseable{rt.Elem()}, nil
	}
	if reflect.PtrTo(rt).Implements(parseableType) {
		return &parseable{rt}, nil
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Ptr:
		t = indirectType(t.Elem())
		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected a struct but got %T", t)
		}
		fallthrough

	case reflect.Struct:
		slexer, err := lexStruct(t)
		if err != nil {
			return nil, err
		}
		out := &strct{typ: t}
		g.typeNodes[t] = out // Ensure we avoid infinite recursion.
		if slexer.NumField() == 0 {
			return nil, fmt.Errorf("can not parse into empty struct %s", t)
		}
		defer decorate(&returnedError, func() string { return slexer.Field().Name })
		e, err := g.parseDisjunction(slexer)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return nil, fmt.Errorf("no grammar found in %s", t)
		}
		if token, _ := slexer.Peek(); !token.EOF() {
			return nil, fmt.Errorf("unexpected input %q", token.Value)
		}
		out.expr = e
		return out, nil
	}
	return nil, fmt.Errorf("%s should be a struct or should implement the Parseable interface", t)
}

func (g *generatorContext) parseDisjunction(slexer *structLexer) (node, error) {
	out := &disjunction{}
	for {
		n, err := g.parseSequence(slexer)
		if err != nil {
			return nil, err
		}
		out.nodes = append(out.nodes, n)
		if token, _ := slexer.Peek(); token.Type != '|' {
			break
		}
		_, err = slexer.Next() // |
		if err != nil {
			return nil, err
		}
	}
	if len(out.nodes) == 1 {
		return out.nodes[0], nil
	}
	return out, nil
}

func (g *generatorContext) parseSequence(slexer *structLexer) (node, error) {
	head := &sequence{}
	cursor := head
loop:
	for {
		if token, err := slexer.Peek(); err != nil {
			return nil, err
		} else if token.Type == lexer.EOF {
			break loop
		}
		term, err := g.parseTerm(slexer)
		if err != nil {
			return nil, err
		}
		if term == nil {
			break loop
		}
		if cursor.node == nil {
			cursor.head = true
			cursor.node = term
		} else {
			cursor.next = &sequence{node: term}
			cursor = cursor.next
		}

		// An optional or repetition result in some magic.
		switch n := term.(type) {
		case *optional:
			n.next, err = g.parseSequence(slexer)
			if err != nil {
				return nil, err
			}
			break loop
		case *repetition:
			n.next, err = g.parseSequence(slexer)
			if err != nil {
				return nil, err
			}
			break loop
		}
	}
	if head.node == nil {
		return nil, nil
	}
	if head.next == nil {
		return head.node, nil
	}
	return head, nil
}

func (g *generatorContext) parseTerm(slexer *structLexer) (node, error) {
	r, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
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
		_, _ = slexer.Next()
		return nil, nil
	default:
		return nil, nil
	}
}

// @<expression> captures <expression> into the current field.
func (g *generatorContext) parseCapture(slexer *structLexer) (node, error) {
	_, _ = slexer.Next()
	token, err := slexer.Peek()
	if err != nil {
		return nil, err
	}
	field := slexer.Field()
	if token.Type == '@' {
		_, _ = slexer.Next()
		n, err := g.parseType(field.Type)
		if err != nil {
			return nil, err
		}
		return &capture{field, n}, nil
	}
	if indirectType(field.Type).Kind() == reflect.Struct && !field.Type.Implements(captureType) {
		return nil, fmt.Errorf("structs can only be parsed with @@ or by implementing the Capture interface")
	}
	n, err := g.parseTerm(slexer)
	if err != nil {
		return nil, err
	}
	return &capture{field, n}, nil
}

// A reference in the form <identifier> refers to a named token from the lexer.
func (g *generatorContext) parseReference(slexer *structLexer) (node, error) { // nolint: interfacer
	token, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if token.Type != scanner.Ident {
		return nil, fmt.Errorf("expected identifier but got %q", token)
	}
	typ, ok := g.Symbols()[token.Value]
	if !ok {
		return nil, fmt.Errorf("unknown token type %q", token)
	}
	return &reference{typ: typ, identifier: token.Value}, nil
}

// [ <expression> ] optionally matches <expression>.
func (g *generatorContext) parseOptional(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // [
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	optional := &optional{node: disj}
	next, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if next.Type != ']' {
		return nil, fmt.Errorf("expected ] but got %q", next)
	}
	return optional, nil
}

// { <expression> } matches 0 or more repititions of <expression>
func (g *generatorContext) parseRepetition(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // {
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	n := &repetition{
		node: disj,
	}
	next, err := slexer.Next()
	if err != nil {
		return nil, err
	}
	if next.Type != '}' {
		return nil, fmt.Errorf("expected } but got %q", next)
	}
	return n, nil
}

// ( <expression> ) groups a sub-expression
func (g *generatorContext) parseGroup(slexer *structLexer) (node, error) {
	_, _ = slexer.Next() // (
	disj, err := g.parseDisjunction(slexer)
	if err != nil {
		return nil, err
	}
	next, err := slexer.Next() // )
	if err != nil {
		return nil, err
	}
	if next.Type != ')' {
		return nil, fmt.Errorf("expected ) but got %q", next)
	}
	return disj, nil
}

// A literal string.
//
// Note that for this to match, the tokeniser must be able to produce this string. For example,
// if the tokeniser only produces individual characters but the literal is "hello", or vice versa.
func (g *generatorContext) parseLiteral(lex *structLexer) (node, error) { // nolint: interfacer
	token, err := lex.Next()
	if err != nil {
		return nil, err
	}
	if token.Type != scanner.String && token.Type != scanner.RawString && token.Type != scanner.Char {
		return nil, fmt.Errorf("expected quoted string but got %q", token)
	}
	s := token.Value
	t := rune(-1)
	token, err = lex.Peek()
	if err != nil {
		return nil, err
	}
	if token.Value == ":" && (token.Type == scanner.Char || token.Type == ':') {
		_, _ = lex.Next()
		token, err = lex.Next()
		if err != nil {
			return nil, err
		}
		if token.Type != scanner.Ident {
			return nil, fmt.Errorf("expected identifier for literal type constraint but got %q", token)
		}
		var ok bool
		t, ok = g.Symbols()[token.Value]
		if !ok {
			return nil, fmt.Errorf("unknown token type %q in literal type constraint", token)
		}
	}
	return &literal{s: s, t: t, tt: g.symbolsToIDs[t]}, nil
}
