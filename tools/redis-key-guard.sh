#!/usr/bin/env bash
# Build the rediskeyguard analyzer once, then run it over every Go service
# module. Non-empty diagnostics → non-zero exit. Run from the repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/rediskeyguard"
BIN="$(mktemp -d)/rediskeyguard"

echo "building rediskeyguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/rediskeyguard )

rc=0
# Every Go module that has a go.mod under services/ is a guard target.
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "rediskeyguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "rediskeyguard: FAIL — raw keyed redis client calls found (use a libs/atlas-redis type)"
fi
exit $rc
