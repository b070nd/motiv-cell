package segment

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func Sentences(text string) []string {
	var out []string
	start := 0
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		end := i
		for end < len(text) {
			c, n := utf8.DecodeRuneInString(text[end:])
			if c != '"' && c != '\'' && c != ')' && c != '”' && c != '’' {
				break
			}
			end += n
		}
		next := end
		spaces := 0
		for next < len(text) {
			c, n := utf8.DecodeRuneInString(text[next:])
			if !unicode.IsSpace(c) {
				break
			}
			next += n
			spaces++
		}
		if next < len(text) {
			if spaces == 0 {
				continue
			}
			c, _ := utf8.DecodeRuneInString(text[next:])
			if !unicode.IsUpper(c) && !unicode.IsDigit(c) && c != '"' && c != '“' && c != '\'' {
				continue
			}
			if r == '.' && abbreviation(text[start:i-1]) {
				continue
			}
		}
		if s := strings.TrimSpace(text[start:end]); s != "" {
			out = append(out, s)
		}
		start = next
		i = next
	}
	if s := strings.TrimSpace(text[start:]); s != "" {
		out = append(out, s)
	}
	return out
}

func abbreviation(before string) bool {
	cut := strings.LastIndexFunc(before, unicode.IsSpace)
	word := before[cut+1:]
	if utf8.RuneCountInString(word) == 1 {
		r, _ := utf8.DecodeRuneInString(word)
		return unicode.IsLetter(r)
	}
	switch strings.ToLower(word) {
	case "mr", "mrs", "ms", "dr", "prof", "inc", "corp", "co", "ltd", "vs", "st", "no", "u.s", "u.k", "e.g", "i.e", "jr", "sr":
		return true
	}
	return false
}
