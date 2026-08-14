#!/usr/bin/env bash
# npc-shop-contract-mirror-guard.sh — enforces that the COMMAND_TOPIC_NPC_SHOP /
# EVENT_TOPIC_NPC_SHOP_STATUS contract is identical in its three copies.
#
# atlas-npc-shops owns the contract; atlas-channel and atlas-saga-orchestrator
# carry mirrors because the three services live in separate Go modules and
# nothing in the compiler links them. A field name or json tag changed in one
# copy and not the others does not fail any build — it decodes into a
# zero-valued body at runtime, silently. task-221.
#
# The files are compared from their `package` clause onward: the only permitted
# difference is the leading doc comment, which names the mirror direction.
#
# Run from the repo root; drift → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go"
CHANNEL_MIRROR="$ROOT/services/atlas-channel/atlas.com/channel/kafka/message/npc/shop/kafka.go"
SAGA_MIRROR="$ROOT/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npcshop/kafka.go"

rc=0
for f in "$OWNER" "$CHANNEL_MIRROR" "$SAGA_MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "npc-shop-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

check_pair() {
    local owner="$1" mirror="$2" label="$3"
    if ! diff -u <(body "$owner") <(body "$mirror"); then
        echo "npc-shop-contract-mirror-guard: FAIL — $label drift (diff above)."
        return 1
    fi
    return 0
}

check_pair "$OWNER" "$CHANNEL_MIRROR" "atlas-channel mirror" || rc=1
check_pair "$OWNER" "$SAGA_MIRROR" "atlas-saga-orchestrator mirror" || rc=1

if [ "$rc" -eq 0 ]; then
    echo "npc-shop-contract-mirror-guard: OK — all three copies identical."
fi
exit "$rc"
