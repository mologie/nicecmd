package nicecmd

import (
	"strings"
	"unicode"
)

func slug(in string, sep rune) string {
	type slugState int
	const (
		start slugState = iota
		word
		punct
	)
	var s strings.Builder
	s.Grow(len(in) + len(in)/4)
	state := start
	runes := []rune(in)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if !unicode.IsPrint(r) || unicode.IsPunct(r) || unicode.IsSpace(r) {
			state = punct
		} else if state == punct || unicode.IsUpper(r) {
			if s.Len() > 0 && (state != start || i > 0 && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				s.WriteRune(sep)
			}
			state = start
			s.WriteRune(unicode.ToLower(r))
		} else {
			state = word
			s.WriteRune(r)
		}
	}
	return s.String()
}

func screamingSnake(in string) string {
	return strings.ToUpper(slug(in, '_'))
}
