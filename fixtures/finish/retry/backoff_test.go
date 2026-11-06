package retry

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	noteRun(t)
	for _, c := range []struct {
		n    int
		want time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{4, 8 * time.Second},
		{7, time.Minute},
		{80, time.Minute},
	} {
		if got := Backoff(c.n, time.Second, time.Minute); got != c.want {
			t.Errorf("Backoff(%d) = %v, want %v", c.n, got, c.want)
		}
	}
}
