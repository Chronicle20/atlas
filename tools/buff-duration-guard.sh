#!/usr/bin/env bash
# tools/buff-duration-guard.sh — buffdurationguard gate.
#
# Thin wrapper: the shared driver in tools/lib/analyzer-guard.sh builds the
# analyzer once (content-keyed), runs it through `go vet -vettool=` so the go
# command caches per-package facts, and walks modules in parallel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tools/lib/analyzer-guard.sh"

GUARD=buffdurationguard
SELFTEST=1
SCAN_ROOTS=("$ROOT/services" "$ROOT/libs")
FAIL_MSG=(
    "buffdurationguard: FAIL — seconds-valued buff duration emitter found"
    "  The COMMAND_TOPIC_CHARACTER_BUFF duration field is MILLISECONDS."
    "  Contract owner: services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go"
)

analyzer_guard_main
