// Package wslpath tells WSL's bash launcher from a bash that runs in the
// Windows filesystem view, as hrn's run_command does before it picks a shell.
package wslpath

import "path/filepath"

// InSystem32 reports whether path's folder is System32. System32\bash.exe
// starts WSL, a different filesystem view, so hrn skips it and looks for Git
// Bash instead.
func InSystem32(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "System32"
}
