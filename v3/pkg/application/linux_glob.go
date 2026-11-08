//go:build linux && !android && !server

package application

import "strings"

// caseGlobs makes a file filter's ';'-separated glob patterns match either letter case: GTK matches patterns
// case-sensitively, so "*.zip" alone hides "MOD.ZIP". Each letter outside a bracket expression becomes a
// two-case class ("*.[zZ][iI][pP]"); characters inside an existing bracket expression are kept as written.
func caseGlobs(pattern string) string {
	var b strings.Builder
	inClass := false
	for _, r := range pattern {
		switch {
		case inClass:
			if r == ']' {
				inClass = false
			}
			b.WriteRune(r)
		case r == '[':
			inClass = true
			b.WriteRune(r)
		default:
			lower, upper := strings.ToLower(string(r)), strings.ToUpper(string(r))
			if lower == upper {
				b.WriteRune(r)
				continue
			}
			b.WriteString("[" + lower + upper + "]")
		}
	}
	return b.String()
}
