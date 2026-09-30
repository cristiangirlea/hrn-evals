package buggy

import "testing"

func TestSum(t *testing.T) {
	for _, tc := range []struct {
		in   []int
		want int
	}{
		{nil, 0},
		{[]int{5}, 5},
		{[]int{1, 2, 3}, 6},
		{[]int{-4, 4, 10}, 10},
	} {
		if got := Sum(tc.in); got != tc.want {
			t.Errorf("Sum(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
