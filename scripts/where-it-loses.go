package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type evalRecord struct {
	ID         string    `json:"id"`
	BatchID    string    `json:"batch_id"`
	Started    time.Time `json:"started"`
	Spec       evalSpec  `json:"spec"`
	SuiteLabel string    `json:"-"`
	Table      struct {
		Variants []variant    `json:"variants"`
		Tasks    []taskResult `json:"tasks"`
	} `json:"table"`
}

// evalSpec is an eval spec as a record or a suite file holds it: its tasks and
// its matrix as entries, and the rest of its top-level fields as JSON with
// sorted keys. dir, parallel and repeats are left out of the rest: they say
// where and how often a cell ran, not what it was given.
type evalSpec struct {
	Matrix []specEntry
	Tasks  []specEntry
	Rest   string
}

func (s *evalSpec) UnmarshalJSON(b []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	var spec evalSpec
	rest := make(map[string]any, len(fields))
	for key, raw := range fields {
		var err error
		switch key {
		case "tasks":
			err = json.Unmarshal(raw, &spec.Tasks)
		case "matrix":
			err = json.Unmarshal(raw, &spec.Matrix)
		case "dir", "parallel", "repeats":
		default:
			var value any
			err = json.Unmarshal(raw, &value)
			rest[key] = value
		}
		if err != nil {
			return err
		}
	}
	canonical, err := json.Marshal(rest)
	if err != nil {
		return err
	}
	spec.Rest = string(canonical)
	*s = spec
	return nil
}

// specEntry is one task or one variant of an eval spec: the task's id, the
// variant's name and the fields hrn names a nameless variant by, and the whole
// entry as JSON with sorted keys, so two evals can be compared for the same
// task or variant definition whatever fields a spec carries. The JSON holds
// the prompt; it is compared, never written to the page.
type specEntry struct {
	ID      string
	Name    string
	Backend string
	Model   string
	Effort  string
	JSON    string
}

func (s *specEntry) UnmarshalJSON(b []byte) error {
	var names struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Backend string `json:"backend"`
		Model   string `json:"model"`
		Effort  string `json:"effort"`
	}
	if err := json.Unmarshal(b, &names); err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return err
	}
	*s = specEntry{ID: names.ID, Name: names.Name, Backend: names.Backend, Model: names.Model, Effort: names.Effort, JSON: string(canonical)}
	return nil
}

// variantName is the name hrn gives the variant and writes into the results
// table: its name, or backend[-model][-effort] when it has none.
func (e specEntry) variantName() string {
	if e.Name != "" {
		return e.Name
	}
	parts := []string{e.Backend}
	if e.Model != "" {
		parts = append(parts, e.Model)
	}
	if e.Effort != "" {
		parts = append(parts, e.Effort)
	}
	return strings.Join(parts, "-")
}

// entry returns the canonical JSON of the first entry whose key is want.
func entry(entries []specEntry, key func(specEntry) string, want string) (string, bool) {
	for _, e := range entries {
		if key(e) == want {
			return e.JSON, true
		}
	}
	return "", false
}

// sameCell reports whether later's spec defines the failing case's cell exactly
// as old's does, so that later counts as a re-run of that cell even when it ran
// that cell alone: the task entry, the variant entry and the spec's top-level
// fields apart from dir, parallel and repeats must be identical, since a cell
// inherits what its task and variant leave out (gate, mode, max_turns, timeout,
// backend, model, effort) from the spec. The rest of those fields, the judge
// among them, only scores or guards a run, and is compared all the same. A
// variant without a name is matched by the name hrn gives it,
// backend[-model][-effort]. What a cell is given outside the spec is not
// compared: the home's default backend and the CLI's default model and effort
// when the variant, the task and the spec all leave them out, the CLI's other
// settings, hrn's and the CLI's version, and the batch's base commit.
func sameCell(old, later evalRecord, task taskResult) bool {
	taskID := func(e specEntry) string { return e.ID }
	oldTask, ok1 := entry(old.Spec.Tasks, taskID, task.Task)
	laterTask, ok2 := entry(later.Spec.Tasks, taskID, task.Task)
	oldVariant, ok3 := entry(old.Spec.Matrix, specEntry.variantName, task.Variant)
	laterVariant, ok4 := entry(later.Spec.Matrix, specEntry.variantName, task.Variant)
	return ok1 && ok2 && ok3 && ok4 && oldTask == laterTask && oldVariant == laterVariant && old.Spec.Rest == later.Spec.Rest
}

type variant struct {
	Name       string  `json:"name"`
	Cost       float64 `json:"total_cost_usd"`
	MedianCost float64 `json:"median_cost_usd"`
	Tokens     float64 `json:"median_context_tokens"`
}

type taskResult struct {
	Task    string `json:"task"`
	Variant string `json:"variant"`
	Passed  int    `json:"passed"`
	Total   int    `json:"total"`
}

type batchRecord struct {
	Tasks []struct {
		ID       string `json:"id"`
		Attempts []struct {
			Result struct {
				Status string `json:"status"`
			} `json:"result"`
			Gates []struct {
				Command  string `json:"command"`
				OK       *bool  `json:"ok"`
				ExitCode *int   `json:"exit_code"`
				Output   string `json:"output"`
			} `json:"gates"`
		} `json:"attempts"`
	} `json:"tasks"`
}

const outsideStart = "<!-- outside reports: start -->"
const outsideEnd = "<!-- outside reports: end -->"

func main() {
	home := os.Getenv("HRN_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		home = filepath.Join(userHome, ".hrn")
	}
	homeFlag := flag.String("home", home, "hrn home containing evals and batches")
	out := flag.String("out", "results/where-it-loses.md", "output page")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(1)
	}
	command := "scripts/where-it-loses.sh"
	flag.Visit(func(f *flag.Flag) { command += " --" + f.Name + " " + shellQuote(f.Value.String()) })
	if err := generate(*homeFlag, *out, command); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func cell(s string) string {
	s = html.EscapeString(s)
	s = strings.NewReplacer("|", "&#124;", "\r", "", "\n", "<br>", "`", "&#96;", "*", "&#42;", "_", "&#95;", "[", "&#91;", "]", "&#93;", "\\", "&#92;").Replace(s)
	return s
}
func (e evalRecord) suite() string {
	ids := make([]string, 0, len(e.Spec.Tasks))
	seen := make(map[string]bool)
	for _, task := range e.Spec.Tasks {
		if !seen[task.ID] {
			ids = append(ids, task.ID)
			seen[task.ID] = true
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ", ")
}

func (e evalRecord) suiteLabel() string {
	if e.SuiteLabel != "" {
		return e.SuiteLabel
	}
	return e.suite()
}

type suiteKey struct {
	tasks, matrix string
}

// labelKey identifies a suite as suite files are labelled: by its task ID set
// and the names hrn gives its matrix variants, so that an unnamed variant keys
// by backend[-model][-effort], the name sameCell and the results table use.
func (e evalRecord) labelKey() suiteKey {
	names := make([]string, 0, len(e.Spec.Matrix))
	seen := make(map[string]bool)
	for _, v := range e.Spec.Matrix {
		name := v.variantName()
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	sort.Strings(names)
	matrix, _ := json.Marshal(names)
	return suiteKey{e.suite(), string(matrix)}
}

func suiteLabels() (map[suiteKey]string, error) {
	labels := make(map[suiteKey]string)
	err := filepath.WalkDir("suites", func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == "suites" {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		var spec evalSpec
		if err := readJSON(path, &spec); err != nil {
			return err
		}
		key := (evalRecord{Spec: spec}).labelKey()
		label, err := filepath.Rel("suites", path)
		if err != nil {
			return err
		}
		label = strings.TrimSuffix(filepath.ToSlash(label), ".json")
		// Two files with one task set and matrix would label each other's
		// records; only the task-set fallback may be shared, and the first
		// file in walk order keeps it.
		if other, exists := labels[key]; exists {
			return fmt.Errorf("suites/%s.json and suites/%s.json have the same task IDs and matrix variant names", other, label)
		}
		for _, k := range []suiteKey{key, {tasks: key.tasks}} {
			if _, exists := labels[k]; !exists {
				labels[k] = label
			}
		}
		return nil
	})
	return labels, err
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func gateMessage(home string, e evalRecord, t taskResult) string {
	// Batch IDs are identifiers, never paths outside the batches directory.
	if e.BatchID == "" {
		return "(gate output not kept)"
	}
	if strings.ContainsAny(e.BatchID, "/\\") {
		return "(invalid batch id)"
	}
	var b batchRecord
	if err := readJSON(filepath.Join(home, "batches", e.BatchID+".json"), &b); err != nil {
		if os.IsNotExist(err) {
			return "(gate output not kept)"
		}
		return "(batch unreadable)"
	}
	noGate := ""
	prefix := e.ID + "." + t.Task + "." + t.Variant + ".r"
	for _, bt := range b.Tasks {
		if !strings.HasPrefix(bt.ID, prefix) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(bt.ID, prefix)); err != nil || n < 1 {
			continue
		}
		for _, attempt := range bt.Attempts {
			if noGate == "" && len(attempt.Gates) == 0 {
				status := "unknown"
				if words := strings.Fields(attempt.Result.Status); len(words) > 0 {
					status = words[0]
				}
				noGate = "(no gate ran: " + cell(status) + ")"
			}
			for _, g := range attempt.Gates {
				// Older saved batches used exit_code instead of ok. Unknown
				// gate status is not evidence of a failing gate.
				failed := g.OK != nil && !*g.OK
				if g.OK == nil && g.ExitCode != nil {
					failed = *g.ExitCode != 0
				}
				if !failed {
					continue
				}
				lines := strings.Split(strings.TrimRight(strings.ReplaceAll(g.Output, "\r\n", "\n"), "\n"), "\n")
				if len(lines) > 3 {
					lines = lines[len(lines)-3:]
				}
				return "<code>" + cell(g.Command) + "</code><br>" + cell(strings.Join(lines, "\n"))
			}
		}
	}
	if noGate != "" {
		return noGate
	}
	return "(attempt not found)"
}

// fixedIn names the earliest later eval in which the failing case's task and
// variant passed every repetition: a later run of the same suite (the same task
// ID set and matrix variant names, as suites are labelled), or a re-run whose
// spec defines the same cell (sameCell). The matrix names keep a one-task
// suite's cell, re-run alone, out of the suite rule.
func fixedIn(evals []evalRecord, old evalRecord, task taskResult) string {
	var fixed *evalRecord
	for i := range evals {
		e := &evals[i]
		if !e.Started.After(old.Started) {
			continue
		}
		if e.labelKey() != old.labelKey() && !sameCell(old, *e, task) {
			continue
		}
		for _, t := range e.Table.Tasks {
			if t.Task == task.Task && t.Variant == task.Variant && t.Total > 0 && t.Passed == t.Total {
				if fixed == nil || e.Started.Before(fixed.Started) || (e.Started.Equal(fixed.Started) && e.ID < fixed.ID) {
					fixed = e
				}
			}
		}
	}
	if fixed != nil {
		return "fixed in eval " + cell(fixed.ID)
	}
	return ""
}

func generate(home, out, command string) error {
	labels, err := suiteLabels()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(home, "evals"))
	if err != nil {
		return err
	}
	var evals []evalRecord
	skipped := 0
	var latest time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var e evalRecord
		if readJSON(filepath.Join(home, "evals", entry.Name()), &e) != nil || e.ID == "" || e.Started.IsZero() {
			skipped++
			continue
		}
		e.SuiteLabel = labels[e.labelKey()]
		if e.SuiteLabel == "" {
			e.SuiteLabel = labels[suiteKey{tasks: e.suite()}]
		}
		evals = append(evals, e)
		if e.Started.After(latest) {
			latest = e.Started
		}
	}
	outside := "\nA run on real work, with date and setup, goes here.\n"
	if existing, err := os.ReadFile(out); err == nil {
		s := string(existing)
		start := strings.Index(s, outsideStart)
		end := strings.Index(s, outsideEnd)
		if start >= 0 && end >= start+len(outsideStart) {
			outside = s[start+len(outsideStart) : end]
		} else {
			return fmt.Errorf("existing page has missing or reversed outside reports markers")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	var page strings.Builder
	date := "no saved evals"
	if !latest.IsZero() {
		date = latest.UTC().Format("2006-01-02")
	}
	fmt.Fprintf(&page, "# Where it loses\n\nDate: %s (latest saved eval, UTC).\n\nProduced by: <code>%s</code>\n\nEvals read: %d. Unreadable or invalid evals skipped: %d.\n\n", date, cell(command), len(evals), skipped)
	page.WriteString("Failures remain when later runs pass. Fix notes use the earliest later fully passing case of the same task and variant, ordered by started: a later run of the same suite (the same task ID set and matrix variant names), or a re-run whose spec holds the identical task and variant definitions and the same top-level fields apart from dir, parallel and repeats, since a cell inherits what its task and variant leave out (gate, mode, max_turns, timeout, backend, model, effort) from the spec; a variant without a name is matched by the name hrn gives it, backend[-model][-effort]; what a cell is given outside the spec is not compared: the home's default backend and the CLI's default model and effort when the variant, the task and the spec all leave them out, the CLI's other settings, hrn's and the CLI's version, and the batch's base commit. Suites are labelled by the file under suites/ matching their task ID set and matrix variant names, falling back to the task ID set alone, then joined task IDs. Only the first failing gate's command and last three output lines are retained here.\n\n## Failing cases\n")
	type failure struct {
		e evalRecord
		t taskResult
	}
	var failures []failure
	for _, e := range evals {
		for _, t := range e.Table.Tasks {
			if t.Passed < t.Total {
				failures = append(failures, failure{e, t})
			}
		}
	}
	sort.Slice(failures, func(i, j int) bool {
		a, b := failures[i], failures[j]
		if a.e.suite() != b.e.suite() {
			return a.e.suite() < b.e.suite()
		}
		if a.e.suiteLabel() != b.e.suiteLabel() {
			return a.e.suiteLabel() < b.e.suiteLabel()
		}
		if a.t.Variant != b.t.Variant {
			return a.t.Variant < b.t.Variant
		}
		if a.t.Task != b.t.Task {
			return a.t.Task < b.t.Task
		}
		return a.e.ID < b.e.ID
	})
	suite, variantName := "", ""
	for _, f := range failures {
		if f.e.suiteLabel() != suite {
			suite = f.e.suiteLabel()
			variantName = ""
			fmt.Fprintf(&page, "\n### %s\n", cell(f.e.suiteLabel()))
		}
		if f.t.Variant != variantName {
			variantName = f.t.Variant
			fmt.Fprintf(&page, "\n#### %s\n\n| Task | Passed/total (failed) | Eval | Gate message | History |\n|---|---|---|---|---|\n", cell(variantName))
		}
		fmt.Fprintf(&page, "| %s | %d/%d (%d failed) | %s | %s | %s |\n", cell(f.t.Task), f.t.Passed, f.t.Total, f.t.Total-f.t.Passed, cell(f.e.ID), gateMessage(home, f.e, f.t), fixedIn(evals, f.e, f.t))
	}
	if len(failures) == 0 {
		page.WriteString("\nNo failing cases in the saved evals.\n")
	}
	writeComparisons(&page, evals)
	fmt.Fprintf(&page, "\n## Outside reports\n\n%s%s%s\n", outsideStart, outside, outsideEnd)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(page.String()), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "where-it-loses: read %d evals; skipped %d unreadable or invalid evals\n", len(evals), skipped)
	return nil
}

func cheaper(a, b variant) bool {
	if a.Cost != b.Cost {
		return a.Cost < b.Cost
	}
	return a.Tokens < b.Tokens
}

func writeComparisons(page *strings.Builder, evals []evalRecord) {
	page.WriteString("\n## Comparisons\n\nCheaper means lower total eval cost, with lower median context tokens breaking equal costs (including zero). Costs and tokens below are variant-wide, not per-task measurements; zero reported cost is not proof of free execution. Every cost shown is a list-price estimate, hrn's or the CLI's, never a bill. Cost overruns mean higher median cost per attempt despite a lower total. Only tasks present for both variants are compared.\n")
	type comparison struct {
		suite, task, id string
		low, high       variant
		a, b            taskResult
		reason          string
	}
	var rows []comparison
	for _, e := range evals {
		for _, a := range e.Table.Variants {
			for _, b := range e.Table.Variants {
				if !cheaper(a, b) {
					continue
				}
				for _, ta := range e.Table.Tasks {
					if ta.Variant != a.Name {
						continue
					}
					for _, tb := range e.Table.Tasks {
						if tb.Variant != b.Name || tb.Task != ta.Task {
							continue
						}
						var reasons []string
						if ta.Passed < tb.Passed {
							reasons = append(reasons, "fewer passes")
						}
						if a.MedianCost > b.MedianCost {
							reasons = append(reasons, "higher median cost")
						}
						if len(reasons) > 0 {
							rows = append(rows, comparison{e.suiteLabel(), ta.Task, e.ID, a, b, ta, tb, strings.Join(reasons, "; ")})
						}
					}
				}
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.suite != b.suite {
			return a.suite < b.suite
		}
		if a.low.Name != b.low.Name {
			return a.low.Name < b.low.Name
		}
		if a.task != b.task {
			return a.task < b.task
		}
		if a.id != b.id {
			return a.id < b.id
		}
		return a.high.Name < b.high.Name
	})
	if len(rows) == 0 {
		page.WriteString("\nNo cheaper-variant losses or median cost overruns in saved comparisons.\n")
		return
	}
	page.WriteString("\n| Suite | Cheaper variant / other | Task | Eval | Passed/total (cheaper / other) | Total USD (cheaper / other) | Median USD (cheaper / other) | Median context tokens (cheaper / other) | Loss |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(page, "| %s | %s / %s | %s | %s | %d/%d / %d/%d | %s / %s | %s / %s | %g / %g | %s |\n", cell(r.suite), cell(r.low.Name), cell(r.high.Name), cell(r.task), cell(r.id), r.a.Passed, r.a.Total, r.b.Passed, r.b.Total, cost(r.low.Cost), cost(r.high.Cost), cost(r.low.MedianCost), cost(r.high.MedianCost), r.low.Tokens, r.high.Tokens, r.reason)
	}
}
func cost(n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if n != 0 {
		s += " (estimate)"
	}
	return s
}
