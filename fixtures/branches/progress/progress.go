// Package progress formats how far a job has got.
package progress

import "fmt"

// Progress counts finished steps out of a job's total.
type Progress struct {
	Done  int
	Total int
}

// Report describes p for a status line.
func Report(p Progress) string {
	if p.Total <= 0 {
		return "nothing to do"
	}
	if p.Done <= 0 {
		return "not started"
	}
	if p.Done >= p.Total {
		return "finished"
	}
	return fmt.Sprintf("%d of %d (%d%%)", p.Done, p.Total, percent(p.Done, p.Total))
}

// percent returns done as a whole percentage of total, rounded down.
func percent(done, total int) int {
	return done * 100 / total
}
