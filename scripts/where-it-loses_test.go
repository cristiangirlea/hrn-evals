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
		"fixed in eval a-new", "| missing | 0/2 (2 failed) | z-old | (attempt not found) |",
		"| exit-status | 0/1 (1 failed) | z-old | <code>exit-check</code><br>exit gate failed |",
		"| broken | small / large | broken | z-old | 1/2 / 2/2 | 1 (estimate) / 2 (estimate) |",
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

func TestNoGateRan(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "batches"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "batches", "batch.json")
	if err := os.WriteFile(path, []byte(`{"tasks":[{"id":"eval.task.small.r1","attempts":[{"result":{"status":"ok"},"gates":[{"ok":true}]}]},{"id":"eval.task.small.r2","attempts":[{"result":{"status":"timeout"},"gates":[]}]}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	e := evalRecord{ID: "eval", BatchID: "batch"}
	task := taskResult{Task: "task", Variant: "small"}
	if got := gateMessage(home, e, task); got != "(no gate ran: timeout)" {
		t.Errorf("gateMessage = %q", got)
	}
	e.BatchID = "missing"
	if got := gateMessage(home, e, task); got != "(gate output not kept)" {
		t.Errorf("missing batch = %q", got)
	}
}

func TestKilledGate(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "batches"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "batches", "batch.json"), []byte(`{"tasks":[{"id":"eval.task.small.r1","attempts":[{"gates":[{"command":"killed-check","exit_code":-1,"output":"killed"}]}]}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := gateMessage(home, evalRecord{ID: "eval", BatchID: "batch"}, taskResult{Task: "task", Variant: "small"}); got != "<code>killed-check</code><br>killed" {
		t.Errorf("killed gate = %q", got)
	}
}

func TestCellEscapes(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"lone carriage return", "a\rb", "ab"},
		{"asterisk", "*", "&#42;"},
		{"underscore", "_", "&#95;"},
		{"backtick", "`", "&#96;"},
		{"backslash", "\\", "&#92;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cell(tc.input); got != tc.want {
				t.Errorf("cell = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestComparisonRequiresSharedTask(t *testing.T) {
	var e evalRecord
	e.Table.Variants = []variant{{Name: "small", Cost: 1, MedianCost: 3}, {Name: "large", Cost: 2, MedianCost: 1}}
	e.Table.Tasks = []taskResult{{Task: "only-small", Variant: "small", Total: 1}, {Task: "only-large", Variant: "large", Passed: 1, Total: 1}}
	var page strings.Builder
	writeComparisons(&page, []evalRecord{e})
	if strings.Contains(page.String(), "| Suite |") || strings.Contains(page.String(), "only-small") || strings.Contains(page.String(), "only-large") {
		t.Errorf("comparison row for unshared tasks:\n%s", page.String())
	}
	if !strings.Contains(page.String(), "No cheaper-variant losses") {
		t.Error("missing empty comparisons note")
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

func TestAllCostsAreEstimates(t *testing.T) {
	var e evalRecord
	if err := json.Unmarshal([]byte(`{"table":{"variants":[
		{"name":"codex","total_cost_usd":0,"median_cost_usd":0},
		{"name":"claude","total_cost_usd":1,"median_cost_usd":0.5},
		{"name":"api","total_cost_usd":2,"median_cost_usd":0.75,"cost_is_estimate":true}],
		"tasks":[{"task":"task","variant":"codex","passed":0,"total":3},
		{"task":"task","variant":"claude","passed":1,"total":3},
		{"task":"task","variant":"api","passed":3,"total":3}]}}`), &e); err != nil {
		t.Fatal(err)
	}
	var page strings.Builder
	writeComparisons(&page, []evalRecord{e})
	for _, want := range []string{
		"Every cost shown is a list-price estimate, hrn's or the CLI's, never a bill.",
		"| 0 / 1 (estimate) | 0 / 0.5 (estimate) |",
		"| 0 / 2 (estimate) | 0 / 0.75 (estimate) |",
		"| 1 (estimate) / 2 (estimate) | 0.5 (estimate) / 0.75 (estimate) |",
	} {
		if !strings.Contains(page.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(page.String(), "0 (estimate)") {
		t.Error("zero cost marked as an estimate")
	}
}

func TestSuiteMatrixLabels(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("suites/routing/first.json", `{"tasks":[{"id":"b"},{"id":"a"}],"matrix":[{"name":"small"},{"name":"large"}]}`)
	write("suites/routing/second.json", `{"tasks":[{"id":"a"},{"id":"b"}],"matrix":[{"name":"small"},{"name":"other"}]}`)
	for _, tc := range []struct{ id, tasks, matrix, want string }{
		{"first", `[{"id":"a"},{"id":"b"},{"id":"a"}]`, `[{"name":"large"},{"name":"small"}]`, "routing/first"},
		{"second", `[{"id":"b"},{"id":"a"}]`, `[{"name":"other"},{"name":"small"}]`, "routing/second"},
		{"task-fallback", `[{"id":"b"},{"id":"a"}]`, `[{"name":"unknown"}]`, "routing/first"},
		{"ids-fallback", `[{"id":"z"},{"id":"a"}]`, `[{"name":"small"},{"name":"large"}]`, "a, z"},
	} {
		write("evals/"+tc.id+".json", `{"id":"`+tc.id+`","started":"2026-10-15T00:00:00Z","spec":{"tasks":`+tc.tasks+`,"matrix":`+tc.matrix+`},"table":{"variants":[{"name":"small"},{"name":"large","total_cost_usd":1}],"tasks":[{"task":"a","variant":"small","total":1},{"task":"a","variant":"large","passed":1,"total":1}]}}`)
	}
	// Two suites share a task set and differ only in an unnamed variant. The
	// key holds the name hrn gives it, codex-m1, so the record matches the file
	// that defines its variant rather than the first file with its task set.
	write("suites/routing/unnamed-claude.json", `{"tasks":[{"id":"c"}],"matrix":[{"backend":"claude"}]}`)
	write("suites/routing/unnamed-codex.json", `{"tasks":[{"id":"c"}],"matrix":[{"backend":"codex","model":"m1"}]}`)
	write("evals/unnamed.json", `{"id":"unnamed","started":"2026-10-15T00:00:00Z","spec":{"tasks":[{"id":"c"}],"matrix":[{"backend":"codex","model":"m1"}]},"table":{"variants":[{"name":"codex-m1"}],"tasks":[{"task":"c","variant":"codex-m1","total":1}]}}`)
	var unnamed evalRecord
	if err := readJSON("evals/unnamed.json", &unnamed); err != nil {
		t.Fatal(err)
	}
	if key := unnamed.labelKey(); key != (suiteKey{"c", `["codex-m1"]`}) {
		t.Errorf("labelKey = %+v, want the derived variant name", key)
	}
	if err := generate(root, "page.md", "generate"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("page.md")
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{
		"| routing/first | small / large | a | first |",
		"| routing/second | small / large | a | second |",
		"| routing/first | small / large | a | task-fallback |",
		"| a, z | small / large | a | ids-fallback |",
		"### routing/first\n", "### routing/second\n", "### a, z\n",
		"### routing/unnamed-codex\n", "| c | 0/1 (1 failed) | unnamed |",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(page, "unnamed-claude") {
		t.Error("the record took the first file with its task set: the key holds the raw variant name")
	}
}

func TestSingleCellRerunFix(t *testing.T) {
	decode := func(s string) evalRecord {
		t.Helper()
		var e evalRecord
		if err := json.Unmarshal([]byte(s), &e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// The cell's gate, mode and backend come from the spec: the task and the
	// variant leave them out. later always ran from another dir with other
	// parallel and repeats, which say where and how often, not what the cell
	// was given.
	const top = `"mode":"yolo","gate":["go test ./a"],"backend":"codex"`
	const task = `{"id":"a","prompt":"PRIVATE PROMPT","max_turns":20}`
	const variant = `{"name":"small","model":"m1","effort":"low"}`
	old := decode(`{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"dir":"/old","parallel":2,"repeats":3,` + top + `,"tasks":[` + task + `,{"id":"b","prompt":"do b"}],"matrix":[` + variant + `,{"name":"large","model":"m2"}]}}`)
	failing := taskResult{Task: "a", Variant: "small", Passed: 1, Total: 3}
	for _, tc := range []struct{ name, started, top, tasks, matrix, want string }{
		{"identical cell alone", "2026-10-02T00:00:00Z", top, `[` + task + `]`, `[` + variant + `]`, "fixed in eval rerun"},
		{"keys reordered", "2026-10-02T00:00:00Z", `"backend":"codex","gate":["go test ./a"],"mode":"yolo"`, `[{"max_turns":20,"prompt":"PRIVATE PROMPT","id":"a"}]`, `[{"effort":"low","model":"m1","name":"small"}]`, "fixed in eval rerun"},
		{"other prompt", "2026-10-02T00:00:00Z", top, `[{"id":"a","prompt":"PRIVATE OTHER","max_turns":20}]`, `[` + variant + `]`, ""},
		{"gate moved into the task", "2026-10-02T00:00:00Z", top, `[{"id":"a","prompt":"PRIVATE PROMPT","max_turns":20,"gate":["go test ./a"]}]`, `[` + variant + `]`, ""},
		{"task field dropped", "2026-10-02T00:00:00Z", top, `[{"id":"a","prompt":"PRIVATE PROMPT"}]`, `[` + variant + `]`, ""},
		{"other model under the same name", "2026-10-02T00:00:00Z", top, `[` + task + `]`, `[{"name":"small","model":"m3","effort":"low"}]`, ""},
		{"other top-level gate", "2026-10-02T00:00:00Z", `"mode":"yolo","gate":["go test ./b"],"backend":"codex"`, `[` + task + `]`, `[` + variant + `]`, ""},
		{"top-level gate dropped", "2026-10-02T00:00:00Z", `"mode":"yolo","backend":"codex"`, `[` + task + `]`, `[` + variant + `]`, ""},
		{"other top-level mode", "2026-10-02T00:00:00Z", `"mode":"auto","gate":["go test ./a"],"backend":"codex"`, `[` + task + `]`, `[` + variant + `]`, ""},
		{"other top-level backend", "2026-10-02T00:00:00Z", `"mode":"yolo","gate":["go test ./a"],"backend":"claude"`, `[` + task + `]`, `[` + variant + `]`, ""},
		{"judge added", "2026-10-02T00:00:00Z", top + `,"judge":{"rubric":"PRIVATE RUBRIC"}`, `[` + task + `]`, `[` + variant + `]`, ""},
		{"ran before the failure", "2026-09-30T00:00:00Z", top, `[` + task + `]`, `[` + variant + `]`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			later := decode(`{"id":"rerun","started":"` + tc.started + `","spec":{"dir":"/rerun","parallel":1,"repeats":1,` + tc.top + `,"tasks":` + tc.tasks + `,"matrix":` + tc.matrix + `},"table":{"tasks":[{"task":"a","variant":"small","passed":1,"total":1}]}}`)
			if got := fixedIn([]evalRecord{later}, old, failing); got != tc.want {
				t.Errorf("fixedIn = %q, want %q", got, tc.want)
			}
		})
	}
	// A re-run that still fails a repetition is not a fix, whatever its spec.
	partial := decode(`{"id":"partial","started":"2026-10-02T00:00:00Z","spec":{"repeats":3,` + top + `,"tasks":[` + task + `],"matrix":[` + variant + `]},"table":{"tasks":[{"task":"a","variant":"small","passed":2,"total":3}]}}`)
	if got := fixedIn([]evalRecord{partial}, old, failing); got != "" {
		t.Errorf("partial re-run counted as a fix: %q", got)
	}
	// A later eval whose matrix lacks the variant defines no such cell. Its
	// table has no row for it either, so sameCell is asked directly.
	missing := decode(`{"id":"missing","started":"2026-10-02T00:00:00Z","spec":{"repeats":1,` + top + `,"tasks":[` + task + `],"matrix":[{"name":"large","model":"m2"}]},"table":{"tasks":[{"task":"a","variant":"large","passed":1,"total":1}]}}`)
	if sameCell(old, missing, failing) {
		t.Error("a matrix without the variant counted as the same cell")
	}
}

func TestUnnamedVariantRerunFix(t *testing.T) {
	decode := func(s string) evalRecord {
		t.Helper()
		var e evalRecord
		if err := json.Unmarshal([]byte(s), &e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// hrn names a variant without a name backend[-model][-effort], and the
	// results table carries that name; one without a backend either is -m1.
	old := decode(`{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"tasks":[{"id":"a"},{"id":"b"}],"matrix":[{"backend":"codex","model":"m1","effort":"low"},{"backend":"claude"},{"model":"m1"}]}}`)
	for _, tc := range []struct{ name, variant, matrix, want string }{
		{"backend, model and effort", "codex-m1-low", `[{"backend":"codex","model":"m1","effort":"low"}]`, "fixed in eval rerun"},
		{"backend alone", "claude", `[{"backend":"claude"}]`, "fixed in eval rerun"},
		{"no backend", "-m1", `[{"model":"m1"}]`, "fixed in eval rerun"},
		{"other effort under the same backend and model", "codex-m1-low", `[{"backend":"codex","model":"m1","effort":"high"}]`, ""},
		// Spelling the name out defines the variant differently: entries are
		// compared, not effective cells.
		{"name spelled out", "codex-m1-low", `[{"name":"codex-m1-low","backend":"codex","model":"m1","effort":"low"}]`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			later := decode(`{"id":"rerun","started":"2026-10-02T00:00:00Z","spec":{"tasks":[{"id":"a"}],"matrix":` + tc.matrix + `},"table":{"tasks":[{"task":"a","variant":"` + tc.variant + `","passed":1,"total":1}]}}`)
			if got := fixedIn([]evalRecord{later}, old, taskResult{Task: "a", Variant: tc.variant, Total: 1}); got != tc.want {
				t.Errorf("fixedIn = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateSingleCellRerun(t *testing.T) {
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
	// The task runs under the spec's gate, and the variant has no name: hrn
	// calls it codex-m1-low.
	const top = `"mode":"yolo","gate":["go test ./..."],"max_turns":20`
	const cell = `"tasks":[{"id":"a","prompt":"PRIVATE PROMPT"}],"matrix":[{"backend":"codex","model":"m1","effort":"low"}]`
	write("evals/old.json", `{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"dir":"/old","parallel":2,"repeats":3,`+top+`,"tasks":[{"id":"a","prompt":"PRIVATE PROMPT"},{"id":"b","prompt":"PRIVATE OTHER"}],"matrix":[{"backend":"codex","model":"m1","effort":"low"},{"name":"large","backend":"codex","model":"m2"}]},"table":{"variants":[{"name":"codex-m1-low"},{"name":"large"}],"tasks":[{"task":"a","variant":"codex-m1-low","passed":1,"total":3},{"task":"a","variant":"large","passed":3,"total":3},{"task":"b","variant":"codex-m1-low","passed":3,"total":3},{"task":"b","variant":"large","passed":3,"total":3}]}}`)
	// The cell alone under another top-level gate passes, but is no fix.
	write("evals/other-gate.json", `{"id":"other-gate","started":"2026-10-02T00:00:00Z","spec":{"dir":"/other","parallel":1,"repeats":3,"mode":"yolo","gate":["true"],"max_turns":20,`+cell+`},"table":{"variants":[{"name":"codex-m1-low"}],"tasks":[{"task":"a","variant":"codex-m1-low","passed":3,"total":3}]}}`)
	// The cell alone from another directory, with other parallel and repeats, is.
	write("evals/rerun.json", `{"id":"rerun","started":"2026-10-03T00:00:00Z","spec":{"dir":"/rerun","parallel":1,"repeats":1,`+top+`,`+cell+`},"table":{"variants":[{"name":"codex-m1-low"}],"tasks":[{"task":"a","variant":"codex-m1-low","passed":1,"total":1}]}}`)
	out := filepath.Join(home, "page.md")
	if err := generate(home, out, "generate"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if want := "| a | 1/3 (2 failed) | old | (gate output not kept) | fixed in eval rerun |"; !strings.Contains(page, want) {
		t.Errorf("missing %q in page:\n%s", want, page)
	}
	for _, unwanted := range []string{"fixed in eval other-gate", "PRIVATE"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("leaked %q", unwanted)
		}
	}
}

func TestOneTaskSuiteRerunFix(t *testing.T) {
	decode := func(s string) evalRecord {
		t.Helper()
		var e evalRecord
		if err := json.Unmarshal([]byte(s), &e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	// A one-task suite's cell, re-run alone, has the suite's task ID set; the
	// matrix names tell that re-run from a run of the suite, so that it is
	// judged as a cell, top-level fields included.
	const task = `{"id":"x","prompt":"PRIVATE PROMPT"}`
	old := decode(`{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"repeats":3,"gate":["go test ./..."],"tasks":[` + task + `],"matrix":[{"name":"claude","backend":"claude"},{"name":"codex","backend":"codex"}]}}`)
	failing := taskResult{Task: "x", Variant: "codex", Total: 3}
	for _, tc := range []struct{ name, spec, want string }{
		{"cell alone", `{"gate":["go test ./..."],"tasks":[` + task + `],"matrix":[{"name":"codex","backend":"codex"}]}`, "fixed in eval later"},
		{"cell alone under another gate", `{"gate":["true"],"tasks":[` + task + `],"matrix":[{"name":"codex","backend":"codex"}]}`, ""},
		{"cell alone without the gate", `{"tasks":[` + task + `],"matrix":[{"name":"codex","backend":"codex"}]}`, ""},
		// A later run of the whole suite is trusted as such, as it always was.
		{"whole suite under another gate", `{"gate":["true"],"tasks":[` + task + `],"matrix":[{"name":"codex","backend":"codex"},{"name":"claude","backend":"claude"}]}`, "fixed in eval later"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			later := decode(`{"id":"later","started":"2026-10-02T00:00:00Z","spec":` + tc.spec + `,"table":{"tasks":[{"task":"x","variant":"codex","passed":1,"total":1}]}}`)
			if got := fixedIn([]evalRecord{later}, old, failing); got != tc.want {
				t.Errorf("fixedIn = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSpecNullOrNotObject(t *testing.T) {
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
	// hrn writes the spec as an object. A null spec decodes to an empty one
	// that defines no cell and names no suite, so the record is read and never
	// credited; a string or an array does not decode, and the record is skipped.
	write("evals/old.json", `{"id":"old","started":"2026-10-01T00:00:00Z","spec":{"repeats":1,"tasks":[{"id":"a","prompt":"PRIVATE PROMPT"}],"matrix":[{"name":"small","backend":"codex"}]},"table":{"variants":[{"name":"small"}],"tasks":[{"task":"a","variant":"small","total":1}]}}`)
	write("evals/null-spec.json", `{"id":"null-spec","started":"2026-10-02T00:00:00Z","spec":null,"table":{"variants":[{"name":"small"}],"tasks":[{"task":"a","variant":"small","passed":1,"total":1}]}}`)
	write("evals/string-spec.json", `{"id":"string-spec","started":"2026-10-02T00:00:00Z","spec":"PRIVATE SPEC","table":{"variants":[{"name":"small"}],"tasks":[{"task":"a","variant":"small","passed":1,"total":1}]}}`)
	write("evals/array-spec.json", `{"id":"array-spec","started":"2026-10-02T00:00:00Z","spec":[],"table":{"variants":[{"name":"small"}],"tasks":[{"task":"a","variant":"small","passed":1,"total":1}]}}`)
	var null evalRecord
	if err := readJSON(filepath.Join(home, "evals", "null-spec.json"), &null); err != nil {
		t.Fatal(err)
	}
	if len(null.Spec.Tasks) != 0 || len(null.Spec.Matrix) != 0 || null.Spec.Rest != "{}" {
		t.Errorf("null spec decoded as %+v, want an empty spec", null.Spec)
	}
	out := filepath.Join(home, "page.md")
	if err := generate(home, out, "generate"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{
		"Evals read: 2. Unreadable or invalid evals skipped: 2.",
		"| a | 0/1 (1 failed) | old | (gate output not kept) |  |\n",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in page:\n%s", want, page)
		}
	}
	for _, unwanted := range []string{"fixed in eval", "PRIVATE"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("leaked %q", unwanted)
		}
	}
}
