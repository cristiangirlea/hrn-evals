#!/usr/bin/env bash
# Check each task's gate three ways. At the commit before the fix it must pass with the tests
# that commit has, and fail once the merged fix's test files are copied in: the copied tests are
# then the only difference, so they are the reason. At the merge, with the same files, it must pass.
#
#   validate-gates.sh [tasks.json]
#
# HRN_VS_DIRECT_WORK holds the harness clone (harness/) and gets the output (validate/);
# HRN_VS_DIRECT_TMP is the temp folder. A setup error stops the script with exit 2; a gate that
# does not behave as above makes it end with exit 1.
set -u -o pipefail
# The gates never need a key, and a test must not be able to bill one.
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY CODEX_API_KEY
HERE=$(cd "$(dirname "$0")" && pwd)
TASKS=${1:-$HERE/../tasks.json}
[ -f "$TASKS" ] || { echo "no task list at $TASKS"; exit 2; }
# Absolute before the cd below, so a relative path means the caller's folder.
TASKS=$(cd "$(dirname "$TASKS")" && pwd)/$(basename "$TASKS")
WORK=${HRN_VS_DIRECT_WORK:-/e/hrn-vs-direct}
WORK=$(cd "$WORK" 2>/dev/null && pwd) || { echo "no work folder at ${HRN_VS_DIRECT_WORK:-/e/hrn-vs-direct}"; exit 2; }
export TMP=${HRN_VS_DIRECT_TMP:-'E:\hrn-vd-tmp'}
export TEMP=$TMP
mkdir -p "$(cygpath -u "$TMP" 2>/dev/null || echo "$TMP")" || { echo "cannot make the temp folder $TMP"; exit 2; }
REPO=$WORK/harness
OUT=$WORK/validate
mkdir -p "$OUT" || exit 2
cd "$REPO" || { echo "no harness clone at $REPO"; exit 2; }

# id issue pr merge pkg testfile...   Paths must stay inside the repository.
list=$(python -c '
import json, re, sys
for t in json.load(open(sys.argv[1], encoding="utf-8"))["tasks"]:
    paths = [t["pkg"], *t["hidden_tests"]]
    for p in paths:
        if not re.fullmatch(r"[A-Za-z0-9_.+-]+(/[A-Za-z0-9_.+-]+)*", p) or ".." in p.split("/"):
            sys.exit("%s: not a plain path inside the repository: %r" % (t["id"], p))
    if not re.fullmatch(r"[0-9a-f]{40}", t["merge"]) or not t["hidden_tests"]:
        sys.exit("%s: needs a full merge commit and at least one test file" % t["id"])
    print(t["id"], int(t["issue"]), int(t["pr"]), t["merge"], *paths)
' "$TASKS" | tr -d '\r') || { echo "cannot use every task in $TASKS"; exit 2; }
[ -n "$list" ] || { echo "no tasks in $TASKS"; exit 2; }

wt=
cleanup() { [ -n "$wt" ] && git worktree remove --force "$wt" 2>/dev/null; wt=; }
trap cleanup EXIT

# What the log shows, for the reader only; the verdict never depends on it.
shown() {
  local log=$1 out
  out=$(grep -E '^--- FAIL: ' "$log" | sed -E 's/^--- FAIL: ([^ ]+).*/\1 fails/' | sed -n '1,3p' | paste -sd ';' -)
  [ -n "$out" ] || out=$(tr '\\' '/' < "$log" | grep -E '^[^ ]+\.go:[0-9]+:[0-9]+: ' | sed -n 1p | sed 's/^/does not build: /')
  echo "${out:-no test named in the output}"
}

bad=0 n=0
while read -r id issue pr merge pkg tests; do
  [ -z "$id" ] && continue
  n=$((n + 1))
  base=$(git rev-parse --verify -q "$merge^1") || { echo "$id: no commit $merge^1"; exit 2; }
  for f in $tests; do
    [ "$(git cat-file -t "$merge:$f" 2>/dev/null)" = blob ] || { echo "$id: no file $f at $merge"; exit 2; }
  done
  for side in own base fix; do
    ref=$base; [ $side = fix ] && ref=$merge
    path="$OUT/wt-$pr-$side"
    git worktree remove --force "$path" 2>/dev/null
    git worktree add -q --detach "$path" "$ref" || { echo "$id $side: worktree failed"; exit 2; }
    wt=$path
    root=$(realpath "$wt")
    if [ $side != own ]; then
      for f in $tests; do
        # The file lands inside the worktree, also when a folder on the way is a link.
        dir=$(realpath -m "$wt/$(dirname "$f")")
        case "$dir/" in "$root"/*) ;; *) echo "$id $side: $f leaves the worktree"; exit 2 ;; esac
        [ ! -d "$dir/$(basename "$f")" ] || { echo "$id $side: $f is a folder at $ref"; exit 2; }
        mkdir -p "$dir" && git show "$merge:$f" > "$dir/$(basename "$f").new" && mv -f "$dir/$(basename "$f").new" "$dir/$(basename "$f")" \
          || { echo "$id $side: cannot copy $f from $merge"; exit 2; }
      done
    fi
    log="$OUT/$pr-$side.log"
    # A log that cannot be written must not let an older one stand in for this run.
    rm -f "$log" 2>/dev/null; : > "$log" || { echo "$id $side: cannot write $log"; exit 2; }
    start=$(date +%s)
    # The build is judged apart from the tests: a build that fails says nothing about the gate.
    # Stdin is closed to both: the loop's own input is the task list.
    ( cd "$wt" && go build ./... ) >> "$log" 2>&1 < /dev/null
    built=$?
    rc=1
    [ $built -ne 0 ] || { ( cd "$wt" && go test -count=1 "./$pkg/" ) >> "$log" 2>&1 < /dev/null; rc=$?; }
    secs=$(( $(date +%s) - start ))
    git worktree remove --force "$wt" || { echo "$id $side: cannot remove the worktree"; exit 2; }
    wt=
    if [ $built -ne 0 ]; then
      echo "task=$id issue=$issue pr=$pr side=$side rc=$built secs=$secs base=${base:0:7} UNEXPECTED: go build fails, so the tests did not run ($(shown "$log"))"
      bad=1
      continue
    fi
    case $side in
      own)  why="passes with its own tests"
            [ $rc -eq 0 ] || { why="UNEXPECTED: fails before the fix without the copied tests ($(shown "$log"))"; bad=1; } ;;
      base) why=$(shown "$log")
            [ $rc -ne 0 ] || { why="UNEXPECTED: passes before the fix"; bad=1; } ;;
      fix)  why=passes
            [ $rc -eq 0 ] || { why="UNEXPECTED: fails at the merge ($(shown "$log"))"; bad=1; } ;;
    esac
    echo "task=$id issue=$issue pr=$pr side=$side rc=$rc secs=$secs base=${base:0:7} $why"
  done
done <<LIST
$list
LIST
[ $bad -eq 0 ] || { echo "FAILED: a gate did not behave as expected ($n tasks)"; exit 1; }
echo "DONE: $n gates pass before their fix, fail there with the copied tests, and pass at the merge"
