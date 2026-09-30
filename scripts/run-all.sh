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

failed=0
for s in "${suites[@]}"; do
	echo "=== $s"
	hrn eval "$s" || failed=1
done
hrn eval list -n "${#suites[@]}"
exit "$failed"
