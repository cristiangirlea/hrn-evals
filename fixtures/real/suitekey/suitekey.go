// Package suitekey identifies an eval's suite as the losses page does: by its
// task IDs and its matrix variant names.
package suitekey

import (
	"encoding/json"
	"sort"
	"strings"
)

// Spec is what identifies a suite: its task IDs and variant names, in spec
// order, possibly repeated.
type Spec struct {
	Tasks    []string
	Variants []string
}

// Tasks is the spec's task IDs, each once, sorted, joined by ", ".
func Tasks(s Spec) string {
	ids := make([]string, 0, len(s.Tasks))
	seen := make(map[string]bool)
	for _, id := range s.Tasks {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ", ")
}

// Key is the suite's key: Tasks, and the variant names, each once and sorted,
// as a JSON array.
func Key(s Spec) (tasks, variants string) {
	names := make([]string, 0, len(s.Variants))
	seen := make(map[string]bool)
	for _, name := range s.Variants {
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	sort.Strings(names)
	matrix, _ := json.Marshal(names)
	return Tasks(s), string(matrix)
}
