package tokenize

import (
	"strings"
	"testing"
	"time"
)

func TestFuzzyContains(t *testing.T) {
	cases := []struct {
		text, pat string
		d         int
		want      bool
	}{
		{"", "", 0, true},
		{"abc", "", 0, true},
		{"", "a", 0, false},
		{"", "a", 1, true},
		{"héllo wörld", "wörld", 0, true},
		{"héllo wörld", "world", 1, true},
		{"héllo wörld", "world", 0, false},
		{"kitten", "sitting", 2, false},
		{"kitten", "sitting", 3, true},
		{"the quick brown fox", "quikc", 1, true}, // drop the k: "quic"
		{"the quick brown fox", "quikc", 0, false},
		{"the quick brown fox", "qxick", 1, true},
		{"the quick brown fox", "qxick", 0, false},
		{"the quick brown fox", "the quick brown fox ju", 2, false},
		{"the quick brown fox", "the quick brown fox j", 2, true},
		{"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx the quick brown fox", "the quikc brown", 2, true},
		{"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx the quick brown fox", "the quikc brown", 1, false},
		{"aaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaab", 2, false},
		{"aaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaab", 2, true},
	}
	for _, c := range cases {
		if got := FuzzyContains(c.text, c.pat, c.d); got != c.want {
			t.Errorf("FuzzyContains(%q, %q, %d) = %v, want %v", c.text, c.pat, c.d, got, c.want)
		}
	}
}

func TestFuzzyContainsBoundsWork(t *testing.T) {
	text := strings.Repeat("b", 4<<20)
	pattern := strings.Repeat("a", 512<<10)
	for _, c := range []struct {
		text, pat string
		d         int
		want      bool
	}{
		{text, pattern, 2, false},
		{text, pattern, 0, false},
		{strings.Repeat("a", 4<<20), pattern, 2, true},
		{strings.Repeat("a", 4<<20), pattern + "xyz", 2, false},
		{text[:len(text)-3] + "abc", pattern[:3] + "bc", 2, true},
		{text[:len(text)-3] + "abc", pattern[:3] + "bc", 1, false},
	} {
		// The unbounded DP needs ~2 trillion cells here (minutes); the work
		// cap makes it ~2^26. Ten seconds leaves room for -race.
		start := time.Now()
		got := FuzzyContains(c.text, c.pat, c.d)
		if elapsed := time.Since(start); got != c.want || elapsed > 10*time.Second {
			t.Fatalf("FuzzyContains(%d bytes, %d bytes, %d) = %v in %v, want %v", len(c.text), len(c.pat), c.d, got, elapsed, c.want)
		}
	}
}
