#!/usr/bin/env bash
# Runs every suite under suites/ (or the ones named) with hrn eval, then lists
# the saved evals so today's run can be compared with an earlier one:
#   hrn eval diff <earlier-id> <today-id>
set -euo pipefail
cd "$(dirname "$0")/.."

if ! git diff --quiet HEAD --; then
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
echo "suite commit: $(git rev-parse HEAD)"
echo "model ids: from each suite's matrix, saved in each eval's spec"
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
