# 2026-11-04: a continued session against a fresh hand-over, implement, review, fix

harness #117 asks for a controlled comparison of a long continued chain against a fresh
hand-over: correctness, the context actually carried, and net usage. This is a small one on
the owner's Claude plan, two repetitions. Every number on this page is **measured** from the
run records unless marked otherwise.

## What was run

The task is `implement-event-line-limit` from `suites/routing/real-work.json`, split into three
phases, each a separate `hrn run` on `claude` with model `sonnet` (`claude-sonnet-5-5`), lean,
`max_turns` 25, timeout 15m, `no_switch`:

1. **implement** (`auto`): "Make ReadEvents in fixtures/real/events/client.go report an event
   line over its 1 MiB limit as an error that says "event line exceeds 1 MiB limit" and wraps
   bufio.ErrTooLong. Keep every other behaviour of the package. Do not change
   fixtures/real/events/ISSUE.md or the checks under fixtures/real/checks. Do not run git."
   This leaves out the task's second half (what `ISSUE.md` reports about `Watch`), so the review
   has something to find.
2. **review** (`readonly`), continuing phase 1: "Review the change to fixtures/real/events against
   fixtures/real/events/ISSUE.md and the package's own tests. List each defect or unresolved point
   with file:line and why. Do not edit anything."
3. **fix** (`auto`), continuing phase 2: "Fix every point the review listed, including what
   fixtures/real/events/ISSUE.md reports about Watch. Keep every other behaviour of the package.
   Do not change ISSUE.md or the checks under fixtures/real/checks. Do not run git."

Chain **continued** passes `fresh: false` on phases 2 and 3 (the saved session is forked);
chain **fresh** passes `fresh: true` (a new session handed the earlier run's record). The two
chains ran side by side, each in its own worktree of this repository at `8a0a69c`. After phase 3
the task's own gate ran unchanged: the protected files untouched, both hidden tests installed,
`go test -count=1 -timeout 120s ./fixtures/real/events/`.

hrn `30b19bd`, Claude Code 2.1.284, Windows 11.

## Results

Both chains passed the gate in both repetitions (4 of 4), and every review found the real defect
(`Watch` still retrying an oversized line), except the fresh chain's second review, whose first
phase had already fixed `Watch` unasked (it read `ISSUE.md`).

| repeat | chain | context at the fix phase's last turn | input over the chain (incl. cache) | output |
|---|---|---:|---:|---:|
| 1 | continued | 50,107 | 530,904 | 4,531 |
| 1 | fresh | 40,859 | 405,599 | 4,575 |
| 2 | continued | 50,051 | 628,776 | 5,940 |
| 2 | fresh | 43,030 | 670,084 | 5,519 |

Per phase:

| repeat | chain | phase | run | turns | tool calls | context at last turn | input (incl. cache) | cache write | output |
|---|---|---|---|---:|---:|---:|---:|---:|---:|
| 1 | continued | implement | `20261104-205437-a79bf8` | 4 | 3 | 40,346 | 156,999 | 40,344 | 1,064 |
| 1 | continued | review | `20261104-205533-da38fd` | 7 | 6 | 48,099 | 224,547 | 7,753 | 2,454 |
| 1 | continued | fix | `20261104-205716-bcabcd` | 3 | 2 | 50,107 | 149,358 | 2,008 | 1,013 |
| 1 | fresh | implement | `20261104-205459-b62b3a` | 6 | 5 | 40,558 | 157,426 | 19,222 | 1,210 |
| 1 | fresh | review | `20261104-205605-ead419` | 4 | 3 | 45,958 | 128,625 | 22,889 | 2,155 |
| 1 | fresh | fix | `20261104-205736-3cdd76` | 5 | 4 | 40,859 | 119,548 | 17,790 | 1,210 |
| 2 | continued | implement | `20261104-205844-222c4c` | 4 | 3 | 40,268 | 156,746 | 17,199 | 1,012 |
| 2 | continued | review | `20261104-205956-7d6125` | 5 | 4 | 46,280 | 176,871 | 6,012 | 2,566 |
| 2 | continued | fix | `20261104-210223-89f69f` | 7 | 6 | 50,051 | 295,159 | 3,771 | 2,362 |
| 2 | fresh | implement | `20261104-205900-cd00f1` | 6 | 5 | 40,941 | 158,412 | 4,037 | 1,514 |
| 2 | fresh | review | `20261104-210059-ccb6d5` | 6 | 5 | 46,667 | 218,672 | 23,598 | 2,093 |
| 2 | fresh | fix | `20261104-210308-4e3577` | 10 | 9 | 43,030 | 293,000 | 19,961 | 1,912 |

## What it shows

- **Correctness:** equal. Four of four passed; neither path lost a finding.
- **Carried context:** the fresh hand-over carried 14-18% less into the fix phase (40.9k and 43.0k
  against 50.1k twice). The continued chain grows by every earlier phase's turns; the fresh one
  starts each phase from the record.
- **Net usage:** mixed at this length. Input over the chain was 24% lower for fresh in repeat 1
  and 7% higher in repeat 2, where the fresh fix phase took 10 turns to re-orient against 7.
  A fresh phase writes more cache (it starts a new prefix each time) and reads less.
- So for a short chain from a ~40k base the choice is about even. The saving grows with each phase
  carried, so a fresh hand-over is the better default once a chain is long or its context large,
  which is what hrn's 150,000-token warning already suggests.

Not measured: chains longer than three phases, other backends (codex already defaults to a fresh
hand-over when continuing codex), and more than two repetitions.
