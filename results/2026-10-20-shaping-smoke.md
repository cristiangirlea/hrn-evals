# 2026-10-20: tool-output shaping on a local model, off and on

The question from #69: does a shaped summary ever cost an agent the answer? hrn's
`shape_tools` replaces a shell result over 30,000 bytes with a summary and a handle the agent
can read in full with `read_original`; without it, hrn keeps the first 20,000 and the last
10,000 bytes and cuts the middle. Shaping runs only on hrn's api backend, so this smoke test
ran on a local model at no cost while the real comparison on Claude and Codex waits for
harness #435. Every number on this page is **measured** on this run unless marked otherwise.

**It answers only that question.** A local model with 32k of context favours shaping, since an
unshaped 30 KB result fills most of that context and two of them overflow it; the token figures
below are therefore not a forecast for Claude or Codex, whose contexts are far larger, and they
cannot decide whether shaping goes on by default.

## What was run

Suite `suites/shaping/smoke.json` at `617ee38`: six tasks, each asking for one exact fact from
one command's output (`go run ./fixtures/shaping/emit <kind>`). Five outputs are over 30,000
bytes, one per shape the summarizer handles (Go test, log, JSON, table, terminal), with the fact
placed where the plain cut drops it; the sixth is small, so shaping has nothing to do. Two
variants on hrn's api backend with the same model, `local-unshaped` and `local-shaped`
(`shape_tools` on), three repetitions, one cell at a time, `max_turns` 12. Gates check the
fact, that the answer is under 300 bytes (so pasting the whole output fails), and that the
fixture was not changed.

Eval id `19bdf`, batch `20261020-095812-61b5c1`, 36 cells, 1h0m wall time, 32 of 36 passed.

## Result

| task | what the shaped agent got | local-unshaped | local-shaped |
|---|---|---|---|
| shape-test | the plain cut (the summarizer refuses this output) | 2/3 | 1/3 |
| shape-log | a summary that keeps the error line | 3/3 | 3/3 |
| shape-json | a summary without the fact | 3/3 | 3/3 |
| shape-table | a summary without the fact | 2/3 | 3/3 |
| shape-terminal | a summary without the fact (26,968 bytes) | 3/3 | 3/3 |
| shape-noop | the output itself (676 bytes) | 3/3 | 3/3 |
| **all** | | **16/18** | **16/18** |

**Shaping never cost an agent the answer.** Every shaped cell that finished passed. Where the
summary left the fact out (JSON, table, terminal), all nine shaped cells still found it: they
re-ran the command and filtered it with `grep`, or saved it to a file and searched that. No
cell in either arm ever called `read_original`.

Faced with a summary that lacked the fact, six shaped cells also opened files the task did not
point them to, which no unshaped cell did. Five read the emitter's source,
`fixtures/shaping/emit/main.go`, which computes the facts and does not hold them; one of those
(terminal, r3) worked the digest out from the source, then confirmed it with `grep` on the
command's output before writing it. One (table, r3) read `suites/shaping/smoke.json`, whose
gates hold the expected answers, but only after its second call had already printed the row
it answered from. In all six the written answer had appeared in a command's output first, so
none of the passes rests on the suite file; the gates live beside the tasks in every suite of
this repository, and an agent that goes looking can read them.

The four failures, none of them a shaped summary losing an answer:

- `shape-test`, unshaped, r2 (`20261020-100119-40cc2f`): after 7 turns and two cut results,
  the next request was 32,918 tokens and the server refused it (`exceed_context_size_error`).
- `shape-table`, unshaped, r2 (`20261020-103840-865be5`): the same overflow after 8 turns, at
  33,919 tokens.
- `shape-test`, shaped, r2 (`20261020-101050-13f6a4`): after one command the model wrote until
  it hit its output limit (15,690 output tokens). This task's output is not summarised, so the
  shaped arm saw the same cut as the unshaped one plus a handle.
- `shape-test`, shaped, r3 (`20261020-102122-016951`): the agent saved the output to a file and
  read the 109 KB file whole with `read_file`, which hrn does not shape, and overflowed at 33,230
  tokens.

## What the summarizer did

- **Go test:** the summary is refused, so `shape_tools` falls back to the plain cut with a
  handle. The summary hrn builds opens `tests: passed=1499 failed=2` (the failed count includes
  the parent test), and its validity check refuses it: `2` never appears in the output as a
  number token, and with the count corrected some further check still refuses it. Plain
  `go test -v` output therefore never gets shaped. Filed as harness #619.
- **Log:** summarised to 487 bytes, keeping the error line; the shaped agents answered in 3 to 5
  turns with a peak context of about 4,000 tokens, against about 18,800 unshaped.
- **JSON and table:** summarised to under 300 bytes without the fact, as designed; the agents
  went back to the command, in 6 to 12 turns.
- **Terminal:** summarised to 26,968 bytes, the head and tail of the output, which is nearly
  the size of the plain cut; the shaped agents then re-ran the command several times and used
  more tokens than the unshaped ones. Also in harness #619.

## Tokens

Provider-reported usage from the run records (llama-server's counts as hrn records them),
summed over each arm's 18 cells, unweighted. The server reports no cache writes. There is no
cost: the model ran locally.

| arm | input | output | cache read | cells |
|---|---:|---:|---:|---:|
| local-unshaped | 336,027 | 27,444 | 1,461,009 | 18 |
| local-shaped | 167,770 | 53,177 | 943,542 | 18 |

Median per cell of input, output and cache read summed, by task:

| task | local-unshaped | local-shaped |
|---|---:|---:|
| shape-test | 182,010 | 38,795 |
| shape-log | 77,580 | 12,041 |
| shape-json | 129,232 | 65,075 |
| shape-table | 123,927 | 40,980 |
| shape-terminal | 79,137 | 176,904 |
| shape-noop | 10,439 | 11,031 |

The `shape-test` medians compare cells that failed early in the shaped arm with cells that
finished in the unshaped one, and the shaped arm's output total holds one runaway answer of
15,690 tokens. These figures are reported as measured, not as a saving: see rule 5 below.

## Against the publication gate

This page claims no saving and no quality difference; it answers whether shaping ever lost an
answer. Checked against [the publication gate](../docs/publication-gate.md):

- [x] 1. **Cases, repetitions and pairing:** met. Six cases, three repetitions per arm, both arms
  on the same tasks, paired by case and repetition.
- [x] 2. **Complete scoring:** met. All 36 cells are in the eval record and in the tables above,
  the four failures included; the failed cells ended in an error before their gates ran.
- [x] 3. **Usage and cost:** met. The token tables sum the provider-reported fields from the run
  records without weighting; there is no cost to report.
- [x] 4. **Negative and no-op cases:** met. `shape-noop` is in every aggregate, and so is
  `shape-terminal`, where shaping used more tokens.
- [ ] 5. **Uncertainty:** not met and not needed: no paired difference is claimed. A token saving
  claim would need the bootstrap interval, and on a 32k local model it would not carry over to
  Claude or Codex anyway.
- [x] 6. **Reproduction pins:** met. hrn 0.2.0 built from harness `751beb4b4915`; the api backend
  against llama.cpp `llama-server` 0.5.0-dev (build 11321, commit `b0aca3c65`, Vulkan) on
  127.0.0.1:8081 with one slot of 32,768 tokens, serving `gemma-4-26B-A4B-it-Q4_0.gguf`
  (SHA-256 `d208665ab1cd3a69f7a9a4bc59430e8448c8093d9b06334f566ac59d6d504a03`) under the
  variant's model name `gemma4-26b-a4b` (the run records report the file the server loaded);
  suite commit `617ee388bf8580df9155b48c648140216a4a1930`; eval id `19bdf`; fixture SHA-256
  `fixtures/shaping/emit/main.go` `414275a58070e8f871d34d52ba494c013eb3d0b468fbade3142a6aaf0af450c5`.
- [x] 7. **Number basis:** met. Every number is measured on this run, except the summary sizes
  in "What the summarizer did", which are **measured** by putting each output through a copy of
  harness `internal/shape` at `751beb4`, the commit hrn was built from.
