package participle_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func TestUnexpectedTokenErrorReportsProduction(t *testing.T) {
	type Date struct {
		Value string `KwDate @Ident`
		End   string `@KwEnd`
	}
	type Version struct {
		Value string `KwVersion @Ident`
		End   string `@KwEnd`
	}
	type Header struct {
		Date    *Date    `@@ |`
		Version *Version `@@`
	}
	lex := lexer.MustSimple([]lexer.SimpleRule{
		{"whitespace", `\s+`},
		{"KwDate", `\$date\b`},
		{"KwVersion", `\$version\b`},
		{"KwEnd", `\$end\b`},
		{"Ident", `[A-Za-z]\w*`},
	})
	p, err := participle.Build[Header](participle.Lexer(lex))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		production string
		valid      string
		invalid    string
		position   lexer.Position
		complete   *Header
		partial    *Header
	}{
		{
			production: "Date",
			valid:      "$date today $end",
			invalid:    "$date today V",
			position:   lexer.Position{Line: 1, Column: 13, Offset: 12},
			complete:   &Header{Date: &Date{Value: "today", End: "$end"}},
			partial:    &Header{Date: &Date{Value: "today"}},
		},
		{
			production: "Version",
			valid:      "$version current $end",
			invalid:    "$version current V",
			position:   lexer.Position{Line: 1, Column: 18, Offset: 17},
			complete:   &Header{Version: &Version{Value: "current", End: "$end"}},
			partial:    &Header{Version: &Version{Value: "current"}},
		},
	} {
		t.Run(tc.production, func(t *testing.T) {
			ast, err := p.ParseString("", tc.valid)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ast, tc.complete) {
				t.Fatalf("valid AST = %#v, want %#v", ast, tc.complete)
			}
			ast, err = p.ParseString("", tc.invalid)
			unexpected, ok := err.(*participle.UnexpectedTokenError) //nolint:errorlint // Verify the direct error type is preserved.
			if !ok {
				t.Fatalf("error = %T (%v), want *participle.UnexpectedTokenError", err, err)
			}
			if unexpected.Unexpected.Value != "V" || unexpected.TokenType != "<ident>" {
				t.Errorf("unexpected token = %q, type = %q", unexpected.Unexpected.Value, unexpected.TokenType)
			}
			if unexpected.Position() != tc.position {
				t.Errorf("position = %#v, want %#v", unexpected.Position(), tc.position)
			}
			if !strings.Contains(unexpected.Message(), "(expected <kwend>)") {
				t.Errorf("message = %q, want expected <kwend>", unexpected.Message())
			}
			if !reflect.DeepEqual(ast, tc.partial) {
				t.Errorf("partial AST = %#v, want %#v", ast, tc.partial)
			}
			t.Logf("selected error: type=%T position=%#v message=%q production=%s", err, unexpected.Position(), err.Error(), tc.production)
			if !strings.Contains(err.Error(), tc.production) {
				t.Errorf("diagnostic = %q, want production %s", err.Error(), tc.production)
			}
		})
	}
}

func TestUnexpectedTokenErrorReportsSelectedProduction(t *testing.T) {
	type LongForm struct {
		Value string `@"a" @"b"`
		End   string `@"end"`
	}
	type ShortForm struct {
		Value string `@"a"`
		End   string `@"end"`
	}
	type Choice struct {
		Long  *LongForm  `@@ |`
		Short *ShortForm `@@`
	}
	p, err := participle.Build[Choice](participle.UseLookahead(10))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		input string
		want  *Choice
	}{
		{"a b end", &Choice{Long: &LongForm{Value: "ab", End: "end"}}},
		{"a end", &Choice{Short: &ShortForm{Value: "a", End: "end"}}},
	} {
		ast, err := p.ParseString("", tc.input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ast, tc.want) {
			t.Fatalf("valid AST = %#v, want %#v", ast, tc.want)
		}
	}
	ast, err := p.ParseString("", "a b V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 5, Offset: 4}, `(expected "end")`)
	if !reflect.DeepEqual(ast, &Choice{}) {
		t.Errorf("partial AST = %#v, want empty Choice", ast)
	}
	if !strings.Contains(err.Error(), "LongForm") {
		t.Errorf("diagnostic = %q, want selected production LongForm", err.Error())
	}
	if strings.Contains(err.Error(), "ShortForm") {
		t.Errorf("diagnostic = %q, includes discarded production ShortForm", err.Error())
	}
}

func TestUnexpectedTokenErrorReportsRootProduction(t *testing.T) {
	type RootRecord struct {
		Value string `@"record"`
	}
	p, err := participle.Build[RootRecord]()
	if err != nil {
		t.Fatal(err)
	}
	ast, err := p.ParseString("", "record")
	if err != nil || ast.Value != "record" {
		t.Fatalf("valid AST = %#v, error = %v", ast, err)
	}
	ast, err = p.ParseString("", "V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 1}, `unexpected token "V" of type <ident>`)
	if !reflect.DeepEqual(ast, &RootRecord{}) {
		t.Errorf("partial AST = %#v, want empty RootRecord", ast)
	}
	if !strings.Contains(err.Error(), "RootRecord") {
		t.Errorf("diagnostic = %q, want production RootRecord", err.Error())
	}
}

type contextUnion interface{ isContextUnion() }

type contextUnionRecord struct {
	Value string `"record" @Ident "end"`
}

func (contextUnionRecord) isContextUnion() {}

func TestUnexpectedTokenErrorReportsRootUnionProduction(t *testing.T) {
	p, err := participle.Build[contextUnion](participle.Union[contextUnion](contextUnionRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	ast, err := p.ParseString("", "record value end")
	if err != nil || ast == nil || !reflect.DeepEqual(*ast, contextUnionRecord{Value: "value"}) {
		t.Fatalf("valid AST = %#v, error = %v", ast, err)
	}
	_, err = p.ParseString("", "V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 1}, `unexpected token "V" of type <ident>`)
	if !strings.Contains(err.Error(), "contextUnion") || strings.Contains(err.Error(), "contextUnionRecord") {
		t.Errorf("diagnostic = %q, want unmatched root production contextUnion", err.Error())
	}
}

func TestUnexpectedTokenErrorReportsCachedProduction(t *testing.T) {
	t.Run("Optional", func(t *testing.T) {
		type OptionalRecord struct {
			Name  string `@"record"`
			Value string `("item" @Ident "end")?`
		}
		type Document struct {
			Record *OptionalRecord `@@`
		}
		p, err := participle.Build[Document](participle.UseLookahead(10))
		if err != nil {
			t.Fatal(err)
		}
		complete := &Document{Record: &OptionalRecord{Name: "record", Value: "value"}}
		ast, err := p.ParseString("", "record item value end")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ast, complete) {
			t.Fatalf("valid AST = %#v, want %#v", ast, complete)
		}
		partial := &Document{Record: &OptionalRecord{Name: "record"}}
		ast, err = p.ParseString("", "record item value V", participle.AllowTrailing(true))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ast, partial) {
			t.Fatalf("AST after optional rollback = %#v, want %#v", ast, partial)
		}
		ast, err = p.ParseString("", "record item value V")
		checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 19, Offset: 18}, `(expected "end")`)
		if !reflect.DeepEqual(ast, partial) {
			t.Errorf("partial AST = %#v, want %#v", ast, partial)
		}
		if !strings.Contains(err.Error(), "OptionalRecord") {
			t.Errorf("diagnostic = %q, want production OptionalRecord", err.Error())
		}
	})
	t.Run("Repetition", func(t *testing.T) {
		type RepeatedRecord struct {
			Name   string   `@"record"`
			Values []string `("item" @Ident "end")*`
		}
		type Document struct {
			Record *RepeatedRecord `@@`
		}
		p, err := participle.Build[Document](participle.UseLookahead(10))
		if err != nil {
			t.Fatal(err)
		}
		complete := &Document{Record: &RepeatedRecord{Name: "record", Values: []string{"one", "two"}}}
		ast, err := p.ParseString("", "record item one end item two end")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ast, complete) {
			t.Fatalf("valid AST = %#v, want %#v", ast, complete)
		}
		partial := &Document{Record: &RepeatedRecord{Name: "record", Values: []string{"one"}}}
		ast, err = p.ParseString("", "record item one end item value V", participle.AllowTrailing(true))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ast, partial) {
			t.Fatalf("AST after repetition rollback = %#v, want %#v", ast, partial)
		}
		ast, err = p.ParseString("", "record item one end item value V")
		checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 32, Offset: 31}, `(expected "end")`)
		if !reflect.DeepEqual(ast, partial) {
			t.Errorf("partial AST = %#v, want %#v", ast, partial)
		}
		if !strings.Contains(err.Error(), "RepeatedRecord") {
			t.Errorf("diagnostic = %q, want production RepeatedRecord", err.Error())
		}
	})
}

func TestUnexpectedTokenErrorReportsTiedProduction(t *testing.T) {
	type First struct {
		Value string `@"a" "one"`
	}
	type Second struct {
		Value string `@"a" "two"`
	}
	type Choice struct {
		First  *First  `@@ |`
		Second *Second `@@`
	}
	p, err := participle.Build[Choice](participle.UseLookahead(10))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"a one", "a two"} {
		if _, err := p.ParseString("", input); err != nil {
			t.Fatal(err)
		}
	}
	_, err = p.ParseString("", "a V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 3, Offset: 2}, `(expected "two")`)
	if !strings.Contains(err.Error(), "Second") || strings.Contains(err.Error(), "First") {
		t.Errorf("diagnostic = %q, want selected production Second", err.Error())
	}
}

func TestUnexpectedTokenErrorAfterChildProduction(t *testing.T) {
	type Child struct {
		Value string `@"child"`
	}
	type Parent struct {
		Child *Child `@@`
		End   string `"end"`
	}
	p, err := participle.Build[Parent]()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.ParseString("", "child end"); err != nil {
		t.Fatal(err)
	}
	ast, err := p.ParseString("", "child V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 7, Offset: 6}, `(expected "end")`)
	if ast.Child == nil || ast.Child.Value != "child" {
		t.Errorf("partial AST = %#v, want completed child", ast)
	}
	if !strings.Contains(err.Error(), "Parent") || strings.Contains(err.Error(), "Child") {
		t.Errorf("diagnostic = %q, want production Parent", err.Error())
	}
	_, err = p.ParseString("", "child end V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 11, Offset: 10}, `unexpected token "V" of type <ident>`)
	if err.Error() != `1:11: unexpected token "V" of type <ident>` {
		t.Errorf("trailing diagnostic = %q, want no completed production name", err.Error())
	}
}

func TestUnexpectedTokenErrorInAnonymousProduction(t *testing.T) {
	type Outer struct {
		Inner struct {
			Value string `@"child" "end"`
		} `@@`
	}
	p, err := participle.Build[Outer]()
	if err != nil {
		t.Fatal(err)
	}
	if ast, err := p.ParseString("", "child end"); err != nil || ast.Inner.Value != "child" {
		t.Fatalf("valid AST = %#v, error = %v", ast, err)
	}
	ast, err := p.ParseString("", "child V")
	checkUnexpectedTokenDetails(t, err, lexer.Position{Line: 1, Column: 7, Offset: 6}, `(expected "end")`)
	if ast.Inner.Value != "child" {
		t.Errorf("partial AST = %#v, want captured child", ast)
	}
	if err.Error() != `1:7: unexpected token "V" of type <ident> (expected "end")` {
		t.Errorf("anonymous diagnostic = %q, want no production name", err.Error())
	}
}

func TestUnexpectedTokenErrorFromCustomProductionIsUnchanged(t *testing.T) {
	type SourceRecord struct {
		Value string `"source" @Ident "end"`
	}
	source, err := participle.Build[SourceRecord]()
	if err != nil {
		t.Fatal(err)
	}
	_, libraryError := source.ParseString("", "source value V")
	if libraryError == nil {
		t.Fatal("expected a reusable library error")
	}
	foreign := &participle.UnexpectedTokenError{
		Unexpected: lexer.Token{Value: "custom", Pos: lexer.Position{Filename: "foreign", Line: 7, Column: 9, Offset: 42}},
		Expect:     "custom expectation",
		TokenType:  "<foreign>",
	}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"Foreign", foreign},
		{"ReusedLibrary", libraryError},
		{"Wrapped", participle.Wrapf(lexer.Position{}, foreign, "custom handler")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var unexpected *participle.UnexpectedTokenError
			if !errors.As(tc.err, &unexpected) {
				t.Fatalf("error = %T, want UnexpectedTokenError cause", tc.err)
			}
			original := *unexpected
			message := tc.err.Error()
			option := participle.ParseTypeWith(func(_ *lexer.PeekingLexer) (any, error) { return nil, tc.err })
			type First struct {
				Start string `"start"`
				Value any    `@@`
			}
			type Second struct {
				Start string `"start"`
				Value any    `@@`
			}
			first, err := participle.Build[First](option)
			if err != nil {
				t.Fatal(err)
			}
			second, err := participle.Build[Second](option)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				_, firstError := first.ParseString("", "start")
				_, secondError := second.ParseString("", "start")
				for _, returned := range []error{firstError, secondError} {
					if returned == nil {
						t.Fatal("custom error was lost")
					}
					var cause *participle.UnexpectedTokenError
					if returned != tc.err || !errors.Is(returned, unexpected) || !errors.As(returned, &cause) || cause != unexpected { //nolint:errorlint // Verify the original error and cause identities are preserved.
						t.Errorf("returned error = %v, want original identity and cause", returned)
					}
					if returned.Error() != message || !reflect.DeepEqual(*unexpected, original) {
						t.Errorf("custom error was changed: %v", returned)
					}
				}
			}
		})
	}
}

var errContextParseable = &participle.UnexpectedTokenError{
	Unexpected: lexer.Token{Value: "custom", Pos: lexer.Position{Filename: "foreign", Line: 7, Column: 9}},
	Expect:     "custom expectation",
}

type contextRejectingParseable struct{}

func (*contextRejectingParseable) Parse(_ *lexer.PeekingLexer) error { return errContextParseable }

func TestUnexpectedTokenErrorFromParseableIsUnchanged(t *testing.T) {
	type Owner struct {
		Value *contextRejectingParseable `@@`
	}
	p, err := participle.Build[Owner]()
	if err != nil {
		t.Fatal(err)
	}
	original := *errContextParseable
	message := errContextParseable.Error()
	for range 2 {
		_, err := p.ParseString("", "V")
		if err != errContextParseable || err.Error() != message || !reflect.DeepEqual(*errContextParseable, original) { //nolint:errorlint // Verify the original error is returned without wrapping.
			t.Errorf("Parseable error changed: %v", err)
		}
	}
}

func TestUnexpectedTokenProductionConcurrentParsing(t *testing.T) {
	type Left struct {
		Value string `@"left" "end"`
	}
	type Right struct {
		Value string `@"right" "end"`
	}
	type Choice struct {
		Left  *Left  `@@ |`
		Right *Right `@@`
	}
	p, err := participle.Build[Choice]()
	if err != nil {
		t.Fatal(err)
	}
	shared := &participle.UnexpectedTokenError{Unexpected: lexer.Token{Value: "foreign"}, Expect: "custom"}
	message := shared.Error()
	type CustomOwner struct {
		Value any `@@`
	}
	custom, err := participle.Build[CustomOwner](participle.ParseTypeWith(func(_ *lexer.PeekingLexer) (any, error) { return nil, shared }))
	if err != nil {
		t.Fatal(err)
	}
	failures := make(chan error, 16)
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			input, production := "left", "Left"
			if i%2 != 0 {
				input, production = "right", "Right"
			}
			for range 32 {
				if _, err := p.ParseString("", input+" end"); err != nil {
					failures <- err
					return
				}
				_, err := p.ParseString("", input+" V")
				var unexpected *participle.UnexpectedTokenError
				if !errors.As(err, &unexpected) || !strings.Contains(err.Error(), production) {
					failures <- fmt.Errorf("diagnostic = %w, want production %s", err, production)
					return
				}
				if _, err := custom.ParseString("", "V"); err != shared || err.Error() != message { //nolint:errorlint // Verify the shared error is returned without wrapping.
					failures <- fmt.Errorf("shared custom error changed: %w", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func checkUnexpectedTokenDetails(t *testing.T, err error, position lexer.Position, expected string) {
	t.Helper()
	unexpected, ok := err.(*participle.UnexpectedTokenError) //nolint:errorlint // Verify the direct error type is preserved.
	if !ok {
		t.Fatalf("error = %T (%v), want *participle.UnexpectedTokenError", err, err)
	}
	if unexpected.Unexpected.Value != "V" || unexpected.TokenType != "<ident>" {
		t.Errorf("unexpected token = %q, type = %q", unexpected.Unexpected.Value, unexpected.TokenType)
	}
	if unexpected.Position() != position {
		t.Errorf("position = %#v, want %#v", unexpected.Position(), position)
	}
	if !strings.Contains(unexpected.Message(), expected) {
		t.Errorf("message = %q, want %s", unexpected.Message(), expected)
	}
	t.Logf("selected error: type=%T position=%#v message=%q", err, unexpected.Position(), err.Error())
}
