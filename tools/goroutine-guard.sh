#!/usr/bin/env bash
# tools/goroutine-guard.sh — goroutineguard gate.
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=goroutineguard
SELFTEST=1
SCAN_ROOTS=("$ROOT/services" "$ROOT/libs")
FAIL_MSG=(
    "goroutineguard: FAIL — bare go statements found (use routine.Go from libs/atlas-routine)"
)

analyzer_guard_main
