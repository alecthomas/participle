/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2015 Ian Coleman
 * Copyright (c) 2018 Ma_124, <github.com/Ma124>
 * Copyright (c) 2021 tooolbox, <github.com/tooolbox>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, Subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or Substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package antlr

import (
	"strings"
)

// Converts a string to CamelCase
func toCamelInitCase(s string, initCase bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := initCase
	wasLow := false
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		} else if vIsCap && !wasLow {
			v += 'a'
			v -= 'A'
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
		wasLow = vIsLow
	}
	return n.String()
}

// ToCamel converts a string to CamelCase
func toCamel(s string) string {
	return toCamelInitCase(s, true)
}

var charReplacer = strings.NewReplacer(
	"!", "_bang_",
	"@", "_at_",
	"#", "_hash_",
	"$", "_dollar_",
	"%", "_percent_",
	"^", "_caret_",
	"&", "_amp_",
	"*", "_star_",
	"(", "_lparen_",
	")", "_rparen_",
	"-", "_minus_",
	"+", "_plus_",
	"=", "_eq_",
	"[", "_lbkt_",
	"]", "_rbkt_",
	"{", "_lbrc_",
	"}", "_rbrc_",
	"|", "_pipe_",
	";", "_semi_",
	":", "_colon_",
	",", "_comma_",
	"'", "_quo_",
	`"`, "_dquo_",
	"<", "_lt_",
	">", "_gt_",
	".", "_stop_",
	"?", "_query_",
	"\n", "_nl_",
	"\\n", "_nl_",
	"\r", "_cr_",
	"\\r", "_cr_",
	"\t", "_tab_",
	"\\t", "_tab_",
	" ", "_space_",
	"\\", "_bslash_",
	"/", "_fslash_",
	"~", "_tilde_",
	"`", "_tick_",
)

func saySymbols(s string) string {
	return charReplacer.Replace(s)
}
