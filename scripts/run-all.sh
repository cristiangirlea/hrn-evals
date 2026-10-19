#!/usr/bin/env bash
# Runs every suite under suites/ (or the ones named) with hrn eval, then lists
# the saved evals so today's run can be compared with an earlier one:
#   hrn eval diff <earlier-id> <today-id>
set -euo pipefail
cd "$(dirname "$0")/.."

# Plain assignments, so that set -e stops the script when git fails: inside the
# if a failure would read as a clean tree, inside echo as an empty commit.
# Nothing in the user's git setup may hide a file from this check, since a
# file left out of HEAD is left out of what the agents branch from: an empty
# core.excludesFile turns off their global ignore files (their
# core.excludesFile and the default ~/.config/git/ignore), and
# --untracked-files=normal overrides a status.showUntrackedFiles=no. The
# repository's own .gitignore still applies.
status=$(git -c core.excludesFile= status --porcelain --untracked-files=normal)
if [ -n "$status" ]; then
	echo "run-all: commit or stash local changes first; the agents branch from HEAD" >&2
	exit 1
fi
# git status does not report a change to a file marked assume-unchanged (a
# lowercase tag in ls-files -v) or skip-worktree (S), so refuse the marks.
files=$(git -c core.quotePath=false ls-files -v)
flagged=()
while IFS= read -r line; do
	case ${line%% *} in
	[a-z] | S) flagged+=("${line#* }") ;;
	esac
done <<<"$files"
if [ ${#flagged[@]} -gt 0 ]; then
	# update-index takes one of the two marks per call, so the fix is two commands.
	echo "run-all: these files are marked assume-unchanged or skip-worktree, so a change to them would not show:" >&2
	printf '  %s\n' "${flagged[@]}" >&2
	echo "clear the marks first, one command each:" >&2
	echo "  git update-index --no-assume-unchanged <file>" >&2
	echo "  git update-index --no-skip-worktree <file>" >&2
	exit 1
fi

suites=("$@")
if [ ${#suites[@]} -eq 0 ]; then
	suites=(suites/*/*.json)
fi

echo "pins:"
echo "hrn version:"
hrn version
# hrn looks in known install folders after PATH (codex's own on Windows), so a
# backend missing from PATH is pinned from the path hrn backends prints for it,
# on a line "  ✓ <name> <state>: <path>". A plain assignment, so a failing hrn
# stops the run.
listed=0
for backend in claude codex; do
	bin= from=
	if command -v "$backend" >/dev/null 2>&1; then
		bin=$backend
	else
		if [ "$listed" = 0 ]; then
			backends=$(hrn backends)
			listed=1
		fi
		while read -r mark name rest; do
			if [ "$mark" = ✓ ] && [ "$name" = "$backend" ]; then
				bin=${rest#*: }
				bin=${bin%$'\r'}
				from=" (hrn backends: $bin)"
			fi
		done <<<"$backends"
	fi
	if [ -n "$bin" ]; then
		echo "$backend version$from:"
		"$bin" --version || echo "$backend: version unavailable"
	else
		echo "$backend: (not installed)"
	fi
done
commit=$(git rev-parse HEAD)
echo "suite commit: $commit"
echo "model ids: a spec's variant asks for a model or for the backend's default (sonnet is an alias); in the batch record (~/.hrn/batches/<id>.json) tasks[].attempts[].result.model is the id the backend reported when result.model_reported is true (claude), otherwise the model asked for; codex reports none, so a codex variant without a model ran codex's configured default, which is not recorded"
echo "fixture SHA-256 hashes:"
if command -v sha256sum >/dev/null 2>&1; then
	hash=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
	hash=(shasum -a 256)
else
	echo "run-all: sha256sum or shasum is required for fixture pins" >&2
	exit 1
fi
while IFS= read -r -d '' fixture; do
	read -r sum _ < <("${hash[@]}" "$fixture")
	printf '%s  %s\n' "$sum" "$fixture"
done < <(find fixtures/ -type f -print0 | LC_ALL=C sort -z)

failed=0
for s in "${suites[@]}"; do
	echo "=== $s"
	hrn eval "$s" || failed=1
done
hrn eval list -n "${#suites[@]}"
exit "$failed"
