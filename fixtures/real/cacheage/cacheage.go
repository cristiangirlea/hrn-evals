// Package cacheage decides which build caches hrn gc removes.
package cacheage

import (
	"fmt"
	"sort"
	"time"
)

// Cache is a build cache and the last time a build wrote to it.
type Cache struct {
	Path      string
	LastBuild time.Time
}

// Decide says whether gc removes c: when it has been idle for at least idle.
// hrn gc --idle 0 means every cache.
func Decide(c Cache, now time.Time, idle time.Duration) (remove bool, why string) {
	age := now.Sub(c.LastBuild)
	if age >= idle {
		return true, "idle " + age.Round(time.Second).String()
	}
	return false, fmt.Sprintf("last written %s ago, inside --idle %s", age.Round(time.Second), idle)
}

// Candidates lists the caches --free-below may take to free space, oldest
// first: those idle for at least minIdle.
func Candidates(caches []Cache, now time.Time, minIdle time.Duration) []Cache {
	var out []Cache
	for _, c := range caches {
		if now.Sub(c.LastBuild) >= minIdle {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].LastBuild.Before(out[b].LastBuild) })
	return out
}
