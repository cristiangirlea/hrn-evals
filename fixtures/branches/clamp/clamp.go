// Package clamp has small integer helpers.
package clamp

// Clamp returns v limited to the inclusive range [lo, hi].
// It panics when lo > hi, since that range is empty.
func Clamp(v, lo, hi int) int {
	if lo > hi {
		panic("clamp: lo > hi")
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
