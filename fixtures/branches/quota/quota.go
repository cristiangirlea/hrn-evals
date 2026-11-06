// Package quota counts units against a fixed budget.
package quota

// Budget allows up to Max units in total.
type Budget struct {
	Max  int
	used int
}

// Allow takes n units and reports whether they fit. A request that would
// reach Max is refused, and refused units are not taken.
func (b *Budget) Allow(n int) bool {
	if n < 0 {
		return false
	}
	if n >= b.Max-b.used {
		return false
	}
	b.used += n
	return true
}

// Remaining returns the units not yet taken.
func (b *Budget) Remaining() int {
	return b.Max - b.used
}
