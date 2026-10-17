#!/usr/bin/env bash
# Runs every suite under suites/ (or the ones named) with hrn eval, then lists
# the saved evals so today's run can be compared with an earlier one:
#   hrn eval diff <earlier-id> <today-id>
set -euo pipefail
cd "$(dirname "$0")/.."

# Plain assignments, so that set -e stops the script when git fails: inside the
# if a failure would read as a clean tree, inside echo as an empty commit.
# An empty core.excludesFile turns off the user's global ignore files (their
# core.excludesFile and the default ~/.config/git/ignore): a file they hide is
# still left out of HEAD, which the agents branch from. The repository's own
# .gitignore still applies.
status=$(git -c core.excludesFile= status --porcelain)
if [ -n "$status" ]; then
	echo "run-all: commit or stash local changes first; the agents branch from HEAD" >&2
	exit 1
fi

suites=("$@")
if [ ${#suites[@]} -eq 0 ]; then
	suites=(suites/*/*.json)
fi

echo "pins:"
echo "hrn version:"
hrn version
for backend in claude codex; do
	if command -v "$backend" >/dev/null 2>&1; then
		echo "$backend version:"
		"$backend" --version || echo "$backend: version unavailable"
	else
		echo "$backend: (not installed)"
	fi
done
commit=$(git rev-parse HEAD)
echo "suite commit: $commit"
echo "model ids: eval specs save the Claude alias sonnet and no Codex model; the resolved Claude id is in the batch record's attempt result; Codex records no model id"
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
