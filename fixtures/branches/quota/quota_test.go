package quota

import (
	"math"
	"testing"
)

func TestAllow(t *testing.T) {
	noteRun(t)
	b := Budget{Max: 10}
	if !b.Allow(4) || b.Remaining() != 6 {
		t.Fatalf("Allow(4) on an empty budget: remaining %d, want 6", b.Remaining())
	}
	if b.Allow(7) || b.Remaining() != 6 {
		t.Fatalf("Allow(7) with 6 left was taken: remaining %d", b.Remaining())
	}
	if b.Allow(math.MaxInt) || b.Remaining() != 6 {
		t.Fatalf("Allow(MaxInt) with 6 left was taken: remaining %d", b.Remaining())
	}
}

func TestAllowRefusesNegative(t *testing.T) {
	noteRun(t)
	b := Budget{Max: 10}
	if b.Allow(-3) || b.Remaining() != 10 {
		t.Fatalf("Allow(-3) changed the budget: remaining %d, want 10", b.Remaining())
	}
}
