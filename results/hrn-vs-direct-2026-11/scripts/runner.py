"""Run one cell or one lane of the hrn-versus-direct comparison.

python runner.py cell <task> <claude|codex> <hrn|direct> <repeat>
python runner.py lane <claude|codex> [task ...]
"""
import datetime
import json
import os
import shutil
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
WORK = os.environ.get("HRN_VS_DIRECT_WORK", r"E:\hrn-vs-direct")
BASE = os.path.join(WORK, "harness")
WT = os.path.join(WORK, "wt")
OUT = os.path.join(WORK, "out")
CELLS = os.path.join(OUT, "cells")
RAW = os.path.join(OUT, "raw")
HRN_HOME = os.path.join(os.path.expanduser("~"), ".hrn")


def _codex():
    # The Codex installer keeps codex.exe under a hashed folder; take the newest, as hrn does.
    root = os.path.join(os.environ.get("LOCALAPPDATA", ""), "OpenAI", "Codex", "bin")
    found = sorted((os.path.getmtime(os.path.join(root, e, "codex.exe")), os.path.join(root, e, "codex.exe"))
                   for e in (os.listdir(root) if os.path.isdir(root) else [])
                   if os.path.isfile(os.path.join(root, e, "codex.exe")))
    return found[-1][1] if found else shutil.which("codex")


CODEX = _codex()
CLAUDE = shutil.which("claude") or os.path.join(os.path.expanduser("~"), ".local", "bin", "claude.exe")
BASH = r"C:\Program Files\Git\bin\bash.exe"
HRN = shutil.which("hrn")
AGENT_TIMEOUT = 15 * 60
GATE_TIMEOUT = 15 * 60
TMPDIR = os.environ.get("HRN_VS_DIRECT_TMP", r"E:\hrn-vd-tmp")

# tasks.json sits beside the scripts folder in the published layout, beside the script in a scratch copy.
TASKS_FILE = next((p for p in (os.path.join(HERE, "tasks.json"), os.path.join(os.path.dirname(HERE), "tasks.json")) if os.path.exists(p)), None)
if TASKS_FILE is None:
    sys.exit("no tasks.json beside runner.py or one folder up")
TASKS ={t["id"]: t for t in json.load(open(TASKS_FILE, encoding="utf-8"))["tasks"]}


def git(*args, cwd=BASE, check=True):
    return subprocess.run(["git", *args], cwd=cwd, check=check, capture_output=True, text=True, encoding="utf-8", errors="replace")


API_KEYS = ("ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "CODEX_API_KEY")


def base_env():
    # No arm may bill per token: a key in the caller's environment never reaches an agent.
    env = {k: v for k, v in os.environ.items() if k.upper() not in API_KEYS}
    env["TMP"] = env["TEMP"] = TMPDIR
    return env


def write_json(path, obj):
    # Through a temporary file of its own, so a stopped run or a second writer never leaves half a record.
    import tempfile
    fd, tmp = tempfile.mkstemp(dir=os.path.dirname(path), prefix=os.path.basename(path) + ".", suffix=".tmp")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(obj, f, indent=1)
        for attempt in range(50):
            try:
                os.replace(tmp, path)
                break
            except PermissionError:  # Windows: another writer holds the target for a moment
                if attempt == 49:
                    raise
                time.sleep(0.05)
    except BaseException:
        if os.path.exists(tmp):
            os.remove(tmp)
        raise


def direct_env():
    # The desktop app's own session variables are not a terminal's; hrn's allowlist drops them too.
    env = base_env()
    for k in list(env):
        ku = k.upper()
        if (ku.startswith(("CLAUDE_CODE_", "CLAUDE_AGENT_SDK", "CLAUDE_PREVIEW"))
                or ku in ("CLAUDECODE", "CLAUDE_EFFORT", "CLAUDE_PID", "ANTHROPIC_BASE_URL")):
            del env[k]
    return env


def run_proc(cmd, cwd, env, stdin_text, stdout_path, stderr_path, timeout):
    with open(stdout_path, "wb") as so, open(stderr_path, "wb") as se:
        t0 = time.monotonic()
        p = subprocess.Popen(cmd, cwd=cwd, env=env, stdin=subprocess.PIPE, stdout=so, stderr=se)
        try:
            p.stdin.write(stdin_text.encode("utf-8"))
            p.stdin.close()
        except OSError:
            pass
        timed_out = False
        try:
            p.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            subprocess.run(["taskkill", "/F", "/T", "/PID", str(p.pid)], capture_output=True)
            p.wait()
        return p.returncode, time.monotonic() - t0, timed_out


def last_json_line(path):
    last = None
    with open(path, encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if line.startswith("{"):
                try:
                    last = json.loads(line)
                except json.JSONDecodeError:
                    pass
    return last


def parse_hrn(stdout_path, stderr_path, cell):
    tail = last_json_line(stdout_path)
    run_id = tail.get("run_id") if isinstance(tail, dict) else None
    cell["hrn_run_id"] = run_id
    stderr = open(stderr_path, encoding="utf-8", errors="replace").read()
    cell["stderr_lines"] = [l for l in stderr.splitlines() if l.strip()][:20]
    if not run_id:
        cell["notes"].append("hrn printed no run_id")
        return
    rec = os.path.join(HRN_HOME, "runs", run_id)
    try:
        res = json.load(open(os.path.join(rec, "result.json"), encoding="utf-8"))
    except OSError as e:
        cell["notes"].append(f"no result.json: {e}")
        return
    r = res.get("result", res)
    cell["status"] = r.get("status")
    cell["error"] = (r.get("error") or "")[:300]
    cell["turns"] = r.get("turns")
    cell["tool_calls"] = r.get("tool_calls")
    cell["model"] = r.get("model")
    cell["tokens"] = {
        "input": r.get("input_tokens", 0),
        "cache_read": r.get("cache_read_tokens", 0),
        "cache_write": r.get("cache_write_tokens", 0),
        "output": r.get("output_tokens", 0),
    }
    cell["usage_status"] = (r.get("usage_metadata") or {}).get("status")
    cell["cost_usd"] = r.get("cost_usd")
    # hrn leaves the flag out when the cost is the provider's own figure.
    cell["cost_estimate"] = bool(r.get("cost_is_estimate")) if r.get("cost_usd") is not None else None
    cell["agent_duration_s"] = round((r.get("duration_ns") or 0) / 1e9, 1)
    cell["refusal_flags"] = len(r.get("refusal_flags") or [])
    cell["final"] = (r.get("final") or "")[:500]
    # Guard refusals and info lines hrn added, from the event log.
    kinds = {}
    refusals = 0
    try:
        with open(os.path.join(rec, "events.jsonl"), encoding="utf-8", errors="replace") as f:
            for line in f:
                try:
                    ev = json.loads(line)
                except json.JSONDecodeError:
                    continue
                k = ev.get("kind")
                kinds[k] = kinds.get(k, 0) + 1
                text = (ev.get("text") or "")
                if k == "tool_result" and text.lstrip().startswith("blocked by the hrn guard"):
                    refusals += 1
    except OSError:
        pass
    cell["event_kinds"] = kinds
    cell["guard_refusals_seen"] = refusals


def parse_direct_claude(stdout_path, stderr_path, cell):
    """Read claude's stream-json (or a single json result): totals from the result line when the run
    finished, else the per-message usage summed once per message id, as hrn does."""
    stderr = open(stderr_path, encoding="utf-8", errors="replace").read()
    cell["stderr_lines"] = [l for l in stderr.splitlines() if l.strip()][:20]
    result = None
    per_message = {}
    models = set()
    with open(stdout_path, encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                continue
            if ev.get("type") == "result":
                result = ev
            elif ev.get("type") == "assistant":
                msg = ev.get("message") or {}
                u = msg.get("usage") or {}
                mid = msg.get("id") or f"anon{len(per_message)}"
                if u:
                    prev = per_message.get(mid, {})
                    per_message[mid] = {k: max(prev.get(k, 0), u.get(k, 0) or 0) for k in
                                        ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "output_tokens")}
                if msg.get("model"):
                    models.add(msg["model"])
    if result is None:
        # A killed run leaves no result line; keep what the messages reported.
        cell["status"] = cell.get("status") or "error"
        cell["tokens"] = {
            "input": sum(u["input_tokens"] for u in per_message.values()),
            "cache_read": sum(u["cache_read_input_tokens"] for u in per_message.values()),
            "cache_write": sum(u["cache_creation_input_tokens"] for u in per_message.values()),
            "output": sum(u["output_tokens"] for u in per_message.values()),
        }
        cell["tokens_source"] = "per-message usage (no result line)"
        cell["turns"] = len(per_message)
        cell["model"] = ",".join(sorted(models)) or None
        cell["cost_usd"] = None
        if not per_message:
            cell["notes"].append("claude printed no JSON result and no messages")
        return
    obj = result
    cell["status"] = "error" if obj.get("is_error") else "ok"
    if obj.get("subtype") and obj.get("subtype") != "success":
        cell["error"] = str(obj.get("subtype"))[:300]
    if obj.get("is_error"):
        cell["error"] = (cell.get("error", "") + " " + str(obj.get("result", ""))[:300]).strip()
    cell["turns"] = obj.get("num_turns")
    u = obj.get("usage") or {}
    cell["tokens"] = {
        "input": u.get("input_tokens", 0),
        "cache_read": u.get("cache_read_input_tokens", 0),
        "cache_write": u.get("cache_creation_input_tokens", 0),
        "output": u.get("output_tokens", 0),
    }
    cell["tokens_source"] = "result line"
    cell["cost_usd"] = obj.get("total_cost_usd")
    cell["cost_estimate"] = False
    cell["agent_duration_s"] = round((obj.get("duration_ms") or 0) / 1000, 1)
    mu = obj.get("modelUsage") or {}
    cell["model"] = ",".join(sorted(mu.keys())) or ",".join(sorted(models)) or None
    cell["final"] = str(obj.get("result") or "")[:500]
    cell["session"] = obj.get("session_id")


def parse_direct_codex(stdout_path, stderr_path, cell):
    stderr = open(stderr_path, encoding="utf-8", errors="replace").read()
    cell["stderr_lines"] = [l for l in stderr.splitlines() if l.strip()][:20]
    tokens = {"input": 0, "cache_read": 0, "cache_write": 0, "output": 0}
    turns = 0
    tool_calls = 0
    errors = []
    final = ""
    thread = None
    usage_seen = False
    with open(stdout_path, encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                continue
            t = ev.get("type")
            if t == "thread.started":
                thread = ev.get("thread_id")
            elif t == "turn.completed":
                turns += 1
                u = ev.get("usage") or {}
                if u:
                    usage_seen = True
                tokens["input"] += u.get("input_tokens", 0) or 0
                tokens["cache_read"] += u.get("cached_input_tokens", 0) or 0
                tokens["cache_write"] += u.get("cache_write_input_tokens", 0) or 0
                tokens["output"] += u.get("output_tokens", 0) or 0
            elif t == "item.completed":
                item = ev.get("item") or {}
                if item.get("type") == "command_execution":
                    tool_calls += 1
                elif item.get("type") == "agent_message":
                    final = item.get("text") or final
            elif t in ("error", "turn.failed"):
                msg = ev.get("message") or (ev.get("error") or {}).get("message") or t
                errors.append(msg)
    cell["tokens"] = tokens
    cell["usage_status"] = "reported" if usage_seen else "missing"
    cell["turns"] = turns
    cell["tool_calls"] = tool_calls
    cell["final"] = final[:500]
    cell["session"] = thread
    real_errors = [e for e in errors if not e.startswith("Reconnecting...")]
    if real_errors:
        cell["error"] = "; ".join(real_errors)[:300]
    cell["status"] = "error" if (real_errors or turns == 0) else "ok"
    cell["cost_usd"] = None
    cell["notes"].append("codex exec reports no cost")


def worktree_add(task, path):
    base = git("rev-parse", task["merge"] + "^1").stdout.strip()
    if os.path.exists(path):
        git("worktree", "remove", "--force", path, check=False)
        shutil.rmtree(path, ignore_errors=True)
    git("worktree", "add", "--detach", path, base)
    return base


def worktree_remove(path):
    git("worktree", "remove", "--force", path, check=False)
    if os.path.exists(path):
        shutil.rmtree(path, ignore_errors=True)


def collect_changes(task, wt, base, cell, patch_path):
    head = git("rev-parse", "HEAD", cwd=wt).stdout.strip()
    cell["agent_committed"] = head != base
    st = git("status", "--porcelain", "-uall", cwd=wt).stdout
    cell["changed_files"] = sorted(l[3:].strip() for l in st.splitlines() if l.strip())
    git("add", "-A", cwd=wt, check=False)
    diff = subprocess.run(["git", "diff", "--cached", base], cwd=wt, capture_output=True)
    with open(patch_path, "wb") as f:
        f.write(diff.stdout)
    cell["patch_bytes"] = len(diff.stdout)
    git("reset", "-q", cwd=wt, check=False)
    if cell["agent_committed"]:
        cell["changed_files"] = sorted(set(cell["changed_files"]) | set(
            git("diff", "--name-only", base, head, cwd=wt).stdout.split()))


def gate(task, wt, log_path):
    for f in task["hidden_tests"]:
        content = subprocess.run(["git", "show", f"{task['merge']}:{f}"], cwd=BASE, capture_output=True, check=True).stdout
        with open(os.path.join(wt, f), "wb") as out:
            out.write(content)
    cmd = f"go build ./... && go test -count=1 ./{task['pkg']}/"
    t0 = time.monotonic()
    with open(log_path, "wb") as log:
        try:
            p = subprocess.run([BASH, "-c", cmd], cwd=wt, env=base_env(), stdout=log, stderr=subprocess.STDOUT, timeout=GATE_TIMEOUT)
            rc = p.returncode
        except subprocess.TimeoutExpired:
            rc = -1
            log.write(b"\ngate timed out\n")
    tail = open(log_path, encoding="utf-8", errors="replace").read().splitlines()[-6:]
    return rc == 0, round(time.monotonic() - t0, 1), tail


def run_cell(task_id, backend, arm, repeat):
    task = TASKS[task_id]
    cell_id = f"{task_id}.{backend}.{arm}.r{repeat}"
    os.makedirs(CELLS, exist_ok=True)
    os.makedirs(RAW, exist_ok=True)
    os.makedirs(WT, exist_ok=True)
    os.makedirs(TMPDIR, exist_ok=True)
    cell_path = os.path.join(CELLS, cell_id + ".json")
    if os.path.exists(cell_path):
        print(f"{cell_id}: done already", flush=True)
        return json.load(open(cell_path, encoding="utf-8"))
    wt = os.path.join(WT, cell_id)
    base = worktree_add(task, wt)
    cell = {
        "cell": cell_id, "task": task_id, "backend": backend, "arm": arm, "repeat": repeat,
        "base": base, "started": datetime.datetime.now().astimezone().isoformat(timespec="seconds"),
        "notes": [], "error": "", "tokens": None, "status": None,
    }
    stdout_path = os.path.join(RAW, cell_id + ".stdout")
    stderr_path = os.path.join(RAW, cell_id + ".stderr")
    if arm == "hrn":
        cmd = [HRN, "run", "-b", backend, "--mode", "yolo", "--dir", wt, "--timeout", "15m", "--no-switch", "--json", "-"]
        env = base_env()
    elif backend == "claude":
        # stream-json, as hrn reads it: a killed run still leaves its per-message usage.
        cmd = [CLAUDE, "-p", "--output-format", "stream-json", "--verbose", "--permission-mode", "bypassPermissions"]
        env = direct_env()
    else:
        cmd = [CODEX, "exec", "--json", "--skip-git-repo-check", "-C", wt, "--dangerously-bypass-approvals-and-sandbox", "-"]
        env = direct_env()
    cell["command"] = [c.replace(wt, "<wt>") for c in cmd]
    print(f"{cell_id}: start {cell['started']}", flush=True)
    rc, wall, timed_out = run_proc(cmd, wt, env, task["prompt"], stdout_path, stderr_path, AGENT_TIMEOUT + (120 if arm == "hrn" else 0))
    cell["exit_code"] = rc
    cell["wall_s"] = round(wall, 1)
    cell["timed_out"] = timed_out
    try:
        if arm == "hrn":
            parse_hrn(stdout_path, stderr_path, cell)
        elif backend == "claude":
            parse_direct_claude(stdout_path, stderr_path, cell)
        else:
            parse_direct_codex(stdout_path, stderr_path, cell)
    except Exception as e:  # keep the cell; the raw output stays for a second look
        cell["notes"].append(f"parse failed: {e!r}")
    if timed_out:
        cell["status"] = "timeout"
    if cell["status"] is None:
        cell["status"] = "ok" if rc == 0 else "error"
    collect_changes(task, wt, base, cell, os.path.join(RAW, cell_id + ".patch"))
    passed, gate_s, tail = gate(task, wt, os.path.join(RAW, cell_id + ".gate.log"))
    cell["gate_pass"] = passed
    cell["gate_s"] = gate_s
    cell["gate_tail"] = tail
    cell["finished"] = datetime.datetime.now().astimezone().isoformat(timespec="seconds")
    worktree_remove(wt)
    write_json(cell_path, cell)
    print(f"{cell_id}: status={cell['status']} gate={'pass' if passed else 'FAIL'} wall={cell['wall_s']}s "
          f"tokens={cell['tokens']} model={cell.get('model')} err={cell['error'][:80]!r}", flush=True)
    return cell


def limited(cell):
    text = (cell.get("error") or "").lower()
    return cell.get("status") == "limited" or ("usage limit" in text or "rate limit" in text or "hit your" in text)


PLAN_FLOOR = {"claude": 1, "codex": 10}  # % of the week to leave; the maintainer set claude's to 1


def plan_left(backend):
    out = subprocess.run([HRN, "backends"], capture_output=True, text=True, encoding="utf-8", errors="replace").stdout
    import re
    m = re.search(backend + r"\s+enabled[^\n]*\n\s+week: (\d+)% left", out)
    return int(m.group(1)) if m else None


def lane(backend, task_ids):
    os.makedirs(OUT, exist_ok=True)
    # One lane per backend: two would share worktrees and records.
    lock = os.path.join(OUT, f"lane-{backend}.lock")
    try:
        fd = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    except FileExistsError:
        sys.exit(f"{backend} lane: {lock} exists; another lane is running, or one was killed (then delete the file)")
    try:
        try:
            os.write(fd, str(os.getpid()).encode())
        finally:
            os.close(fd)
        run_lane(backend, task_ids)
    finally:
        os.remove(lock)


def run_lane(backend, task_ids):
    left = plan_left(backend)
    print(f"{backend} lane: {left}% of the week left", flush=True)
    if left is not None and left < PLAN_FLOOR[backend]:
        print(f"{backend} lane: below the floor of {PLAN_FLOOR[backend]}%; not starting", flush=True)
        return
    with open(os.path.join(OUT, f"backends-{backend}-start.txt"), "w", encoding="utf-8") as f:
        f.write(subprocess.run([HRN, "backends"], capture_output=True, text=True, encoding="utf-8", errors="replace").stdout)
    order = [(1, "hrn"), (1, "direct"), (2, "direct"), (2, "hrn")]
    for task_id in task_ids:
        for repeat, arm in order:
            cell = run_cell(task_id, backend, arm, repeat)
            if limited(cell):
                print(f"{backend} lane: plan limit reached at {cell['cell']}; stopping the lane", flush=True)
                return
            left = plan_left(backend)
            print(f"{backend} lane: {left}% of the week left after {cell['cell']}", flush=True)
            if left is not None and left < PLAN_FLOOR[backend]:
                print(f"{backend} lane: below the floor of {PLAN_FLOOR[backend]}%; stopping the lane", flush=True)
                return
    with open(os.path.join(OUT, f"backends-{backend}-end.txt"), "w", encoding="utf-8") as f:
        f.write(subprocess.run([HRN, "backends"], capture_output=True, text=True, encoding="utf-8", errors="replace").stdout)
    print(f"{backend} lane: done", flush=True)


def recount(cells_dir):
    """Recount guard refusals in recorded hrn cells with the exact refusal text (older cells used a loose match)."""
    import glob
    for p in glob.glob(os.path.join(cells_dir, "*.hrn.*.json")):
        c = json.load(open(p, encoding="utf-8"))
        rid = c.get("hrn_run_id")
        if not rid:
            continue
        n = 0
        try:
            for line in open(os.path.join(HRN_HOME, "runs", rid, "events.jsonl"), encoding="utf-8", errors="replace"):
                try:
                    ev = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if ev.get("kind") == "tool_result" and (ev.get("text") or "").lstrip().startswith("blocked by the hrn guard"):
                    n += 1
        except OSError:
            continue
        changed = c.get("guard_refusals_seen") != n
        c["guard_refusals_seen"] = n
        t = c.get("tokens") or {}
        if not any(t.values()) and c.get("tokens_source") is None:
            tot = {"input": 0, "cache_read": 0, "cache_write": 0, "output": 0}
            calls = 0
            for line in open(os.path.join(HRN_HOME, "runs", rid, "events.jsonl"), encoding="utf-8", errors="replace"):
                try:
                    ev = json.loads(line)
                except json.JSONDecodeError:
                    continue
                u = ev.get("usage") if ev.get("kind") == "usage" else None
                if not u:
                    continue
                calls += 1
                tot["input"] += u.get("input_tokens", 0) or 0
                tot["cache_read"] += u.get("cache_read_tokens", 0) or 0
                tot["cache_write"] += u.get("cache_write_tokens", 0) or 0
                tot["output"] += u.get("output_tokens", 0) or 0
            if calls:
                c["tokens"] = tot
                c["turns"] = c.get("turns") or calls
                c["tokens_source"] = f"usage events ({calls} model calls; the run was killed before totals)"
                changed = True
        if changed:
            write_json(p, c)
            print(p, "guard_refusals_seen", n, "tokens", c.get("tokens"))


def codex_models(cells_dir):
    """Fill the model of direct codex cells from codex's rollout file, as hrn does for its own runs."""
    import glob
    home = os.environ.get("CODEX_HOME") or os.path.join(os.path.expanduser("~"), ".codex")
    for p in glob.glob(os.path.join(cells_dir, "*.codex.direct.*.json")):
        c = json.load(open(p, encoding="utf-8"))
        session = c.get("session")
        if c.get("model") or not session:
            continue
        files = glob.glob(os.path.join(home, "sessions", "*", "*", "*", f"rollout-*-{session}.jsonl"))
        if not files:
            continue
        models, version = rollout_models(files[0])
        if models:
            c["model"] = models.most_common(1)[0][0]
            if len(models) > 1:
                c["notes"].append("codex named several models in its turns: " + ", ".join(sorted(models)))
        if version:
            c["cli_version"] = version
        if models or version:
            write_json(p, c)
            print(p, "model ->", c.get("model"), "cli ->", c.get("cli_version"))


def rollout_models(path):
    """The model of each turn (turn_context) and the CLI version (session_meta) in a codex rollout file."""
    from collections import Counter
    models, version = Counter(), None
    with open(path, encoding="utf-8", errors="replace") as f:
        for line in f:
            try:
                event = json.loads(line)
            except ValueError:
                continue
            payload = event.get("payload") if isinstance(event, dict) else None
            if not isinstance(payload, dict):
                continue
            if event.get("type") == "turn_context" and payload.get("model"):
                models[payload["model"]] += 1
            if event.get("type") == "session_meta" and payload.get("cli_version"):
                version = payload["cli_version"]
    return models, version


if __name__ == "__main__":
    if sys.argv[1] == "cell":
        run_cell(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]))
    elif sys.argv[1] == "lane":
        ids = sys.argv[3:] or list(TASKS)
        lane(sys.argv[2], ids)
    elif sys.argv[1] == "recount":
        recount(sys.argv[2] if len(sys.argv) > 2 else CELLS)
        codex_models(sys.argv[2] if len(sys.argv) > 2 else CELLS)
    else:
        sys.exit(__doc__)
