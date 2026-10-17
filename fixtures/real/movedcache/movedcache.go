// Package movedcache recognises the folders hrn gc leaves behind.
package movedcache

import (
	"path/filepath"
	"strings"
)

// IsMovedCache reports whether dir is a build cache an earlier gc run moved
// aside to remove it. It accepts both the old .gc-<pid>-<name> and the newer
// .gc-<pid>-<time>-<name> form. Both start with a decimal PID and a hyphen.
func IsMovedCache(dir string) bool {
	name, ok := strings.CutPrefix(filepath.Base(dir), ".gc-")
	if !ok || name == "" {
		return false
	}
	i := 0
	for i < len(name) && name[i] >= '0' && name[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(name) && name[i] == '-'
}
