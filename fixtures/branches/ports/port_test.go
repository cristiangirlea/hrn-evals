package ports

import "testing"

func TestParsePort(t *testing.T) {
	noteRun(t)
	if p, err := ParsePort("8080"); err != nil || p != 8080 {
		t.Errorf("ParsePort(8080) = %d, %v", p, err)
	}
	for _, s := range []string{"http", "70000", "-1"} {
		if _, err := ParsePort(s); err == nil {
			t.Errorf("ParsePort(%q) accepted it", s)
		}
	}
}
