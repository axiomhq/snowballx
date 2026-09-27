// Package stopwords holds the stop word lists snowballstem.org publishes
// alongside its stemmers, one list per language, embedded and parsed once.
package stopwords

import (
	"embed"
	"strings"
)

//go:embed lists/*.txt
var files embed.FS

var byLanguage = load()

func load() map[string]map[string]bool {
	entries, err := files.ReadDir("lists")
	if err != nil {
		panic(err)
	}
	out := make(map[string]map[string]bool, len(entries))
	for _, entry := range entries {
		data, err := files.ReadFile("lists/" + entry.Name())
		if err != nil {
			panic(err)
		}
		words := strings.Fields(string(data))
		set := make(map[string]bool, len(words))
		for _, w := range words {
			set[w] = true
		}
		out[strings.TrimSuffix(entry.Name(), ".txt")] = set
	}
	return out
}

// Languages lists the languages with a stop word list, sorted.
func Languages() []string {
	out := make([]string, 0, len(byLanguage))
	for l := range byLanguage {
		out = append(out, l)
	}
	sortStrings(out)
	return out
}

// Set returns language's stop words (lowercase, as published), or nil when
// the language has no list. The map is shared: do not modify it.
func Set(language string) map[string]bool { return byLanguage[language] }

// Is reports whether word is a stop word of language.
func Is(language, word string) bool { return byLanguage[language][word] }

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
