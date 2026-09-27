package tokenize

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CompileGlob compiles a pattern in the syntax of Rust's globset crate into
// an anchored regular expression that matches whole values:
//
//   - `?` matches any single character and `*` any (possibly empty) sequence;
//     with globset's default literal_separator=false both cross `/`.
//   - `**` as a whole path component is a recursive wildcard: `**/` at the
//     start matches zero or more directories (`**/foo` matches `foo` and
//     `bar/foo` but not `foo/bar`), `/**` at the end matches all sub-entries
//     (`foo/**` matches `foo/a` but not `foo`), `/**/` inside matches zero or
//     more directories, and a lone `**` matches everything. A `**` anywhere
//     else (`**.tsx`) is globset's "two consecutive `*` patterns" — one `*`.
//   - `{a,b}` alternates (nesting allowed), `[ab]`/`[!ab]` character classes,
//     `[*]` class notation, and `\` escapes any metacharacter (before a
//     non-meta character the backslash is ignored; `\\` matches one `\`).
//
// foldCase compiles Unicode case-insensitive matching, globset's
// case_insensitive option.
func CompileGlob(pattern string, foldCase bool) (*regexp.Regexp, error) {
	body, err := globBody(pattern)
	if err != nil {
		return nil, err
	}
	flag := "s"
	if foldCase {
		flag = "si"
	}
	re, err := regexp.Compile("(?" + flag + ")^(?:" + body + ")$")
	if err != nil {
		return nil, fmt.Errorf("glob %q: %w", pattern, err)
	}
	return re, nil
}

// globBody translates the pattern into regex source.
func globBody(pattern string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(pattern); {
		r, size := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		case '\\':
			next, nextSize := utf8.DecodeRuneInString(pattern[i+size:])
			if nextSize == 0 {
				return "", fmt.Errorf("glob %q ends with a bare backslash", pattern)
			}
			b.WriteString(regexp.QuoteMeta(string(next)))
			i += size + nextSize
			continue
		case '*':
			j := i
			for j < len(pattern) && pattern[j] == '*' {
				j++
			}
			stars := j - i
			atStart := i == 0
			atEnd := j == len(pattern)
			afterSlash := !atStart && pattern[i-1] == '/'
			beforeSlash := !atEnd && pattern[j] == '/'
			if stars >= 2 && (atStart || afterSlash) && (atEnd || beforeSlash) {
				// A whole-component globstar, in one of the legal positions.
				switch {
				case atStart && beforeSlash:
					b.WriteString(`(?:.*/)?`)
					j++ // the following '/' is part of `**/`
				case afterSlash && atEnd:
					b.WriteString(`.*`)
				case afterSlash && beforeSlash:
					b.WriteString(`(?:.*/)?`)
					j++
				default: // atStart && atEnd: the glob `**` matches everything
					b.WriteString(`.*`)
				}
				i = j
				continue
			}
			// Two consecutive `*` patterns inside a larger component
			// (`**.tsx`) are one wildcard.
			b.WriteString(`.*`)
			i = j
			continue
		case '?':
			b.WriteString(`.`)
		case '[':
			class, classSize, err := globClass(pattern[i:])
			if err != nil {
				return "", err
			}
			b.WriteString(class)
			i += classSize
			continue
		case '}':
			return "", fmt.Errorf("glob %q: unopened alternate group; missing '{'", pattern)
		case '{':
			end, err := matchGlobBrace(pattern, i)
			if err != nil {
				return "", err
			}
			inner := pattern[i+1 : end]
			var options []string
			braceDepth := 0
			for j := 0; j < len(inner); j++ {
				switch inner[j] {
				case '\\':
					j++
				case '{':
					braceDepth++
				case '}':
					braceDepth--
				case ',':
					if braceDepth == 0 {
						options = append(options, inner[:j])
						inner = inner[j+1:]
						j = -1
					}
				}
			}
			options = append(options, inner)
			b.WriteString("(?:")
			for k, option := range options {
				if k > 0 {
					b.WriteString("|")
				}
				part, err := globBody(option)
				if err != nil {
					return "", err
				}
				b.WriteString(part)
			}
			b.WriteString(")")
			i = end + 1
			continue
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
		i += size
	}
	return b.String(), nil
}

// globClass translates one `[...]` class, already positioned at the '['.
func globClass(pattern string) (string, int, error) {
	runes := []rune(pattern)
	var b strings.Builder
	b.WriteString("[")
	i := 1
	if i < len(runes) && runes[i] == '!' { // globset negation; '^' stays a literal member
		b.WriteString("^")
		i++
	}
	first := true
	for {
		if i >= len(runes) {
			return "", 0, fmt.Errorf("glob %q: unclosed character class; missing ']'", pattern)
		}
		r := runes[i]
		if r == ']' && !first {
			i++
			break
		}
		first = false
		if r == '\\' {
			if i+1 >= len(runes) {
				return "", 0, fmt.Errorf("glob %q ends with a bare backslash", pattern)
			}
			b.WriteString(regexp.QuoteMeta(string(runes[i+1])))
			i += 2
			continue
		}
		switch r {
		case ']', '^', '&':
			b.WriteString("\\" + string(r))
		default:
			b.WriteString(string(r)) // '-' stays raw so Go reads ranges
		}
		i++
	}
	b.WriteString("]")
	return b.String(), len(string(runes[:i])), nil
}

// matchGlobBrace returns the index of the '}' closing the '{' at pattern[i],
// honoring escapes and nested braces.
func matchGlobBrace(pattern string, open int) (int, error) {
	depth := 0
	for i := open; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("glob %q: unclosed alternate group; missing '}'", pattern)
}
