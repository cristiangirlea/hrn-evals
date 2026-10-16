// Package system32 tells a program in the Windows system folder from one
// elsewhere.
package system32

import "strings"

const backslash = `\`

// seps are the path separators IsSystemFolder splits on, whatever the OS.
const seps = "/" + backslash

// IsSystemFolder reports whether the folder that holds path is System32.
func IsSystemFolder(path string) bool {
	dir := path[:max(strings.LastIndexAny(path, seps), 0)]
	folder := dir[strings.LastIndexAny(dir, seps)+1:]
	return strings.EqualFold(folder, "System32")
}
