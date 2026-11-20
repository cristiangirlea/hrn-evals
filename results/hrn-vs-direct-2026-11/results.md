## claude

| arm | cells | gate passed | median wall | median tokens | median output tokens | median reported cost (runs that reported one) | median turns | timeouts |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| direct | 16 | 15/16 | 391 s | 628,464 | 4,482 | $0.71 (13 of 16) | 13.0 | 4 |
| hrn | 16 | 16/16 | 468 s | 586,590 | 3,863 | $0.43 (11 of 16) | 13.0 | 5 |

Paired by task and repeat (16 pairs): both pass 15, hrn only 1, direct only 0, neither 0.

| hrn minus direct | mean paired difference | 95% task bootstrap |
|---|---:|---:|
| gate pass (fraction) | +0.06 | [+0.00, +0.19] |
| wall seconds | +60.16 s | [-86.19, +194.63] |
| total tokens | -46943.62 | [-176686.44, +88182.75] |
| output tokens | -363.25 | [-2185.88, +1051.75] |

| task | r | direct pass | direct wall | direct tokens | hrn pass | hrn wall | hrn tokens | hrn model | direct model |
|---|---|---|---:|---:|---|---:|---:|---|---|
| codex-reconnect-keeps-error | 1 | pass | 54.3 s | 391,714 | pass | 865 s | 163,817 | claude-opus-5-5 | claude-opus-5-5 |
| codex-reconnect-keeps-error | 2 | pass | 371 s | 396,571 | pass | 128 s | 379,351 | claude-opus-5-5 | claude-opus-5-5 |
| durable-partial-usage | 1 | pass | 901 s | 1,523,863 | pass | 902 s | 1,094,706 | claude-opus-5-5 | claude-opus-5-5 |
| durable-partial-usage | 2 | pass | 901 s | 2,016,939 | pass | 900 s | 1,842,843 | claude-opus-5-5 | claude-opus-5-5 |
| gc-idle-zero | 1 | pass | 298 s | 975,332 | pass | 240 s | 457,038 | claude-opus-5-5 | claude-opus-5-5 |
| gc-idle-zero | 2 | pass | 901 s | 598,043 | pass | 234 s | 553,703 | claude-opus-5-5 | claude-opus-5-5 |
| guard-case | 1 | pass | 671 s | 604,573 | pass | 901 s | 595,128 | claude-opus-5-5 | claude-opus-5-5 |
| guard-case | 2 | pass | 901 s | 652,355 | pass | 901 s | 662,786 | claude-opus-5-5 | claude-opus-5-5 |
| privacy-check-dedupe | 1 | pass | 237 s | 733,338 | pass | 259 s | 798,416 | claude-opus-5-5 | claude-opus-5-5 |
| privacy-check-dedupe | 2 | pass | 260 s | 845,304 | pass | 114 s | 457,652 | claude-opus-5-5 | claude-opus-5-5 |
| remote-oversized-line | 1 | pass | 120 s | 407,761 | pass | 140 s | 467,271 | claude-opus-5-5 | claude-opus-5-5 |
| remote-oversized-line | 2 | pass | 133 s | 284,321 | pass | 357 s | 302,350 | claude-opus-5-5 | claude-opus-5-5 |
| watch-dest-document | 1 | pass | 122 s | 759,169 | pass | 579 s | 792,595 | claude-opus-5-5 | claude-opus-5-5 |
| watch-dest-document | 2 | pass | 716 s | 233,262 | pass | 323 s | 585,308 | claude-opus-5-5 | claude-opus-5-5 |
| watch-quoted-path | 1 | pass | 485 s | 674,943 | pass | 701 s | 1,057,286 | claude-opus-5-5 | claude-opus-5-5 |
| watch-quoted-path | 2 | FAIL | 412 s | 451,733 | pass | 901 s | 587,873 | claude-opus-5-5 | claude-opus-5-5 |

Failures, with the gate's last line and the agent's status:

- `watch-quoted-path.claude.direct.r2`: status ok; gate: FAIL

## codex

| arm | cells | gate passed | median wall | median tokens | median output tokens | median reported cost (runs that reported one) | median turns | timeouts |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| direct | 16 | 16/16 | 196 s | 683,168 | 2,788 | - (0 of 16) | 1.0 | 0 |
| hrn | 16 | 16/16 | 169 s | 764,315 | 3,007 | - (0 of 16) | 1.0 | 0 |

Paired by task and repeat (16 pairs): both pass 16, hrn only 0, direct only 0, neither 0.

| hrn minus direct | mean paired difference | 95% task bootstrap |
|---|---:|---:|
| gate pass (fraction) | +0.00 | [+0.00, +0.00] |
| wall seconds | +23.21 s | [-37.70, +101.03] |
| total tokens | +103388.62 | [-101173.31, +337127.75] |
| output tokens | +341.25 | [-252.56, +1076.19] |

| task | r | direct pass | direct wall | direct tokens | hrn pass | hrn wall | hrn tokens | hrn model | direct model |
|---|---|---|---:|---:|---|---:|---:|---|---|
| codex-reconnect-keeps-error | 1 | pass | 64.8 s | 270,274 | pass | 168 s | 653,980 | gpt-6.1-terra | gpt-6.1-terra |
| codex-reconnect-keeps-error | 2 | pass | 256 s | 490,625 | pass | 146 s | 477,346 | gpt-6.1-terra | gpt-6.1-terra |
| durable-partial-usage | 1 | pass | 207 s | 1,138,522 | pass | 265 s | 1,268,707 | gpt-6.1-terra | gpt-6.1-terra |
| durable-partial-usage | 2 | pass | 217 s | 1,102,069 | pass | 534 s | 1,904,371 | gpt-6.1-terra | gpt-6.1-terra |
| gc-idle-zero | 1 | pass | 71.7 s | 300,894 | pass | 143 s | 829,848 | gpt-6.1-terra | gpt-6.1-terra |
| gc-idle-zero | 2 | pass | 228 s | 1,197,126 | pass | 53.2 s | 255,854 | gpt-6.1-terra | gpt-6.1-terra |
| guard-case | 1 | pass | 253 s | 810,779 | pass | 149 s | 584,362 | gpt-6.1-terra | gpt-6.1-terra |
| guard-case | 2 | pass | 176 s | 772,305 | pass | 208 s | 765,819 | gpt-6.1-terra | gpt-6.1-terra |
| privacy-check-dedupe | 1 | pass | 63.2 s | 234,058 | pass | 166 s | 502,609 | gpt-6.1-terra | gpt-6.1-terra |
| privacy-check-dedupe | 2 | pass | 71.7 s | 286,465 | pass | 363 s | 1,481,879 | gpt-6.1-terra | gpt-6.1-terra |
| remote-oversized-line | 1 | pass | 148 s | 465,988 | pass | 138 s | 576,495 | gpt-6.1-terra | gpt-6.1-terra |
| remote-oversized-line | 2 | pass | 147 s | 629,489 | pass | 46.6 s | 164,050 | gpt-6.1-terra | gpt-6.1-terra |
| watch-dest-document | 1 | pass | 198 s | 1,127,373 | pass | 170 s | 769,689 | gpt-6.1-terra | gpt-6.1-terra |
| watch-dest-document | 2 | pass | 206 s | 614,973 | pass | 188 s | 1,176,409 | gpt-6.1-terra | gpt-6.1-terra |
| watch-quoted-path | 1 | pass | 193 s | 736,847 | pass | 193 s | 786,764 | gpt-6.1-terra | gpt-6.1-terra |
| watch-quoted-path | 2 | pass | 260 s | 1,128,988 | pass | 200 s | 762,811 | gpt-6.1-terra | gpt-6.1-terra |
