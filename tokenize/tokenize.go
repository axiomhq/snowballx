// Package tokenize turns text into full-text search terms: word
// segmentation (word_v0 to word_v4), case folding, a byte length bound,
// Snowball stop word removal, Snowball stemming for eighteen languages and
// ASCII folding, each term with its word position. CompileGlob turns a
// globset pattern into an anchored regular expression.
package tokenize

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/axiomhq/snowballx/greek"
	"github.com/axiomhq/snowballx/stopwords"
	"github.com/blevesearch/snowballstem"
	"github.com/blevesearch/snowballstem/arabic"
	"github.com/blevesearch/snowballstem/danish"
	"github.com/blevesearch/snowballstem/dutch"
	"github.com/blevesearch/snowballstem/english"
	"github.com/blevesearch/snowballstem/finnish"
	"github.com/blevesearch/snowballstem/french"
	"github.com/blevesearch/snowballstem/german"
	"github.com/blevesearch/snowballstem/hungarian"
	"github.com/blevesearch/snowballstem/italian"
	"github.com/blevesearch/snowballstem/norwegian"
	"github.com/blevesearch/snowballstem/portuguese"
	"github.com/blevesearch/snowballstem/romanian"
	"github.com/blevesearch/snowballstem/russian"
	"github.com/blevesearch/snowballstem/spanish"
	"github.com/blevesearch/snowballstem/swedish"
	"github.com/blevesearch/snowballstem/tamil"
	"github.com/blevesearch/snowballstem/turkish"
	"golang.org/x/text/unicode/norm"
)

// DefaultTokenizer is the tokenizer an empty Config.Tokenizer selects.
const DefaultTokenizer = "word_v4"

// defaultMaxTokenLength is the byte bound an empty Config.MaxTokenLength
// selects.
const defaultMaxTokenLength = 39

// Config selects the tokenizer and its filters. The zero value is word_v4,
// lowercased, tokens of at most 39 bytes, no stop word removal, no stemming,
// no ASCII folding. Config is not validated: an unknown tokenizer segments
// like word_v4, an unknown language neither stems nor removes stop words.
type Config struct {
	// Tokenizer is word_v0, word_v1, word_v2, word_v3, word_v4 or
	// pre_tokenized_array; empty means DefaultTokenizer.
	Tokenizer string
	// Language picks the stemmer and the stop word list; empty means english.
	Language string
	// Stemming applies the language's Snowball stemmer.
	Stemming bool
	// RemoveStopwords drops the language's Snowball stop words. Languages
	// without a list (arabic, greek, romanian, tamil, turkish) drop nothing.
	RemoveStopwords bool
	// CaseSensitive skips lowercasing.
	CaseSensitive bool
	// ASCIIFolding folds each term to ASCII where an equivalent exists
	// (é to e), after stemming and stop word removal.
	ASCIIFolding bool
	// MaxTokenLength drops words longer than this many bytes; 0 means 39.
	MaxTokenLength int
}

func (c Config) maxTokenLength() int {
	if c.MaxTokenLength == 0 {
		return defaultMaxTokenLength
	}
	return c.MaxTokenLength
}

// Token is one term with the position of the word it came from. Positions
// count every word, including ones a stop word or length filter drops, so
// phrase distances keep their gaps.
type Token struct {
	Term string
	Pos  int
}

// Terms applies cfg's tokenizer and filters and returns the terms. Index
// and query sides must call it with the same Config to see the same terms.
func Terms(s string, cfg Config) []string {
	tokens := Positions(s, cfg)
	if len(tokens) == 0 {
		return nil
	}
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = t.Term
	}
	return out
}

// Positions is Terms with each term's word position. A pre_tokenized_array
// Config has no string tokenizer and returns nil: its terms come from
// Pretokenized.
func Positions(s string, cfg Config) []Token {
	tokenizer := cfg.Tokenizer
	if tokenizer == "" {
		tokenizer = DefaultTokenizer
	}
	if tokenizer == "pre_tokenized_array" {
		return nil
	}
	if !cfg.CaseSensitive {
		s = strings.ToLower(s)
	}
	words := segmentWords(s, tokenizer)
	maxLen := cfg.maxTokenLength()
	var stops map[string]bool
	if cfg.RemoveStopwords {
		stops = stopwords.Set(language(cfg.Language))
	}
	out := make([]Token, 0, len(words))
	for i, word := range words {
		if len(word) > maxLen { // the bound is in bytes
			continue
		}
		if stops[word] {
			continue
		}
		if cfg.Stemming {
			word = stem(word, cfg.Language)
		}
		if cfg.ASCIIFolding { // after stemming and stop word removal
			word = asciiFold(word)
		}
		out = append(out, Token{Term: word, Pos: i})
	}
	return out
}

// Pretokenized applies the rules that hold for already split tokens: empty
// tokens and tokens over the length bound are dropped, then ASCIIFolding
// applies. Case, stemming and stop words are left alone.
func Pretokenized(tokens []string, cfg Config) []string {
	maxLen := cfg.maxTokenLength()
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" && len(token) <= maxLen {
			if cfg.ASCIIFolding {
				token = asciiFold(token)
			}
			out = append(out, token)
		}
	}
	return out
}

func language(l string) string {
	if l == "" {
		return "english"
	}
	return l
}

// segmentWords splits s into word-like tokens under the tokenizer's rules.
// Each emitted token consumes one position whether or not a filter keeps it.
// The scan decodes runes lazily and slices the input instead of
// materializing runes, so tokenizing allocates only the result slice.
func segmentWords(s, tokenizer string) []string {
	switch tokenizer {
	case "word_v0", "word_v1", "word_v2":
		return segmentLegacy(s, tokenizer)
	default: // word_v3, word_v4: UAX #29-style word segmentation
		return segmentUAX(s)
	}
}

// segmentLegacy implements word_v0..word_v2: contiguous alphanumeric runs.
// word_v1 treats ideographic codepoints as alphanumeric (they join the run);
// word_v2 makes each ideographic codepoint its own token. Emoji glyph
// sequences are tokens — except in word_v0, which discards them.
func segmentLegacy(s, tokenizer string) []string {
	var out []string
	start := -1
	flush := func(end int) {
		if start >= 0 {
			out = append(out, s[start:end])
			start = -1
		}
	}
	for i := 0; i < len(s); {
		if n, ok := emojiRun(s, i); ok {
			flush(i)
			if tokenizer != "word_v0" {
				out = append(out, s[i:i+n])
			}
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.Is(unicode.Ideographic, r):
			if tokenizer == "word_v2" {
				flush(i)
				out = append(out, s[i:i+size])
			} else if start < 0 { // word_v0/v1: ideographic is alphanumeric
				start = i
			}
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if start < 0 {
				start = i
			}
		default:
			flush(i)
		}
		i += size
	}
	flush(len(s))
	return out
}

// midNumLet keeps interior '.' and apostrophes between word runes, per UAX #29
// (MidNumLet | Single_Quote): "can't", "l’été", "e.g", "3.14" are one token.
func midNumLet(r rune) bool { return r == '\'' || r == '’' || r == '.' }

func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

func katakanaJoin(r rune) bool {
	return unicode.Is(unicode.Katakana, r) || r == 0x30FC || r == 0x30A0
}

// segmentUAX implements word_v3/word_v4 over UAX #29 word-break rules,
// pragmatically: alphanumeric runs keep interior MidNumLet characters;
// Han and Hiragana codepoints segment individually (they are not letters
// under the word-break classes); Katakana runs stay together; emoji glyph
// sequences are tokens. Combining marks attach to the preceding token.
func segmentUAX(s string) []string {
	var out []string
	start := -1
	prev := rune(0)
	flush := func(end int) {
		if start >= 0 {
			out = append(out, s[start:end])
			start = -1
		}
	}
	for i := 0; i < len(s); {
		if n, ok := emojiRun(s, i); ok {
			flush(i)
			out = append(out, s[i:i+n])
			i += n
			prev, _ = utf8.DecodeLastRuneInString(s[i-n : i])
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r):
			flush(i)
			out = append(out, s[i:i+size])
		case katakanaJoin(r):
			if start >= 0 && !katakanaJoin(prev) {
				flush(i)
			}
			if start < 0 {
				start = i
			}
		case wordRune(r):
			if start < 0 {
				start = i
			}
		case midNumLet(r):
			next, _ := utf8.DecodeRuneInString(s[i+size:])
			if start < 0 || i+size >= len(s) || !wordRune(prev) || !wordRune(next) {
				flush(i)
			}
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r):
			if start < 0 { // an orphan mark separates and leaves prev untouched
				i += size
				continue
			}
		default:
			flush(i)
			prev = r
			i += size
			continue
		}
		prev = r
		i += size
	}
	flush(len(s))
	return out
}

// emojiRun reports the byte length of the emoji glyph sequence starting at
// s[i:]: one emoji base rune, plus variation selectors, skin tones and
// ZWJ-joined further bases (a ZWJ sequence forms a single glyph). ok is
// false when s[i:] does not start an emoji.
func emojiRun(s string, i int) (int, bool) {
	r, size := utf8.DecodeRuneInString(s[i:])
	if !isEmojiBase(r) {
		return 0, false
	}
	n := size
	for i+n < len(s) {
		r, size := utf8.DecodeRuneInString(s[i+n:])
		if r == 0x200D { // joiner: absorbed only when another emoji base follows
			if next, nextSize := utf8.DecodeRuneInString(s[i+n+size:]); nextSize != 0 && isEmojiBase(next) {
				n += size + nextSize
				continue
			}
			break
		}
		if r == 0xFE0F || r == 0xFE0E || r == 0x20E3 || (r >= 0x1F3FB && r <= 0x1F3FF) {
			n += size
			continue
		}
		break
	}
	return n, true
}

func isEmojiBase(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF: // pictographs, supplemental symbols
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols and dingbats
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // regional indicators
		return true
	case r == 0x00A9 || r == 0x00AE || r == 0x2122 || r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299:
		return true
	}
	return false
}

func asciiFold(s string) string {
	return strings.Map(func(r rune) rune {
		if f, ok := asciiFolds[r]; ok {
			return f
		}
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(s))
}

// asciiFolds covers the Latin characters whose ASCII equivalent exists but
// which NFD decomposition cannot produce: stroke letters, ligatures, and
// sharp s.
var asciiFolds = map[rune]rune{
	'ø': 'o', 'Ø': 'o',
	'đ': 'd', 'Đ': 'd',
	'ł': 'l', 'Ł': 'l',
	'ð': 'd', 'Ð': 'd',
	'ħ': 'h', 'Ħ': 'h',
	'ı': 'i',
	'þ': 't', 'Þ': 't',
	'ß': 's',
	'æ': 'a', 'Æ': 'a',
	'œ': 'o', 'Œ': 'o',
	'ŧ': 't', 'Ŧ': 't',
	'ŉ': 'n',
	'ﬀ': 'f', 'ﬁ': 'f', 'ﬂ': 'f', 'ﬃ': 'f', 'ﬄ': 'f', 'ﬅ': 'f', 'ﬆ': 'f',
}

// stemmers are the Snowball stemmers, one per supported language; greek
// comes from this module, the rest from snowballstem.
var stemmers = map[string]func(*snowballstem.Env) bool{
	"arabic": arabic.Stem, "danish": danish.Stem, "dutch": dutch.Stem,
	"english": english.Stem, "finnish": finnish.Stem, "french": french.Stem,
	"german": german.Stem, "greek": greek.Stem, "hungarian": hungarian.Stem, "italian": italian.Stem,
	"norwegian": norwegian.Stem, "portuguese": portuguese.Stem, "romanian": romanian.Stem,
	"russian": russian.Stem, "spanish": spanish.Stem, "swedish": swedish.Stem,
	"tamil": tamil.Stem, "turkish": turkish.Stem,
}

// stem applies language's Snowball stemmer, english by default. A language
// without a stemmer keeps the word as is.
func stem(word, lang string) string {
	stemmer, ok := stemmers[language(lang)]
	if !ok {
		return word
	}
	env := snowballstem.NewEnv(word)
	stemmer(env)
	if out := env.Current(); out != "" {
		return out
	}
	return word
}
