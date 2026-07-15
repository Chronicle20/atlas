#!/usr/bin/env bash
# Live-config patch: add the hired-merchant field-NPC clientbound writers
# (SpawnHiredMerchant / DestroyHiredMerchant / UpdateHiredMerchant) to a LIVE
# tenant's atlas-configurations socket config.
#
# WHY: these three opcodes ship in the seed templates, which apply only at tenant
# CREATION. A tenant created before this change has no writer entry for them, so
# atlas-channel silently drops the spawn/despawn/update packets. This patch adds
# them to the live config and (because PATCH enqueues a config-status event) the
# fleet picks them up; restart atlas-channel if it does not reload live.
#
# The patch is a FULL-DOCUMENT replace (atlas-configurations UpdateById marshals
# the whole tenant RestModel), so we GET the current doc, inject the writers, and
# PATCH the whole thing back. It is idempotent: re-running is a no-op.
#
# Usage:
#   CONFIG_BASE=https://<pr-env-host>/api/configurations \
#   TENANT_ID=<uuid> \
#   ./patch-hired-merchant-writers.sh            # add the writers
#
#   ... REMOVE=1 ./patch-hired-merchant-writers.sh   # revert (remove them)
#
# CONFIG_BASE can also be a port-forward, e.g.:
#   kubectl -n <ns> port-forward svc/atlas-configurations 8080:8080 &
#   CONFIG_BASE=http://localhost:8080/configurations TENANT_ID=<uuid> ./patch-...
#
# Requires: curl, jq.
set -euo pipefail

: "${CONFIG_BASE:?set CONFIG_BASE (e.g. https://host/api/configurations)}"
: "${TENANT_ID:?set TENANT_ID (tenant uuid); list via: curl \$CONFIG_BASE/tenants | jq '.data[].id'}"
REMOVE="${REMOVE:-0}"

# opCode per (region majorVersion) — source: docs/packets/audits/status.json.
# v48 has no hired-merchant feature and is intentionally absent.
opcodes_for() { # $1=region $2=major  -> "SPAWN DESTROY UPDATE" hex
  case "$1 $2" in
    "GMS 61") echo "0xCA 0xCB 0xCC" ;;
    "GMS 72") echo "0xEB 0xEC 0xED" ;;
    "GMS 79") echo "0xF3 0xF4 0xF5" ;;
    "GMS 83"|"GMS 84") [ "$2" = 83 ] && echo "0x109 0x10A 0x10B" || echo "0x110 0x111 0x112" ;;
    "GMS 87") echo "0x11A 0x11B 0x11C" ;;
    "GMS 95") echo "0x13F 0x140 0x141" ;;
    "JMS 185") echo "0x11E 0x11F 0x120" ;;
    *) echo "" ;;
  esac
}

echo ">> GET current config for tenant ${TENANT_ID}"
DOC="$(curl -fsS "${CONFIG_BASE}/tenants/${TENANT_ID}")"
ATTR="$(jq -c '.data.attributes' <<<"$DOC")"
REGION="$(jq -r '.region' <<<"$ATTR")"
MAJOR="$(jq -r '.majorVersion' <<<"$ATTR")"
echo ">> tenant is ${REGION} v${MAJOR}"

OPS="$(opcodes_for "$REGION" "$MAJOR")"
if [ -z "$OPS" ]; then
  echo "!! no hired-merchant opcodes for ${REGION} v${MAJOR} (v48 has no feature); nothing to do." >&2
  exit 1
fi
read -r SPAWN DESTROY UPDATE <<<"$OPS"
echo ">> opcodes: Spawn=${SPAWN} Destroy=${DESTROY} Update=${UPDATE}"

if [ "$REMOVE" = 1 ]; then
  NEW_ATTR="$(jq -c \
    '.socket.writers |= map(select(.writer|test("^(Spawn|Destroy|Update)HiredMerchant$")|not))' \
    <<<"$ATTR")"
  echo ">> removing the three hired-merchant writers"
else
  NEW_ATTR="$(jq -c \
    --arg sp "$SPAWN" --arg de "$DESTROY" --arg up "$UPDATE" '
    .socket.writers as $w
    | ($w | map(.writer)) as $have
    | .socket.writers = ($w
        + (if ($have|index("SpawnHiredMerchant"))   then [] else [{opCode:$sp,writer:"SpawnHiredMerchant"}]   end)
        + (if ($have|index("DestroyHiredMerchant")) then [] else [{opCode:$de,writer:"DestroyHiredMerchant"}] end)
        + (if ($have|index("UpdateHiredMerchant"))  then [] else [{opCode:$up,writer:"UpdateHiredMerchant"}]  end))
    ' <<<"$ATTR")"
fi

# No-op guard.
if [ "$(jq -cS . <<<"$ATTR")" = "$(jq -cS . <<<"$NEW_ATTR")" ]; then
  echo ">> config already up to date; nothing to PATCH."
  exit 0
fi

BODY="$(jq -nc --arg id "$TENANT_ID" --argjson attr "$NEW_ATTR" \
  '{data:{type:"tenants",id:$id,attributes:$attr}}')"

echo ">> PATCH config back"
curl -fsS -X PATCH "${CONFIG_BASE}/tenants/${TENANT_ID}" \
  -H 'Content-Type: application/vnd.api+json' \
  -d "$BODY" >/dev/null
echo ">> done. The PATCH enqueues a config-status event; if atlas-channel does not"
echo ">> reload live, restart it:  kubectl -n <ns> rollout restart deploy/atlas-channel"
