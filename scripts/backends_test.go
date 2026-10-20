package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// runAllAPI reads the api pattern from run-all.sh, so the two cannot drift.
func runAllAPI(t *testing.T) *regexp.Regexp {
	t.Helper()
	b, err := os.ReadFile("run-all.sh")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^api_variant='([^']*)'$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("run-all.sh has no api_variant='...' line")
	}
	return regexp.MustCompile(string(m[1]))
}

// backendsAgree fails a spec whose backends hrn would read differently from
// run-all's text match: hrn decodes keys without regard to case and decodes
// escapes, and a variant or judge without a backend runs hrn's default, which
// is api when a key is set.
func backendsAgree(text string, api *regexp.Regexp) error {
	var spec struct {
		Matrix []struct {
			Name    string `json:"name"`
			Backend string `json:"backend"`
		} `json:"matrix"`
		Judge *struct {
			Backend string `json:"backend"`
		} `json:"judge"`
	}
	if err := json.Unmarshal([]byte(text), &spec); err != nil {
		return err
	}
	decoded := false
	for _, v := range spec.Matrix {
		if v.Backend == "" {
			return fmt.Errorf("variant %q names no backend", v.Name)
		}
		decoded = decoded || strings.EqualFold(v.Backend, "api")
	}
	if spec.Judge != nil {
		if spec.Judge.Backend == "" {
			return fmt.Errorf("the judge names no backend")
		}
		decoded = decoded || strings.EqualFold(spec.Judge.Backend, "api")
	}
	if matched := api.MatchString(text); matched != decoded {
		return fmt.Errorf("run-all.sh reads api=%v, hrn reads api=%v: write the backend key and value in lowercase, without escapes", matched, decoded)
	}
	return nil
}

func TestSuiteBackendsMatchRunAll(t *testing.T) {
	api := runAllAPI(t)
	files, err := filepath.Glob(filepath.Join("..", "suites", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no suites found: %v", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := backendsAgree(string(b), api); err != nil {
			t.Errorf("%s: %v", filepath.ToSlash(f), err)
		}
	}
}

func TestBackendsAgreeRefusesHiddenAPI(t *testing.T) {
	api := runAllAPI(t)
	for _, tc := range []struct{ name, spec string }{
		{"key case", `{"matrix": [{"name": "a", "Backend": "api"}]}`},
		{"escaped value", `{"matrix": [{"name": "a", "backend": "\u0061pi"}]}`},
		{"escaped key", `{"matrix": [{"name": "a", "b\u0061ckend": "api"}]}`},
		{"no backend", `{"matrix": [{"name": "a", "model": "m"}]}`},
		{"judge without backend", `{"matrix": [{"name": "a", "backend": "claude"}], "judge": {"model": "m"}}`},
		{"api judge", `{"matrix": [{"name": "a", "backend": "claude"}], "judge": {"Backend": "api"}}`},
	} {
		if err := backendsAgree(tc.spec, api); err == nil {
			t.Errorf("%s: accepted %s", tc.name, tc.spec)
		}
	}
	for _, spec := range []string{
		`{"matrix": [{"name": "a", "backend": "api"}]}`,
		`{"matrix": [{"name": "a", "backend": "claude"}, {"name": "b", "backend" : "codex"}]}`,
	} {
		if err := backendsAgree(spec, api); err != nil {
			t.Errorf("refused %s: %v", spec, err)
		}
	}
}
