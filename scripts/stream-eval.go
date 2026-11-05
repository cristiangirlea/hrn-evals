package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var streamLabels = []string{"bug", "billing", "feature", "question"}
var streamName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

type streamTicket struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Label string `json:"label"`
}

type streamResult struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	Output       *string `json:"output"`
	Duration     *int64  `json:"duration_ms"`
	InputTokens  *int64  `json:"input_tokens"`
	OutputTokens *int64  `json:"output_tokens"`
}

type streamLabelScore struct {
	Items    int     `json:"items"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy"`
}

type streamScore struct {
	Name           string                      `json:"name"`
	Items          int                         `json:"items"`
	OK             int                         `json:"ok"`
	Correct        int                         `json:"correct"`
	Accuracy       float64                     `json:"accuracy"`
	InvalidAnswers int                         `json:"invalid_answers"`
	MedianMS       float64                     `json:"median_duration_ms"`
	P90MS          int64                       `json:"p90_duration_ms"`
	ItemsPerSecond *float64                    `json:"items_per_second"`
	InputTokens    int64                       `json:"input_tokens"`
	OutputTokens   int64                       `json:"output_tokens"`
	PerLabel       map[string]streamLabelScore `json:"per_label"`
}

type streamMeta struct {
	Name        string  `json:"name"`
	WallSeconds float64 `json:"wall_seconds"`
}

func streamJSONL(path string, visit func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 4*1024*1024)
	line := 0
	for s.Scan() {
		line++
		if err := visit(s.Bytes()); err != nil {
			return fmt.Errorf("%s:%d: %w", path, line, err)
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if line == 0 {
		return fmt.Errorf("%s: empty file", path)
	}
	return nil
}

func streamLabel(s string) bool {
	for _, label := range streamLabels {
		if s == label {
			return true
		}
	}
	return false
}

func readStreamTickets(path string) ([]streamTicket, error) {
	var tickets []streamTicket
	seen := map[string]bool{}
	err := streamJSONL(path, func(b []byte) error {
		var t streamTicket
		if err := json.Unmarshal(b, &t); err != nil {
			return err
		}
		if t.ID == "" || strings.TrimSpace(t.Text) == "" || !streamLabel(t.Label) || seen[t.ID] {
			return fmt.Errorf("invalid ticket or duplicate id %q", t.ID)
		}
		seen[t.ID] = true
		tickets = append(tickets, t)
		return nil
	})
	return tickets, err
}

func writeStreamInput(w io.Writer, tickets []streamTicket) error {
	enc := json.NewEncoder(w)
	for _, t := range tickets {
		// A separate type ensures the answer key cannot reach the model.
		if err := enc.Encode(struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}{t.ID, t.Text}); err != nil {
			return err
		}
	}
	return nil
}

func streamAnswer(output string) string {
	words := strings.Fields(output)
	if len(words) == 0 {
		return ""
	}
	return strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return r
	}, words[0]))
}

func scoreStream(path, name string, tickets []streamTicket, wallSeconds float64) (streamScore, error) {
	score := streamScore{Name: name, Items: len(tickets), PerLabel: map[string]streamLabelScore{}}
	labels := map[string]string{}
	for _, label := range streamLabels {
		score.PerLabel[label] = streamLabelScore{}
	}
	for _, t := range tickets {
		labels[t.ID] = t.Label
		p := score.PerLabel[t.Label]
		p.Items++
		score.PerLabel[t.Label] = p
	}
	seen := map[string]bool{}
	var durations []int64
	err := streamJSONL(path, func(b []byte) error {
		var r streamResult
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		label, exists := labels[r.ID]
		if !exists || seen[r.ID] {
			return fmt.Errorf("unknown or duplicate result id %q", r.ID)
		}
		if r.Output == nil || r.Duration == nil || r.InputTokens == nil || r.OutputTokens == nil {
			return fmt.Errorf("%s: missing output or counters", r.ID)
		}
		if *r.Duration < 0 || *r.InputTokens < 0 || *r.OutputTokens < 0 {
			return fmt.Errorf("%s: negative counter", r.ID)
		}
		switch r.Status {
		case "ok":
			score.OK++
			answer := streamAnswer(*r.Output)
			if !streamLabel(answer) {
				score.InvalidAnswers++
			}
			if answer == label {
				score.Correct++
				p := score.PerLabel[label]
				p.Correct++
				score.PerLabel[label] = p
			}
		case "error", "timeout", "invalid":
		default:
			return fmt.Errorf("%s: unknown status %q", r.ID, r.Status)
		}
		seen[r.ID] = true
		durations = append(durations, *r.Duration)
		score.InputTokens += *r.InputTokens
		score.OutputTokens += *r.OutputTokens
		return nil
	})
	if err != nil {
		return score, err
	}
	if len(seen) != len(tickets) {
		return score, fmt.Errorf("%s: got %d results, want %d", path, len(seen), len(tickets))
	}
	score.Accuracy = float64(score.Correct) / float64(score.Items)
	for label, p := range score.PerLabel {
		if p.Items > 0 {
			p.Accuracy = float64(p.Correct) / float64(p.Items)
		}
		score.PerLabel[label] = p
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	n := len(durations)
	score.MedianMS = float64(durations[n/2])
	if n%2 == 0 {
		score.MedianMS = (float64(durations[n/2-1]) + float64(durations[n/2])) / 2
	}
	score.P90MS = durations[int(math.Ceil(0.9*float64(n)))-1]
	if wallSeconds > 0 {
		rate := float64(score.Items) / wallSeconds
		score.ItemsPerSecond = &rate
	}
	return score, nil
}

func streamEval(args []string, out, stderr io.Writer) (retErr error) {
	if len(args) == 0 {
		return fmt.Errorf("usage: stream-eval.sh run <name> -- <hrn feed args...> | score <results.jsonl>")
	}
	tickets, err := readStreamTickets("fixtures/stream/tickets.jsonl")
	if err != nil {
		return err
	}
	var path string
	var meta streamMeta
	switch args[0] {
	case "score":
		if len(args) != 2 {
			return fmt.Errorf("score requires one results.jsonl path")
		}
		path = args[1]
		meta.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		b, err := os.ReadFile(path + ".meta.json")
		if err == nil {
			if err := json.Unmarshal(b, &meta); err != nil {
				return fmt.Errorf("%s.meta.json: %w", path, err)
			}
			if meta.Name == "" || meta.WallSeconds <= 0 {
				return fmt.Errorf("%s.meta.json: invalid name or wall time", path)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	case "run":
		if len(args) < 4 || args[2] != "--" || !streamName.MatchString(args[1]) {
			return fmt.Errorf("run requires a name (letters, digits, underscores or hyphens) and -- <hrn feed args...>")
		}
		for _, arg := range args[3:] {
			// Go's flag parser accepts both -out and --out, including = forms.
			flag := strings.TrimLeft(strings.SplitN(arg, "=", 2)[0], "-")
			switch flag {
			case "in", "out", "task", "task-file", "":
				return fmt.Errorf("%s is managed by stream-eval", flag)
			}
		}
		meta.Name = args[1]
		path = filepath.Join("results", "stream", time.Now().Format("2006-01-02")+"-"+meta.Name+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		// Never let a second run silently replace an earlier measurement.
		reserved, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		// Handle interruption through the command context so deferred cleanup runs.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		defer func() {
			if retErr != nil {
				if info, err := os.Stat(path); err == nil {
					if info.Size() == 0 {
						if err := os.Remove(path); err != nil {
							retErr = fmt.Errorf("%w; remove empty results: %v", retErr, err)
						}
					} else {
						retErr = fmt.Errorf("%w (results at %s)", retErr, path)
					}
				}
			}
		}()
		if err := reserved.Close(); err != nil {
			return err
		}
		input, err := os.CreateTemp("", "stream-eval-*.jsonl")
		if err != nil {
			return err
		}
		defer os.Remove(input.Name())
		err = writeStreamInput(input, tickets)
		closeErr := input.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		feedArgs := append([]string{"feed", "--task-file", "fixtures/stream/task.md", "--in", input.Name(), "--out", path}, args[3:]...)
		cmd := exec.CommandContext(ctx, "hrn", feedArgs...)
		cmd.Stdout, cmd.Stderr = stderr, stderr
		start := time.Now()
		err = cmd.Run()
		meta.WallSeconds = time.Since(start).Seconds()
		if err != nil {
			return fmt.Errorf("hrn feed failed: %w", err)
		}
		b, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path+".meta.json", append(b, '\n'), 0644); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown command %q; use run or score", args[0])
	}
	score, err := scoreStream(path, meta.Name, tickets, meta.WallSeconds)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(score)
}
