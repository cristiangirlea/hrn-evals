# 2026-11-16: the gate and shaping on claude and codex

The eval harness#435 asks for (`docs/terminal-through-the-gate.md`, "How shaping reaches every
backend, and the eval"): does routing commands through adanext-exec cost anything, and does
shaping large output then save tokens or cost answers, on the two agents in daily use? Every
number on this page is **measured** on this run unless marked **inferred**.

## What was run

The six tasks of `suites/shaping/smoke.json`, unchanged: one exact fact buried in a large Go
test, log, JSON, table or terminal output (`go run ./fixtures/shaping/emit <kind>`), plus a
small output with nothing to shape. Gates check the fact, that the answer is at most 300 bytes,
and that the fixture was not changed. Three arms per agent, three repetitions, `yolo`,
`max_turns` 12, three cells at a time:

| arm | suite | variants |
|---|---|---|
| no gate (the native shell) | `suites/shaping/gate-baseline.json` | `claude-sonnet`, `codex` |
| gate, raw output | `suites/shaping/gate.json` (`command_gate`) | `claude-sonnet-gate-raw`, `codex-gate-raw` |
| gate, shaped | `suites/shaping/gate.json` (`command_gate`) | `claude-sonnet-gate-shaped`, `codex-gate-shaped` (`shape_tools`) |

An eval variant cannot switch the gate, only `shape_tools`, so the no-gate arm is its own spec
with the same tasks; it ran first, the gated spec straight after. In the gated eval the task
gates ran through adanext-exec too, under hrn's operator profile.

| | |
|---|---|
| no-gate eval | `0278c`, batch `20261116-152629-93942e`, 36 cells, 6m23s |
| gated eval | `74040`, batch `20261116-153314-d89a5b`, 72 cells, 12m19s |
| hrn | 0.2.0, harness `1a9a15fe` (go1.26.10) |
| adanext-exec | `d90bbe15` |
| Claude Code | 2.1.284, model `claude-sonnet-5-5` (reported) |
| Codex CLI | 0.167.0-alpha.3.2, model `gpt-6.1-sol` (reported) |
| suite commit | `c42f8ba` |
| fixture | `fixtures/shaping/emit/main.go` sha256 `414275a58070e8f871d34d52ba494c013eb3d0b468fbade3142a6aaf0af450c5` |

## Pass rate

**108 of 108 cells passed, every arm 18/18, each on its first attempt.** No arm lost an answer.
This suite is at its ceiling on both agents, so it cannot show a quality difference between the
arms, only that there was none to see here.

## Tokens

Tokens are the provider-reported fields from hrn's run records. Codex reports input inclusive
of its cached input (hrn records `inclusive_cache`): its cache-read is part of its input, and it
reports no cache-write.

Mean per cell, by arm:

| arm | input | output | cache-read | cache-write |
|---|---|---|---|---|
| claude no gate | 7 | 676 | 104,979 | 17,471 |
| claude gate raw | 7 | 630 | 70,112 | 16,728 |
| claude gate shaped | 6 | 674 | 66,135 | 14,829 |
| codex no gate | 108,370 | 220 | (87,310 of input) | none reported |
| codex gate raw | 135,299 | 300 | (110,592 of input) | none reported |
| codex gate shaped | 103,302 | 271 | (88,114 of input) | none reported |

The total per cell sums those fields unweighted: input + output + cache-read + cache-write for
claude, input + output for codex. Differences are paired by task. The interval is a 95%
bootstrap over the six tasks (10,000 resamples, seed 7, percentiles 250 and 9,749), each sampled
task keeping its three repetitions. A difference is claimed only when the interval excludes zero.

| comparison | mean tokens per cell | difference | 95% interval | claimed |
|---|---|---|---|---|
| claude: gate raw vs no gate | 123,132 → 87,476 | −35,657 (−29.0%) | −43,988 to −27,144 (−35.7% to −22.0%) | yes |
| claude: gate shaped vs gate raw | 87,476 → 81,644 | −5,831 (−6.7%) | −11,555 to −1,314 | yes, but not caused by shaping (below) |
| claude: gate shaped vs no gate | 123,132 → 81,644 | −41,488 (−33.7%) | −49,905 to −33,362 | yes |
| codex: gate raw vs no gate | 108,589 → 135,599 | +27,010 (+24.9%) | +14,486 to +38,135 (+13.3% to +35.1%) | yes |
| codex: gate shaped vs gate raw | 135,599 → 103,573 | −32,026 (−23.6%) | −68,760 to +838 (−50.7% to +0.6%) | no |
| codex: gate shaped vs no gate | 108,589 → 103,573 | −5,016 (−4.6%) | −34,887 to +21,332 (−32.1% to +19.6%) | no |

Mean total tokens per cell by task (three repetitions each):

| task | claude no gate | claude gate raw | claude gate shaped | codex no gate | codex gate raw | codex gate shaped |
|---|---|---|---|---|---|---|
| shape-test | 153,292 | 106,305 | 96,523 | 140,939 | 186,046 | 80,449 |
| shape-log | 111,688 | 78,201 | 78,311 | 127,229 | 162,970 | 80,573 |
| shape-json | 112,180 | 78,735 | 78,899 | 66,887 | 105,431 | 101,337 |
| shape-table | 112,088 | 95,882 | 78,833 | 98,529 | 113,102 | 122,326 |
| shape-terminal | 137,063 | 87,060 | 78,544 | 139,694 | 165,461 | 156,185 |
| shape-noop | 112,481 | 78,670 | 78,754 | 78,258 | 80,586 | 80,570 |

## What the numbers come from

**Claude never had anything shaped.** In all 45 claude cells with a large output the agent
filtered the command itself (`go run ./fixtures/shaping/emit log | grep -n ERROR`), so no result
came near the 30,000-byte threshold: the median claude tool result was 239 to 356 bytes per cell
in every arm, and the 18 shaped cells saved no original. The −6.7% between claude's shaped and
raw arms is therefore run-to-run variation, not shaping. It follows how many cells ran a third
command, a second search to confirm the fact: 6 of 18 raw cells against 2 of 18 shaped, each
such cell about 26k tokens more. A six-case percentile bootstrap still called it a difference,
which is the limit of six cases.

**Claude used about 30% fewer tokens under the gate, with the same commands.** Turns and tool
calls were the same with and without the gate (60 turns and 42 tool calls over 18 cells in both
arms), while the median context per cell fell from 37,496 to 26,258 tokens and cache-read fell
by 34,867 tokens a cell (−33.2%, interval −43,148 to −26,378). Under `--gate`, hrn starts claude
with `--tools Bash,PowerShell,Read,Edit,Write,...` (20 tools) to keep commands on the routed shell
tools; without the gate claude loads its full tool set. **Inferred:** the 11k tokens are the tool
definitions `--tools` leaves out, so the same saving would come from that list without the gate;
this was not tested, and the size depends on the tools and skills installed where the run
happens. Claude Code's reported median cost per cell fell less, by about 8% ($0.085 to $0.078),
since most of the saving is cache-read.

**Codex used about a quarter more tokens under the gate with raw output.** The median tool
result hrn recorded per codex cell was 44,779 bytes with no gate, 30,140 under the gate raw and
611 shaped. With shaping, 15 of 18 codex cells saved an original (the three no-op cells had
nothing to shape), and five read one back with `read_original` (all three table cells and two
terminal cells). By task, shaping took codex's test and log tasks from 163–186k tokens a cell
(gate raw) to about 80k, and left JSON, table and terminal above the native shell (101k, 122k and
156k against 67k, 99k and 140k). Neither codex difference involving shaping could be claimed:
shaped against raw −23.6% (−50.7% to +0.6%), shaped against no gate −4.6% (−32.1% to +19.6%).

Costs, for reference only: claude's are what Claude Code reports per run, codex's are hrn's
**estimate** (`~$`). Median per cell: claude $0.085 no gate, $0.078 raw, $0.079 shaped; codex
~$0.063, ~$0.063, ~$0.032. Both ran on subscription plans; nothing was billed per token.

## What this means for harness#435

- **Quality:** no arm lost an answer on either agent (108/108), and the suite is at its ceiling.
  That shaping did no harm here rests on the 15 shaped codex cells alone, since nothing was
  shaped on claude.
- **claude:** shaping had nothing to do, because claude filters large output on its own. Runs
  under the gate used about 30% fewer tokens, with the tool list as the **inferred** cause.
- **codex:** the gate with raw output costs about a quarter more tokens than the native shell.
  With shaping the measured mean is lower, but neither shaping comparison can be claimed.
- On this suite neither `--gate` nor `shape_tools` shows a claimable token saving of its own over
  the native shell on codex, and shaping never fired on claude; whether either becomes a default
  is decision 6 in the design, for a maintainer.

## Publication gate

- [x] 1. At least six cases, three repetitions per arm, paired identical tasks. (The no-gate arm
  ran as its own eval minutes before the gated one, same tasks and suite commit.)
- [x] 2. Every run gate-scored; none dropped. (108 cells, one attempt each.)
- [x] 3. Provider-only unweighted token usage; costs reported or marked estimated. (Codex input
  is inclusive of its cache-read, so cache-read is not added again.)
- [x] 4. Negative and no-op cases included in the aggregate. (`shape-noop` is in every figure.)
- [x] 5. 95% case-bootstrap interval, 10,000 resamples; a difference is claimed only where it
  excludes zero.
- [x] 6. All reproduction pins and eval ids recorded.
- [x] 7. Every number labelled measured, inferred or verified.

Reproduce: `hrn eval suites/shaping/gate-baseline.json` and `hrn eval suites/shaping/gate.json`
(needs `adanext-exec` in `PATH`; both run on the Claude and Codex plans); compare with
`hrn eval show 0278c` and `hrn eval show 74040`.
