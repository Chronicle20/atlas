#!/usr/bin/env bash
# Self-test the goroutineguard analyzer fixtures, build it once, then run it
# over every Go module under services/ and libs/. Non-empty diagnostics →
# non-zero exit. Run from the repo root. tools/ is deliberately not swept —
# the analyzer's own testdata must be allowed to contain bare go statements.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/goroutineguard"
BIN="$(mktemp -d)/goroutineguard"

echo "self-testing goroutineguard..."
( cd "$GUARD_SRC" && GOWORK=off go test ./... )

echo "building goroutineguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/goroutineguard )

rc=0
# Every Go module with a go.mod under services/ or libs/ is a guard target.
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "goroutineguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "goroutineguard: FAIL — bare go statements found (use routine.Go from libs/atlas-routine)"
fi
exit $rc
