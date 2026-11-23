"""Offline checks of the published runner. Starts no agent.

Run from the scripts folder with HRN_VS_DIRECT_WORK naming a new or empty scratch folder and a
fake value in each of the four key variables (ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN,
OPENAI_API_KEY, CODEX_API_KEY). It refuses any other folder, so it can never write into a real
run's output.
"""
import json
import os
import subprocess
import sys
import threading

work = os.environ.get("HRN_VS_DIRECT_WORK")
if not work or (os.path.exists(work) and os.listdir(work)):
    sys.exit("set HRN_VS_DIRECT_WORK to a new or empty scratch folder")
os.makedirs(work, exist_ok=True)

import runner  # noqa: E402  (after the folder check, since importing it fixes its paths)

ok = True


def check(name, cond, detail=""):
    global ok
    ok = ok and bool(cond)
    print(("pass  " if cond else "FAIL  ") + name + (f"  [{detail}]" if detail and not cond else ""))


def no_keys(env):
    return env is not None and not [k for k in env if k.upper() in runner.API_KEYS]


os.makedirs(runner.OUT, exist_ok=True)
# Nothing here depends on an installed CLI or on the default temp folder.
runner.HRN, runner.CLAUDE, runner.CODEX = "hrn-stub", "claude-stub", "codex-stub"
runner.TMPDIR = os.path.join(work, "tmp")

# 1. No key reaches an agent launch or a gate, whatever the case of its name.
names = [k for k in os.environ if k.upper() in runner.API_KEYS]
check("the test set all four key names", len({k.upper() for k in names}) == 4, str(names))
for fn in (runner.base_env, runner.direct_env):
    check(f"{fn.__name__} drops every key", no_keys(fn()) and fn()["TMP"] == runner.TMPDIR)

# run_cell picks the environment for each arm: catch it where the process would start.
launched = []
real = {n: getattr(runner, n) for n in ("worktree_add", "worktree_remove", "run_proc", "parse_hrn", "parse_direct_claude",
                                        "parse_direct_codex", "collect_changes", "gate")}
runner.worktree_add = lambda task, wt: "0" * 40
runner.worktree_remove = lambda wt: None
runner.run_proc = lambda cmd, cwd, env, *a: (launched.append(env), (0, 1.0, False))[1]
runner.parse_hrn = runner.parse_direct_claude = runner.parse_direct_codex = lambda so, se, cell: cell.update(status="ok")
runner.collect_changes = lambda task, wt, base, cell, patch: cell.update(changed_files=[], agent_committed=False)
runner.gate = lambda task, wt, log: (True, 0.1, [])
task_id = next(iter(runner.TASKS))
for backend, arm in (("claude", "hrn"), ("claude", "direct"), ("codex", "direct")):
    runner.run_cell(task_id, backend, arm, 9)
check("run_cell starts all three kinds of agent without a key", len(launched) == 3 and all(no_keys(e) for e in launched))
for n, f in real.items():
    setattr(runner, n, f)

# gate() starts the gate's shell: catch the environment it hands over.
gate_envs = []
real_run = subprocess.run


class Done:
    stdout, returncode = b"", 0


def fake_run(cmd, **kw):
    if cmd and cmd[0] == runner.BASH:
        gate_envs.append(kw.get("env"))
    return Done()


task = runner.TASKS[task_id]
wt = os.path.join(runner.OUT, "gate-wt")
for f in task["hidden_tests"]:
    os.makedirs(os.path.dirname(os.path.join(wt, f)), exist_ok=True)
subprocess.run = fake_run
try:
    runner.gate(task, wt, os.path.join(runner.OUT, "gate.log"))
finally:
    subprocess.run = real_run
check("gate starts its shell without a key", len(gate_envs) == 1 and no_keys(gate_envs[0]))

# 2. Records: two writers at once leave one whole record, no error and no temporary file.
path = os.path.join(runner.OUT, "cell.json")
big = {"k": ["x" * 200] * 2000}
errors = []


def writer(i):
    try:
        for _ in range(40):
            runner.write_json(path, dict(big, who=i))
    except Exception as e:
        errors.append(repr(e))


threads = [threading.Thread(target=writer, args=(i,)) for i in range(2)]
[t.start() for t in threads]
[t.join() for t in threads]
whole = json.load(open(path, encoding="utf-8"))
check("two writers leave a whole record", whole["who"] in (0, 1) and len(whole["k"]) == 2000)
check("neither writer failed", not errors, str(errors))


def temps():
    return [f for f in os.listdir(runner.OUT) if f.endswith(".tmp")]


check("no temporary file is left", not temps(), str(temps()))

real_replace, real_sleep, calls = os.replace, runner.time.sleep, []
runner.time.sleep = lambda s: None


def refuse_once(a, b):
    calls.append(1)
    if len(calls) == 1:
        raise PermissionError("held")
    return real_replace(a, b)


os.replace = refuse_once
try:
    runner.write_json(path, {"after": "one refusal"})
finally:
    os.replace = real_replace
check("a replace refused once is tried again", json.load(open(path)) == {"after": "one refusal"} and len(calls) == 2)


def refuse_always(a, b):
    raise PermissionError("held")


os.replace = refuse_always
try:
    runner.write_json(path, {"never": "written"})
    check("a replace refused every time fails", False)
except PermissionError:
    check("a replace refused every time fails", True)
finally:
    os.replace, runner.time.sleep = real_replace, real_sleep
check("and keeps the earlier record", json.load(open(path)) == {"after": "one refusal"})
check("and leaves no temporary file", not temps(), str(temps()))

# 3. hrn's estimate flag.
home = os.path.join(runner.OUT, "hrnhome")
runner.HRN_HOME = home
for rid, res, want in (("r-true", {"cost_usd": 0.5, "cost_is_estimate": True}, True),
                       ("r-false", {"cost_usd": 0.5, "cost_is_estimate": False}, False),
                       ("r-absent", {"cost_usd": 0.5}, False),
                       ("r-nocost", {}, None)):
    os.makedirs(os.path.join(home, "runs", rid), exist_ok=True)
    json.dump({"result": dict(res, status="ok")}, open(os.path.join(home, "runs", rid, "result.json"), "w"))
    so, se = os.path.join(runner.OUT, rid + ".out"), os.path.join(runner.OUT, rid + ".err")
    open(so, "w").write(json.dumps({"run_id": rid}) + "\n")
    open(se, "w").write("")
    cell = {"notes": [], "error": ""}
    try:
        runner.parse_hrn(so, se, cell)
    except OSError:
        pass  # no event log in the made-up record; the fields above it are set first
    check(f"cost_estimate for {rid} is {want}", cell.get("cost_estimate") is want, repr(cell.get("cost_estimate")))

# 4. A direct codex run's model and version, through codex_models.
codex_home = os.path.join(runner.OUT, "codexhome")
day = os.path.join(codex_home, "sessions", "2026", "11", "07")
os.makedirs(day, exist_ok=True)
os.environ["CODEX_HOME"] = codex_home
cells = os.path.join(runner.OUT, "model-cells")
os.makedirs(cells, exist_ok=True)


def rollout(session, events):
    open(os.path.join(day, f"rollout-2026-11-07T00-00-00-{session}.jsonl"), "w", encoding="utf-8").write(
        "\n".join(json.dumps(e) for e in events) + "\nnot json\n42\n")


def cell_for(name, session):
    p = os.path.join(cells, f"{name}.codex.direct.r1.json")
    json.dump({"cell": name, "session": session, "notes": []}, open(p, "w"))
    return p


meta = {"type": "session_meta", "payload": {"cli_version": "9.9.9"}}
decoy = {"type": "response_item", "payload": {"model": "decoy-field", "item": {"model": "decoy-nested"}}}
turn = lambda m: {"type": "turn_context", "payload": {"model": m}}  # noqa: E731
rollout("s-one", [meta, turn("m-one"), decoy, decoy, decoy, turn("m-one")])  # the decoys outnumber the turns
rollout("s-two", [meta, turn("m-one"), turn("m-two"), turn("m-two")])
rollout("s-meta", [meta, decoy])
p1, p2, p3 = cell_for("one", "s-one"), cell_for("two", "s-two"), cell_for("meta", "s-meta")
runner.codex_models(cells)
c1, c2, c3 = (json.load(open(p)) for p in (p1, p2, p3))
check("one model: taken from the turns, not from a message", c1.get("model") == "m-one" and c1.get("cli_version") == "9.9.9" and not c1["notes"])
check("two models: the commoner one, with a note", c2.get("model") == "m-two" and any("m-one, m-two" in n for n in c2["notes"]), str(c2))
check("no turn: the version is kept and the model stays unset", c3.get("cli_version") == "9.9.9" and not c3.get("model"), str(c3))

# 5. One lane per backend; the lock goes when the lane ends, also when it fails early.
runner.plan_left = lambda b: 0  # under the floor: the lane returns before any agent starts
lock = os.path.join(runner.OUT, "lane-claude.lock")
runner.lane("claude", [])
check("the lock is removed when a lane ends", not os.path.exists(lock))
real_write = os.write


def failing_write(fd, data):
    raise OSError("disk full")


os.write = failing_write
try:
    runner.lane("claude", [])
    check("a failed lock write stops the lane", False)
except OSError:
    check("a failed lock write stops the lane", True)
finally:
    os.write = real_write
check("and releases the lock", not os.path.exists(lock))
open(lock, "w").write("1")
try:
    runner.lane("claude", [])
    check("a second lane is refused", False)
except SystemExit as e:
    check("a second lane is refused", "exists" in str(e))
check("and leaves the first lane's lock", os.path.exists(lock))
os.remove(lock)

# 6. The task list is found beside the script or one folder up, and its absence is said plainly.
import shutil  # noqa: E402

here = os.path.dirname(os.path.abspath(runner.__file__))
count = "import runner, sys; print(len(runner.TASKS))"
for name, place, want_rc, want in (("beside the script", "", 0, str(len(runner.TASKS))),
                                   ("one folder up", "..", 0, str(len(runner.TASKS))),
                                   ("nowhere", None, 1, "no tasks.json")):
    copy = os.path.join(work, "layout-" + name.replace(" ", "-"), "scripts")
    os.makedirs(copy)
    shutil.copyfile(os.path.join(here, "runner.py"), os.path.join(copy, "runner.py"))
    if place is not None:
        shutil.copyfile(runner.TASKS_FILE, os.path.normpath(os.path.join(copy, place, "tasks.json")))
    p = subprocess.run([sys.executable, "-c", count], cwd=copy, capture_output=True, text=True, env=runner.base_env())
    check(f"tasks.json {name}", p.returncode == want_rc and want in (p.stdout + p.stderr), repr((p.returncode, p.stdout, p.stderr[-200:])))

print("ALL PASS" if ok else "SOME FAILED")
sys.exit(0 if ok else 1)
