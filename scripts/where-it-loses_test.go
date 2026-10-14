package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	home := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		path = filepath.Join(home, path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// IDs sort opposite to chronology; fixes must use started, not filenames.
	write("evals/z-old.json", `{
	  "id":"z-old", "batch_id":"batch-old", "started":"2026-10-01T23:30:00-02:00",
	  "spec":{"name":"sample", "tasks":[{"id":"broken","prompt":"PRIVATE PROMPT"}]},
	  "table":{
	    "variants":[
	      {"name":"small","total_cost_usd":1,"median_cost_usd":0.6,"median_context_tokens":20,"cost_is_estimate":true},
	      {"name":"large","total_cost_usd":2,"median_cost_usd":0.5,"median_context_tokens":10}],
	    "tasks":[
	      {"task":"missing","variant":"small","passed":0,"total":2},
	      {"task":"exit-status","variant":"small","passed":0,"total":1},
	      {"task":"broken","variant":"small","passed":1,"total":2},
	      {"task":"broken","variant":"large","passed":2,"total":2}]}}
	`)
	write("evals/a-new.json", `{
	  "id":"a-new", "started":"2026-10-03T00:00:00Z", "spec":{"tasks":[{"id":"broken"}]},
	  "table":{"variants":[{"name":"small","total_cost_usd":1}],
	    "tasks":[{"task":"broken","variant":"small","passed":2,"total":2}]}}
	`)
	write("batches/batch-old.json", `{
	  "tasks":[
	    {"id":"z-old.exit-status.small.r1","attempts":[{"gates":[
	      {"command":"UNKNOWN STATUS","output":"UNKNOWN OUTPUT"},
	      {"command":"EXIT SUCCESS","exit_code":0,"output":"SUCCESS OUTPUT"},
	      {"command":"exit-check","exit_code":1,"output":"exit gate failed"}]}]},
	    {"id":"z-old.broken.small.r1-extra","attempts":[{"gates":[{"command":"WRONG GATE","ok":false,"output":"WRONG OUTPUT"}]}]},
	    {"id":"z-old.broken.large.r1","attempts":[{"gates":[{"command":"OTHER VARIANT","ok":false,"output":"WRONG OUTPUT"}]}]},
	    {"id":"z-old.broken.small.r1","prompt":"PRIVATE BATCH PROMPT","attempts":[
	      {"result":{"output":"PRIVATE AGENT OUTPUT"},"gates":[
	        {"command":"PASSING COMMAND","ok":true,"output":"PASSING OUTPUT"},
	        {"command":"check | verify <safe>","ok":false,"output":"OMITTED LINE\r\nlast one\r\nlast two | <tag>\r\nlast three\r\n"},
	        {"command":"SECOND FAILURE","ok":false,"output":"SECOND OUTPUT"}]}]}]}
	`)
	write("evals/invalid.json", `{bad json`)
	write("evals/null.json", `null`)
	write("page.md", "old generated content\n"+outsideStart+"\nReal run: 2026-10-04, local setup.\n\n<!-- custom -->\n"+outsideEnd+"\n")
	out := filepath.Join(home, "page.md")
	if err := generate(home, out, "scripts/where-it-loses.sh"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	page := string(first)
	for _, want := range []string{
		"Date: 2026-10-03", "Evals read: 2. Unreadable or invalid evals skipped: 2.",
		"### broken", "#### small", "| broken | 1/2 (1 failed) | z-old |",
		"<code>check &#124; verify &lt;safe&gt;</code><br>last one<br>last two &#124; &lt;tag&gt;<br>last three",
		"fixed in eval a-new", "| missing | 0/2 (2 failed) | z-old | (gate output not kept) |",
		"| exit-status | 0/1 (1 failed) | z-old | <code>exit-check</code><br>exit gate failed |",
		"| broken | small / large | broken | z-old | 1/2 / 2/2 | 1 (estimate) / 2 |",
		"fewer passes; higher median cost",
		outsideStart + "\nReal run: 2026-10-04, local setup.\n\n<!-- custom -->\n" + outsideEnd,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in page:\n%s", want, page)
		}
	}
	for _, unwanted := range []string{"PRIVATE", "OMITTED LINE", "PASSING COMMAND", "PASSING OUTPUT", "SECOND FAILURE", "SECOND OUTPUT", "WRONG", "OTHER VARIANT", "old generated content", "UNKNOWN", "EXIT SUCCESS", "SUCCESS OUTPUT"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("leaked %q", unwanted)
		}
	}
	if strings.Index(page, "| broken |") > strings.Index(page, "| missing |") {
		t.Error("tasks not sorted")
	}
	if err := generate(home, out, "scripts/where-it-loses.sh"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("second generation changed the page")
	}
}

func TestCheaper(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b variant
		want bool
	}{
		{"cost first", variant{Cost: 1, Tokens: 100}, variant{Cost: 2, Tokens: 1}, true},
		{"equal zero cost", variant{Tokens: 10}, variant{Tokens: 20}, true},
		{"equal paid cost", variant{Cost: 1, Tokens: 10}, variant{Cost: 1, Tokens: 20}, true},
		{"equal metrics", variant{Cost: 1, Tokens: 10}, variant{Cost: 1, Tokens: 10}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cheaper(tc.a, tc.b); got != tc.want {
				t.Errorf("cheaper = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEmptyHome(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "evals"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(home, "results", "page.md")
	if err := generate(home, out, "generate"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), outsideStart+"\nA run on real work, with date and setup, goes here.\n"+outsideEnd) {
		t.Error("missing first-run placeholder")
	}
	if err := os.WriteFile(out, []byte("outside reports without markers"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(home, out, "generate"); err == nil {
		t.Error("must not overwrite outside reports with malformed markers")
	}
}

func TestUnnamedSuiteFix(t *testing.T) {
	decode := func(s string) evalRecord {
		t.Helper()
		var e evalRecord
		if err := json.Unmarshal([]byte(s), &e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	old := decode(`{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"tasks":[{"id":"b"},{"id":"a"}]}}`)
	task := taskResult{Task: "a", Variant: "small", Total: 1}
	for _, tc := range []struct{ name, spec, variant, want string }{
		{"same set reordered", `[{"id":"a"},{"id":"b"}]`, "small", "fixed in eval later"},
		{"different set", `[{"id":"a"}]`, "small", ""},
		{"other variant", `[{"id":"a"},{"id":"b"}]`, "large", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			later := decode(`{"id":"later","started":"2026-10-02T00:00:00Z","spec":{"tasks":` + tc.spec + `},"table":{"tasks":[{"task":"a","variant":"` + tc.variant + `","passed":1,"total":1}]}}`)
			if got := fixedIn([]evalRecord{later}, old, task); got != tc.want {
				t.Errorf("fixedIn = %q, want %q", got, tc.want)
			}
		})
	}
	if got := old.suite(); got != "a, b" {
		t.Errorf("suite = %q, want sorted ids", got)
	}
}

func TestSuiteFileLabel(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for path, content := range map[string]string{
		"suites/golden/sample.json": `{"tasks":[{"id":"b"},{"id":"a"}]}`,
		"evals/old.json":            `{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"tasks":[{"id":"a"},{"id":"b"}]},"table":{"tasks":[{"task":"a","variant":"small","total":1}]}}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := generate(root, "page.md", "generate"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("page.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "### golden/sample\n") {
		t.Errorf("missing suite file label:\n%s", b)
	}
}
