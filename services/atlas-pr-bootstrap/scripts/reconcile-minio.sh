#!/usr/bin/env bash
# Daily reconciliation orchestrator. Enumerates the live-tenant UUID union
# across atlas-main + atlas-pr-* namespaces (FAIL-CLOSED: any unreachable env
# aborts the run), then POSTs the keep-list to atlas-data's reconcile endpoint.
#
# Env:
#   KUBECTL                 (default kubectl)
#   CURL                    (default curl)
#   ATLAS_DATA_BASE         (default http://atlas-data.atlas-main.svc.cluster.local:8080)
#   RECONCILE_DRY_RUN       (default true)
#   RECONCILE_MIN_AGE_HOURS (default 48)
set -uo pipefail

. "$(dirname "$0")/lib.sh"

: "${KUBECTL:=kubectl}"
: "${CURL:=curl}"
: "${ATLAS_DATA_BASE:=http://atlas-data.atlas-main.svc.cluster.local:8080}"
: "${RECONCILE_DRY_RUN:=true}"
: "${RECONCILE_MIN_AGE_HOURS:=48}"

do_reconcile() {
  local namespaces ns url ids all=""
  if ! namespaces=$("$KUBECTL" get ns -o name 2>/dev/null \
        | sed 's|^namespace/||' \
        | grep -E '^(atlas-main|atlas-pr-.+)$'); then
    record_error reconcile "could not list namespaces"
    return 1
  fi
  if [ -z "$namespaces" ]; then
    record_error reconcile "no atlas namespaces found"
    return 1
  fi

  while IFS= read -r ns; do
    [ -z "$ns" ] && continue
    url="http://atlas-ingress.${ns}.svc.cluster.local/api/tenants"
    ATLAS_STEP=reconcile log info "enumerating tenants in ${ns}"
    if ! ids=$("$CURL" -fsS -H 'Accept: application/vnd.api+json' "$url" 2>/dev/null \
          | jq -r '.data[].id' 2>/dev/null); then
      # FAIL-CLOSED: a discovered env we cannot read must not be treated as orphaned.
      record_error reconcile "could not enumerate tenants in ${ns}; aborting (fail-closed)"
      return 1
    fi
    all="${all}${ids}"$'\n'
  done <<<"$namespaces"

  local union
  union=$(printf '%s\n' "$all" | sed '/^$/d' | sort -u)
  if [ -z "$union" ]; then
    record_error reconcile "empty tenant union; refusing to reconcile"
    return 1
  fi

  local keep_json body
  keep_json=$(printf '%s\n' "$union" | jq -R . | jq -cs .)
  body=$(jq -cn --argjson keep "$keep_json" \
      --argjson age "$RECONCILE_MIN_AGE_HOURS" \
      --argjson dry "$RECONCILE_DRY_RUN" \
      '{data:{type:"minioReconciles",attributes:{keepTenantIds:$keep,minAgeHours:$age,dryRun:$dry}}}')

  ATLAS_STEP=reconcile log info "posting keep-list ($(printf '%s\n' "$union" | wc -l | tr -d ' ') tenants, dryRun=${RECONCILE_DRY_RUN}, minAgeHours=${RECONCILE_MIN_AGE_HOURS})"
  # ParseTenant gates EVERY atlas-data route (400 without these four headers),
  # even this cross-tenant sweep. Send a synthetic tenant it accepts but ignores
  # (verified live: nil-UUID + GMS/83/1 → 200). Overridable for tests/other envs.
  : "${RECONCILE_TENANT_ID:=00000000-0000-0000-0000-000000000000}"
  : "${RECONCILE_REGION:=GMS}"
  : "${RECONCILE_MAJOR_VERSION:=83}"
  : "${RECONCILE_MINOR_VERSION:=1}"
  # -f (--fail): curl exits non-zero on any HTTP >=400 response, so success/
  # failure is signalled by curl's own exit code rather than a parsed status
  # string. Mirrors the -fsS enumeration call above and predelete-purge.sh's
  # GET pattern.
  local resp
  if resp=$("$CURL" -sf -X POST \
      -H 'X-Atlas-Operator: 1' \
      -H "TENANT_ID: ${RECONCILE_TENANT_ID}" \
      -H "REGION: ${RECONCILE_REGION}" \
      -H "MAJOR_VERSION: ${RECONCILE_MAJOR_VERSION}" \
      -H "MINOR_VERSION: ${RECONCILE_MINOR_VERSION}" \
      -H 'Content-Type: application/vnd.api+json' \
      -d "$body" \
      "${ATLAS_DATA_BASE}/api/data/minio/reconcile" 2>/dev/null); then
    ATLAS_STEP=reconcile log info "reconcile ok: ${resp}"
  else
    record_error reconcile "reconcile POST failed"
    return 1
  fi
  return 0
}

ATLAS_PHASE_ERRORS=()
run_phase reconcile do_reconcile
summarize_phases 1
exit $?
