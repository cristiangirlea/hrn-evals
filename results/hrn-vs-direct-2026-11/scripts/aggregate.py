"""Aggregate the cells into paired per-backend tables and bootstrap intervals.

python aggregate.py <cells dir> [--json out.json]  -> Markdown on stdout

Stops when a cell appears twice, has no pair, or a task lacks a repeat that another task or
backend has. A task missing from every backend cannot be noticed: check the count against tasks.json.
"""
import glob
import json
import os
import random
import statistics
import sys


def total_tokens(cell):
    t = cell.get("tokens") or {}
    if cell["backend"] == "claude":
        return (t.get("input", 0) or 0) + (t.get("cache_read", 0) or 0) + (t.get("cache_write", 0) or 0) + (t.get("output", 0) or 0)
    # codex: input already includes cached reads
    return (t.get("input", 0) or 0) + (t.get("output", 0) or 0)


def med(xs):
    xs = [x for x in xs if x is not None]
    return statistics.median(xs) if xs else None


def fmt(x, unit=""):
    if x is None:
        return "-"
    if isinstance(x, float) and abs(x) < 100:
        return f"{x:.1f}{unit}"
    return f"{int(round(x)):,}{unit}"


def bootstrap(pairs_by_task, key, n=10000, seed=7):
    """pairs_by_task: {task: [(hrn_cell, direct_cell), ...]}; returns (mean diff, lo, hi)."""
    rnd = random.Random(seed)
    tasks = sorted(pairs_by_task)
    if not tasks:
        return None, None, None

    def diffs(task_list):
        out = []
        for t in task_list:
            for h, d in pairs_by_task[t]:
                a, b = key(h), key(d)
                if a is None or b is None:
                    continue
                out.append(a - b)
        return out

    base = diffs(tasks)
    if not base:
        return None, None, None
    mean = statistics.fmean(base)
    means = []
    for _ in range(n):
        sample = [rnd.choice(tasks) for _ in tasks]
        d = diffs(sample)
        if d:
            means.append(statistics.fmean(d))
    means.sort()
    lo = means[int(0.025 * len(means))]
    hi = means[min(len(means) - 1, int(0.975 * len(means)))]
    return mean, lo, hi


def main():
    cells_dir = sys.argv[1]
    cells = [json.load(open(p, encoding="utf-8")) for p in sorted(glob.glob(os.path.join(cells_dir, "*.json")))]
    out = {"backends": {}}
    lines = []
    # Every task seen anywhere must have every repeat seen anywhere, in each backend that ran.
    matrix = {(t, r) for t in {c["task"] for c in cells} for r in {c["repeat"] for c in cells}}
    for backend in ("claude", "codex"):
        bc = [c for c in cells if c["backend"] == backend]
        if not bc:
            continue
        by = {}
        for c in bc:
            arms = by.setdefault((c["task"], c["repeat"]), {})
            if c["arm"] in arms:
                sys.exit(f"two records for {c['task']} {backend} {c['arm']} repeat {c['repeat']}")
            arms[c["arm"]] = c
        pairs_by_task = {}
        for task, rep in sorted(matrix - set(by)):
            sys.exit(f"{task} {backend} repeat {rep} has no record, though another task or backend has that repeat")
        for (task, rep), arms in sorted(by.items()):
            if set(arms) != {"hrn", "direct"}:
                sys.exit(f"{task} {backend} repeat {rep} has no pair: only {', '.join(sorted(arms))}")
            pairs_by_task.setdefault(task, []).append((arms["hrn"], arms["direct"]))
        hrn = [c for c in bc if c["arm"] == "hrn"]
        direct = [c for c in bc if c["arm"] == "direct"]
        summary = {}
        for name, arm in (("hrn", hrn), ("direct", direct)):
            summary[name] = {
                "cells": len(arm),
                "passed": sum(1 for c in arm if c.get("gate_pass")),
                "median_wall_s": med([c.get("wall_s") for c in arm]),
                "median_tokens": med([total_tokens(c) for c in arm if c.get("tokens")]),
                "median_output_tokens": med([(c.get("tokens") or {}).get("output") for c in arm if c.get("tokens")]),
                "median_cost_usd": med([c.get("cost_usd") for c in arm if c.get("cost_usd") is not None]),
                "cells_with_cost": sum(1 for c in arm if c.get("cost_usd") is not None),
                "median_turns": med([c.get("turns") for c in arm]),
                "timeouts": sum(1 for c in arm if c.get("timed_out") or c.get("status") == "timeout"),
                "errors": sum(1 for c in arm if c.get("status") not in ("ok", None)),
            }
        pairs = [p for ps in pairs_by_task.values() for p in ps]
        paired = {
            "pairs": len(pairs),
            "both_pass": sum(1 for h, d in pairs if h.get("gate_pass") and d.get("gate_pass")),
            "hrn_only": sum(1 for h, d in pairs if h.get("gate_pass") and not d.get("gate_pass")),
            "direct_only": sum(1 for h, d in pairs if d.get("gate_pass") and not h.get("gate_pass")),
            "neither": sum(1 for h, d in pairs if not h.get("gate_pass") and not d.get("gate_pass")),
        }
        boots = {
            "pass": bootstrap(pairs_by_task, lambda c: 1.0 if c.get("gate_pass") else 0.0),
            "wall_s": bootstrap(pairs_by_task, lambda c: c.get("wall_s")),
            "tokens": bootstrap(pairs_by_task, lambda c: total_tokens(c) if c.get("tokens") else None),
            "output_tokens": bootstrap(pairs_by_task, lambda c: (c.get("tokens") or {}).get("output") if c.get("tokens") else None),
        }
        out["backends"][backend] = {"summary": summary, "paired": paired, "bootstrap": boots}

        lines.append(f"## {backend}\n")
        lines.append("| arm | cells | gate passed | median wall | median tokens | median output tokens | median reported cost (runs that reported one) | median turns | timeouts |")
        lines.append("|---|---:|---:|---:|---:|---:|---:|---:|---:|")
        for name in ("direct", "hrn"):
            s = summary[name]
            cost = f"${s['median_cost_usd']:.2f}" if s["median_cost_usd"] is not None else "-"
            cost += f" ({s['cells_with_cost']} of {s['cells']})"
            lines.append(f"| {name} | {s['cells']} | {s['passed']}/{s['cells']} | {fmt(s['median_wall_s'], ' s')} | {fmt(s['median_tokens'])} | {fmt(s['median_output_tokens'])} | {cost} | {fmt(s['median_turns'])} | {s['timeouts']} |")
        lines.append("")
        lines.append(f"Paired by task and repeat ({paired['pairs']} pairs): both pass {paired['both_pass']}, hrn only {paired['hrn_only']}, direct only {paired['direct_only']}, neither {paired['neither']}.\n")
        lines.append("| hrn minus direct | mean paired difference | 95% task bootstrap |")
        lines.append("|---|---:|---:|")
        for k, label, unit in (("pass", "gate pass (fraction)", ""), ("wall_s", "wall seconds", " s"), ("tokens", "total tokens", ""), ("output_tokens", "output tokens", "")):
            m, lo, hi = boots[k]
            if m is None:
                lines.append(f"| {label} | - | - |")
            else:
                lines.append(f"| {label} | {m:+.2f}{unit} | [{lo:+.2f}, {hi:+.2f}] |")
        lines.append("")
        lines.append("| task | r | direct pass | direct wall | direct tokens | hrn pass | hrn wall | hrn tokens | hrn model | direct model |")
        lines.append("|---|---|---|---:|---:|---|---:|---:|---|---|")
        for (task, rep), arms in sorted(by.items()):
            d, h = arms.get("direct"), arms.get("hrn")

            def cols(c):
                if not c:
                    return "-", "-", "-"
                return ("pass" if c.get("gate_pass") else "FAIL"), fmt(c.get("wall_s"), " s"), fmt(total_tokens(c) if c.get("tokens") else None)
            dp, dw, dt = cols(d)
            hp, hw, ht = cols(h)
            lines.append(f"| {task} | {rep} | {dp} | {dw} | {dt} | {hp} | {hw} | {ht} | {(h or {}).get('model') or '-'} | {(d or {}).get('model') or '-'} |")
        lines.append("")
        fails = [c for c in bc if not c.get("gate_pass")]
        if fails:
            lines.append("Failures, with the gate's last line and the agent's status:\n")
            for c in fails:
                tail = (c.get("gate_tail") or ["-"])[-1]
                lines.append(f"- `{c['cell']}`: status {c.get('status')}; gate: {tail[:160].strip()}")
            lines.append("")
    sys.stdout.reconfigure(newline="\n")
    print("\n".join(lines).rstrip("\n"))
    if "--json" in sys.argv:
        with open(sys.argv[sys.argv.index("--json") + 1], "w", encoding="utf-8") as f:
            json.dump(out, f, indent=1)


if __name__ == "__main__":
    main()
