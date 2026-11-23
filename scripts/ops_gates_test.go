package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// opsCase is one file an agent might leave for an ops task. A case the gates
// must fail names words the failing gate's output has to carry.
type opsCase struct {
	name string
	file string
	pass bool
	why  string
}

// checkOpsBothWays writes each file where the task expects it and runs the
// task's gates on it as hrn does.
func checkOpsBothWays(t *testing.T, id, target string, cases []opsCase) {
	bash, found := gateBash()
	if !found {
		t.Skip("no bash that hrn would use")
	}
	gates := taskGates(t, "routing/real-work.json", id)
	path := filepath.Join("..", filepath.FromSlash(target))
	mustBeAbsent(t, path)
	// Only folders this test makes are removed again.
	var made []string
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(dir); err == nil {
			break
		}
		made = append(made, dir)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Remove(path)
		for _, dir := range made {
			os.Remove(dir)
		}
	})
	for _, c := range cases {
		if err := os.WriteFile(path, []byte(c.file), 0o644); err != nil {
			t.Fatal(err)
		}
		why, ok := runGates(t, bash, gates)
		switch {
		case c.pass && !ok:
			t.Errorf("%s: %s: the gates fail a file they should pass: %s", id, c.name, why)
		case !c.pass && ok:
			t.Errorf("%s: %s: the gates pass a file they should fail", id, c.name)
		case !c.pass && !strings.Contains(why, c.why):
			t.Errorf("%s: %s: the failing gate does not say %q: %s", id, c.name, c.why, why)
		}
	}
}

const workflowHead = `name: CI

on:
  push:
  pull_request:

jobs:
`

// workflowJob is one job as RUNNERS.md asks for it; rest is everything after
// runs-on.
func workflowJob(name, variable, rest string) string {
	cond := "vars." + variable + " != ''"
	if variable == "CI_RUNNER_WINDOWS" {
		cond += " && github.event_name != 'pull_request'"
	}
	return "  " + name + ":\n    if: ${{ " + cond + " }}\n    runs-on: ${{ vars." + variable + " }}\n" + rest
}

const (
	setup = `    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
`
	vetAndTest = `      - run: go vet ./...
      - run: go test -count=1 ./...
`
	namedOnly = `      - name: go vet
        run: go build ./...
      - name: go test
        run: go build ./...
`
	// What the gates print for a job that lacks a command, up to the command.
	linuxLacks   = "no step of the Linux job that runs only when CI_RUNNER is set runs go "
	windowsLacks = "no step of the Windows job that is kept off pull requests runs go "
)

func TestOpsRunnerWorkflowGates(t *testing.T) {
	linux := workflowJob("linux", "CI_RUNNER", setup+vetAndTest)
	windows := workflowJob("windows", "CI_RUNNER_WINDOWS", setup+vetAndTest)
	checkOpsBothWays(t, "ops-runner-workflow", "fixtures/real/ops-runner/.github/workflows/ci.yml", []opsCase{
		{name: "a run step for each command in both jobs", pass: true, file: workflowHead + linux + "\n" + windows},
		{name: "the same with CRLF line ends", pass: true, file: strings.ReplaceAll(workflowHead+linux+windows, "\n", "\r\n")},
		{name: "named steps, a block and a folded run, list items level with steps", pass: true, file: `name: ci
on: [push, pull_request]
jobs:
  test-linux:
    name: Linux
    runs-on: ${{ vars.CI_RUNNER }}
    if: ${{ vars.CI_RUNNER != '' }}
    steps:
    - name: Check out
      uses: actions/checkout@v4
    - name: Vet and test
      shell: bash
      run: |
        set -euo pipefail
        go vet ./...
        CGO_ENABLED=0 go test ./...
  test-windows:
    name: Windows
    runs-on: ${{ vars.CI_RUNNER_WINDOWS }}
    if: ${{ vars.CI_RUNNER_WINDOWS != '' && github.event_name != 'pull_request' }}
    steps:
    - name: Check out
      uses: actions/checkout@v4
    - name: Vet
      run: go vet ./...
    - name: Test
      run: >-
        go test
        ./...
`},
		{name: "both commands in one run value, under a name that also has them", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", setup+`      - name: go vet and go test
        run: go vet ./... && go test ./...
`) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - name: go vet and go test
        run: go vet ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; go test ./...
`)},
		{name: "a run value that starts or goes on below its key, and a quoted one", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", setup+`      - run:
          go vet ./...
      - run: go build ./...
          && go test ./...
`) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - run: "go vet ./..."
      - run: if ! go test ./...; then exit 1; fi
`)},
		{name: "other job keys between runs-on and steps", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", `    timeout-minutes: 10
    strategy:
      fail-fast: false
    env:
      GOFLAGS: -mod=readonly
    defaults:
      run:
        shell: bash
`+setup+vetAndTest) + windows},
		{name: "one job per command and runner", pass: true, file: workflowHead +
			workflowJob("vet-linux", "CI_RUNNER", setup+"      - run: go vet ./...\n") +
			workflowJob("test-linux", "CI_RUNNER", setup+"      - run: go test ./...\n") +
			workflowJob("vet-windows", "CI_RUNNER_WINDOWS", setup+"      - run: go vet ./...\n") +
			workflowJob("test-windows", "CI_RUNNER_WINDOWS", setup+"      - run: go test ./...\n")},
		{name: "the list of steps shared through an anchor", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", strings.Replace(setup, "steps:", "steps: &steps", 1)+vetAndTest) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", "    steps: *steps\n")},
		{name: "single steps shared through anchors", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", setup+`      - &vet
        run: go vet ./...
      - &test
        name: Test
        run: go test ./...
`) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - *vet
      - *test
`)},
		{name: "steps in flow style", pass: true, file: workflowHead +
			workflowJob("linux", "CI_RUNNER", "    steps: [{uses: actions/checkout@v4}, {run: go vet ./...}, {run: go test ./...}]\n") +
			workflowJob("windows", "CI_RUNNER_WINDOWS", `    steps:
      - {uses: actions/checkout@v4}
      - {name: Vet, run: go vet ./...}
      - {run: "go test ./...", name: Test}
`)},

		{name: "step names carry the commands, no step runs them", why: linuxLacks + "vet", file: workflowHead +
			workflowJob("linux", "CI_RUNNER", setup+namedOnly) + workflowJob("windows", "CI_RUNNER_WINDOWS", setup+namedOnly)},
		{name: "both commands in the Linux job alone", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+"      - run: go build ./...\n")},
		{name: "the Windows job only names the commands", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - name: go vet and go test
        run: echo todo
`)},
		{name: "the Windows job only prints the commands", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - run: echo go vet and go test are not set up yet
      - run: |
          echo "go test is skipped on Windows"
          Write-Host 'go vet later'
`)},
		{name: "the Windows job tests and never vets", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+"      - run: go test ./...\n")},
		{name: "the Linux job vets and never tests", why: linuxLacks + "test", file: workflowHead +
			workflowJob("linux", "CI_RUNNER", setup+"      - run: go vet ./...\n") + windows},
		{name: "the commands run only in a Windows job that pull requests reach", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+"      - run: go build ./...\n") +
			"  windows-extra:\n    runs-on: ${{ vars.CI_RUNNER_WINDOWS }}\n" + setup + vetAndTest},
		{name: "the commands are an input of an action, not a run step", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", `    steps:
      - uses: actions/checkout@v4
      - uses: example/go-action@v1
        with:
          run: go vet ./... && go test ./...
`)},
		{name: "the commands are in an env value", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", setup+`      - name: checks
        env:
          CHECKS: go vet ./... && go test ./...
        run: echo "$CHECKS"
`)},
		{name: "an anchor shares steps that never vet", why: linuxLacks + "vet", file: workflowHead +
			workflowJob("linux", "CI_RUNNER", strings.Replace(setup, "steps:", "steps: &steps", 1)+`      - name: go vet
        run: go build ./...
      - run: go test ./...
`) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", "    steps: *steps\n")},
		{name: "the Windows job shares only the checkout step", why: windowsLacks + "vet", file: workflowHead +
			workflowJob("linux", "CI_RUNNER", `    steps:
      - &checkout
        uses: actions/checkout@v4
`+vetAndTest) +
			workflowJob("windows", "CI_RUNNER_WINDOWS", `    steps:
      - *checkout
`)},
		{name: "flow style steps that only name the commands", why: windowsLacks + "vet", file: workflowHead + linux +
			workflowJob("windows", "CI_RUNNER_WINDOWS", `    steps:
      - {uses: actions/checkout@v4}
      - {run: go build ./..., name: go vet}
      - {name: go test, run: echo go test}
`)},
	})
}

func TestOpsDockerignoreGates(t *testing.T) {
	const (
		moduleFiles = "*\n!go.mod\n!go.sum\n"
		allowed     = moduleFiles + "!cmd\n!internal\n"
	)
	checkOpsBothWays(t, "ops-dockerignore", "fixtures/real/ops-docker/.dockerignore", []opsCase{
		{name: "the two folders by name", pass: true, file: "# Only what the Dockerfile copies.\n" + allowed},
		{name: "the same with CRLF line ends", pass: true, file: strings.ReplaceAll(allowed, "\n", "\r\n")},
		{name: "a byte-order mark before the star", pass: true, file: bom + allowed},
		{name: "a byte-order mark before a comment", pass: true, file: bom + "# Only what the Dockerfile copies.\n" + allowed},
		{name: "double stars", pass: true, file: "**\n\n!go.mod\n!go.sum\n# sources\n!cmd/**\n!internal/**\n"},
		{name: "rooted, with trailing slashes", pass: true, file: "*\n!/go.mod\n!/go.sum\n!/cmd/\n!/internal/\n"},
		{name: "the source files one by one", pass: true, file: moduleFiles + "!cmd/server/main.go\n!internal/greet/greet.go\n"},
		{name: "both forms and the Docker files", pass: true, file: "**\n!Dockerfile\n!.dockerignore\n!go.mod\n!go.sum\n!cmd/\n!cmd/**\n!internal/\n!internal/**"},
		{name: "later rules that take nothing the build needs", pass: true, file: moduleFiles + "!cmd/\n!internal/\n\n**/*_test.go\n**/.env*\n**/*.md\n"},
		{name: "later rules written with ./ and doubled slashes", pass: true, file: allowed + "./docs\nscripts//\n/./.env.example\ncmd/./missing\n"},

		{name: "exceptions for paths that do not exist", why: "cmd/server/main.go is left out of the build context", file: moduleFiles + "!cmd/missing\n!internal/missing\n"},
		{name: "one folder let in, the other by a missing path", why: "internal/greet/greet.go is left out of the build context", file: moduleFiles + "!cmd\n!internal/missing\n"},
		{name: "exceptions one level too shallow", why: "cmd/server/main.go is left out of the build context", file: moduleFiles + "!cmd/*.go\n!internal/*.go\n"},
		{name: "a later rule takes the sources out again", why: "last matched by the rule **/*.go", file: allowed + "**/*.go\n"},
		{name: "a later rule written with ./ takes a source folder out", why: "last matched by the rule cmd/server", file: allowed + "./cmd/./server/\n"},
		{name: "a comment after an exception, which Docker reads as part of the path", why: "cmd/server/main.go is left out of the build context", file: moduleFiles + "!cmd # the server\n!internal # its packages\n"},
		{name: "a comment after the star, so no rule leaves anything out", why: "docs/NOTES.md is still in the build context", file: "* # everything\n!go.mod\n!go.sum\n!cmd\n!internal\n"},
		{name: "the module files left out", why: "go.mod is left out of the build context", file: "*\n!cmd\n!internal\n"},
		{name: "a rule with a backslash", why: `the rule docs\ has a backslash`, file: allowed + "docs\\\n"},
	})
}
