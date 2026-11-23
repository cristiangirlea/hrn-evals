#!/usr/bin/env bash
# Try validate-gates.sh on a made-up repository: two gates it must accept and the cases it must
# refuse. Needs git, go and python; starts no agent. Prints one line per case and ends with exit 0
# only when every case behaved.
#
#   test-validate-gates.sh <new empty folder>
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
[ $# -eq 1 ] || { echo "usage: $0 <new empty folder>"; exit 2; }
mkdir -p "$1" && [ -z "$(ls -A "$1")" ] || { echo "$1 is not empty"; exit 2; }
W=$(cd "$1" && pwd)
export HRN_VS_DIRECT_WORK=$W
# The temp folder does not exist yet: the validator has to make it.
HRN_VS_DIRECT_TMP=$(cygpath -w "$W/tmp" 2>/dev/null || echo "$W/tmp")
export HRN_VS_DIRECT_TMP
R=$W/harness
PKGS="good build other otherbuild same never swap"
mkdir -p "$R" && cd "$R" || exit 2
git init -q . && git config core.autocrlf false && git config user.name test && git config user.email test@example.invalid
printf 'module example.invalid/fake\n\ngo 1.21\n' > go.mod
old='import "testing"\n\nfunc TestOld(t *testing.T) {}\n'
check='import "testing"\n\nfunc %s(t *testing.T) {\n\tif Answer() != 2 {\n\t\tt.Fatal("wrong")\n\t}\n}\n'

# Before the fix: every package's function is wrong, and every package's own tests pass,
# except in the two packages made to be refused for that.
for p in $PKGS; do mkdir -p $p; printf 'package %s\n\nfunc Answer() int { return 1 }\n' $p > $p/code.go; done
# good: the copied test will fail before the fix. Its own test fails if a key variable reaches it.
printf 'package good\n\nimport (\n\t"os"\n\t"testing"\n)\n\nfunc TestOld(t *testing.T) {\n\tfor _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "CODEX_API_KEY"} {\n\t\tif os.Getenv(k) != "" {\n\t\t\tt.Fatal(k + " reached a gate")\n\t\t}\n\t}\n}\n' > good/a_test.go
printf "package build\n\n$old" > build/cache_test.go       # the copied test will not build before the fix
printf "package other\n\n$old" > other/a_test.go           # another file's test fails before the fix
printf "package other\n\n$check" TestElsewhere > other/b_test.go
printf "package otherbuild\n\n$old" > otherbuild/cache_test.go   # another file, named alike, does not build
printf 'package otherbuild\n\nimport "testing"\n\nfunc TestElsewhere(t *testing.T) { _ = helper() }\n' > otherbuild/other_cache_test.go
printf "package same\n\n$old" > same/a_test.go             # the copied test passes on both sides
printf "package never\n\n$old" > never/a_test.go           # the copied test fails at the merge too
mkdir -p swap/a_test.go && echo keep > swap/a_test.go/keep.txt   # a folder where the fix has a test file
git add -A && git commit -q -m base

for p in $PKGS; do printf 'package %s\n\nfunc Answer() int { return 2 }\n\nfunc helper() int { return 2 }\n' $p > $p/code.go; done
printf "package good\n\n$check" TestNew > good/a_test.go
printf 'package build\n\nimport "testing"\n\nfunc TestNew(t *testing.T) { _ = helper() }\n' > build/cache_test.go
# The copied file names the other file's test only inside a comment.
printf 'package other\n\nimport "testing"\n\n/*\nfunc TestElsewhere(t *testing.T) {}\n*/\n\nfunc TestCopied(t *testing.T) {}\n' > other/a_test.go
printf 'package never\n\nimport "testing"\n\nfunc TestNever(t *testing.T) { t.Fatal("always") }\n' > never/a_test.go
rm -rf swap/a_test.go && printf "package swap\n\n$check" TestNew > swap/a_test.go
git add -A && git commit -q -m fix
MERGE=$(git rev-parse HEAD)

task() { printf '{"id":"%s","issue":1,"pr":%s,"merge":"%s","pkg":"%s","hidden_tests":["%s"]}' "$1" "$2" "$MERGE" "$3" "$4"; }
list() { local out=$1; shift; local IFS=,; printf '{"tasks":[%s]}\n' "$*" > "$out"; }
GOOD=$(task good 1 good good/a_test.go)
list "$W/accept.json" "$GOOD" "$(task build 2 build build/cache_test.go)"
list "$W/other.json" "$(task other 3 other other/a_test.go)"
list "$W/otherbuild.json" "$(task otherbuild 4 otherbuild otherbuild/cache_test.go)"
list "$W/same.json" "$(task same 8 same same/a_test.go)"
list "$W/never.json" "$(task never 9 never never/a_test.go)"
list "$W/nofile.json" "$(task nofile 5 good good/missing_test.go)"
list "$W/escape.json" "$(task escape 10 good ../sentinel_test.go)"
list "$W/onlygood.json" "$GOOD"
list "$W/swap.json" "$(task swap 12 swap swap/a_test.go)"
# A folder listed beside a real test file.
printf '{"tasks":[{"id":"folder","issue":1,"pr":11,"merge":"%s","pkg":"good","hidden_tests":["good/a_test.go","good"]}]}\n' "$MERGE" > "$W/folder.json"
# A go that fails the build it is told to (by count) and is the real one otherwise.
mkdir -p "$W/bin" && REAL_GO=$(command -v go) && export REAL_GO COUNT="$W/builds"
cat > "$W/bin/go" <<'STUB'
#!/usr/bin/env bash
if [ "${1:-}" = build ] && [ -n "${FAIL_BUILD_AT:-}" ]; then
  n=$(( $(cat "$COUNT" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$COUNT"
  [ "$n" -ne "$FAIL_BUILD_AT" ] || { echo "stub: this build fails"; exit 1; }
fi
exec "$REAL_GO" "$@"
STUB
chmod +x "$W/bin/go"
list "$W/nocommit.json" "$(task nocommit 6 good good/a_test.go | sed "s/$MERGE/0000000000000000000000000000000000000000/")"
printf '{"tasks":[%s,{"id":"broken","issue":1,"pr":7,"merge":"%s","pkg":"good"}]}\n' "$GOOD" "$MERGE" > "$W/halfbroken.json"
# A different list inside the clone under the same name: a relative path must not find this one.
mkdir -p "$W/caller" && cp "$W/accept.json" "$W/caller/tasks.json" && cp "$W/other.json" "$R/tasks.json"

bad=0
expect() { # name, wanted exit, wanted text, list...
  local name=$1 want=$2 text=$3; shift 3
  local out rc
  out=$(bash "$HERE/validate-gates.sh" "$@" 2>&1); rc=$?
  if [ $rc -eq "$want" ] && grep -q -F -- "$text" <<< "$out"; then echo "pass  $name (exit $rc)"
  else echo "FAIL  $name: exit $rc, wanted $want with '$text'"; echo "$out" | sed 's/^/        /'; bad=1; return 1; fi
}
also() { if eval "$2"; then echo "pass  $1"; else echo "FAIL  $1"; bad=1; fi; }
clean() { [ "$(git -C "$R" worktree list | wc -l)" -eq 1 ]; }

export ANTHROPIC_API_KEY=fake ANTHROPIC_AUTH_TOKEN=fake OPENAI_API_KEY=fake CODEX_API_KEY=fake
expect "two good gates are accepted, and no key variable reaches a gate" 0 "DONE: 2 gates" "$W/accept.json"
also "  and the validator made its temp folder" '[ -d "$W/tmp" ]'
expect "  with the failing test shown" 0 "TestNew fails" "$W/accept.json"
expect "  and the build error in the copied file shown" 0 "does not build: build/cache_test.go:" "$W/accept.json"
expect "a test of another file that fails before the fix is refused" 1 "UNEXPECTED: fails before the fix without the copied tests" "$W/other.json"
expect "a build error in another file is refused" 1 "UNEXPECTED: fails before the fix without the copied tests" "$W/otherbuild.json"
expect "a gate that passes before the fix is refused" 1 "UNEXPECTED: passes before the fix" "$W/same.json"
expect "a gate that fails at the merge is refused" 1 "UNEXPECTED: fails at the merge" "$W/never.json"
for at in 1 2 3; do
  rm -f "$COUNT"
  PATH="$W/bin:$PATH" FAIL_BUILD_AT=$at expect "a build that fails on side $at of 3 is refused, not read as a failing test" 1 \
    "UNEXPECTED: go build fails, so the tests did not run" "$W/onlygood.json"
done
expect "a test file missing at the merge stops the run" 2 "no file good/missing_test.go at" "$W/nofile.json"
expect "a folder listed as a test file stops the run" 2 "no file good at" "$W/folder.json"
expect "a test file that is a folder before the fix stops the run" 2 "is a folder at" "$W/swap.json"
echo keep > "$W/validate/sentinel_test.go"
expect "a test path that leaves the repository stops the run" 2 "not a plain path inside the repository" "$W/escape.json"
also "  and the file outside is untouched" '[ "$(cat "$W/validate/sentinel_test.go")" = keep ]'
expect "an unknown commit stops the run" 2 "no commit" "$W/nocommit.json"
also "no worktree is left behind by the refusals" clean
rm -rf "$W/validate"
expect "a task without its test files stops the run" 2 "cannot use every task" "$W/halfbroken.json"
also "  before any task ran" '[ -z "$(ls -A "$W/validate" 2>/dev/null)" ]'
expect "a missing list stops the run" 2 "no task list" "$W/none.json"
mkdir -p "$W/validate/1-own.log"
expect "a log that cannot be written stops the run" 2 "cannot write" "$W/accept.json"
also "  and leaves no worktree behind" clean
rm -rf "$W/validate"
( cd "$W/caller" && expect "a relative list is the caller's, not the clone's" 0 "DONE: 2 gates" tasks.json ) || bad=1
( cd "$W/.." && HRN_VS_DIRECT_WORK=$(basename "$W") expect "a relative work folder is the caller's" 0 "DONE: 2 gates" "$W/accept.json" ) || bad=1
[ $bad -eq 0 ] && echo "ALL PASS" || { echo "SOME FAILED"; exit 1; }
