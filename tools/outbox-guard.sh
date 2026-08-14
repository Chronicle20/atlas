#!/usr/bin/env bash
# tools/outbox-guard.sh — outboxguard gate.
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=outboxguard
SELFTEST=0
SCAN_ROOTS=("$ROOT/services")
FAIL_MSG=(
    "outboxguard: FAIL — direct producer calls inside DB transactions (use outbox.EmitProvider)"
)

analyzer_guard_main
