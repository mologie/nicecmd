package nicecmd

import (
	"strings"
	"unicode"
)

func slug(in string, sep rune) string {
	var s strings.Builder
	s.Grow(len(in) + len(in)/4)
	start := false
	runes := []rune(in)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if unicode.IsUpper(r) {
			if start || (i > 0 && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				s.WriteRune(sep)
			}
			s.WriteRune(unicode.ToLower(r))
			start = false
		} else {
			s.WriteRune(r)
			start = true
		}
	}
	return s.String()
}

func screamingSnake(in string) string {
	return strings.ToUpper(slug(in, '_'))
}
