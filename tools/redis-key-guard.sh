#!/usr/bin/env bash
# tools/redis-key-guard.sh — rediskeyguard gate.
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=rediskeyguard
SELFTEST=0
SCAN_ROOTS=("$ROOT/services")
FAIL_MSG=(
    "rediskeyguard: FAIL — raw keyed redis client calls found (use a libs/atlas-redis type)"
)

analyzer_guard_main
