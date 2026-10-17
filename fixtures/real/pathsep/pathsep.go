// Package pathsep finds path separators in a command line, which may be plain
// text or JSON text, where a backslash is written \\.
package pathsep

import "strings"

// SepAt is the length of the path separator at s[i], 0 for none. On Windows a
// backslash is a separator too, and two of them are one, as JSON holds it.
func SepAt(s string, i int, windows bool) int {
	switch {
	case i >= len(s):
		return 0
	case windows && strings.HasPrefix(s[i:], `\\`):
		return 2
	case s[i] == '/' || windows && s[i] == '\\':
		return 1
	}
	return 0
}
