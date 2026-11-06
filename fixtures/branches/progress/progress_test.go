package progress

import "testing"

func TestReport(t *testing.T) {
	noteRun(t)
	for _, c := range []struct {
		p    Progress
		want string
	}{
		{Progress{0, 0}, "nothing to do"},
		{Progress{3, 0}, "nothing to do"},
		{Progress{0, 5}, "not started"},
		{Progress{1, 3}, "1 of 3 (33%)"},
		{Progress{2, 3}, "2 of 3 (66%)"},
		{Progress{5, 5}, "finished"},
	} {
		if got := Report(c.p); got != c.want {
			t.Errorf("Report(%+v) = %q, want %q", c.p, got, c.want)
		}
	}
}

func TestPercent(t *testing.T) {
	noteRun(t)
	for _, c := range []struct{ done, total, want int }{
		{1, 4, 25},
		{1, 3, 33},
		{999, 1000, 99},
	} {
		if got := percent(c.done, c.total); got != c.want {
			t.Errorf("percent(%d, %d) = %d, want %d", c.done, c.total, got, c.want)
		}
	}
}
