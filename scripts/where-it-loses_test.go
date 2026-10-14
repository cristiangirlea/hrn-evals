package main

import (
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
	      {"task":"broken","variant":"small","passed":1,"total":2},
	      {"task":"broken","variant":"large","passed":2,"total":2}]}}
	`)
	write("evals/a-new.json", `{
	  "id":"a-new", "started":"2026-10-03T00:00:00Z", "spec":{"name":"sample"},
	  "table":{"variants":[{"name":"small","total_cost_usd":1}],
	    "tasks":[{"task":"broken","variant":"small","passed":2,"total":2}]}}
	`)
	write("batches/batch-old.json", `{
	  "tasks":[
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
		"### sample", "#### small", "| broken | 1/2 (1 failed) | z-old |",
		"<code>check &#124; verify &lt;safe&gt;</code><br>last one<br>last two &#124; &lt;tag&gt;<br>last three",
		"fixed in eval a-new", "| missing | 0/2 (2 failed) | z-old | (gate output not kept) |",
		"| sample | small / large | broken | z-old | 1/2 / 2/2 | 1 (estimate) / 2 |",
		"fewer passes; higher median cost",
		outsideStart + "\nReal run: 2026-10-04, local setup.\n\n<!-- custom -->\n" + outsideEnd,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in page:\n%s", want, page)
		}
	}
	for _, unwanted := range []string{"PRIVATE", "OMITTED LINE", "PASSING COMMAND", "PASSING OUTPUT", "SECOND FAILURE", "SECOND OUTPUT", "WRONG", "OTHER VARIANT", "old generated content"} {
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
	if (evalRecord{ID: "unnamed"}).suite() != "unnamed" {
		t.Error("unnamed suite must use eval ID")
	}
	if err := os.WriteFile(out, []byte("outside reports without markers"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(home, out, "generate"); err == nil {
		t.Error("must not overwrite outside reports with malformed markers")
	}
}
