#!/usr/bin/env bash
# Build the outboxguard analyzer once, then run it over every Go service
# module. Non-empty diagnostics → non-zero exit. Run from the repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/outboxguard"
BIN="$(mktemp -d)/outboxguard"

echo "building outboxguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/outboxguard )

rc=0
# Every Go module that has a go.mod under services/ is a guard target.
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "outboxguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "outboxguard: FAIL — direct producer calls inside DB transactions (use outbox.EmitProvider)"
fi
exit $rc
