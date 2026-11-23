"""Write the public copy of the run records: cells without the agents' own text, and the plan readings.

python publish.py <runner out dir> <new empty dir>

Only the fields in KEEP leave the runner's records. The agents' final answers, stderr, error text
and session ids stay behind, and so do the raw transcripts: codex reads the maintainer's private
memory file in every run and quotes it. Program paths become program names, and the work, temp
and home folders become <work>, <tmp> and ~. The destination must be new or empty; copy its
contents over the published folder by hand, then rebuild results.md with aggregate.py.
"""
import glob
import json
import os
import re
import sys
import time

KEEP = [
    "cell", "task", "backend", "arm", "repeat", "base", "started", "notes",
    "tokens", "status", "command", "exit_code", "wall_s", "timed_out",
    "hrn_run_id", "turns", "tool_calls", "tokens_source", "model", "cli_version",
    "usage_status", "cost_usd", "cost_estimate", "agent_duration_s",
    "refusal_flags", "event_kinds", "guard_refusals_seen",
    "agent_committed", "changed_files", "patch_bytes", "gate_pass", "gate_s", "gate_tail",
    "finished",
]
DROP = {"final", "stderr_lines", "error", "session"}


def spellings(path, mark):
    # A Windows path as written raw, JSON-escaped, with slashes, and as Git Bash writes it.
    path = os.path.abspath(path)
    out = [(path.replace("\\", "\\\\"), mark), (path, mark), (path.replace("\\", "/"), mark)]
    if len(path) > 2 and path[1] == ":":
        out.append(("/" + path[0].lower() + path[2:].replace("\\", "/"), mark))
    return out


def scrubber(src):
    work = os.path.dirname(os.path.abspath(src))
    tmp = os.environ.get("HRN_VS_DIRECT_TMP", r"E:\hrn-vd-tmp")  # the runner's own default
    pairs = [(work, "<work>"), (tmp, "<tmp>"), (os.path.expanduser("~"), "~")]
    table = sorted((s for p, m in pairs for s in spellings(p, m)), key=lambda s: -len(s[0]))

    def scrub(text):
        for a, b in table:
            text = text.replace(a, b)
        return text
    return scrub


def program(arg):
    # A program's path becomes its name: no install folder leaves the machine.
    return re.split(r"[\\/]", arg)[-1].rsplit(".", 1)[0].lower()


def public_cell(cell):
    unknown = sorted(set(cell) - set(KEEP) - DROP)
    if unknown:
        sys.exit(f"{cell.get('cell')}: fields neither kept nor dropped: {', '.join(unknown)}")
    out = {k: cell[k] for k in cell if k in KEEP}
    if out.get("command"):
        out["command"] = [program(out["command"][0])] + out["command"][1:]
    out["stderr_line_count"] = len(cell.get("stderr_lines") or [])
    return out


def plan_readings(src):
    rows = []
    for p in sorted(glob.glob(os.path.join(src, "backends-*.txt")), key=os.path.getmtime):
        left, name = {}, None
        with open(p, encoding="utf-8", errors="replace") as f:
            for line in f:
                m = re.match(r"\s*\S\s+(\w+)\s+(enabled|disabled)", line)
                if m:
                    name = m.group(1)
                m = re.search(r"week: (\d+)% left", line)
                if m and name:
                    left[name] = m.group(1) + "%"
        label = os.path.basename(p)[len("backends-"):-len(".txt")].replace("-", " lane ")
        taken = time.strftime("%Y-%m-%d %H:%M", time.localtime(os.path.getmtime(p)))
        rows.append(f"{taken}  {label:17}  claude week left {left.get('claude', '?'):>4}  codex week left {left.get('codex', '?'):>4}")
    return "\n".join(rows) + "\n"


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    src, dest = sys.argv[1], sys.argv[2]
    if os.path.isdir(dest) and os.listdir(dest):
        sys.exit(f"{dest} is not empty; give a new folder")
    scrub = scrubber(src)
    cells = sorted(glob.glob(os.path.join(src, "cells", "*.json")))
    if not cells:
        sys.exit(f"no cells under {src}")
    # Every record is checked before the first one is written, so a refusal leaves nothing behind.
    public = []
    for p in cells:
        with open(p, encoding="utf-8") as f:
            public.append((os.path.basename(p), public_cell(json.load(f))))
    os.makedirs(os.path.join(dest, "cells"))
    for name, cell in public:
        with open(os.path.join(dest, "cells", name), "w", encoding="utf-8", newline="\n") as f:
            f.write(scrub(json.dumps(cell, indent=1)) + "\n")
    with open(os.path.join(dest, "plan-readings.txt"), "w", encoding="utf-8", newline="\n") as f:
        f.write(plan_readings(src))
    print(f"wrote {len(cells)} cells and plan-readings.txt to {dest}")


if __name__ == "__main__":
    main()
