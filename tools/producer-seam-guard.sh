#!/usr/bin/env bash
# tools/producer-seam-guard.sh — producerseamguard gate.
#
# Bans new direct producer.Produce calls under services/. producer.Produce is
# the raw seam beneath producer.ProviderImpl, the canonical composed producer
# (span + tenant + environment header decorators — task-232 FR-4.1). A direct
# call bypasses that composition and silently drops whichever header the
# call site forgot to pass by hand. The four call sites that predate the
# composed decorator are allowlisted in tools/producerseamguard/analyzer.go.
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=producerseamguard
SELFTEST=1
SCAN_ROOTS=("$ROOT/services")
FAIL_MSG=(
    "producerseamguard: FAIL — direct producer.Produce call outside libs/ found"
    "  compose headers through producer.ProviderImpl instead, or add the call"
    "  site to tools/producerseamguard's allowlist with a written reason."
)

analyzer_guard_main
