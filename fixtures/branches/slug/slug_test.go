package slug

import "testing"

func TestMake(t *testing.T) {
	noteRun(t)
	for title, want := range map[string]string{
		"Hello, World!":     "hello-world",
		"  Go 1.26 notes  ": "go-1-26-notes",
		"Crème brûlée":      "crème-brûlée",
		"---":               "",
		"":                  "",
		"a--b":              "a-b",
	} {
		if got := Make(title); got != want {
			t.Errorf("Make(%q) = %q, want %q", title, got, want)
		}
	}
}
