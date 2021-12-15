package lexer

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

// TextScannerLexer is a lexer that uses the text/scanner module.
var (
	TextScannerLexer Definition = &textScannerLexerDefinition{}

	// DefaultDefinition defines properties for the default lexer.
	DefaultDefinition = TextScannerLexer
)

// NewTextScannerLexer constructs a Definition that uses an underlying scanner.Scanner
//
// "configure" will be called after the scanner.Scanner.Init(r) is called. If "configure"
// is nil a default scanner.Scanner will be used.
func NewTextScannerLexer(configure func(*scanner.Scanner)) Definition {
	return &textScannerLexerDefinition{configure: configure}
}

type textScannerLexerDefinition struct {
	configure func(*scanner.Scanner)
}

func (d *textScannerLexerDefinition) Lex(filename string, r io.Reader) (Lexer, error) {
	l := Lex(filename, r)
	if d.configure != nil {
		d.configure(l.(*textScannerLexer).scanner)
	}
	return l, nil
}

func (d *textScannerLexerDefinition) Symbols() map[string]TokenType {
	return map[string]TokenType{
		"EOF":       EOF,
		"Char":      scanner.Char,
		"Ident":     scanner.Ident,
		"Int":       scanner.Int,
		"Float":     scanner.Float,
		"String":    scanner.String,
		"RawString": scanner.RawString,
		"Comment":   scanner.Comment,
	}
}

func count16(rang unicode.Range16) int {
	return int(((rang.Hi - rang.Lo) / rang.Stride) + 1)
}

func count32(rang unicode.Range32) int {
	return int(((rang.Hi - rang.Lo) / rang.Stride) + 1)
}

func totalRunesInRange(tables []*unicode.RangeTable) int {
	total := 0
	for _, table := range tables {
		for _, r16 := range table.R16 {
			total += count16(r16)
		}
		for _, r32 := range table.R32 {
			total += count32(r32)
		}
	}
	return total
}

// we're pretending the tables are smushed up against
// eachother here
func nthRuneFromTables(at int, tables []*unicode.RangeTable) (ret rune) {
	n := at

	for _, table := range tables {
		for _, rang := range table.R16 {
			num := count16(rang)
			if n <= num-1 {
				return rune(int(rang.Lo) + (int(rang.Stride) * n))
			}
			n -= num
		}
		for _, rang := range table.R32 {
			num := count32(rang)
			if n <= num-1 {
				return rune(int(rang.Lo) + (int(rang.Stride) * n))
			}
			n -= num
		}
	}

	return ' '
}

func randomRune(len int, tables ...*unicode.RangeTable) rune {
	return nthRuneFromTables(
		rand.Intn(len),
		tables)
}

var cleaner = strings.NewReplacer(
	"\x00", "",
)

var defaultTableCount = totalRunesInRange([]*unicode.RangeTable{unicode.Letter, unicode.Symbol, unicode.Number})
var letterTableCount = totalRunesInRange([]*unicode.RangeTable{unicode.Letter})
var letterNumberTableCount = totalRunesInRange([]*unicode.RangeTable{unicode.Letter, unicode.Number})

func randomString(length int, tableLength int, tables ...*unicode.RangeTable) string {
	s := make([]rune, length)

	if len(tables) == 0 {
		tables = append(tables, unicode.Letter, unicode.Symbol, unicode.Number)
	}

	for i := 0; i < length; i++ {
		char := randomRune(tableLength, tables...)
		s = append(s, char)
	}

	return cleaner.Replace(string(s))
}

func (d *textScannerLexerDefinition) Fuzz(t TokenType) string {
	switch t {
	case EOF:
		return ""
	case scanner.Char:
		return string(rune(rand.Intn(math.MaxInt)))
	case scanner.Ident:
		return string(randomRune(letterTableCount, unicode.Letter)) + randomString(rand.Intn(100), letterNumberTableCount, unicode.Letter, unicode.Number)
	case scanner.Int:
		return strconv.Itoa(rand.Int())
	case scanner.Float:
		return strconv.FormatFloat(rand.Float64(), 'f', -1, 64)
	case scanner.String:
		return `"` + strings.ReplaceAll(randomString(rand.Intn(50), defaultTableCount), "\n", " ") + `"`
	case scanner.RawString:
		return "`" + randomString(rand.Intn(50), defaultTableCount) + "`"
	case scanner.Comment:
		return randomString(rand.Intn(50), defaultTableCount)
	default:
		return string(rune(t))
	}
}

// textScannerLexer is a Lexer based on text/scanner.Scanner
type textScannerLexer struct {
	scanner  *scanner.Scanner
	filename string
	err      error
}

// Lex an io.Reader with text/scanner.Scanner.
//
// This provides very fast lexing of source code compatible with Go tokens.
//
// Note that this differs from text/scanner.Scanner in that string tokens will be unquoted.
func Lex(filename string, r io.Reader) Lexer {
	s := &scanner.Scanner{}
	s.Init(r)
	lexer := lexWithScanner(filename, s)
	lexer.scanner.Error = func(s *scanner.Scanner, msg string) {
		lexer.err = errorf(Position(lexer.scanner.Pos()), msg)
	}
	return lexer
}

// LexWithScanner creates a Lexer from a user-provided scanner.Scanner.
//
// Useful if you need to customise the Scanner.
func LexWithScanner(filename string, scan *scanner.Scanner) Lexer {
	return lexWithScanner(filename, scan)
}

func lexWithScanner(filename string, scan *scanner.Scanner) *textScannerLexer {
	scan.Filename = filename
	lexer := &textScannerLexer{
		filename: filename,
		scanner:  scan,
	}
	return lexer
}

// LexBytes returns a new default lexer over bytes.
func LexBytes(filename string, b []byte) Lexer {
	return Lex(filename, bytes.NewReader(b))
}

// LexString returns a new default lexer over a string.
func LexString(filename, s string) Lexer {
	return Lex(filename, strings.NewReader(s))
}

func (t *textScannerLexer) Next() (Token, error) {
	typ := t.scanner.Scan()
	text := t.scanner.TokenText()
	pos := Position(t.scanner.Position)
	pos.Filename = t.filename
	if t.err != nil {
		return Token{}, t.err
	}
	return Token{
		Type:  TokenType(typ),
		Value: text,
		Pos:   pos,
	}, nil
}
