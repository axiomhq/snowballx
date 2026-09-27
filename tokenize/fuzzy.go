package tokenize

import "strings"

// fuzzyMaxCells bounds the dynamic-programming work of one FuzzyContains
// call. With the cutoff below ordinary text costs O(len(text)*maxDist)
// cells, but text that keeps nearly matching the pattern (a long run of one
// rune against a slightly different run) degrades to O(len(text)*len(pattern)).
// Past this many cells the call gives up and reports no match, so a huge
// text against a huge pattern cannot run unbounded; the exact-substring fast
// path is unaffected.
const fuzzyMaxCells = 1 << 26

// FuzzyContains reports whether text has a substring within maxDist
// Levenshtein edits of pattern (Sellers' algorithm): the row for the empty
// pattern prefix is zero at every text position, so a match may start
// anywhere. Exact substrings short-circuit through strings.Contains. The DP
// keeps Ukkonen's last active cell: a cell past last+1 can only reach
// maxDist through the pattern-deletion transition cur[j-1]+1, so each row
// evaluates O(last+maxDist) cells rather than len(pattern), and cells left
// unevaluated are known to exceed maxDist. Work is capped at 2^26 cells.
func FuzzyContains(text, pattern string, maxDist int) bool {
	if strings.Contains(text, pattern) {
		return true
	}
	if maxDist <= 0 {
		return false
	}
	p := []rune(pattern)
	if len(p) <= maxDist {
		return true
	}
	prev := make([]int, len(p)+1)
	cur := make([]int, len(p)+1)
	for j := range prev {
		prev[j] = j
	}
	last := maxDist // largest j with prev[j] <= maxDist; prev[last+1] exceeds it
	cells := 0
	for _, r := range text {
		cur[0] = 0
		active := 0
		j := 1
		for ; j <= last+1 && j <= len(p); j++ {
			cost := 1
			if p[j-1] == r {
				cost = 0
			}
			cur[j] = min(prev[j-1]+cost, prev[j]+1, cur[j-1]+1)
			if cur[j] <= maxDist {
				active = j
			}
		}
		for ; j <= len(p) && cur[j-1] < maxDist; j++ {
			cur[j] = cur[j-1] + 1
			active = j
		}
		if active == len(p) {
			return true
		}
		if active+1 <= len(p) {
			cur[active+1] = maxDist + 1
		}
		cells += j
		if cells > fuzzyMaxCells {
			return false
		}
		last = active
		prev, cur = cur, prev
	}
	return false
}
