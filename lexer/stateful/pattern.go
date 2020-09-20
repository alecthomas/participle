package stateful

import (
	"fmt"
	"regexp/syntax"
	"unicode"
)

///////////////////////////////////////////
// These functions analyze a Regexp AST to determine for a given pattern ;
//   - what possible first runes it would match on
//   - what possible match size can be expected (possibly zero, exactly one, one or more)
// The possible match size is for now not utilized, but could potentially be used to not
// even launch a regexp match if the rune is found is in the input.
///////////////////////////////////////////

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// If a character class has a range bigger than this, then give up on it and mark it as always to test,
// since otherwise it would take too much memory.
const (
	charclassSizeLimit = 4096
	mayMatchZero       = 0
	mayMatchOneOrMore  = 1
	matchesExactlyOne  = 2
)

// Get all the possible starting points of a regexp that resolve to runes
// Returns
//   - the list of nodes that represent a potential starting point
//   - an int indicating the match nature
//       * a zero length match is possible
//       * the match can be one or more characters
//			 * the whole pattern matches only exactly one character
func getPotentialStarts(r *syntax.Regexp) ([]*syntax.Regexp, int) {
	var (
		result []*syntax.Regexp
	)

	switch r.Op {
	// Disjunction. The match nature returned is by priority ; zero-match, one-or-more, exactly one.
	case syntax.OpAlternate:
		var mayresult = matchesExactlyOne // will go downward with the arms that reply less
		for _, reg := range r.Sub {
			res, may := getPotentialStarts(reg)
			mayresult = min(mayresult, may)
			result = append(result, res...)
		}
		return result, mayresult

	// A sequence. Checks that it has at least one non-zero length pattern and always returns one or more.
	case syntax.OpConcat:
		for _, reg := range r.Sub {
			res, may := getPotentialStarts(reg)
			result = append(result, res...)
			if may != mayMatchZero {
				return result, mayMatchOneOrMore
			}
		}
		return result, mayMatchZero

	case syntax.OpBeginLine,
		syntax.OpBeginText,
		syntax.OpWordBoundary,
		syntax.OpNoWordBoundary,
		syntax.OpNoMatch,
		syntax.OpEmptyMatch,
		syntax.OpEndLine,
		syntax.OpEndText:
		// We just ignore those as being "irrelevant" in the search for runes, as they are just "helpers"
		// to perform certain cases of matches.
		// Moreover, they're zero-length matches, which by itself will trigger an error for the regexp analysis
		// if unused along stuff that actually consume runes.
		return []*syntax.Regexp{}, mayMatchZero

	case syntax.OpQuest, syntax.OpStar:
		res, _ := getPotentialStarts(r.Sub[0])
		return res, mayMatchZero

	case syntax.OpRepeat:
		res, nb := getPotentialStarts(r.Sub[0])
		if r.Min == 0 {
			return res, mayMatchZero
		} else if r.Min == 1 && r.Max == 1 {
			return res, nb
		}
		return res, mayMatchOneOrMore

	case syntax.OpCapture:
		return getPotentialStarts(r.Sub[0])

	case syntax.OpPlus:
		var nodes, nb = getPotentialStarts(r.Sub[0])
		return nodes, min(nb, mayMatchOneOrMore) // (^\b+) is a potential zero length match

	case syntax.OpLiteral:
		// Literal can also visibly be a sequence of characters, hence the check on the length
		// of the Runes.
		res := []*syntax.Regexp{r}
		if len(r.Rune) > 1 {
			return res, mayMatchOneOrMore
		}
		return res, matchesExactlyOne

	case syntax.OpCharClass, syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		// These are one character matches.
		return []*syntax.Regexp{r}, matchesExactlyOne
	}

	panic("should not get here")
}

// appendRange returns the result of appending the range lo-hi to the class r.
// function stolen from syntax/parse.go
func appendRange(r []rune, lo, hi rune) []rune {
	// Expand last range or next to last range if it overlaps or abuts.
	// Checking two ranges helps when appending case-folded
	// alphabets, so that one range can be expanding A-Z and the
	// other expanding a-z.
	n := len(r)
	for i := 2; i <= 4; i += 2 { // twice, using i=2, i=4
		if n >= i {
			rlo, rhi := r[n-i], r[n-i+1]
			if lo <= rhi+1 && rlo <= hi+1 {
				if lo < rlo {
					r[n-i] = lo
				}
				if hi > rhi {
					r[n-i+1] = hi
				}
				return r
			}
		}
	}

	return append(r, lo, hi)
}

func computeNumberOfRunes(arg []rune) int {
	var acc = 0
	for i, l := 0, len(arg); i < l; i += 2 {
		acc += int(arg[i+1] - arg[i])
	}
	return acc
}

type computedRuneRange struct {
	pattern string
	runes   []rune
	size    int
	nbmatch int
}

// computeRuneRanges gives the rune ranges that are potential starts for the regexp
// it returns the ranges in a single slice as [start,stop,start,stop,...]. The ranges
// are inclusive.
// The second return value is the number of runes included.
// FIXME ; the ranges should be simplified when added
func computeRuneRanges(pattern string) (*computedRuneRange, error) {
	var (
		syn   *syntax.Regexp
		res   []rune
		count = 0
		err   error
	)

	if syn, err = syntax.Parse(pattern, syntax.Perl); err != nil {
		return nil, fmt.Errorf("could not compute rune ranges for pattern /%s/: %w", pattern, err)
	}
	// this step is not required and may actually take up time for nothing, as the pattern seems to be
	// somewhat simplified by default
	// syn = syn.Simplify()

	// should also check for match length
	var startNodes, nbmatch = getPotentialStarts(syn)

	if nbmatch == mayMatchZero || startNodes == nil {
		// If there is at least one non-definite match, then we have an error on our hands !
		// FIXME this has to be handled
		return nil, fmt.Errorf(`pattern /%s/ may match zero times`, pattern)
	}

	for _, start := range startNodes {
		switch start.Op {
		case syntax.OpCharClass:
			for i, l := 0, len(start.Rune); i < l; i += 2 {
				var lo, hi = start.Rune[i], start.Rune[i+1]
				res = appendRange(res, lo, hi)
			}

		case syntax.OpLiteral:
			var val = start.Rune[0]
			res = appendRange(res, val, val)
			if start.Flags&syntax.FoldCase != 0 {
				// If the match is case insensitive, add also its case counterpart.
				folded := unicode.SimpleFold(val)
				res = appendRange(res, folded, folded)
			}
			count++
		case syntax.OpAnyChar:
			return &computedRuneRange{pattern, []rune{0, 1114111}, 1114111, nbmatch}, nil
		case syntax.OpAnyCharNotNL:
			res = appendRange(res, 0, 9)
			res = appendRange(res, 11, 1114111)
		default:
			// This should not happen, the 0 cases should be handled
			return nil, fmt.Errorf("an unknown error while looking for pattern starts has occurred")
		}
	}

	count = computeNumberOfRunes(res)
	return &computedRuneRange{pattern, res, count, nbmatch}, nil
}
