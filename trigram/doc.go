// Package trigram is the rune trigrams of strings for a substring index, and
// the trigrams a glob, a regular expression or a fuzzy pattern requires of
// any value it matches.
//
// Folding rule: strings.ToLower, then every window of three runes. Index
// values with Of and query with Glob, Regex or Fuzzy, which fold the same
// way, so both sides see the same grams.
package trigram
