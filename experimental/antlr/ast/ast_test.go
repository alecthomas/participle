package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAntlr(t *testing.T) {
	tt := []struct {
		code string
		sr   string
	}{
		{
			code: `/*
			T-SQL (Transact-SQL, MSSQL) grammar.
			The MIT License (MIT).
			Copyright (c) 2017, Mark Adams (madams51703@gmail.com)
			Copyright (c) 2015-2017, Ivan Kochurkin (kvanttt@gmail.com), Positive Technologies.
			Copyright (c) 2016, Scott Ure (scott@redstormsoftware.com).
			Copyright (c) 2016, Rui Zhang (ruizhang.ccs@gmail.com).
			Copyright (c) 2016, Marcus Henriksson (kuseman80@gmail.com).
			Permission is hereby granted, free of charge, to any person obtaining a copy
			of this software and associated documentation files (the "Software"), to deal
			in the Software without restriction, including without limitation the rights
			to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
			copies of the Software, and to permit persons to whom the Software is
			furnished to do so, subject to the following conditions:
			The above copyright notice and this permission notice shall be included in
			all copies or substantial portions of the Software.
			THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
			IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
			FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
			AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
			LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
			OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
			THE SOFTWARE.
			*/
			
			lexer grammar TSqlLexer;
			
			// Basic keywords (from https://msdn.microsoft.com/en-us/library/ms189822.aspx)
			
			ADD:                                   'ADD';
			ALL:                                   'ALL';
			ALTER:                                 'ALTER';
			AND:                                   'AND';
			ANY:                                   'ANY';
			AS:                                    'AS';
			ASC:                                   'ASC';
			AUTHORIZATION:                         'AUTHORIZATION';
			BACKSLASH:                             '\\';
			BACKUP:                                'BACKUP';`,
			sr: `lexer grammar TSqlLexer;

// Lexer Rules

ADD: 'ADD';
ALL: 'ALL';
ALTER: 'ALTER';
AND: 'AND';
ANY: 'ANY';
AS: 'AS';
ASC: 'ASC';
AUTHORIZATION: 'AUTHORIZATION';
BACKSLASH: '\\';
BACKUP: 'BACKUP';

// Parser Rules

// None

`,
		},
	}

	for _, test := range tt {
		ast := &AntlrFile{}
		if err := Parser.ParseString("", test.code, ast); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, orStr(test.sr, test.code), NewPrinter().Visit(ast))
	}
}

func TestParseGrammarStmt(t *testing.T) {
	tt := []struct {
		code       string
		lexerOnly  bool
		parserOnly bool
		name       string
	}{
		{
			code:       `lexer grammar alpha;`,
			lexerOnly:  true,
			parserOnly: false,
			name:       "alpha",
		},
		{
			code:       `parser grammar Beta;`,
			lexerOnly:  false,
			parserOnly: true,
			name:       "Beta",
		},
		{
			code:       `grammar GAMMA;`,
			lexerOnly:  false,
			parserOnly: false,
			name:       "GAMMA",
		},
	}

	for _, test := range tt {
		ast := &AntlrFile{}
		if err := Parser.ParseString("", test.code, ast); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, test.lexerOnly, ast.Grammar.LexerOnly, "%s: lexer only", test.name)
		assert.Equal(t, test.parserOnly, ast.Grammar.ParserOnly, "%s: parser only", test.name)
		assert.Equal(t, test.name, ast.Grammar.Name, "%s: name", test.name)
	}
}

func TestParseOptionsStmt(t *testing.T) {
	tt := []struct {
		code    string
		keyvals map[string]string
	}{
		{
			code: `options { tokenVocab=TSqlLexer; }`,
			keyvals: map[string]string{
				"tokenVocab": "TSqlLexer",
			},
		},
	}

	p := MustBuildParser(&OptionStmt{})
	for _, test := range tt {
		ast := &OptionStmt{}
		if err := p.ParseString("", test.code, ast); err != nil {
			t.Fatal(err)
		}
		for _, opt := range ast.Opts {
			assert.Equal(t, opt.Value, test.keyvals[opt.Key], "option %s", opt.Key)
		}
	}
}

func TestParseLexRule(t *testing.T) {
	tt := []struct {
		code     string
		fragment bool
		name     string
		skip     bool
		channel  string
		sr       string
	}{
		{
			code: `ADD: 'ADD';`,
			name: "ADD",
		},
		{
			code: `ADD: 'A' 'D' 'D';`,
			name: "ADD",
		},
		{
			code: `DOUBLE_QUOTE_ID:    '"' ~'"'+ '"';`,
			name: "DOUBLE_QUOTE_ID",
			sr:   `DOUBLE_QUOTE_ID: '"' ~'"'+ '"';`,
		},
		{
			code: `SPACE:              [ \t\r\n]+    -> skip;`,
			name: "SPACE",
			skip: true,
			sr:   `SPACE: [ \t\r\n]+ -> skip;`,
		},
		{
			code:    `COMMENT: '/*' (COMMENT | .)*? '*/' -> channel(HIDDEN);`,
			name:    "COMMENT",
			channel: "HIDDEN",
		},
		{
			code: `SQUARE_BRACKET_ID: '[' (~']' | ']' ']')* ']';`,
			name: "SQUARE_BRACKET_ID",
		},
		{
			code: `DECIMAL: DEC_DIGIT+;`,
			name: "DECIMAL",
		},
		{
			code: `REAL: (DECIMAL | DEC_DOT_DEC) ('E' [+-]? DEC_DIGIT+);`,
			name: "REAL",
		},
		{
			code:     `fragment LETTER: [A-Z_];`,
			name:     "LETTER",
			fragment: true,
		},
		{
			code: `fragment FullWidthLetter
: '\u00c0'..'\u00d6'
| '\u00d8'..'\u00f6'
| '\u00f8'..'\u00ff'
| '\u0100'..'\u1fff'
| '\u2c00'..'\u2fff'
| '\u3040'..'\u318f'
| '\u3300'..'\u337f'
| '\u3400'..'\u3fff'
| '\u4e00'..'\u9fff'
| '\ua000'..'\ud7ff'
| '\uf900'..'\ufaff'
| '\uff00'..'\ufff0'
// | '\u10000'..'\u1F9FF'  //not support four bytes chars
// | '\u20000'..'\u2FA1F'
;`,
			name:     "FullWidthLetter",
			fragment: true,
			sr:       `fragment FullWidthLetter: '\u00c0'..'\u00d6' | '\u00d8'..'\u00f6' | '\u00f8'..'\u00ff' | '\u0100'..'\u1fff' | '\u2c00'..'\u2fff' | '\u3040'..'\u318f' | '\u3300'..'\u337f' | '\u3400'..'\u3fff' | '\u4e00'..'\u9fff' | '\ua000'..'\ud7ff' | '\uf900'..'\ufaff' | '\uff00'..'\ufff0';`,
		},
		{
			code: `DISK_DRIVE: [A-Z][:];`,
			name: "DISK_DRIVE",
			sr:   `DISK_DRIVE: [A-Z] [:];`,
		},
		{
			code: `DEFAULT_DOUBLE_QUOTE:                  ["]'DEFAULT'["];`,
			name: "DEFAULT_DOUBLE_QUOTE",
			sr:   `DEFAULT_DOUBLE_QUOTE: ["] 'DEFAULT' ["];`,
		},
		{
			code: `DOUBLE_BACK_SLASH:                     '\\\\';`,
			name: "DOUBLE_BACK_SLASH",
			sr:   `DOUBLE_BACK_SLASH: '\\\\';`,
		},
		{
			code: `BACKSLASH:                             '\\';`,
			name: "BACKSLASH",
			sr:   `BACKSLASH: '\\';`,
		},
	}

	p := MustBuildParser(&LexerRule{})
	for _, test := range tt {

		ast := &LexerRule{}

		toks, e := p.Lex("", strings.NewReader(test.code))
		if e != nil {
			t.Fatal(e)
		}
		if err := p.ParseString("", test.code, ast); err != nil {
			t.Logf("Tokens: %+v", toks)
			t.Fatal(err)
		}

		assert.Equal(t, test.fragment, ast.Fragment, "%s: fragment", test.name)
		assert.Equal(t, test.name, ast.Name, "%s: name", test.name)
		assert.Equal(t, test.skip, ast.Skip, "%s: skip", test.name)
		assert.Equal(t, test.channel, ast.Channel, "%s: channel", test.name)
		assert.Equal(t, orStr(test.sr, test.code), NewPrinter().Visit(ast), "%s: string representation", test.name)
	}
}

func TestParsePrsRule(t *testing.T) {
	tt := []struct {
		title string
		code  string
		name  string
		sr    string
	}{
		{
			title: "plain",
			code: `insert_statement_value
			: table_value_constructor
			| derived_table
			| execute_statement
			| DEFAULT VALUES
			;`,
			name: "insert_statement_value",
			sr:   `insert_statement_value: table_value_constructor | derived_table | execute_statement | DEFAULT VALUES;`,
		},
		{
			title: "more complex",
			code: `select_statement
			: query_expression order_by_clause? for_clause? option_clause? ';'?
			;`,
			name: "select_statement",
			sr:   `select_statement: query_expression order_by_clause? for_clause? option_clause? ';'?;`,
		},
		{
			title: "interleaved comments",
			code: `cursor_statement
			// https://msdn.microsoft.com/en-us/library/ms175035(v=sql.120).aspx
			: CLOSE GLOBAL? cursor_name ';'?
			// https://msdn.microsoft.com/en-us/library/ms188782(v=sql.120).aspx
			| DEALLOCATE GLOBAL? CURSOR? cursor_name ';'?
			// https://msdn.microsoft.com/en-us/library/ms180169(v=sql.120).aspx
			| declare_cursor
			// https://msdn.microsoft.com/en-us/library/ms180152(v=sql.120).aspx
			| fetch_cursor
			// https://msdn.microsoft.com/en-us/library/ms190500(v=sql.120).aspx
			| OPEN GLOBAL? cursor_name ';'?
			;`,
			name: "cursor_statement",
			sr:   `cursor_statement: CLOSE GLOBAL? cursor_name ';'? | DEALLOCATE GLOBAL? CURSOR? cursor_name ';'? | declare_cursor | fetch_cursor | OPEN GLOBAL? cursor_name ';'?;`,
		},
		{
			title: "literals",
			code: `comparison_operator
			: '=' | '>' | '<' | '<' '=' | '>' '=' | '<' '>' | '!' '=' | '!' '>' | '!' '<'
			;`,
			name: "comparison_operator",
			sr:   `comparison_operator: '=' | '>' | '<' | '<' '=' | '>' '=' | '<' '>' | '!' '=' | '!' '>' | '!' '<';`,
		},
		{
			title: "alternative labels", // https://github.com/antlr/antlr4/blob/master/doc/parser-rules.md#alternative-labels
			code: `function_call
			: ranking_windowed_function                         #RANKING_WINDOWED_FUNC
			| aggregate_windowed_function                       #AGGREGATE_WINDOWED_FUNC
			| analytic_windowed_function                        #ANALYTIC_WINDOWED_FUNC
			| built_in_functions                                #BUILT_IN_FUNC
			| scalar_function_name '(' expression_list? ')'     #SCALAR_FUNCTION
			| freetext_function                                 #FREE_TEXT
			| partition_function                                #PARTITION_FUNC
			;`,
			name: "function_call",
			sr:   `function_call: ranking_windowed_function #RANKING_WINDOWED_FUNC | aggregate_windowed_function #AGGREGATE_WINDOWED_FUNC | analytic_windowed_function #ANALYTIC_WINDOWED_FUNC | built_in_functions #BUILT_IN_FUNC | scalar_function_name '(' expression_list? ')' #SCALAR_FUNCTION | freetext_function #FREE_TEXT | partition_function #PARTITION_FUNC;`,
		},
		{
			title: "rule element label", // https://github.com/antlr/antlr4/blob/master/doc/parser-rules.md#rule-element-labels
			code: `try_catch_statement
			: BEGIN TRY ';'? try_clauses=sql_clauses+ END TRY ';'? BEGIN CATCH ';'? catch_clauses=sql_clauses* END CATCH ';'?
			;`,
			name: "try_catch_statement",
			sr:   `try_catch_statement: BEGIN TRY ';'? try_clauses=sql_clauses+ END TRY ';'? BEGIN CATCH ';'? catch_clauses=sql_clauses* END CATCH ';'?;`,
		},
		{
			title: "list label operator",
			code: `create_partition_scheme
			: CREATE PARTITION SCHEME partition_scheme_name=id_
			  AS PARTITION partition_function_name=id_
			  ALL? TO '(' file_group_names+=id_ (',' file_group_names+=id_)* ')'
			;`,
			name: "create_partition_scheme",
			sr:   `create_partition_scheme: CREATE PARTITION SCHEME partition_scheme_name=id_ AS PARTITION partition_function_name=id_ ALL? TO '(' file_group_names+=id_ (',' file_group_names+=id_)* ')';`,
		},
		{
			title: "more labels",
			code: `low_priority_lock_wait
			: WAIT_AT_LOW_PRIORITY '('
			  MAX_DURATION '=' max_duration=time MINUTES? ','
			  ABORT_AFTER_WAIT '=' abort_after_wait=(NONE | SELF | BLOCKERS) ')'
			;`,
			name: "low_priority_lock_wait",
			sr:   `low_priority_lock_wait: WAIT_AT_LOW_PRIORITY '(' MAX_DURATION '=' max_duration=time MINUTES? ',' ABORT_AFTER_WAIT '=' abort_after_wait=(NONE | SELF | BLOCKERS) ')';`,
		},
		{
			title: "empty alternate",
			code: `principal_id:
			| id_
			| PUBLIC
			;`,
			name: "principal_id",
			sr:   `principal_id: | id_ | PUBLIC;`,
		},
	}

	p := MustBuildParser(&ParserRule{})
	for _, test := range tt {

		ast := &ParserRule{}

		toks, e := p.Lex("", strings.NewReader(test.code))
		if e != nil {
			t.Fatal(e)
		}
		if err := p.ParseString("", test.code, ast); err != nil {
			t.Logf("Tokens: %+v", toks)
			t.Fatal(err)
		}

		assert.Equal(t, test.name, ast.Name, "%s: name", test.name)
		assert.Equal(t, orStr(test.sr, test.code), NewPrinter().Visit(ast), "%s: string representation", test.name)
	}
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
