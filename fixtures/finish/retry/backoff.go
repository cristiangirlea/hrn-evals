// Package retry spaces out attempts at a flaky call.
package retry

import "time"

// Backoff returns the wait before attempt n, counting from 1: base, doubled
// for each earlier attempt, and never more than limit.
func Backoff(n int, base, limit time.Duration) time.Duration {
	d := base
	for i := 1; i < n; i++ {
		d *= 2
	}
	if d > limit {
		return limit
	}
	return d
}
