// Package slug turns titles into URL path segments.
package slug

import (
	"strings"
	"unicode"
)

// Make lowercases title, keeps its letters and digits, and joins the runs
// between them with single hyphens, with none at either end.
func Make(title string) string {
	var b strings.Builder
	gap := false
	for _, r := range strings.ToLower(title) {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			gap = true
			continue
		}
		if gap && b.Len() > 0 {
			b.WriteByte('-')
		}
		gap = false
		b.WriteRune(r)
	}
	return b.String()
}
