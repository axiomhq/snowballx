package tokenize

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Hello, World!", []string{"hello", "world"}},
		{"foo_bar baz-2000", []string{"foo", "bar", "baz", "2000"}},
		{"  ", nil},
		{"Grüße münchen", []string{"grüße", "münchen"}},
	}
	for _, c := range cases {
		got := Terms(c.in, Config{})
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Terms(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestTokenizeConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		in   string
		want []string
	}{
		// word_v4/v3 segment CJK ideographs per codepoint (they are not
		// letters under UAX #29's word classes) and keep emoji glyphs.
		{"unicode boundaries", Config{}, "CAFÉ 東京—42", []string{"café", "東", "京", "42"}},
		{"case sensitive", Config{CaseSensitive: true}, "Go go", []string{"Go", "go"}},
		{"english stopwords", Config{RemoveStopwords: true}, "the quick and brown", []string{"quick", "brown"}},
		{"max bytes", Config{MaxTokenLength: 4}, "four café five", []string{"four", "five"}},
		{"english stemming", Config{Stemming: true}, "consign consigned consigning relational conditional skies dying", []string{"consign", "consign", "consign", "relat", "condit", "sky", "die"}},
		// word_v2 classifies by Unicode: é is alphanumeric, ideographic
		// codepoints are their own tokens, emoji glyphs are tokens.
		{"word v2 unicode alnum", Config{Tokenizer: "word_v2"}, "café 42", []string{"café", "42"}},
		{"word v3 unicode apostrophe", Config{Tokenizer: "word_v3"}, "l’été 東京", []string{"l’été", "東", "京"}},
		{"word v4 unicode apostrophe", Config{Tokenizer: "word_v4"}, "can't stop", []string{"can't", "stop"}},
		{"ascii folding", Config{ASCIIFolding: true}, "Crème déjà", []string{"creme", "deja"}},
		{"default max bytes", Config{}, strings.Repeat("a", 40) + " ok", []string{"ok"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Terms(tc.in, tc.cfg); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Terms(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestTokenizerVersionsGolden encodes the splitting rules of each tokenizer
// version: a sample sentence, punctuation and apostrophes, ideographs and
// emoji, and the v2/v1/v0 differences.
func TestTokenizerVersionsGolden(t *testing.T) {
	playground := "Puffy (🐡) visited a café, e.g. searching for clams, but left disappointed."
	cases := []struct {
		name      string
		tokenizer string
		in        string
		want      []string
	}{
		// word_v4 (default) and word_v3: UAX #29 word segmentation. Punctuation
		// is discarded, apostrophes join letters, MidNumLet '.' joins letters
		// ("e.g."), emoji glyph sequences are tokens, Han codepoints segment
		// individually, Katakana runs stay together.
		{"v4 playground", "word_v4", playground, []string{"puffy", "🐡", "visited", "a", "café", "e.g", "searching", "for", "clams", "but", "left", "disappointed"}},
		{"v4 equals v3", "word_v3", playground, []string{"puffy", "🐡", "visited", "a", "café", "e.g", "searching", "for", "clams", "but", "left", "disappointed"}},
		{"v4 ascii apostrophe", "word_v4", "don't stop", []string{"don't", "stop"}},
		{"v4 curly apostrophe", "word_v4", "l’été", []string{"l’été"}},
		{"v4 numeric MidNumLet", "word_v4", "3.14 is pi", []string{"3.14", "is", "pi"}},
		{"v4 han per codepoint", "word_v4", "東京", []string{"東", "京"}},
		{"v4 katakana run", "word_v4", "ウォール", []string{"ウォール"}},
		{"v4 hiragana per codepoint", "word_v4", "はを", []string{"は", "を"}},
		{"v4 emoji zwj glyph", "word_v4", "hi 🧑‍💻!", []string{"hi", "🧑‍💻"}},
		// word_v2: tokens from ideographic codepoints, contiguous alphanumeric
		// sequences, and single-glyph emoji sequences; all else discarded.
		{"v2 alnum runs", "word_v2", "Hello, World! foo_bar", []string{"hello", "world", "foo", "bar"}},
		{"v2 ideographs split", "word_v2", "abc東京def", []string{"abc", "東", "京", "def"}},
		{"v2 emoji kept", "word_v2", "a🐡b", []string{"a", "🐡", "b"}},
		{"v2 punctuation discarded", "word_v2", "e.g. (x)", []string{"e", "g", "x"}},
		// word_v1: like word_v2, but ideographic codepoints are alphanumeric,
		// so they merge into adjacent runs. Emoji still tokenize.
		{"v1 ideographs merge", "word_v1", "abc東京def", []string{"abc東京def"}},
		{"v1 emoji kept", "word_v1", "a🐡b", []string{"a", "🐡", "b"}},
		// word_v0: like word_v1, but emoji codepoints are discarded.
		{"v0 emoji discarded", "word_v0", "a🐡b", []string{"a", "b"}},
		{"v0 ideographs merge", "word_v0", "abc東京def", []string{"abc東京def"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Terms(tc.in, Config{Tokenizer: tc.tokenizer})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Terms(%q, %q) = %q, want %q", tc.in, tc.tokenizer, got, tc.want)
			}
		})
	}
	if got := Terms(playground, Config{}); !reflect.DeepEqual(got, []string{"puffy", "🐡", "visited", "a", "café", "e.g", "searching", "for", "clams", "but", "left", "disappointed"}) {
		t.Fatalf("empty tokenizer must default to word_v4; got %q", got)
	}
}

// TestMaxTokenLengthGolden pins MaxTokenLength: a token's length in BYTES
// (default 39); tokens larger than it are filtered out but still consume a
// position.
func TestMaxTokenLengthGolden(t *testing.T) {
	long := strings.Repeat("ab", 25) // 50 bytes, 25 runes
	if got := Terms("ok "+long+" end", Config{}); !reflect.DeepEqual(got, []string{"ok", "end"}) {
		t.Fatalf("default max_token_length dropped more than the long token: %q", got)
	}
	if got := Terms("café x", Config{MaxTokenLength: 4}); !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("MaxTokenLength is in bytes: café is 5 bytes; got %q", got)
	}
	tokens := Positions("café x y", Config{MaxTokenLength: 4})
	want := []Token{{Term: "x", Pos: 1}, {Term: "y", Pos: 2}}
	if !reflect.DeepEqual(tokens, want) {
		t.Fatalf("dropped tokens must keep their positions: got %+v, want %+v", tokens, want)
	}
}

// TestStopwordsAndLanguagesGolden pins the language behavior: stopword
// removal is language-scoped and a no-op for arabic, greek, romanian, tamil
// and turkish; stemming, ASCII folding and case sensitivity compose.
func TestStopwordsAndLanguagesGolden(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		in   string
		want []string
	}{
		{"english default list", Config{RemoveStopwords: true}, "The quick and the dead", []string{"quick", "dead"}},
		{"english snowball list beyond the old subset", Config{RemoveStopwords: true}, "i will be very quick just because", []string{"will", "quick", "just"}},
		{"english contractions are list words", Config{RemoveStopwords: true}, "you can't do that", nil},
		{"french list is language-scoped", Config{Language: "french", RemoveStopwords: true}, "le chat et the dog", []string{"chat", "the", "dog"}},
		{"french stemming", Config{Language: "french", Stemming: true}, "chevaux", []string{"cheval"}},
		{"arabic removal unsupported", Config{Language: "arabic", RemoveStopwords: true}, "the in", []string{"the", "in"}},
		{"greek removal unsupported", Config{Language: "greek", RemoveStopwords: true}, "the in", []string{"the", "in"}},
		{"romanian removal unsupported", Config{Language: "romanian", RemoveStopwords: true}, "the in", []string{"the", "in"}},
		{"tamil removal unsupported", Config{Language: "tamil", RemoveStopwords: true}, "the in", []string{"the", "in"}},
		{"turkish removal unsupported", Config{Language: "turkish", RemoveStopwords: true}, "the in", []string{"the", "in"}},
		{"no removal by default", Config{}, "the quick", []string{"the", "quick"}},
		{"ascii folding example", Config{ASCIIFolding: true}, "café", []string{"cafe"}},
		{"ascii folding after stemming", Config{Language: "french", Stemming: true, ASCIIFolding: true}, "chevaux", []string{"cheval"}},
		{"case sensitive keeps capital stopword", Config{CaseSensitive: true, RemoveStopwords: true}, "The the cat", []string{"The", "cat"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Terms(tc.in, tc.cfg); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Terms(%q, %+v) = %q, want %q", tc.in, tc.cfg, got, tc.want)
			}
		})
	}
}

// TestTokenizePositionsGolden pins position accounting: every word-like
// token consumes a position even when a filter drops it.
func TestTokenizePositionsGolden(t *testing.T) {
	tokens := Positions("the quick brown fox", Config{RemoveStopwords: true})
	want := []Token{{Term: "quick", Pos: 1}, {Term: "brown", Pos: 2}, {Term: "fox", Pos: 3}}
	if !reflect.DeepEqual(tokens, want) {
		t.Fatalf("got %+v, want %+v", tokens, want)
	}
	tokens = Positions("walrus is lazy", Config{RemoveStopwords: true})
	want = []Token{{Term: "walrus", Pos: 0}, {Term: "lazy", Pos: 2}}
	if !reflect.DeepEqual(tokens, want) {
		t.Fatalf("stopword gaps must survive: got %+v, want %+v", tokens, want)
	}
}

func TestEnglishStemKnownPairs(t *testing.T) {
	for word, want := range map[string]string{
		"generously": "generous", "communism": "communism", "triplicate": "triplic",
		"formative": "format", "formalize": "formal", "electriciti": "electr",
		"hopefulness": "hope", "goodness": "good", "proceed": "proceed",
	} {
		if got := stem(word, "english"); got != want {
			t.Errorf("stem(%q) = %q, want %q", word, got, want)
		}
	}
}

// TestStemEveryLanguage: all 18 languages stem. Each pair is Snowball's
// reference output (snowball-data <language>/output.txt), so a language that
// tokenizes but silently skips stemming fails here.
func TestStemEveryLanguage(t *testing.T) {
	pairs := map[string][2]string{
		"arabic": {"أابيضوا", "ايض"}, "danish": {"fløjten", "fløjt"}, "dutch": {"bijmenging", "bijmeng"},
		"english": {"burgomaster", "burgomast"}, "finnish": {"alkuunkaan", "alku"}, "french": {"cuistres", "cuistr"},
		"german": {"blutgieriges", "blutgier"}, "greek": {"αλλεργιογόνων", "αλλεργιογον"}, "hungarian": {"bevizeztük", "bevizezt"},
		"italian": {"brucerebbe", "bruc"}, "norwegian": {"forsikringsselskapet", "forsikringsselskap"}, "portuguese": {"capas", "cap"},
		"romanian": {"arababură", "arabab"}, "russian": {"вскинулась", "вскинул"}, "spanish": {"chicas", "chic"},
		"swedish": {"ensamen", "ensam"}, "tamil": {"அச்சங்கத்தின்", "சங்கம்"}, "turkish": {"asabiyetin", "asabiyet"},
	}
	if len(pairs) != len(stemmers) {
		t.Fatalf("%d pairs for %d stemmers", len(pairs), len(stemmers))
	}
	for language, pair := range pairs {
		cfg := Config{Language: language, Stemming: true, CaseSensitive: true}
		if got := Terms(pair[0], cfg); len(got) != 1 || got[0] != pair[1] {
			t.Errorf("%s: %q stems to %q, want %q", language, pair[0], got, pair[1])
		}
	}
}

// Every word of up to three letters (with apostrophes, which the stemmer
// strips first) must stem without panicking: the suffix rules slice by
// fixed offsets, and "s" alone used to index before the start of the word.
func TestEnglishStemShortWordsNeverPanic(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz'"
	var words []string
	for _, a := range alphabet {
		words = append(words, string(a))
		for _, b := range alphabet {
			words = append(words, string(a)+string(b))
			for _, c := range alphabet {
				words = append(words, string(a)+string(b)+string(c))
			}
		}
	}
	for _, w := range words {
		if got := stem(w, "english"); len(got) > len(w)+1 {
			t.Fatalf("stem(%q) = %q grew unexpectedly", w, got)
		}
	}
	if got := Terms("s", Config{Stemming: true}); len(got) != 1 {
		t.Fatalf("Terms(\"s\") = %q", got)
	}
}

// FuzzTokenize: Terms is the text trust boundary. Arbitrary UTF-8 and
// invalid bytes must yield only non-empty tokens — word_v0's terms are
// lowercased alphanumeric runs (a separator leaking in would silently
// corrupt the postings keys), and the default tokenizer's UAX #29 terms may
// additionally hold MidNumLet, apostrophes and emoji runes, never whitespace.
func FuzzTokenize(f *testing.F) {
	for _, s := range []string{
		"", " ", "Hello, World!", "foo_bar baz-2000", "Grüße münchen",
		"\xff\xfe\xfd", "\x00\x00", strings.Repeat("a", 4096),
		"ǅǅ ǆ", "𝕳𝖊𝖑𝖑𝖔", "12345", "İSTANBUL", "\xed\xa0\x80",
		"Puffy (🐡) visited a café, e.g. searching", "can't l’été 東京 3.14 🧑‍💻",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, tok := range Terms(string(data), Config{Tokenizer: "word_v0"}) {
			if tok == "" {
				t.Fatalf("empty token from %q", data)
			}
			if tok != strings.ToLower(tok) {
				t.Fatalf("token %q not lowercased (input %q)", tok, data)
			}
			for _, r := range tok {
				if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
					t.Fatalf("token %q holds separator %q (input %q)", tok, r, data)
				}
			}
		}
		for _, tok := range Terms(string(data), Config{}) {
			if tok == "" {
				t.Fatalf("empty token from %q", data)
			}
			if tok != strings.ToLower(tok) {
				t.Fatalf("token %q not lowercased (input %q)", tok, data)
			}
			for _, r := range tok {
				if unicode.IsSpace(r) {
					t.Fatalf("token %q holds whitespace %q (input %q)", tok, r, data)
				}
			}
		}
	})
}
