// Package cellescape makes text safe for one cell of a Markdown table on the
// losses page, whatever a gate printed.
package cellescape

import (
	"html"
	"strings"
)

// Cell escapes s for a Markdown table cell: HTML is escaped, a pipe cannot end
// the cell, a line break becomes <br>, and Markdown's emphasis, code and link
// characters are written as character references.
func Cell(s string) string {
	s = html.EscapeString(s)
	s = strings.NewReplacer("|", "&#124;", "\r", "", "\n", "<br>", "`", "&#96;", "*", "&#42;", "_", "&#95;", "[", "&#91;", "]", "&#93;", "\\", "&#92;").Replace(s)
	return s
}
