# snowballx

```sh
go get github.com/axiomhq/snowballx
```

Three packages on top of the Go Snowball runtime, `github.com/blevesearch/snowballstem`:

1. `greek`: the Snowball Greek stemmer, which snowballstem does not ship. `greek.Stem(env)`, same shape as every other language package.
2. `stopwords`: the stop word lists snowballstem.org publishes beside its stemmers, embedded. Thirteen languages.
3. `tokenize`: text to full-text search terms. Word segmentation, case folding, stop words, stemming in eighteen languages, ASCII folding, word positions. Plus a glob compiler.

## Tokenize

1. Pick a `tokenize.Config`. The zero value works: word_v4, lowercased, tokens of at most 39 bytes.
2. Index: `terms := tokenize.Terms(text, cfg)`.
3. Query: tokenize the query with the same `cfg`, so both sides see the same terms.
4. Phrases: `tokenize.Positions(text, cfg)` returns `[]Token{Term, Pos}`. Dropped words keep their position, so phrase gaps stay right.

```go
import "github.com/axiomhq/snowballx/tokenize"

cfg := tokenize.Config{Language: "english", Stemming: true, RemoveStopwords: true}
tokenize.Terms("The foxes are running", cfg)          // [fox run]
tokenize.Positions("the quick fox", cfg)              // [{quick 1} {fox 2}]
tokenize.Pretokenized([]string{"Café", ""}, cfg)      // [Café]
re, err := tokenize.CompileGlob("src/**/*.{go,md}", false)
```

## Config

| field | zero value | effect |
| --- | --- | --- |
| `Tokenizer` | `word_v4` | segmenter, see the next table |
| `Language` | `english` | stemmer and stop word list |
| `Stemming` | off | Snowball stemmer of `Language` |
| `RemoveStopwords` | off | drop Snowball stop words of `Language`; no-op for arabic, greek, romanian, tamil, turkish |
| `CaseSensitive` | off | off lowercases the text first |
| `ASCIIFolding` | off | `é` to `e`, `ß` to `s`, `æ` to `a`; runs after stemming and stop words |
| `MaxTokenLength` | 39 | words longer than this many BYTES are dropped |

`Config` is not validated. An unknown tokenizer segments like word_v4; an unknown language neither stems nor drops stop words.

## Tokenizers

| name | splits |
| --- | --- |
| `word_v4`, `word_v3` | UAX #29 style: letter and digit runs keep inner `'`, `’`, `.` (`can't`, `e.g`, `3.14`); each Han and Hiragana codepoint is a token; Katakana runs stay together; emoji glyphs are tokens |
| `word_v2` | letter and digit runs; each ideograph is a token; emoji glyphs are tokens |
| `word_v1` | like word_v2, but ideographs join the run |
| `word_v0` | like word_v1, but emoji are dropped |
| `pre_tokenized_array` | none: `Terms` returns nil; pass your tokens to `Pretokenized`, which only drops empty and over-long tokens and applies `ASCIIFolding` |

Languages: arabic, danish, dutch, english, finnish, french, german, greek, hungarian, italian, norwegian, portuguese, romanian, russian, spanish, swedish, tamil, turkish.

## Glob

`CompileGlob(pattern, foldCase)` compiles Rust globset syntax to an anchored `*regexp.Regexp`:

| pattern | means |
| --- | --- |
| `*`, `?` | any run, any one character; both cross `/` |
| `**/x`, `x/**`, `a/**/b`, `**` | globstar: zero or more directories, everything below, everything |
| `{a,b}` | alternates, nesting allowed |
| `[ab]`, `[!ab]`, `[a-z]` | classes |
| `\*` | escape |

`foldCase` matches case-insensitively.

## Facts

- `greek/greek_stemmer.go` is Snowball v3.1.1 compiler output, unedited. Regenerate it, never edit it; the command is in `greek/doc.go`.
- On snowball-data's `greek/voc.txt` (90,727 words) it matches `greek/output.txt` exactly. `greek/stem_test.go` keeps a sample.
- The stop word lists are the published ones, one word per line, lowercase.
- `tokenize` pins every stemmer against a Snowball reference pair and fuzzes `Terms` for empty or whitespace tokens.

## Use greek and stopwords directly

```go
env := snowballstem.NewEnv("γλώσσες")
greek.Stem(env)
word := env.Current()

stopwords.Is("english", "the")   // true
stopwords.Set("french")          // map[string]bool, shared, read only
stopwords.Languages()            // danish dutch english ... swedish
```

## Test

```sh
go test ./...
```

## License

`greek` and `stopwords`: BSD-3-Clause, Snowball's terms, see [LICENSE](LICENSE); their loader and tests carry the same license.
`tokenize`: MIT, see [tokenize/LICENSE](tokenize/LICENSE).
