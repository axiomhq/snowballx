# snowballx

```sh
go get github.com/axiomhq/snowballx
```

Two packages the Go Snowball runtime, `github.com/blevesearch/snowballstem`, does not ship:

1. `greek`: the Snowball Greek stemmer. `greek.Stem(env)`, same shape as every other language package.
2. `stopwords`: the stop word lists snowballstem.org publishes beside its stemmers, embedded. Thirteen languages.

## Use

```go
import (
	"github.com/blevesearch/snowballstem"
	"github.com/axiomhq/snowballx/greek"
	"github.com/axiomhq/snowballx/stopwords"
)

env := snowballstem.NewEnv("γλώσσες")
greek.Stem(env)
word := env.Current()

stopwords.Is("english", "the")   // true
stopwords.Set("french")          // map[string]bool, shared, read only
stopwords.Languages()            // danish dutch english ... swedish
```

## Facts

- `greek/greek_stemmer.go` is Snowball v3.1.1 compiler output, unedited. Regenerate it, never edit it; the command is in `greek/doc.go`.
- On snowball-data's `greek/voc.txt` (90,727 words) it matches `greek/output.txt` exactly. `greek/stem_test.go` keeps a sample.
- The lists are the published ones, one word per line, lowercase.

## Test

```sh
go test ./...
```

## License

BSD-3-Clause, Snowball's terms, see [LICENSE](LICENSE). The loader and the tests carry the same license.
