#!/usr/bin/env bash
# Validate each candidate's hidden test both ways: fails at the merge parent, passes at the merge.
set -u
export TMP='E:\hrn-vd-tmp' TEMP='E:\hrn-vd-tmp'
REPO=/e/hrn-vs-direct/harness
OUT=/e/hrn-vs-direct/validate
mkdir -p "$OUT"
cd "$REPO" || exit 1
# issue pr merge pkg testfiles...
while read -r issue pr merge pkg tests; do
  [ -z "$issue" ] && continue
  base=$(git rev-parse "$merge^1")
  for side in base fix; do
    ref=$base; [ $side = fix ] && ref=$merge
    wt="$OUT/wt-$pr-$side"
    git worktree remove --force "$wt" 2>/dev/null
    git worktree add -q --detach "$wt" "$ref" || { echo "$pr $side: worktree failed"; continue; }
    for f in $tests; do git show "$merge:$f" > "$wt/$f"; done
    start=$(date +%s)
    ( cd "$wt" && go build ./... && go test -count=1 "./$pkg/" ) > "$OUT/$pr-$side.log" 2>&1
    rc=$?
    echo "issue=$issue pr=$pr side=$side rc=$rc secs=$(( $(date +%s) - start )) base=${base:0:7}"
    git worktree remove --force "$wt"
  done
done <<'LIST'
211 268 9c39fca48e183a514b21cd66764e9db404c54a15 internal/tools internal/tools/guard_test.go
245 278 593e2a680e856fc92b8600701b15ad32aad6e07c internal/gc internal/gc/cache_test.go
228 259 3b8d5fa223b01b24f58ec3aeb1467c36265518cf internal/watch internal/watch/access_test.go
164 271 68cfd813c57b7af734c6f021a01166da492ecbc3 internal/remote internal/remote/client_test.go
237 363 3fbd99f51c21c610c76065c11027a887b741f5bb internal/watch internal/watch/paths_test.go
349 350 b7ba498900748e0051e0ee00aba4bab4e976f058 internal/agent internal/agent/session_test.go
149 300 1dde2c96d8b828a008fe5e85aad2d941e4968a27 internal/agent internal/agent/durable_test.go
429 532 e442134b8b7da59538ced77907f0e723eaa67165 internal/tools internal/tools/guard_test.go
607 639 7e8fd639028f377fb0fe26ad6f78d1e36174b9ba internal/agent internal/agent/lean_test.go
348 393 a11013959b439a13bf3f9c045c6e3d9e8d01fd22 internal/agent internal/agent/cli_run_test.go
648 658 0a2f881fd9744a1b8209eb556a2bc7085b89c3a6 scripts/privacy-check scripts/privacy-check/main_test.go
122 490 7bc9266140c6ac6dc34afdef35061b5aa35dd917 internal/watch internal/watch/purposes_test.go
LIST
echo DONE
