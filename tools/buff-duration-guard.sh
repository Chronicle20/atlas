#!/usr/bin/env bash
# Self-test the buffdurationguard analyzer fixtures, build it once, then run it
# over every Go module under services/ and libs/. Non-empty diagnostics →
# non-zero exit. Run from the repo root. tools/ is deliberately not swept —
# the analyzer's own testdata must be allowed to contain the defective forms.
#
# Guards the COMMAND_TOPIC_CHARACTER_BUFF duration unit (milliseconds; contract
# owner atlas-buffs kafka/message/character/kafka.go). task-190 FR-3.2/FR-3.3.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/buffdurationguard"
BIN="$(mktemp -d)/buffdurationguard"

echo "self-testing buffdurationguard..."
( cd "$GUARD_SRC" && GOWORK=off go test ./... )

echo "building buffdurationguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/buffdurationguard )

rc=0
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "buffdurationguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "buffdurationguard: FAIL — seconds-valued buff duration emitter found"
    echo "  The COMMAND_TOPIC_CHARACTER_BUFF duration field is MILLISECONDS."
    echo "  Contract owner: services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go"
fi
exit $rc
