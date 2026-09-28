package trigram

import (
	"reflect"
	"testing"
)

func TestOf(t *testing.T) {
	for s, want := range map[string][]string{
		"":       nil,
		"ab":     nil,
		"éà":     nil,
		"abc":    {"abc"},
		"HeLLo":  {"hel", "ell", "llo"},
		"Straße": {"str", "tra", "raß", "aße"},
		"ΣΟΦΙΑ":  {"σοφ", "οφι", "φια"},
		"aaaa":   {"aaa", "aaa"},
	} {
		if got := Of(s); !reflect.DeepEqual(got, want) {
			t.Errorf("Of(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestGlob(t *testing.T) {
	for pattern, want := range map[string][]string{
		"*":         nil,
		"ab*cde":    {"cde"},
		"ABC?def":   {"abc", "def"},
		`a\*b`:      {"a*b"},
		`ab\`:       nil,
		"abc[xyz]d": {"abc"},
		"ab[!c]de":  nil,
		"[abc":      nil,
	} {
		if got := Glob(pattern); !reflect.DeepEqual(got, want) {
			t.Errorf("Glob(%q) = %q, want %q", pattern, got, want)
		}
	}
}

func TestRegexFoldCase(t *testing.T) {
	for pattern, want := range map[string][]string{
		`(?i)σσσ`:          nil,                          // σ/ς/Σ orbit: ToLower keeps ς distinct
		`(?i)abc`:          {"abc"},                      // ASCII stays indexed
		`(?i)KKK`:          {"kkk"},                      // Kelvin sign lowercases to k
		`(?i)needles`:      {"nee", "eed", "edl", "dle"}, // s/ſ ends the sound run
		`(?i)pre-σσσ-post`: {"pre", "re-", "-po"},        // unsound runes (σ, s) bound the runs
		`ΣΣΣ`:              {"σσσ"},                      // case-sensitive literal folds like the index
	} {
		if got := Regex(pattern); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %q want %q", pattern, got, want)
		}
	}
}

func TestFuzzy(t *testing.T) {
	for _, c := range []struct {
		pattern  string
		distance int
		need     int
	}{
		{"hello", 0, 3},
		{"hello", 1, 0},
		{"needle", 1, 1},
		{"straße", 2, -2},
		{"ab", 0, 0},
	} {
		grams, need := Fuzzy(c.pattern, c.distance)
		if need != c.need {
			t.Errorf("Fuzzy(%q, %d) need = %d, want %d", c.pattern, c.distance, need, c.need)
		}
		if !reflect.DeepEqual(grams, Of(c.pattern)) {
			t.Errorf("Fuzzy(%q, %d) grams = %q, want Of", c.pattern, c.distance, grams)
		}
	}
}
