#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
exec go run scripts/stream-eval-main.go scripts/stream-eval.go "$@"
