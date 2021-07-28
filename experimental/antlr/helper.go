package antlr

import (
	"regexp"
	"strings"
)

var reAntlrLiteral = regexp.MustCompile(`(\\n|\\r|\\t|\\\\|\\'|.)`)

func antlrLiteralLen(s string) int {
	res := reAntlrLiteral.FindAllIndex([]byte(s), -1)
	return len(res)
}

func orStr(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

func strIn(s string, opt ...string) bool {
	for _, v := range opt {
		if s == v {
			return true
		}
	}
	return false
}

func ifStrPtr(sp *string, quotes ...string) string {
	if sp != nil {
		esc := *sp
		if len(quotes) > 1 {
			return quotes[0] + esc + quotes[1]
		}
		if len(quotes) > 0 {
			return quotes[0] + esc + quotes[0]
		}
		return esc
	}
	return ""
}

func isLexerRule(s string) bool {
	return strings.ToUpper(s[0:1]) == s[0:1]
}

func max(i ...int) (ret int) {
	ret = i[0]
	for _, v := range i {
		if v > ret {
			ret = v
		}
	}
	return
}

func strTern(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func ifStr(b bool, s string) string {
	if b {
		return s
	}
	return ""
}

func strToRegex(s string) string {
	s = stripQuotes(s)
	s = fixUnicodeEscapes(s)
	s = strings.Replace(s, `\'`, "'", -1)
	s = escapeRegexMeta(s)
	return s
}

func stripQuotes(s string) string {
	return s[1 : len(s)-1]
}

func escapeRegexMeta(s string) string {
	for _, c := range ".+*?^$()[]{}|" {
		cs := string(c)
		s = strings.Replace(s, cs, `\`+cs, -1)
	}
	return s
}

func fixUnicodeEscapes(s string) string {
	return reUnicodeEscape.ReplaceAllString(s, `\x{$1}`)
}

func withParens(ret string) string {
	if ret[0] != '(' || ret[len(ret)-1] != ')' {
		ret = "(" + ret + ")"
	}
	return ret
}

func regexCharLen(s string) (count int) {
	for _, c := range s {
		if c != '\\' {
			count++
		}
	}
	return
}
