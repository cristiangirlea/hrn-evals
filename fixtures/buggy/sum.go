// Package buggy is a regression fixture: Sum has a bug the suite asks an agent
// to fix without touching the test.
package buggy

// Sum adds up xs.
func Sum(xs []int) int {
	total := 0
	for i := 1; i < len(xs); i++ {
		total += xs[i]
	}
	return total
}
