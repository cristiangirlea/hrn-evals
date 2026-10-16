// Package idleage measures how long a build cache has been idle.
package idleage

import "time"

// AgeAt is how long before now a cache was last written. A write time a hair
// after now (file times and the clock differ by their resolution) is age 0.
func AgeAt(now, written time.Time) time.Duration {
	if d := now.Sub(written); d > 0 {
		return d
	}
	return 0
}
