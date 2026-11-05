package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStreamFixtureAndInput(t *testing.T) {
	tickets, err := readStreamTickets("../fixtures/stream/tickets.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 40 {
		t.Fatalf("got %d tickets", len(tickets))
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, ticket := range tickets {
		if seen[ticket.ID] {
			t.Fatalf("duplicate %s", ticket.ID)
		}
		seen[ticket.ID] = true
		counts[ticket.Label]++
		// Even another category's label could be an artificial hint.
		for _, label := range streamLabels {
			if strings.Contains(strings.ToLower(ticket.Text), label) {
				t.Fatalf("%s gives away %q", ticket.ID, label)
			}
		}
	}
	for _, label := range streamLabels {
		if counts[label] != 10 {
			t.Fatalf("%s: %d items", label, counts[label])
		}
	}
	var input bytes.Buffer
	if err := writeStreamInput(&input, tickets); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(input.String()), "\n")
	if len(lines) != 40 {
		t.Fatalf("got %d input lines", len(lines))
	}
	for i, line := range lines {
		var fields map[string]string
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatal(err)
		}
		if len(fields) != 2 || fields["id"] != tickets[i].ID || fields["text"] != tickets[i].Text {
			t.Fatalf("input changed or leaked a label: %s", line)
		}
	}
}

func streamTestFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "results.jsonl")
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStreamScoring(t *testing.T) {
	// Completion order differs from fixture order. Errors carry plausible labels
	// to prove only successful results can contribute to correct answers.
	path := filepath.Join("testdata", "stream-results.jsonl")
	tickets := []streamTicket{
		{ID: "a", Label: "bug"}, {ID: "b", Label: "billing"},
		{ID: "c", Label: "feature"}, {ID: "d", Label: "question"},
		{ID: "e", Label: "bug"}, {ID: "f", Label: "billing"}, {ID: "g", Label: "question"},
	}
	score, err := scoreStream(path, "fixture", tickets, 2)
	if err != nil {
		t.Fatal(err)
	}
	if score.Name != "fixture" || score.Items != 7 || score.OK != 5 || score.Correct != 3 || score.InvalidAnswers != 1 || score.Accuracy != 3.0/7 {
		t.Fatalf("bad counts: %+v", score)
	}
	if score.MedianMS != 40 || score.P90MS != 70 || score.InputTokens != 28 || score.OutputTokens != 7 || score.ItemsPerSecond == nil || *score.ItemsPerSecond != 3.5 {
		t.Fatalf("bad measurements: %+v", score)
	}
	if score.PerLabel["bug"].Accuracy != 0.5 || score.PerLabel["billing"].Accuracy != 0 || score.PerLabel["feature"].Accuracy != 1 || score.PerLabel["question"].Accuracy != 0.5 {
		t.Fatalf("bad per-label scores: %+v", score.PerLabel)
	}
	score, err = scoreStream(path, "fixture", tickets, 0)
	if err != nil || score.ItemsPerSecond != nil {
		t.Fatalf("unknown wall time: %+v %v", score, err)
	}
}

func TestStreamMedianAndInvalidStatus(t *testing.T) {
	path := streamTestFile(t, `{"id":"a","status":"ok","output":"**BUG** extra words","duration_ms":10,"input_tokens":1,"output_tokens":1}
{"id":"b","status":"invalid","output":"bug","duration_ms":30,"input_tokens":0,"output_tokens":0}
`)
	score, err := scoreStream(path, "even", []streamTicket{{ID: "a", Label: "bug"}, {ID: "b", Label: "bug"}}, 0)
	if err != nil || score.MedianMS != 20 || score.Correct != 1 || score.InvalidAnswers != 0 {
		t.Fatalf("%+v %v", score, err)
	}
}

func TestStreamMalformedResults(t *testing.T) {
	valid := `{"id":"a","status":"ok","output":"bug","duration_ms":1,"input_tokens":2,"output_tokens":3}`
	for name, content := range map[string]string{
		"empty": "", "blank": "\n", "syntax": "{", "null": "null",
		"missing counters": `{"id":"a","status":"ok","output":"bug"}`,
		"null output":      strings.Replace(valid, `"bug"`, `null`, 1),
		"negative":         strings.Replace(valid, `"duration_ms":1`, `"duration_ms":-1`, 1),
		"unknown id":       strings.Replace(valid, `"a"`, `"z"`, 1),
		"unknown status":   strings.Replace(valid, `"ok"`, `"success"`, 1),
		"duplicate":        valid + "\n" + valid,
		"missing result":   valid,
		"wrong type":       strings.Replace(valid, `"duration_ms":1`, `"duration_ms":"1"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := scoreStream(streamTestFile(t, content), "bad", []streamTicket{{ID: "a", Label: "bug"}, {ID: "b", Label: "bug"}}, 0)
			if err == nil {
				t.Fatal("accepted malformed results")
			}
		})
	}
	if _, err := scoreStream(filepath.Join(t.TempDir(), "missing"), "bad", []streamTicket{{ID: "a", Label: "bug"}}, 0); err == nil {
		t.Fatal("accepted missing file")
	}
}

func TestStreamMalformedTickets(t *testing.T) {
	for _, content := range []string{"", "null", "{", `{"id":"a","text":"hello","label":"other"}`, `{"id":"a","text":"","label":"bug"}`, `{"text":"hello","label":"bug"}`, "{\"id\":\"a\",\"text\":\"hello\",\"label\":\"bug\"}\n{\"id\":\"a\",\"text\":\"hello\",\"label\":\"bug\"}"} {
		if _, err := readStreamTickets(streamTestFile(t, content)); err == nil {
			t.Fatalf("accepted %q", content)
		}
	}
}

func TestStreamRunAndRescore(t *testing.T) {
	// A compiled stub works with exec.Command on Windows as well as Unix.
	// No harness command or model is invoked by this test.
	dir := t.TempDir()
	source := `package main
import ("encoding/json"; "os"; "fmt")
func main() {
 if os.Getenv("STREAM_STUB_FAIL") == "1" { os.Exit(7) }
 args := os.Args[1:]
 if len(args)!=13 || args[0]!="feed" || args[1]!="--task-file" || args[2]!="fixtures/stream/task.md" || args[3]!="--in" || args[5]!="--out" || args[7]!="-b" || args[8]!="claude" || args[9]!="--model" || args[10]!="haiku" || args[11]!="--parallel" || args[12]!="4" { panic(fmt.Sprint(args)) }
 in, err := os.Open(args[4]); if err!=nil { panic(err) }; defer in.Close()
 out, err := os.Create(args[6]); if err!=nil { panic(err) }; defer out.Close()
 if os.Getenv("STREAM_STUB_FAIL") == "partial" { out.WriteString("partial result\n"); out.Close(); os.Exit(7) }
 dec, enc := json.NewDecoder(in), json.NewEncoder(out)
 for dec.More() {
  var item map[string]string; if err:=dec.Decode(&item); err!=nil { panic(err) }
  if len(item)!=2 || item["id"]=="" || item["text"]=="" { panic("answer key leaked") }
  enc.Encode(map[string]any{"id":item["id"],"status":"ok","output":"question","duration_ms":10,"input_tokens":2,"output_tokens":1})
 }
}
`
	src := filepath.Join(dir, "stub.go")
	if err := os.WriteFile(src, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "hrn.exe"), src)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build stub: %v\n%s", err, b)
	}
	// exec.LookPath on Unix needs the unsuffixed name.
	b, err := os.ReadFile(filepath.Join(dir, "hrn.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hrn"), b, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(root, "fixtures", "stream"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"tickets.jsonl", "task.md"} {
		b, err := os.ReadFile(filepath.Join("..", "fixtures", "stream", file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "fixtures", "stream", file), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, stderr bytes.Buffer
	args := []string{"run", "stub", "--", "-b", "claude", "--model", "haiku", "--parallel", "4"}
	if err := streamEval(args, &out, &stderr); err != nil {
		t.Fatalf("run: %v\n%s", err, &stderr)
	}
	var score streamScore
	if err := json.Unmarshal(out.Bytes(), &score); err != nil {
		t.Fatal(err)
	}
	if score.Items != 40 || score.Correct != 10 || score.ItemsPerSecond == nil || *score.ItemsPerSecond <= 0 {
		t.Fatalf("%+v", score)
	}
	path := filepath.Join("results", "stream", time.Now().Format("2006-01-02")+"-stub.jsonl")
	var rescored bytes.Buffer
	if err := streamEval([]string{"score", path}, &rescored, &stderr); err != nil {
		t.Fatal(err)
	}
	if out.String() != rescored.String() {
		t.Fatalf("rescore differs: %s / %s", &out, &rescored)
	}
	if err := streamEval(args, &rescored, &stderr); err == nil {
		t.Fatal("overwrote results")
	}
	if err := os.Remove(path + ".meta.json"); err != nil {
		t.Fatal(err)
	}
	rescored.Reset()
	if err := streamEval([]string{"score", path}, &rescored, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rescored.String(), `"items_per_second":null`) {
		t.Fatal("invented throughput")
	}
	if err := os.WriteFile(path+".meta.json", []byte(`{"name":"stub","wall_seconds":0}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := streamEval([]string{"score", path}, &rescored, &stderr); err == nil {
		t.Fatal("accepted invalid metadata")
	}
	for _, bad := range [][]string{nil, {"score"}, {"other"}, {"run", "../bad", "--", "-b", "claude"}, {"run", "x", "--", "--out=x"}, {"run", "x", "--", "--in", "x"}, {"run", "x", "--", "--task=x"}, {"run", "x", "--", "--task-file=x"}, {"run", "x", "--", "-out=x"}, {"run", "x", "--", "-in", "x"}, {"run", "x", "--", "-task=x"}, {"run", "x", "--", "-task-file=x"}, {"run", "x", "--", "--"}} {
		if err := streamEval(bad, &rescored, &stderr); err == nil {
			t.Fatalf("accepted %v", bad)
		}
	}
	t.Setenv("STREAM_STUB_FAIL", "1")
	args[1] = "failure"
	if err := streamEval(args, &rescored, &stderr); err == nil || !strings.Contains(err.Error(), "hrn feed failed") {
		t.Fatalf("lost feed failure: %v", err)
	}
	failurePath := filepath.Join("results", "stream", time.Now().Format("2006-01-02")+"-failure.jsonl")
	if _, err := os.Stat(failurePath); !os.IsNotExist(err) {
		t.Fatalf("empty failed results remain: %v", err)
	}
	t.Setenv("STREAM_STUB_FAIL", "")
	if err := streamEval(args, &rescored, &stderr); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	t.Setenv("STREAM_STUB_FAIL", "partial")
	args[1] = "partial"
	partialPath := filepath.Join("results", "stream", time.Now().Format("2006-01-02")+"-partial.jsonl")
	if err := streamEval(args, &rescored, &stderr); err == nil || !strings.Contains(err.Error(), "results at "+partialPath) {
		t.Fatalf("lost partial results location: %v", err)
	}
	if b, err := os.ReadFile(partialPath); err != nil || string(b) != "partial result\n" {
		t.Fatalf("lost partial results: %q, %v", b, err)
	}
	t.Setenv("STREAM_STUB_FAIL", "")
	t.Run("temporary input failure", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		t.Setenv("TMP", missing)
		t.Setenv("TEMP", missing)
		t.Setenv("TMPDIR", missing)
		args[1] = "temp-failure"
		if err := streamEval(args, &rescored, &stderr); err == nil {
			t.Fatal("accepted missing temporary directory")
		}
		path := filepath.Join("results", "stream", time.Now().Format("2006-01-02")+"-temp-failure.jsonl")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("empty results remain after temporary input failure: %v", err)
		}
	})
}

func TestStreamAnswer(t *testing.T) {
	for _, input := range []string{"`bug`", "<|~$=+BUG+=$~|>"} {
		if got := streamAnswer(input); got != "bug" {
			t.Errorf("%q: got %q, want bug", input, got)
		}
	}
	for input, want := range map[string]string{"": "", "  \n": "", "Bug.": "bug", "‘BILLING,’": "billing", "feature please": "feature", "question": "question", "unknown": "unknown"} {
		if got := streamAnswer(input); got != want {
			t.Error(fmt.Sprintf("%q: got %q, want %q", input, got, want))
		}
	}
}
