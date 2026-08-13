#!/usr/bin/env bash
# mist-contract-mirror-guard.sh — enforces that the COMMAND_TOPIC_MIST /
# EVENT_TOPIC_MIST contract is identical in its two copies.
#
# atlas-maps owns the contract; atlas-channel carries a mirror because the two
# services live in separate Go modules and nothing in the compiler links them.
# A field name or json tag changed in one copy and not the other does not fail
# any build — it decodes into a zero-valued body at runtime, silently: a mist
# with no bounds, no lifetime, or (task-218) no recovery magnitude and no party
# scope. Modelled on tools/trade-contract-mirror-guard.sh, which exists for the
# same failure mode on a different pair of modules.
#
# The two files are compared from their `package` clause onward: the only
# permitted difference is the leading doc comment, which names the mirror
# direction and therefore differs by design.
#
# Run from the repo root; drift → non-zero exit. task-218.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go"
MIRROR="$ROOT/services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go"

rc=0
for f in "$OWNER" "$MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "mist-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

if diff -u --label "owner: ${OWNER#"$ROOT"/}" --label "mirror: ${MIRROR#"$ROOT"/}" \
        <(body "$OWNER") <(body "$MIRROR"); then
    echo "OK: the mist contract mirror matches its owner."
    exit 0
fi

echo ""
echo "mist-contract-mirror-guard: FAIL — the mist Kafka contract has drifted."
echo "  owner : ${OWNER#"$ROOT"/}"
echo "  mirror: ${MIRROR#"$ROOT"/}"
echo ""
echo "  These two files are one cross-service wire contract. Struct names, field"
echo "  names and json tags must match exactly; only the leading doc comment,"
echo "  which names the mirror direction, may differ."
echo ""
echo "  FIX: apply the change to the owner, then re-copy it to the mirror and"
echo "  restore the mirror's doc-comment header:"
echo "    cp ${OWNER#"$ROOT"/} ${MIRROR#"$ROOT"/}"
echo "  (then edit the copied header back to \"Mirrors <owner path>\")"
exit 1
