package clamp

import "testing"

func TestClamp(t *testing.T) {
	noteRun(t)
	for _, c := range []struct{ v, lo, hi, want int }{
		{-5, 0, 10, 0},
		{0, 0, 10, 0},
		{7, 0, 10, 7},
		{10, 0, 10, 10},
		{11, 0, 10, 10},
		{3, 3, 3, 3},
		{-2, -4, -1, -2},
	} {
		if got := Clamp(c.v, c.lo, c.hi); got != c.want {
			t.Errorf("Clamp(%d, %d, %d) = %d, want %d", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

func TestClampEmptyRangePanics(t *testing.T) {
	noteRun(t)
	defer func() {
		if recover() == nil {
			t.Error("Clamp(1, 5, 4) did not panic")
		}
	}()
	Clamp(1, 5, 4)
}
