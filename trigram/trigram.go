package trigram

import (
	"regexp/syntax"
	"strings"
	"unicode"
)

// Of is the trigrams of s: strings.ToLower, then every window of three
// runes, in order and with repeats. A string of fewer than three runes has
// none. The same fold must be used when indexing and when querying.
func Of(s string) []string {
	r := []rune(strings.ToLower(s))
	var out []string
	for i := 0; i+3 <= len(r); i++ {
		out = append(out, string(r[i:i+3]))
	}
	return out
}

// Glob is the trigrams every match of a glob holds: the literal runs
// between `*`, `?` and `[...]`, with `\` escaping the next rune, each run
// through Of.
func Glob(pattern string) []string {
	var out []string
	var literal []rune
	flush := func() {
		out = append(out, Of(string(literal))...)
		literal = literal[:0]
	}
	r := []rune(pattern)
	for i := 0; i < len(r); i++ {
		switch r[i] {
		case '\\':
			if i+1 < len(r) {
				i++
				literal = append(literal, r[i])
			}
		case '*', '?', '[':
			flush()
			if r[i] == '[' {
				for i+1 < len(r) && r[i+1] != ']' {
					i++
				}
				if i+1 < len(r) {
					i++
				}
			}
		default:
			literal = append(literal, r[i])
		}
	}
	flush()
	return out
}

// Regex is the trigrams every match of a regular expression holds, parsed
// with regexp/syntax Perl flags: literals of three or more runes in
// concatenations and captures, each through Of. A pattern that does not
// parse gives nil.
func Regex(pattern string) []string {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil
	}
	var literals []string
	var required func(*syntax.Regexp)
	required = func(re *syntax.Regexp) {
		switch re.Op {
		case syntax.OpConcat:
			for _, child := range re.Sub {
				required(child)
			}
		case syntax.OpCapture:
			required(re.Sub[0])
		case syntax.OpLiteral:
			if re.Flags&syntax.FoldCase == 0 {
				if len(re.Rune) >= 3 {
					literals = append(literals, string(re.Rune))
				}
				return
			}
			// A case-insensitive literal matches every rune of each
			// SimpleFold orbit, but Of folds values with ToLower, which
			// keeps some orbits (σ/ς/Σ, s/ſ) distinct. Only runs of runes
			// whose whole orbit lowercases alike are sound grams.
			start := 0
			for i, r := range re.Rune {
				if lowerFolds(r) {
					continue
				}
				if i-start >= 3 {
					literals = append(literals, string(re.Rune[start:i]))
				}
				start = i + 1
			}
			if len(re.Rune)-start >= 3 {
				literals = append(literals, string(re.Rune[start:]))
			}
		}
	}
	required(re)
	var out []string
	for _, literal := range literals {
		out = append(out, Of(literal)...)
	}
	return out
}

// Fuzzy is the trigrams of pattern and need, how many of them a value within
// distance edits still shares, counted with multiplicity: each edit destroys
// at most three grams. need <= 0 means the grams cannot prune.
func Fuzzy(pattern string, distance int) (grams []string, need int) {
	return Of(pattern), len([]rune(pattern)) - 2 - 3*distance
}

// lowerFolds reports whether every rune in r's SimpleFold orbit lowercases
// to the same rune, so the ToLower-folded trigrams treat them alike.
func lowerFolds(r rune) bool {
	lower := unicode.ToLower(r)
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		if unicode.ToLower(f) != lower {
			return false
		}
	}
	return true
}
