#!/usr/bin/env bash
# tools/operator-cancel-path-guard.sh — enforces that the OPERATOR pending-change
# cancel route (DELETE .../pending-changes/{id}, reason "operator_cancelled",
# services/atlas-character/atlas.com/character/pending_change/resource.go:164)
# is never reachable from a game-client socket handler.
#
# NARROWED PROPERTY. This guard once asserted "no client cancel path exists at
# all" — that the cancel route was operator-only REST. That property is FALSE:
# commit 4a5d9ff65 landed a real client-initiated cancel path, on the strength
# of two IDA derivations committed at
# docs/tasks/task-227-cash-name-change-world-transfer/cancel-entry-point.md and
# cancel-confirm-semantics.md. The client's name-change/world-transfer coupon
# item-use arm builds a double-confirm CANCELREQUESTS_* dialog chain and, on
# full confirmation, appends an invariant trailing byte to the generic cash
# item-use packet, which atlas-channel resolves to the SELF-SCOPED cancel route
# (POST /characters/{id}/pending-changes/cancel, reason "player_cancelled",
# pending_change/processor.go:349). That route is the feature working as
# derived from the client and must NEVER be flagged by this guard.
#
# A green guard asserting a falsehood is worse than no guard — it manufactures
# confidence in a property the code does not have. The property is narrowed to
# what is actually true and still worth enforcing: the id-based, operator-only
# DELETE route must stay unreachable from any socket handler.
#
# Two checks, both scoped to Go files under
# services/atlas-channel/atlas.com/channel/socket/handler/:
#
#   (a) No file may reference the literal reason string "operator_cancelled".
#   (b) No file may combine an HTTP DELETE call with the "pending-changes"
#       resource path — that pairing only exists to reach the id-based
#       operator route; the legitimate self-scoped route is a POST to a fixed
#       "/cancel" sub-path and never appears with a DELETE call.
#
# Plus a template-side check: no tenant socket-config template may bind
# CashShopCancelNameChangeResultWriter or CashShopCancelTransferWorldResultWriter
# (libs/atlas-packet/cash/clientbound/cancel_name_change_result.go,
# cancel_transfer_world_result.go) as a "handler" entry. Both are CLIENTBOUND
# writers; a template naming one as a handler is the defect this catches.
#
# Run from anywhere; non-empty diagnostics -> non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HANDLER_DIR="$ROOT/services/atlas-channel/atlas.com/channel/socket/handler"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

FAIL=0

# Scope to production handler files, not this guard's own test suite -- the
# test file that proves this guard works (Step 2's red runs) necessarily
# contains the banned literals inside string/comment fixtures, which would
# otherwise make the guard fail against itself. Real socket handlers are
# never *_test.go.

# --- (a) operator_cancelled reason string ----------------------------------
while IFS= read -r hit; do
    [ -z "$hit" ] && continue
    echo "OPERATOR-CANCEL REASON REACHABLE FROM SOCKET HANDLER: $hit"
    FAIL=1
done <<EOF
$(grep -rn "operator_cancelled" "$HANDLER_DIR" --include='*.go' --exclude='*_test.go' || true)
EOF

# --- (b) DELETE call combined with the pending-changes resource path -------
for f in "$HANDLER_DIR"/*.go; do
    [ -e "$f" ] || continue
    case "$f" in
        *_test.go) continue ;;
    esac
    if grep -q "pending-changes" "$f" && grep -qE 'MakeDeleteRequest|http\.MethodDelete|"DELETE"' "$f"; then
        line="$(grep -n "pending-changes" "$f" | head -1)"
        echo "OPERATOR-CANCEL DELETE ROUTE REACHABLE FROM SOCKET HANDLER: $f:$line"
        FAIL=1
    fi
done

# --- template check: clientbound cancel writers bound as handlers ----------
TEMPLATE_HITS="$(python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys

tmpl_dir = sys.argv[1]
banned = {"CashShopCancelNameChangeResult", "CashShopCancelTransferWorldResult"}
bad = 0
for path in sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json"))):
    d = json.load(open(path))
    sock = d.get("socket", {})
    for e in sock.get("handlers") or []:
        if not isinstance(e, dict):
            continue
        name = e.get("handler")
        if name in banned:
            print("CLIENTBOUND CANCEL WRITER BOUND AS HANDLER: %s handlers: %s @ opCode %s"
                  % (os.path.basename(path), name, e.get("opCode")))
            bad += 1
if bad:
    sys.exit(1)
PY
)" || true
if [ -n "$TEMPLATE_HITS" ]; then
    echo "$TEMPLATE_HITS"
    FAIL=1
fi

if [ "$FAIL" -ne 0 ]; then
    echo ""
    echo "FAIL: the operator-only pending-change cancel route must never be"
    echo "reachable from a socket handler or bound by a tenant template. The"
    echo "self-scoped player-cancel route (reason \"player_cancelled\", POST"
    echo ".../pending-changes/cancel) is unaffected and legitimate."
    exit 1
fi

echo "OK: operator pending-change cancel route is unreachable from socket handlers and templates."
