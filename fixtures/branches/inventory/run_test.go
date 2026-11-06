package inventory

import (
	"os"
	"testing"
)

// noteRun appends the running test's name to TESTS-RAN.txt.
func noteRun(t *testing.T) {
	f, err := os.OpenFile("TESTS-RAN.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(t.Name() + "\n"); err != nil {
		t.Fatal(err)
	}
}
