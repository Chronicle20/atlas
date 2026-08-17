#!/usr/bin/env bash
# npc-conversation-contract-mirror-guard.sh — enforces that the
# COMMAND_TOPIC_NPC / EVENT_TOPIC_NPC_CONVERSATION_STATUS contract is identical
# in its two copies.
#
# atlas-npc-conversations owns the contract; atlas-saga-orchestrator carries a
# mirror because the two services live in separate Go modules and nothing in the
# compiler links them. A field name or json tag changed in one copy and not the
# other does not fail any build — it decodes into a zero-valued body at runtime,
# silently: a conversation start with no item id, no avatar, or no
# transactionId, so the awaiting saga step never completes. task-230.
#
# The files are compared from their `package` clause onward: the only permitted
# difference is the leading doc comment, which names the mirror direction.
#
# Run from the repo root; drift → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go"
SAGA_MIRROR="$ROOT/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go"

rc=0
for f in "$OWNER" "$SAGA_MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "npc-conversation-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

if ! diff -u <(body "$OWNER") <(body "$SAGA_MIRROR"); then
    echo "npc-conversation-contract-mirror-guard: FAIL — atlas-saga-orchestrator mirror drift (diff above)."
    exit 1
fi

echo "npc-conversation-contract-mirror-guard: OK — both copies identical."
