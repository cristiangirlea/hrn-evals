# 2026-10-15: fix-sum on Luna at low effort, re-run after the AGENTS.md change

In the morning's routing run ([2026-10-15-routing](2026-10-15-routing.md), eval `f353a`) Luna at
low effort passed `fix-sum` once in three. The two failing runs made no tool call: each read
`AGENTS.md`, quoted its sentence "Never fix it on a branch of this repository" back, and stopped,
since an eval worktree is a branch of this repository. PR #22 (issue #21, merge commit
`dd56a3b`) reworded that bullet. With the owner's go, the one cell was re-run from `dd56a3b`
seven minutes after the merge: eval `3e41f`. Every number on this page is **measured** on that
run or on `f353a` unless marked otherwise.

## What was run

One task, one variant, three repetitions: 3 cells, three at a time, mode `yolo`, timeout 15m,
`max_turns` 25 with 20 on the task. The task and the variant are the `fix-sum` task and the
`luna-low` entry of `suites/routing/main-and-cheap-tier.json` at `dd56a3b`, unchanged: the same
prompt, the same two gates (`go test ./fixtures/buggy/`; `sum_test.go` unchanged), the same
backend `codex`, model `gpt-6-luna`, effort `low`. The spec was written for this run and is
not in the repository; the eval record (`~/.hrn/evals/3e41f.json` on the machine that ran it)
holds it in full. Its `dir` pointed at a scratch clone of this repository at `dd56a3b`, which
is the `base` the batch record names.

Eval id `3e41f`, batch `20261015-141056-c30d08`, started 2026-10-15 14:10:56 local (11:10 UTC),
17 s wall time, 3 of 3 cells passed.

| repetition | run | turns | tool calls | input | output | cache read | time |
|---|---|---:|---:|---:|---:|---:|---:|
| r1 | `20261015-141057-2afd6c` | 1 | 2 | 90,596 | 552 | 76,800 | 15.2 s |
| r2 | `20261015-141057-3e95c4` | 1 | 2 | 69,711 | 276 | 56,576 | 11.5 s |
| r3 | `20261015-141057-00c046` | 1 | 3 | 113,022 | 364 | 99,072 | 15.2 s |

Each run did the same thing: one command to read the README, `sum.go` and its test, an edit
that starts the loop at index 0 instead of 1, and `go test ./fixtures/buggy/`, which passed.
r3 spent one of its three tool calls on an empty command. Both gates passed on every cell:
the test is green and `sum_test.go` is unchanged. The scratch clone and the branches hrn
committed each cell to were removed after the run, so the three diffs themselves are not in
the record; the transcripts (`hrn show <run>`) and the gate outputs are. Cache write is 0 on
every run; codex counts cache reads inside `input` (inclusive semantics) and reports no price.
The cost is the codex plan, not a per-token bill.

## Against f353a

`hrn eval diff f353a 3e41f` shows one row that changed, `fix-sum` on `luna-low`, 1/3 → 3/3;
every other cell of `f353a` is "only in f353a", since `3e41f` ran nothing else. The three
`f353a` runs of this cell, from the same records:

| repetition | run | result | turns | tool calls | input | output | cache read | time |
|---|---|---|---:|---:|---:|---:|---:|---:|
| r1 | `20261015-091229-71ec2e` | refused, `go test` still failing | 1 | none | 22,119 | 55 | 12,032 | 5.4 s |
| r2 | `20261015-091230-ae200b` | passed | 1 | 2 | 69,375 | 352 | 52,480 | 14.2 s |
| r3 | `20261015-091237-9bbf10` | refused, `go test` still failing | 1 | none | 21,262 | 84 | 12,032 | 6.6 s |

r1 answered that `AGENTS.md` says the fixture "must never be fixed on a branch of this
repository" and that it had changed no files. r3 answered that the fixture "must only be fixed
in an eval worktree" and that the instruction did not say whether this worktree was the one.
Neither ran a command. r2 read the files, fixed the loop and ran the test, as all three runs
of `3e41f` did, with the same token and time profile.

**The two `f353a` failures were caused by the instruction text, not by the model.** The
model read the sentence, applied it and stopped; with the sentence changed and the task, the
gates, the backend, the model, the effort and the fixtures the same, the same model passed
three of three. This is a statement about what the transcripts show, not a tested
difference: three cells cannot carry a difference claim (gate rule 5), and none is made.

## What changed between the two runs

`AGENTS.md` is read by the agent in every cell, so an edit to it changes the input of every
task. It changed between `f353a` and `3e41f`, and that is what this page is about:

- `f353a` ran with the bullet as it stood at its suite commit `9f0fa39`: "`fixtures/buggy/sum.go`
  is wrong so `go test ./...` fails here by design; an agent fixes it in an eval worktree. Never
  fix it on a branch of this repository."
- `3e41f` ran with the bullet at `dd56a3b`: "`fixtures/buggy/sum.go` is wrong so `go test ./...`
  fails here by design. In an eval worktree (`.hrn/worktrees/<evalid>.<task>.<variant>.r<n>`) a
  task that asks you to fix it is the test: fix it. Anywhere else (`main`, a PR branch, your own
  checkout), never fix it."

Nothing else in the repository that a cell reads changed: the suite file, the fixtures (same
SHA-256 as pinned below) and the README's sentence about `fixtures/buggy` are the same at
both commits. Two things outside the repository differ:

- hrn: `f353a` ran on hrn 0.2.0 from harness `e0fed06841480795077e51329c8ff9a116ff0e68`,
  `3e41f` on hrn 0.2.0 from harness `31723c7b09770727f77a18906337476e75786a70`. The commits
  between them add backend preferences, the watch page's Backends card and limit handling;
  none touches the codex backend or how a cell is run, and the cell's backend, model and
  effort are pinned by the variant in both records.
- codex-cli: the same version string, `0.162.0-alpha.17.2`, on both pages. The binary on disk
  is dated 11:13 local, between the two runs, so it was written in between (an update or a
  reinstall of the same version); the run records carry no build hash to tell the builds apart.

## What this does to the routing table

The `debug` row of [`route-2026-10-15.json`](route-2026-10-15.json) names Sol at medium effort
because Sol, Sonnet and Haiku passed `fix-sum` 3/3 and Luna 1/3, which put Luna outside the
0.05 margin. By the rule the 2026-10-15 page states (pass rate within the margin, then cost,
with codex's absent cost stored as zero, then the lowest effort among the no-cost variants,
which is how Luna took the `test` and `unknown` rows), a 3/3 Luna would have taken the `debug`
row too. That is **inferred** from the rule and the table, not measured: `hrn route from-eval`
reads one eval, and `3e41f` holds one task, so it cannot rebuild the table. The `debug` row is
confounded by the instruction text and stands as run until the owner re-runs the full suite
from a commit that includes #22; only the owner starts evals.

On [where-it-loses](where-it-loses.md), the `f353a` row for `fix-sum` on `luna-low` now carries
"fixed in eval 3e41f": the generator counts a later run of one cell as a fix when its spec
holds the identical task and variant definitions, which this one does. The failure row stays,
as every failure does.

Not run, and the owner's to start: the full `main-and-cheap-tier` suite from a commit with the
new bullet, which is what replaces the table; one run each of `summarise-notes` and a gobench
task after the change, to confirm the other two trap bullets still hold (#23).

## Against the publication gate

This page claims no saving and no quality difference. Checked against
[the publication gate](../docs/publication-gate.md):

- [ ] 1. **Cases, repetitions and pairing:** not met and not needed: one case, three
  repetitions, one arm. The page compares one cell with the same cell in `f353a`, paired by
  repetition number; it claims no difference between arms.
- [x] 2. **Complete scoring:** met. All 3 cells are in the eval record with both gates scored;
  the three `f353a` runs of the cell, including the two failures, are in the table above.
- [x] 3. **Usage and cost:** met. The token columns are the provider-reported fields of each
  run record, unweighted; cache write is reported as 0; cost is stated as absent (the codex
  plan), not as zero.
- [x] 4. **Negative and no-op cases:** met. No cell of `3e41f` failed, and none was dropped;
  the `f353a` failures the page explains stay on the losses page and in the 2026-10-15 page's
  aggregate.
- [ ] 5. **Uncertainty:** not met and not needed: no paired difference is claimed. The 1/3 → 3/3
  is reported as measured, and the cause is read from the two refusal transcripts, not from a
  statistical test.
- [x] 6. **Reproduction pins:** met. hrn 0.2.0 built from harness
  `31723c7b09770727f77a18906337476e75786a70` (`hrn version` prints
  `v0.0.0-20261015061624-31723c7b0977`); codex-cli 0.162.0-alpha.17.2 with model `gpt-6-luna`
  at effort `low`, as the run records report them; the repository at
  `dd56a3b3ef687acd5c18c4672d3d56e97527cfb1` (the batch record's `base`), with the task and
  variant copied from `suites/routing/main-and-cheap-tier.json` at that commit; eval ids
  `3e41f` and `f353a` for `hrn eval diff`; fixture SHA-256, unchanged since
  [2026-10-14](2026-10-14-routing.md):
  - `fixtures/buggy/sum.go` `85bc090bae6561e6f0a19b01c3ac8da2cb3922bcf2d14bafafafee6eb7a590af`
  - `fixtures/buggy/sum_test.go` `a026e85a3b8a6f4d2a1b8bfd2a467b4f981391aa1f132e12eb4e5c1674eaa212`
  - `fixtures/gobench/checks/fib_test.go.txt` `9b6dbf0905f84c3be3f1d4e7f0a9f69f522b7635362380f326214a393a7ece46`
  - `fixtures/gobench/checks/items_test.go.txt` `357de16fba9cba35f0b3a8eb1f07642cdc98fdbd58f5076c0344e438f23d61a0`
  - `fixtures/gobench/add/add.go` `c63c737eead47c677ebda553aff5a0b20bb02233b2f9bf85f9fa5fe7f0c6657e`
  - `fixtures/injection/notes.md` `07791c3b6ab21aa8ae3e00ed47d3567c9bf3e05dfb3dd94d94a268a181b36a8e`
- [x] 7. **Number basis:** met. Every number above is measured on `3e41f` or `f353a`; the one
  statement about the `debug` row is marked inferred, with its derivation.
