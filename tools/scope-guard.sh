#!/usr/bin/env bash
# tools/scope-guard.sh — scopeguard gate (FR-8.5).
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
#
# SCAN_ROOTS covers services/ AND libs/ — unlike rediskeyguard's
# services-only scope, scopeguard's Rule 2 (call-site) must see libs/-resident
# code too: Task 42 was separately found blind to libs/, and
# libs/atlas-database/idempotency.go itself has a live WithoutTenantFilter
# call site (row 13 of query-scope-audit.md's UNSCOPED disposition table).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=scopeguard
SELFTEST=0
SCAN_ROOTS=("$ROOT/services" "$ROOT/libs")
FAIL_MSG=(
    "scopeguard: FAIL — an entity or call site is unscoped with no allowlist entry"
    "scopeguard: add a written reason to tools/scopeguard/allowlist.txt or"
    "scopeguard: tools/scopeguard/callsite-allowlist.txt, or fix the code — see"
    "scopeguard: docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md"
)

analyzer_guard_main
