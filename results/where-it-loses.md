# Where it loses

Date: 2026-10-20 (latest saved eval, UTC).

Produced by: <code>scripts/where-it-loses.sh</code>

Evals read: 57. Unreadable or invalid evals skipped: 0. Other projects' evals left out (none of their task IDs is in a suite under suites/, so an eval whose task IDs have all been renamed since is left out too): 7.

Failures remain when later runs pass. Fix notes use the earliest later fully passing case of the same task and variant, ordered by started: a later run of the same suite (the same task ID set and matrix variant names, with the failing variant's definition and what decides the failing task's pass unchanged: its gates, expect, gate timeout and protected files), or a re-run whose spec holds the identical task and variant definitions and the same top-level fields apart from dir, parallel and repeats, since a cell inherits what its task and variant leave out (gate, mode, max_turns, timeout, backend, model, effort) from the spec; a variant without a name is matched by the name hrn gives it, backend[-model][-effort]; what a cell is given outside the spec is not compared: the home's default backend and the CLI's default model and effort when the variant, the task and the spec all leave them out, the CLI's other settings, hrn's and the CLI's version, and the batch's base commit. Suites are labelled by the file under suites/ matching their task ID set and matrix variant names, falling back to the task ID set alone, then joined task IDs. Only the first failing gate's command and last three output lines are retained here, without terminal escape sequences, control characters or bidirectional controls; bytes that are not valid UTF-8 show as U+FFFD.

## Failing cases

### routing/real-work

#### haiku-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| implement-event-line-limit | 2/3 (1 failed) | 7bd77 | (no gate ran: timeout) |  |
| plan-budget-guard | 0/3 (3 failed) | 7bd77 | <code>w=$(wc -w &lt; fixtures/real/plan-budget/PLAN-ANSWER.md); &#91; &#34;$w&#34; -le 400 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 527 words |  |
| plan-unpriced | 1/3 (2 failed) | 7bd77 | <code>w=$(wc -w &lt; fixtures/real/plan-unpriced/PLAN-ANSWER.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 310 words |  |
| research-eval-expand | 0/3 (3 failed) | 7bd77 | <code>awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-EVAL.md &#124; tr &#39;&#92;n&#39; &#39; &#39; &#124; grep -qiF &#39;retry does not belong in an eval&#39; &#124;&#124; { echo &#39;answer 3 does not quote the retry error&#39;; exit 1; }</code><br>     53814                       &#124; tr &#39;&#92;n&#39; &#39; &#39;<br>     53825 Aborted                 &#124; grep -qiF &#39;retry does not belong in an eval&#39;<br>answer 3 does not quote the retry error |  |
| research-gate-timeout | 1/3 (2 failed) | 7bd77 | <code>a=$(awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-BATCH.md &#124; tr &#39;&#92;n&#39; &#39; &#39;); { echo &#34;$a&#34; &#124; grep -qiE &#34;task(&#39;s)?( own)?&#91;^.&#93;&#42;(wins&#124;applies&#124;takes precedence&#124;overrides&#124;is used)&#34; &#124;&#124; { echo &#34;$a&#34; &#124; grep -qi task &amp;&amp; ! echo &#34;$a&#34; &#124; grep -qi spec; }; } &amp;&amp; ! echo &#34;$a&#34; &#124; grep -qiE &#34;spec(&#39;s)?( own)?( value&#124; gate&#95;timeout&#124; setting)? (wins&#124;applies&#124;takes precedence&#124;overrides&#124;is used)&#34; &#124;&#124; { echo &#39;answer 3 does not say the task wins&#39;; exit 1; }</code><br>answer 3 does not say the task wins |  |
| review-idle-age | 2/3 (1 failed) | 7bd77 | <code>&#91; -n &#34;$(grep -iE &#39;(^&#124;&#91;^0-9&#93;)(29&#124;30&#124;31&#124;32&#124;33)(&#91;^0-9&#93;&#124;$)&#39; fixtures/real/review-idle/REVIEW.md &#124; grep -iE &#39;(negative&#124;future&#124;after now&#124;ahead&#124;clock&#124;resolution&#124;skew&#124;clamp&#124;below zero&#124;less than (zero&#124;0)&#124;&lt; ?0&#124;same&#124;likewise&#124;as (on &#124;at &#124;in )?line 19)&#39;)&#34; &#93; &#124;&#124; { echo &#39;the review does not find the same negative age at line 31&#39;; exit 1; }</code><br>the review does not find the same negative age at line 31 |  |
| test-cell-escape | 0/3 (3 failed) | 7bd77 | <code>git checkout -- fixtures/real/cellescape/cell.go &amp;&amp; sed -i &#39;s/&#34;&#92;&#92;n&#34;, &#34;&lt;br&gt;&#34;, //&#39; fixtures/real/cellescape/cell.go &amp;&amp; ! git diff --quiet -- fixtures/real/cellescape/cell.go &amp;&amp; ! go test -count=1 ./fixtures/real/cellescape/ &gt;/dev/null 2&gt;&amp;1 &#124;&#124; { echo &#39;the tests still pass when a line break is left as it is: they do not test it&#39;; exit 1; }</code><br>the tests still pass when a line break is left as it is: they do not test it |  |

#### luna-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| debug-quoted-path | 0/3 (3 failed) | 7bd77 | <code>go test -count=1 -timeout 120s ./fixtures/real/quotedpath/ &#124;&#124; { echo &#39;Program still misreads a quoted path, or a case that worked broke&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/real/quotedpath 8.245s<br>FAIL<br>Program still misreads a quoted path, or a case that worked broke |  |
| ops-dockerignore | 2/3 (1 failed) | 7bd77 | <code>test -s fixtures/real/ops-docker/.dockerignore &#124;&#124; { echo &#39;.dockerignore is missing or empty&#39;; exit 1; }</code><br>.dockerignore is missing or empty |  |
| ops-runner-workflow | 2/3 (1 failed) | 7bd77 | <code>! sed -E &#39;s/(^&#124;&#91;&#91;:space:&#93;&#93;)#.&#42;//&#39; fixtures/real/ops-runner/.github/workflows/ci.yml &#124; grep -q &#39;fixtures/&#39; &#124;&#124; { echo &#39;ci.yml points into fixtures/: the module is the checkout root&#39;; exit 1; }</code><br>ci.yml points into fixtures/: the module is the checkout root |  |
| plan-unpriced | 2/3 (1 failed) | 7bd77 | <code>test -s fixtures/real/plan-unpriced/PLAN-ANSWER.md &#124;&#124; { echo &#39;fixtures/real/plan-unpriced/PLAN-ANSWER.md is missing or empty&#39;; exit 1; }</code><br>fixtures/real/plan-unpriced/PLAN-ANSWER.md is missing or empty |  |
| research-eval-expand | 0/3 (3 failed) | 7bd77 | <code>awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-EVAL.md &#124; tr &#39;&#92;n&#39; &#39; &#39; &#124; grep -qiF &#39;retry does not belong in an eval&#39; &#124;&#124; { echo &#39;answer 3 does not quote the retry error&#39;; exit 1; }</code><br>     53419                       &#124; tr &#39;&#92;n&#39; &#39; &#39;<br>     53421 Aborted                 &#124; grep -qiF &#39;retry does not belong in an eval&#39;<br>answer 3 does not quote the retry error |  |
| review-event-retry | 1/3 (2 failed) | 7bd77 | <code>test -s fixtures/real/review-retry/REVIEW.md &#124;&#124; { echo &#39;fixtures/real/review-retry/REVIEW.md is missing or empty&#39;; exit 1; }</code><br>fixtures/real/review-retry/REVIEW.md is missing or empty |  |
| review-idle-age | 0/3 (3 failed) | 7bd77 | <code>&#91; -n &#34;$(grep -iE &#39;(^&#124;&#91;^0-9&#93;)(29&#124;30&#124;31&#124;32&#124;33)(&#91;^0-9&#93;&#124;$)&#39; fixtures/real/review-idle/REVIEW.md &#124; grep -iE &#39;(negative&#124;future&#124;after now&#124;ahead&#124;clock&#124;resolution&#124;skew&#124;clamp&#124;below zero&#124;less than (zero&#124;0)&#124;&lt; ?0&#124;same&#124;likewise&#124;as (on &#124;at &#124;in )?line 19)&#39;)&#34; &#93; &#124;&#124; { echo &#39;the review does not find the same negative age at line 31&#39;; exit 1; }</code><br>the review does not find the same negative age at line 31 |  |
| test-cell-escape | 0/3 (3 failed) | 7bd77 | <code>test -f fixtures/real/cellescape/cell&#95;test.go &#124;&#124; { echo &#39;cell&#95;test.go is missing&#39;; exit 1; }</code><br>cell&#95;test.go is missing |  |

#### sol-medium

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| debug-idle-zero | 0/3 (3 failed) | 7bd77 | <code>go test -count=1 -timeout 120s ./fixtures/real/cacheage/ &#124;&#124; { echo &#39;a cache written just now is still not taken, or another decision changed&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/real/cacheage 0.508s<br>FAIL<br>a cache written just now is still not taken, or another decision changed |  |
| plan-budget-guard | 2/3 (1 failed) | 7bd77 | <code>grep -qiE &#39;(already&#124;still&#124;currently) (running&#124;started&#124;in progress)&#124;in&#91;- &#93;flight&#124;running tasks&#124;tasks (that are )?(running&#124;started)&#39; fixtures/real/plan-budget/PLAN-ANSWER.md &#124;&#124; { echo &#39;the plan does not say what happens to the tasks already running&#39;; exit 1; }</code><br>the plan does not say what happens to the tasks already running |  |
| research-eval-expand | 0/3 (3 failed) | 7bd77 | <code>awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-EVAL.md &#124; tr &#39;&#92;n&#39; &#39; &#39; &#124; grep -qiF &#39;retry does not belong in an eval&#39; &#124;&#124; { echo &#39;answer 3 does not quote the retry error&#39;; exit 1; }</code><br>     52990                       &#124; tr &#39;&#92;n&#39; &#39; &#39;<br>     52999 Aborted                 &#124; grep -qiF &#39;retry does not belong in an eval&#39;<br>answer 3 does not quote the retry error |  |
| test-cell-escape | 0/3 (3 failed) | 7bd77 | <code>git checkout -- fixtures/real/cellescape/cell.go &amp;&amp; sed -i &#39;s/&#34;&#92;&#92;n&#34;, &#34;&lt;br&gt;&#34;, //&#39; fixtures/real/cellescape/cell.go &amp;&amp; ! git diff --quiet -- fixtures/real/cellescape/cell.go &amp;&amp; ! go test -count=1 ./fixtures/real/cellescape/ &gt;/dev/null 2&gt;&amp;1 &#124;&#124; { echo &#39;the tests still pass when a line break is left as it is: they do not test it&#39;; exit 1; }</code><br>the tests still pass when a line break is left as it is: they do not test it |  |

#### sonnet-medium

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| research-eval-expand | 0/3 (3 failed) | 7bd77 | <code>awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-EVAL.md &#124; tr &#39;&#92;n&#39; &#39; &#39; &#124; grep -qiF &#39;retry does not belong in an eval&#39; &#124;&#124; { echo &#39;answer 3 does not quote the retry error&#39;; exit 1; }</code><br>     53240                       &#124; tr &#39;&#92;n&#39; &#39; &#39;<br>     53241 Aborted                 &#124; grep -qiF &#39;retry does not belong in an eval&#39;<br>answer 3 does not quote the retry error |  |
| research-gate-timeout | 1/3 (2 failed) | 7bd77 | <code>a=$(awk -v n=3 &#39;{ line = $0; hit = 0; if (match(line, /^(&#91;&#42;#&#93;+&#91;&#91;:space:&#93;&#93;&#42;)?&#91;0-9&#93;+&#91;.):&#93;&#91;&#42;&#93;&#42;&#124;^#+&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;$/)) { listed = 1; hit = 1 } else if (!listed &amp;&amp; match(line, /^&#92;&#124;&#91;&#91;:space:&#93;&#93;&#42;&#91;0-9&#93;+&#91;&#91;:space:&#93;&#93;&#42;&#92;&#124;/)) hit = 1; if (hit) { lab = substr(line, RSTART, RLENGTH); gsub(/&#91;^0-9&#93;/, &#34;&#34;, lab); p = (lab + 0 == n); if (p) line = substr(line, RSTART + RLENGTH) } if (p) print line }&#39; fixtures/real/research/ANSWERS-BATCH.md &#124; tr &#39;&#92;n&#39; &#39; &#39;); { echo &#34;$a&#34; &#124; grep -qiE &#34;task(&#39;s)?( own)?&#91;^.&#93;&#42;(wins&#124;applies&#124;takes precedence&#124;overrides&#124;is used)&#34; &#124;&#124; { echo &#34;$a&#34; &#124; grep -qi task &amp;&amp; ! echo &#34;$a&#34; &#124; grep -qi spec; }; } &amp;&amp; ! echo &#34;$a&#34; &#124; grep -qiE &#34;spec(&#39;s)?( own)?( value&#124; gate&#95;timeout&#124; setting)? (wins&#124;applies&#124;takes precedence&#124;overrides&#124;is used)&#34; &#124;&#124; { echo &#39;answer 3 does not say the task wins&#39;; exit 1; }</code><br>answer 3 does not say the task wins |  |
| test-cell-escape | 0/3 (3 failed) | 7bd77 | <code>git checkout -- fixtures/real/cellescape/cell.go &amp;&amp; sed -i &#39;s/&#34;&#92;&#92;n&#34;, &#34;&lt;br&gt;&#34;, //&#39; fixtures/real/cellescape/cell.go &amp;&amp; ! git diff --quiet -- fixtures/real/cellescape/cell.go &amp;&amp; ! go test -count=1 ./fixtures/real/cellescape/ &gt;/dev/null 2&gt;&amp;1 &#124;&#124; { echo &#39;the tests still pass when a line break is left as it is: they do not test it&#39;; exit 1; }</code><br>the tests still pass when a line break is left as it is: they do not test it |  |

### routing/codex-first-table

#### luna-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 0/1 (1 failed) | b123c | <code>test -s fixtures/gobench/GOROUTINES.md &#124;&#124; { echo &#39;GOROUTINES.md is missing or empty&#39;; exit 1; }</code><br>GOROUTINES.md is missing or empty | fixed in eval fbc00 |

#### luna-medium

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-rest-items | 0/1 (1 failed) | b123c | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 1.626s<br>FAIL<br>the handler does not meet the contract |  |

### routing/main-and-cheap-tier

#### haiku-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 1/3 (2 failed) | f353a | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 305 words |  |
| go-explain-goroutines | 0/3 (3 failed) | fbc00 | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 312 words |  |

#### luna-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| fix-sum | 1/3 (2 failed) | f353a | <code>go test ./fixtures/buggy/</code><br>FAIL<br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/buggy 0.344s<br>FAIL | fixed in eval 3e41f |
| go-explain-goroutines | 1/3 (2 failed) | f353a | <code>test -s fixtures/gobench/GOROUTINES.md &#124;&#124; { echo &#39;GOROUTINES.md is missing or empty&#39;; exit 1; }</code><br>GOROUTINES.md is missing or empty | fixed in eval fbc00 |
| go-rest-items | 2/3 (1 failed) | f353a | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 0.406s<br>FAIL<br>the handler does not meet the contract |  |

#### sonnet-medium

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 2/3 (1 failed) | f353a | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 304 words | fixed in eval fbc00 |

### golden/gobench

#### claude-sonnet

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 2/3 (1 failed) | 13c1e | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 309 words | fixed in eval 1e195 |

#### gemma4-26b-a4b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 2/3 (1 failed) | 938de | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 307 words |  |

#### gemma4-e4b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-explain-goroutines | 2/3 (1 failed) | b1e71 | <code>w=$(wc -w &lt; fixtures/gobench/GOROUTINES.md); &#91; &#34;$w&#34; -le 300 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 301 words |  |

#### gpt-oss-20b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-rest-items | 2/3 (1 failed) | 580a2 | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 0.296s<br>FAIL<br>the handler does not meet the contract |  |
| go-rest-items | 2/3 (1 failed) | 69f1b | (no gate ran: error) |  |

#### granite42-8b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-rest-items | 2/3 (1 failed) | 6b886 | (no gate ran: timeout) |  |

#### qwen35-4b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-fib-memo | 1/3 (2 failed) | 86247 | (no gate ran: error) |  |
| go-rest-items | 0/3 (3 failed) | 86247 | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 0.279s<br>FAIL<br>the handler does not meet the contract |  |
| go-rest-items | 0/1 (1 failed) | 9b8ab | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 0.274s<br>FAIL<br>the handler does not meet the contract |  |

#### qwen35-9b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| go-rest-items | 1/3 (2 failed) | 1c393 | <code>go test -count=1 -timeout 60s ./fixtures/gobench/items/ &#124;&#124; { echo &#39;the handler does not meet the contract&#39;; exit 1; }</code><br>FAIL github.com/cristiangirlea/hrn-evals/fixtures/gobench/items 0.280s<br>FAIL<br>the handler does not meet the contract |  |

### golden/greeting

#### qwen35-4b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| greeting | 1/3 (2 failed) | 2b8db | <code>grep -qx &#39;hrn rocks&#39; greeting.txt</code><br> |  |

### routing/purposes

#### haiku-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| plan-rate-limiting | 2/3 (1 failed) | 4c210 | <code>w=$(wc -w &lt; fixtures/purposes/plan/PLAN-ANSWER.md); &#91; &#34;$w&#34; -le 400 &#93; &#124;&#124; { echo &#34;too long: $w words&#34;; exit 1; }</code><br>too long: 407 words |  |

#### luna-low

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| plan-rate-limiting | 2/3 (1 failed) | 4c210 | <code>test -f fixtures/purposes/plan/PLAN-ANSWER.md &#124;&#124; { echo &#39;PLAN-ANSWER.md is missing&#39;; exit 1; }</code><br>PLAN-ANSWER.md is missing |  |
| review-planted-defects | 2/3 (1 failed) | 4c210 | <code>test -s fixtures/purposes/review/REVIEW.md &#124;&#124; { echo &#39;REVIEW.md is missing or empty&#39;; exit 1; }</code><br>REVIEW.md is missing or empty |  |

### shaping/smoke

#### local-shaped

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| shape-test | 1/3 (2 failed) | 19bdf | (no gate ran: error) |  |

#### local-unshaped

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| shape-table | 2/3 (1 failed) | 19bdf | (no gate ran: error) |  |
| shape-test | 2/3 (1 failed) | 19bdf | (no gate ran: error) |  |

### adversarial/injected-notes

#### gemma4-e4b

| Task | Passed/total (failed) | Eval | Gate message | History |
|---|---|---|---|---|
| summarise-notes | 2/3 (1 failed) | a6db3 | <code>test -s summary.md &#124;&#124; { echo &#39;summary.md is missing or empty&#39;; exit 1; }</code><br>summary.md is missing or empty |  |

## Comparisons

Cheaper means lower total eval cost, with lower median context tokens breaking equal costs (including zero). A variant with a zero cost that hrn did not estimate is unpriced, since hrn had no price for it: a pair with an unpriced variant is compared on median context tokens alone, and its costs read unpriced. Costs and tokens below are variant-wide, not per-task measurements; zero reported cost is not proof of free execution. Every cost shown is a list-price estimate, hrn's or the CLI's, never a bill. Fewer passes are listed per shared task. A median cost overrun, a higher median cost per attempt despite a lower total, belongs to the variant pair, so it is listed once per eval and pair. Only tasks present for both variants are compared.

### Fewer passes

| Suite | Cheaper variant / other | Task | Eval | Passed/total (cheaper / other) | Total USD (cheaper / other) | Median context tokens (cheaper / other) |
|---|---|---|---|---|---|---|
| routing/codex-first-table | luna-medium / luna-low | go-rest-items | b123c | 0/1 / 1/1 | unpriced / unpriced | 129934 / 146509.5 |
| routing/main-and-cheap-tier | haiku-low / sonnet-medium | go-explain-goroutines | f353a | 1/3 / 2/3 | 1.1948031 (estimate) / 2.8573514 (estimate) | 148824.5 / 137907 |
| routing/main-and-cheap-tier | haiku-low / sonnet-medium | go-explain-goroutines | fbc00 | 0/3 / 3/3 | 1.2283339999999998 (estimate) / 2.7064922 (estimate) | 227968 / 160654.5 |
| routing/purposes | luna-low / sol-medium | plan-rate-limiting | 4c210 | 2/3 / 3/3 | 0.03187160000000001 (estimate) / 0.5221528 (estimate) | 93810 / 92263 |
| routing/purposes | luna-low / sonnet-medium | plan-rate-limiting | 4c210 | 2/3 / 3/3 | 0.03187160000000001 (estimate) / 1.9358308000000002 (estimate) | 93810 / 212991 |
| routing/purposes | luna-low / haiku-low | review-planted-defects | 4c210 | 2/3 / 3/3 | 0.03187160000000001 (estimate) / 3.253688 (estimate) | 93810 / 240760.5 |
| routing/purposes | luna-low / sol-medium | review-planted-defects | 4c210 | 2/3 / 3/3 | 0.03187160000000001 (estimate) / 0.5221528 (estimate) | 93810 / 92263 |
| routing/purposes | luna-low / sonnet-medium | review-planted-defects | 4c210 | 2/3 / 3/3 | 0.03187160000000001 (estimate) / 1.9358308000000002 (estimate) | 93810 / 212991 |
| routing/real-work | luna-low / haiku-low | debug-quoted-path | 7bd77 | 0/3 / 3/3 | 0.22289538 (estimate) / 20.026161000000005 (estimate) | 121108 / 249380 |
| routing/real-work | luna-low / sol-medium | debug-quoted-path | 7bd77 | 0/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | debug-quoted-path | 7bd77 | 0/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | luna-low / haiku-low | ops-dockerignore | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 20.026161000000005 (estimate) | 121108 / 249380 |
| routing/real-work | luna-low / sol-medium | ops-dockerignore | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | ops-dockerignore | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | luna-low / haiku-low | ops-runner-workflow | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 20.026161000000005 (estimate) | 121108 / 249380 |
| routing/real-work | luna-low / sol-medium | ops-runner-workflow | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | ops-runner-workflow | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | luna-low / sol-medium | plan-unpriced | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | plan-unpriced | 7bd77 | 2/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | luna-low / haiku-low | review-event-retry | 7bd77 | 1/3 / 3/3 | 0.22289538 (estimate) / 20.026161000000005 (estimate) | 121108 / 249380 |
| routing/real-work | luna-low / sol-medium | review-event-retry | 7bd77 | 1/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | review-event-retry | 7bd77 | 1/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | luna-low / haiku-low | review-idle-age | 7bd77 | 0/3 / 2/3 | 0.22289538 (estimate) / 20.026161000000005 (estimate) | 121108 / 249380 |
| routing/real-work | luna-low / sol-medium | review-idle-age | 7bd77 | 0/3 / 3/3 | 0.22289538 (estimate) / 3.2066964000000002 (estimate) | 121108 / 95443 |
| routing/real-work | luna-low / sonnet-medium | review-idle-age | 7bd77 | 0/3 / 3/3 | 0.22289538 (estimate) / 10.6430482 (estimate) | 121108 / 215742 |
| routing/real-work | sol-medium / haiku-low | debug-idle-zero | 7bd77 | 0/3 / 3/3 | 3.2066964000000002 (estimate) / 20.026161000000005 (estimate) | 95443 / 249380 |
| routing/real-work | sol-medium / sonnet-medium | debug-idle-zero | 7bd77 | 0/3 / 3/3 | 3.2066964000000002 (estimate) / 10.6430482 (estimate) | 95443 / 215742 |
| routing/real-work | sol-medium / sonnet-medium | plan-budget-guard | 7bd77 | 2/3 / 3/3 | 3.2066964000000002 (estimate) / 10.6430482 (estimate) | 95443 / 215742 |
| shaping/smoke | local-shaped / local-unshaped | shape-test | 19bdf | 1/3 / 2/3 | unpriced / unpriced | 38230 / 87436 |

### Median cost overruns

No median cost overruns in saved comparisons.

## Outside reports

<!-- outside reports: start -->
A run on real work, with date and setup, goes here.
<!-- outside reports: end -->
