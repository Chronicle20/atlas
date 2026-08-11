#!/usr/bin/env bash
# trade-contract-mirror-guard.sh — enforces that the COMMAND_TOPIC_TRADE /
# EVENT_TOPIC_TRADE_STATUS contract is identical in its two copies.
#
# atlas-trades owns the contract; atlas-channel carries a mirror because the two
# services live in separate Go modules and nothing in the compiler links them.
# A field name or json tag changed in one copy and not the other does not fail
# any build — it decodes into a zero-valued body at runtime, silently. That is
# the entire reason the mirror convention exists (see the package doc comment in
# both files), so it is checked mechanically here.
#
# The two files are compared from their `package` clause onward: the only
# permitted difference is the leading doc comment, which names the mirror
# direction and therefore differs by design.
#
# Run from the repo root; drift → non-zero exit. task-205.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# Pair 1 — the atlas-channel command/status contract.
OWNER="$ROOT/services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka.go"
MIRROR="$ROOT/services/atlas-channel/atlas.com/channel/kafka/message/trade/kafka.go"

# Pair 2 — the atlas-saga-orchestrator escrow custody contract (task-205 §5A.2).
# Same failure mode, different pair of modules: atlas-trades owns the contract,
# the orchestrator carries a copy because it cannot import the atlas-trades
# module.
CUSTODY_OWNER="$ROOT/services/atlas-trades/atlas.com/trades/kafka/message/custody/kafka.go"
CUSTODY_MIRROR="$ROOT/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/trade/custody/kafka.go"

rc=0
for f in "$OWNER" "$MIRROR" "$CUSTODY_OWNER" "$CUSTODY_MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "trade-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

# Everything from the `package` clause onward; the doc-comment header above it
# is the one sanctioned difference.
body() { awk '/^package /{p=1} p' "$1"; }

# check_pair <owner> <mirror> <label>
check_pair() {
    local owner="$1" mirror="$2" label="$3"
    if diff -u --label "owner: ${owner#"$ROOT"/}" --label "mirror: ${mirror#"$ROOT"/}" \
            <(body "$owner") <(body "$mirror"); then
        echo "OK: the $label mirror matches its owner."
        return 0
    fi

    echo ""
    echo "trade-contract-mirror-guard: FAIL — the $label Kafka contract has drifted."
    echo "  owner : ${owner#"$ROOT"/}"
    echo "  mirror: ${mirror#"$ROOT"/}"
    echo ""
    echo "  These two files are one cross-service wire contract. Struct names, field"
    echo "  names and json tags must match exactly; only the leading doc comment,"
    echo "  which names the mirror direction, may differ."
    echo ""
    echo "  FIX: apply the change to the owner, then re-copy it to the mirror and"
    echo "  restore the mirror's doc-comment header:"
    echo "    cp ${owner#"$ROOT"/} ${mirror#"$ROOT"/}"
    echo "  (then edit the copied header back to \"Mirrors <owner path>\")"
    return 1
}

rc=0
check_pair "$OWNER" "$MIRROR" "trade contract" || rc=1
check_pair "$CUSTODY_OWNER" "$CUSTODY_MIRROR" "trade escrow custody contract" || rc=1
exit "$rc"
