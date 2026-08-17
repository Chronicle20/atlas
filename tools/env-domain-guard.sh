#!/usr/bin/env bash
# tools/env-domain-guard.sh — envguard gate (task-232 NG5/FR-4.5).
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel. Mirrors
# tools/redis-key-guard.sh.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=envguard
SELFTEST=0
SCAN_ROOTS=("$ROOT/services")
FAIL_MSG=(
    "envguard: FAIL — libs/atlas-env imported from a domain package; only main.go and files under kafka/ or rest/ may import it (task-232 NG5/FR-4.5)"
)

analyzer_guard_main
