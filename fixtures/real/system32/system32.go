// Package system32 tells WSL's bash launcher from a bash that runs in the
// Windows filesystem view.
package system32

import "strings"

const backslash = `\`

// seps are the path separators InSystem32 splits on, whatever the OS.
const seps = "/" + backslash

// InSystem32 reports whether path's folder is System32 in any case: PATH often
// spells it C:\WINDOWS\system32. Both separators count, so it works anywhere.
func InSystem32(path string) bool {
	dir := path[:max(strings.LastIndexAny(path, seps), 0)]
	folder := dir[strings.LastIndexAny(dir, seps)+1:]
	return strings.EqualFold(folder, "System32")
}
