package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The gate eval compares arms by task, so its specs must stay paired with smoke.json.
func TestShapingGateSuitesArePaired(t *testing.T) {
	load := func(name string) map[string]json.RawMessage {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "suites", "shaping", name))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return m
	}
	same := func(a, b json.RawMessage) bool {
		var x, y bytes.Buffer
		if json.Compact(&x, a) != nil || json.Compact(&y, b) != nil {
			return false
		}
		return bytes.Equal(x.Bytes(), y.Bytes())
	}
	smoke, base, gate := load("smoke.json"), load("gate-baseline.json"), load("gate.json")

	for _, s := range []struct {
		name string
		spec map[string]json.RawMessage
	}{{"gate-baseline.json", base}, {"gate.json", gate}} {
		if !same(s.spec["tasks"], smoke["tasks"]) {
			t.Errorf("%s: tasks differ from smoke.json", s.name)
		}
	}
	for _, k := range []string{"dir", "mode", "parallel", "max_turns", "timeout", "repeats"} {
		if !same(base[k], gate[k]) {
			t.Errorf("%s differs between gate-baseline.json and gate.json", k)
		}
	}
	if string(gate["command_gate"]) != "true" {
		t.Errorf("gate.json: command_gate = %s, want true", gate["command_gate"])
	}
	if _, ok := base["command_gate"]; ok {
		t.Error("gate-baseline.json sets command_gate; the native arm must run without the gate")
	}

	type variant map[string]any
	var matrix []variant
	if err := json.Unmarshal(gate["matrix"], &matrix); err != nil {
		t.Fatal(err)
	}
	byName := map[string]variant{}
	for _, v := range matrix {
		byName[v["name"].(string)] = v
	}
	shaped := 0
	for name, v := range byName {
		if !strings.HasSuffix(name, "-shaped") {
			continue
		}
		shaped++
		raw, ok := byName[strings.TrimSuffix(name, "-shaped")+"-raw"]
		if !ok {
			t.Errorf("%s has no -raw twin", name)
			continue
		}
		if v["shape_tools"] != true || raw["shape_tools"] != nil {
			t.Errorf("%s: shape_tools must be on for the shaped variant and absent on its raw twin", name)
		}
		for k, x := range v {
			if k == "name" || k == "shape_tools" {
				continue
			}
			if raw[k] != x {
				t.Errorf("%s: %s differs from its raw twin", name, k)
			}
		}
		for k := range raw {
			if _, ok := v[k]; !ok && k != "name" {
				t.Errorf("%s: its raw twin sets %s", name, k)
			}
		}
	}
	if shaped == 0 {
		t.Error("gate.json has no -shaped variant")
	}
}
