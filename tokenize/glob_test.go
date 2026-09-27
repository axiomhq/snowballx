package tokenize

import (
	"testing"
)

// TestGlobGolden encodes globset's documented syntax and semantics
// (https://docs.rs/globset), including the doc examples: `*` crosses `/` at the default literal_separator=false,
// `**`'s three legal positions, alternates, classes, and backslash escapes.
func TestGlobGolden(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		// globset's own doc example: `*.rs` matches foo/bar.rs.
		{"star crosses slash", "*.rs", "foo/bar.rs", true},
		{"star zero chars", "*.rs", ".rs", true},
		{"star no match other ext", "*.rs", "Cargo.toml", false},
		{"question one char", "a?c", "abc", true},
		{"question crosses slash", "a?c", "a/c", true},
		{"question needs one", "a?c", "ac", false},
		// `**/` at the start: zero or more directories.
		{"doublestar prefix zero", "**/foo", "foo", true},
		{"doublestar prefix one", "**/foo", "bar/foo", true},
		{"doublestar prefix many", "**/foo", "bar/baz/foo", true},
		{"doublestar prefix not suffix", "**/foo", "foo/bar", false},
		// `/**` at the end: all sub-entries, not the entry itself.
		{"doublestar suffix one", "foo/**", "foo/a", true},
		{"doublestar suffix deep", "foo/**", "foo/a/b", true},
		{"doublestar suffix not self", "foo/**", "foo", false},
		// `/**/` inside: zero or more directories.
		{"doublestar middle zero", "a/**/b", "a/b", true},
		{"doublestar middle one", "a/**/b", "a/x/b", true},
		{"doublestar middle many", "a/**/b", "a/x/y/b", true},
		{"doublestar middle no word absorb", "a/**/b", "a/xb", false},
		// The lone `**` matches everything; `**` inside a larger component
		// is two consecutive `*` patterns (globset 0.4.20 semantics).
		{"doublestar alone", "**", "any/thing at all", true},
		{"doublestar in component", "**.tsx", "app.tsx", true},
		{"doublestar in component crosses", "**.tsx", "dir/app.tsx", true},
		{"doublestar in component wrong ext", "**.tsx", "dir/app.js", false},
		// Alternates and classes.
		{"alternates", "*.{rs,c}", "main.rs", true},
		{"alternates second", "*.{rs,c}", "main.c", true},
		{"alternates neither", "*.{rs,c}", "main.go", false},
		{"nested alternates allowed", "{a,b{c,d}}", "bc", true},
		{"class", "[ab]c", "ac", true},
		{"class negated", "[!ab]c", "ac", false},
		{"class negated match", "[!ab]c", "xc", true},
		{"class range", "[a-z]9", "f9", true},
		{"class caret is literal member", "[^a]b", "^b", true},
		{"class escapes metachar", "[*]x", "*x", true},
		{"class escapes metachar no match", "[*]x", "ax", false},
		// Backslash escapes (globset's Unix default).
		{"escape star", `a\*c`, "a*c", true},
		{"escape star literal", `a\*c`, "abc", false},
		{"escape question", `\?`, "?", true},
		{"escape non-meta ignores slash", `a\b`, "ab", true},
		{"escaped backslash", `a\\b`, `a\b`, true},
		{"unicode literal", "héllo", "héllo", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			re, err := CompileGlob(tc.pattern, false)
			if err != nil {
				t.Fatalf("CompileGlob(%q): %v", tc.pattern, err)
			}
			if got := re.MatchString(tc.value); got != tc.want {
				t.Fatalf("Glob %q vs %q = %v, want %v", tc.pattern, tc.value, got, tc.want)
			}
		})
	}
}

// TestGlobIllegal encodes globset's parse errors: unclosed classes and
// alternates, dangling escapes, unopened alternates, and invalid ranges.
func TestGlobIllegal(t *testing.T) {
	for _, pattern := range []string{"[ab", "{a,b", "a}", `a\`, "[z-a]", "[]}"} {
		if _, err := CompileGlob(pattern, false); err == nil {
			t.Errorf("CompileGlob(%q) accepted an illegal pattern", pattern)
		}
	}
}

// TestIGlobCase pins IGlob on Unicode case insensitivity and Glob on case
// sensitivity, both full-value matches.
func TestIGlobCase(t *testing.T) {
	re, err := CompileGlob("*.TXT", true)
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("notes.txt") || !re.MatchString("NOTES.txt") {
		t.Fatal("IGlob must fold case")
	}
	glob, err := CompileGlob("*.TXT", false)
	if err != nil {
		t.Fatal(err)
	}
	if glob.MatchString("notes.txt") {
		t.Fatal("Glob must stay case sensitive")
	}
}
