package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
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
	Name           string  `json:"name"`
	Cost           float64 `json:"total_cost_usd"`
	MedianCost     float64 `json:"median_cost_usd"`
	Tokens         float64 `json:"median_context_tokens"`
	CostIsEstimate bool    `json:"cost_is_estimate"`
}

// unpriced reports a variant hrn had no price for: a zero cost that hrn did
// not estimate, as codex's runs read when hrn lacks its model's price. A zero
// that hrn estimated, a local model's, is a price.
func (v variant) unpriced() bool { return v.Cost == 0 && !v.CostIsEstimate }

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
	flag.Visit(func(f *flag.Flag) { command += " --" + f.Name + " " + shellQuote(shownPath(f.Value.String())) })
	if err := generate(*homeFlag, *out, command); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// shownPath keeps a local path off the page: an absolute or home-relative
// path (Unix or Windows, drive-relative too, whatever the OS the page is made
// on) is shown as .../ and, when it is a dot directory or a page, its last
// element. A relative path says nothing about the machine, unless it climbs
// out of the repository with a .. element (../../Users/someone/.hrn names a
// user): that one is shown the same way.
func shownPath(p string) string {
	local := strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) || strings.HasPrefix(p, "~") ||
		len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
	// A drive-relative Windows path (C:Users\someone) is on the machine too.
	local = local || len(p) >= 2 && p[1] == ':' && ('a' <= p[0]|0x20 && p[0]|0x20 <= 'z')
	for _, element := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		local = local || element == ".."
	}
	if !local {
		return p
	}
	// The last element is shown only when it is a dot directory or a page
	// (.hrn, page.md): any other (/home/someone, ~user, C:\Users\john.doe)
	// may be a user name.
	trimmed := strings.TrimRight(p, `/\`)
	last := trimmed[strings.LastIndexAny(trimmed, `/\`)+1:]
	if len(p) >= 2 && p[1] == ':' && strings.LastIndexAny(trimmed, `/\`) < 0 {
		last = last[2:]
	}
	if !strings.HasPrefix(last, ".") && !strings.HasSuffix(last, ".md") {
		last = ""
	}
	return ".../" + last
}

// controls matches terminal escape sequences a test runner may print: CSI
// (colours, cursor moves), OSC (titles, links), the other ESC sequences
// (intermediate bytes, then a final byte: ESC ( B from tput sgr0, ESC 7 and
// ESC 8, ESC =), the 8-bit CSI with its parameters, and the control strings
// with their text (DCS, SOS, PM and APC after ESC P, X, ^ or _, and the 8-bit
// DCS, SOS, OSC, PM and APC), ended by BEL, ESC \ or the 8-bit ST. A string
// that is never ended loses only its opening sequence.
var controls = regexp.MustCompile(`\x1b(\[[0-?]*[ -/]*[@-~]|[\]PX^_][^\x07\x1b\x{9c}]*(\x07|\x1b\\|\x{9c})|[ -/]*[0-~])|\x{9b}[0-?]*[ -/]*[@-~]|[\x{90}\x{98}\x{9d}-\x{9f}][^\x07\x1b\x{9c}]*(\x07|\x1b\\|\x{9c})`)

// bidi holds the Unicode Bidi_Control characters, which reorder how the text
// around them is displayed (marks, embeddings, overrides, isolates); gate
// output keeps none. They are written as escapes so the source shows them.
func bidi(r rune) bool {
	return r == '\u061c' || r == '\u200e' || r == '\u200f' || r >= '\u202a' && r <= '\u202e' || r >= '\u2066' && r <= '\u2069'
}

func cell(s string) string {
	s = controls.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r == '\n' || r == '\r':
			return r
		case r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f || bidi(r):
			return -1
		}
		return r
	}, s)
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

// suiteIndex is what suites/ holds: each suite's label by its key and by its
// task-ID set, and every task ID a suite defines.
type suiteIndex struct {
	labels map[suiteKey]string
	tasks  map[string]bool
}

// ours reports whether an eval ran this repository's work: whether any of its
// task IDs is defined by a suite here. The hrn home is shared, so other
// projects' evals are in it too; with no suites to tell by, every eval is ours.
// An older eval of this repository whose task IDs have all been renamed since
// counts as another project's; one that still has a task ID in a suite stays.
func (s suiteIndex) ours(e evalRecord) bool {
	if len(s.tasks) == 0 {
		return true
	}
	for _, task := range e.Spec.Tasks {
		if s.tasks[task.ID] {
			return true
		}
	}
	return false
}

// suiteLabels reads the files under suites/ and indexes them by task ID set
// and matrix variant names, and by task ID set alone (the first file keeps
// that one), refusing two files with the same task ID set and matrix variant
// names. With it, generate and suiteLabel label evals by the file under
// suites/ matching their task ID set and matrix variant names, falling back to
// the task ID set alone, then joined task IDs.
func suiteLabels() (suiteIndex, error) {
	labels := make(map[suiteKey]string)
	tasks := make(map[string]bool)
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
		for _, task := range spec.Tasks {
			tasks[task.ID] = true
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
	return suiteIndex{labels, tasks}, err
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
				// hrn writes exit_code for every gate, -1 when it was killed. A
				// gate without one is not evidence of a failure.
				if g.ExitCode == nil || *g.ExitCode == 0 {
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

// taskGates is what decides a pass for one task, as JSON: its gates, gate
// timeout, protected files and protect mode, each its own or else the spec's
// top-level one, as hrn picks them, and its expect pattern. An own empty gate
// list is kept: hrn runs that task with no gates.
func (e evalRecord) taskGates(taskID string) (string, bool) {
	raw, ok := entry(e.Spec.Tasks, func(s specEntry) string { return s.ID }, taskID)
	if !ok {
		return "", false
	}
	var task, top map[string]any
	if json.Unmarshal([]byte(raw), &task) != nil || json.Unmarshal([]byte(e.Spec.Rest), &top) != nil {
		return "", false
	}
	rules := map[string]any{"expect": task["expect"]}
	for _, key := range []string{"gate", "gate_timeout", "protect", "protect_mode"} {
		value := task[key]
		if value == nil || value == "" {
			value = top[key]
		}
		rules[key] = value
	}
	gates, err := json.Marshal(rules)
	return string(gates), err == nil
}

// variantDef is the named variant's definition for one task as JSON, without
// its name and with backend, model and effort resolved as hrn picks them: the
// variant's, else the task's, else the spec's, an empty one counting as unset.
// It reports false for a variant the matrix does not hold.
func (e evalRecord) variantDef(name, taskID string) (string, bool) {
	raw, ok := entry(e.Spec.Matrix, specEntry.variantName, name)
	if !ok {
		return "", false
	}
	rawTask, _ := entry(e.Spec.Tasks, func(s specEntry) string { return s.ID }, taskID)
	var v, task, top map[string]any
	if json.Unmarshal([]byte(raw), &v) != nil || json.Unmarshal([]byte(e.Spec.Rest), &top) != nil {
		return raw, true
	}
	if rawTask != "" && json.Unmarshal([]byte(rawTask), &task) != nil {
		return raw, true
	}
	delete(v, "name")
	own := maps.Clone(v)
	for _, key := range []string{"backend", "model", "effort"} {
		delete(v, key)
		for _, level := range []map[string]any{own, task, top} {
			if value := level[key]; value != nil && value != "" {
				v[key] = value
				break
			}
		}
	}
	def, err := json.Marshal(v)
	if err != nil {
		return raw, true
	}
	return string(def), true
}

// fixedIn names the earliest later eval in which the failing case's task and
// variant passed every repetition: a later run of the same suite (the same task
// ID set and matrix variant names, as suites are labelled, with the failing
// variant's definition and what decides the failing task's pass unchanged: its
// gates, expect, gate timeout and protected files; see taskGates), or a re-run
// whose spec defines the same cell (sameCell). The matrix names keep a one-task
// suite's cell, re-run alone, out of the suite rule; taskGates keeps a pass
// under other gates from counting, and the variant's definition a pass by
// another model behind the same name.
func fixedIn(evals []evalRecord, old evalRecord, task taskResult) string {
	var fixed *evalRecord
	oldGates, oldOK := old.taskGates(task.Task)
	for i := range evals {
		e := &evals[i]
		if !e.Started.After(old.Started) {
			continue
		}
		laterGates, laterOK := e.taskGates(task.Task)
		oldVariant, oldVOK := old.variantDef(task.Variant, task.Task)
		laterVariant, laterVOK := e.variantDef(task.Variant, task.Task)
		sameSuite := e.labelKey() == old.labelKey() && oldOK && laterOK && oldGates == laterGates &&
			oldVOK == laterVOK && oldVariant == laterVariant
		if !sameSuite && !sameCell(old, *e, task) {
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
	suites, err := suiteLabels()
	if err != nil {
		return err
	}
	labels := suites.labels
	entries, err := os.ReadDir(filepath.Join(home, "evals"))
	if err != nil {
		return err
	}
	var evals []evalRecord
	skipped, others := 0, 0
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
		if !suites.ours(e) {
			others++
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
	fmt.Fprintf(&page, "# Where it loses\n\nDate: %s (latest saved eval, UTC).\n\nProduced by: <code>%s</code>\n\nEvals read: %d. Unreadable or invalid evals skipped: %d. Other projects' evals left out (none of their task IDs is in a suite under suites/, so an eval whose task IDs have all been renamed since is left out too): %d.\n\n", date, cell(command), len(evals), skipped, others)
	page.WriteString("Failures remain when later runs pass. Fix notes use the earliest later fully passing case of the same task and variant, ordered by started: a later run of the same suite (the same task ID set and matrix variant names, with the failing variant's definition and what decides the failing task's pass unchanged: its gates, expect, gate timeout and protected files), or a re-run whose spec holds the identical task and variant definitions and the same top-level fields apart from dir, parallel and repeats, since a cell inherits what its task and variant leave out (gate, mode, max_turns, timeout, backend, model, effort) from the spec; a variant without a name is matched by the name hrn gives it, backend[-model][-effort]; what a cell is given outside the spec is not compared: the home's default backend and the CLI's default model and effort when the variant, the task and the spec all leave them out, the CLI's other settings, hrn's and the CLI's version, and the batch's base commit. Suites are labelled by the file under suites/ matching their task ID set and matrix variant names, falling back to the task ID set alone, then joined task IDs. Only the first failing gate's command and last three output lines are retained here, without terminal escape sequences, control characters or bidirectional controls; bytes that are not valid UTF-8 show as U+FFFD.\n\n## Failing cases\n")
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
	fmt.Fprintf(os.Stderr, "where-it-loses: read %d evals; skipped %d unreadable or invalid evals; left out %d of other projects\n", len(evals), skipped, others)
	return nil
}

func cheaper(a, b variant) bool {
	if a.Cost != b.Cost && !a.unpriced() && !b.unpriced() {
		return a.Cost < b.Cost
	}
	return a.Tokens < b.Tokens
}

func writeComparisons(page *strings.Builder, evals []evalRecord) {
	page.WriteString("\n## Comparisons\n\nCheaper means lower total eval cost, with lower median context tokens breaking equal costs (including zero). A variant with a zero cost that hrn did not estimate is unpriced, since hrn had no price for it: a pair with an unpriced variant is compared on median context tokens alone, and its costs read unpriced. Costs and tokens below are variant-wide, not per-task measurements; zero reported cost is not proof of free execution. Every cost shown is a list-price estimate, hrn's or the CLI's, never a bill. Fewer passes are listed per shared task. A median cost overrun, a higher median cost per attempt despite a lower total, belongs to the variant pair, so it is listed once per eval and pair. Only tasks present for both variants are compared.\n")
	type loss struct {
		suite, task, id string
		low, high       variant
		a, b            taskResult
	}
	type overrun struct {
		suite, id string
		low, high variant
		shared    int
	}
	var losses []loss
	var overruns []overrun
	for _, e := range evals {
		for _, a := range e.Table.Variants {
			for _, b := range e.Table.Variants {
				if !cheaper(a, b) {
					continue
				}
				shared := 0
				for _, ta := range e.Table.Tasks {
					if ta.Variant != a.Name {
						continue
					}
					for _, tb := range e.Table.Tasks {
						if tb.Variant != b.Name || tb.Task != ta.Task {
							continue
						}
						shared++
						if ta.Passed < tb.Passed {
							losses = append(losses, loss{e.suiteLabel(), ta.Task, e.ID, a, b, ta, tb})
						}
					}
				}
				if shared > 0 && !a.unpriced() && !b.unpriced() && a.MedianCost > b.MedianCost {
					overruns = append(overruns, overrun{e.suiteLabel(), e.ID, a, b, shared})
				}
			}
		}
	}
	sort.Slice(losses, func(i, j int) bool {
		a, b := losses[i], losses[j]
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
	sort.Slice(overruns, func(i, j int) bool {
		a, b := overruns[i], overruns[j]
		if a.suite != b.suite {
			return a.suite < b.suite
		}
		if a.low.Name != b.low.Name {
			return a.low.Name < b.low.Name
		}
		if a.id != b.id {
			return a.id < b.id
		}
		return a.high.Name < b.high.Name
	})
	page.WriteString("\n### Fewer passes\n")
	if len(losses) == 0 {
		page.WriteString("\nNo cheaper-variant losses in saved comparisons.\n")
	} else {
		page.WriteString("\n| Suite | Cheaper variant / other | Task | Eval | Passed/total (cheaper / other) | Total USD (cheaper / other) | Median context tokens (cheaper / other) |\n|---|---|---|---|---|---|---|\n")
		for _, r := range losses {
			fmt.Fprintf(page, "| %s | %s / %s | %s | %s | %d/%d / %d/%d | %s / %s | %g / %g |\n", cell(r.suite), cell(r.low.Name), cell(r.high.Name), cell(r.task), cell(r.id), r.a.Passed, r.a.Total, r.b.Passed, r.b.Total, cost(r.low, r.low.Cost), cost(r.high, r.high.Cost), r.low.Tokens, r.high.Tokens)
		}
	}
	page.WriteString("\n### Median cost overruns\n")
	if len(overruns) == 0 {
		page.WriteString("\nNo median cost overruns in saved comparisons.\n")
		return
	}
	page.WriteString("\n| Suite | Cheaper variant / other | Eval | Shared tasks | Total USD (cheaper / other) | Median USD (cheaper / other) |\n|---|---|---|---|---|---|\n")
	for _, r := range overruns {
		fmt.Fprintf(page, "| %s | %s / %s | %s | %d | %s / %s | %s / %s |\n", cell(r.suite), cell(r.low.Name), cell(r.high.Name), cell(r.id), r.shared, cost(r.low, r.low.Cost), cost(r.high, r.high.Cost), cost(r.low, r.low.MedianCost), cost(r.high, r.high.MedianCost))
	}
}

// cost shows one of a variant's costs: "unpriced" when hrn had no price for
// the variant, otherwise the number, marked as an estimate unless it is zero.
func cost(v variant, n float64) string {
	if v.unpriced() {
		return "unpriced"
	}
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if n != 0 {
		s += " (estimate)"
	}
	return s
}
