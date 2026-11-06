package discount

import (
	"errors"
	"testing"
)

func TestApply(t *testing.T) {
	noteRun(t)
	for _, c := range []struct{ total, pct, want int }{
		{1000, 10, 900},
		{1000, 0, 1000},
		{999, 50, 500},
		{1000, 100, 0},
	} {
		if got, err := Apply(c.total, c.pct); err != nil || got != c.want {
			t.Errorf("Apply(%d, %d) = %d, %v; want %d", c.total, c.pct, got, err, c.want)
		}
	}
	if _, err := Apply(1000, -5); !errors.Is(err, ErrPercent) {
		t.Errorf("Apply(1000, -5) error = %v, want ErrPercent", err)
	}
}
