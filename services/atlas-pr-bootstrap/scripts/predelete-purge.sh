#!/usr/bin/env bash
# Argo CD PreDelete hook. Runs in the per-PR namespace while atlas-data /
# atlas-tenants / atlas-ingress are still alive. Purges every tenant the env
# owns via atlas-data's DELETE /api/data/tenants/{id}. On any failure it exits
# non-zero so the hook Job fails visibly — NO silent skip.
#
# Required env:
#   PR_NUMBER          — PR number; ATLAS_ENV is derived as sha256("pr-N")[:4]
#   ATLAS_INGRESS_BASE — http://atlas-ingress.<ns>.svc.cluster.local

set -uo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

require_env PR_NUMBER ATLAS_INGRESS_BASE

ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
export ATLAS_ENV
ATLAS_STEP=init log info "derived ATLAS_ENV=${ATLAS_ENV} for PR ${PR_NUMBER}"

# atlas-data's ParseTenant middleware requires TENANT_ID/REGION/MAJOR_VERSION/
# MINOR_VERSION on every request, including operator-authenticated ones — a
# request missing any of them 400s before the handler runs. The purge target
# is cross-tenant (tenantpurge reads the id from the URL path via mux.Vars,
# not the tenant context), so any well-formed synthetic tenant is accepted
# and ignored; the defaults below match the combination verified live.
: "${PURGE_TENANT_ID:=00000000-0000-0000-0000-000000000000}"
: "${PURGE_REGION:=GMS}"
: "${PURGE_MAJOR_VERSION:=83}"
: "${PURGE_MINOR_VERSION:=1}"

# Bounded retry budget for the per-tenant DELETE. Overridable for tests.
: "${PURGE_DELETE_RETRIES:=3}"
: "${PURGE_DELETE_RETRY_SLEEP:=2}"

# delete_tenant_once <id>
#
# Single DELETE attempt against atlas-data. curl uses -o/-w to capture
# %{http_code} explicitly (rather than -f) so we can log the status and
# decide success/failure ourselves without swallowing the response.
delete_tenant_once() {
    local id="$1"
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
        -H 'X-Atlas-Operator: 1' \
        -H "TENANT_ID: ${PURGE_TENANT_ID}" \
        -H "REGION: ${PURGE_REGION}" \
        -H "MAJOR_VERSION: ${PURGE_MAJOR_VERSION}" \
        -H "MINOR_VERSION: ${PURGE_MINOR_VERSION}" \
        "${ATLAS_INGRESS_BASE}/api/data/tenants/${id}" 2>/dev/null || echo 000)
    case "$status" in
        2*)
            ATLAS_STEP=predelete-purge log info "purged tenant ${id} (status ${status})"
            return 0 ;;
        *)
            ATLAS_STEP=predelete-purge log info "purge tenant ${id} attempt failed (status ${status})"
            return 1 ;;
    esac
}

# do_purge_tenants
#
# 1. Fetches the tenant list from atlas-tenants via the ingress.
# 2. Fails visibly if the list is empty — a live PR env always owns >=1 tenant.
# 3. Issues DELETE /api/data/tenants/{id} for each, retrying transient
#    failures up to PURGE_DELETE_RETRIES times; only a final non-2xx fails
#    the phase.
#
# curl uses -f (fail-on-4xx/5xx) for the GET so a non-200 response causes a
# non-zero exit.
do_purge_tenants() {
    local ids
    ATLAS_STEP=predelete-purge log info "enumerating tenants from ${ATLAS_INGRESS_BASE}/api/tenants"
    if ! ids=$(curl -fsS -H 'Accept: application/vnd.api+json' \
            "${ATLAS_INGRESS_BASE}/api/tenants" 2>/dev/null \
            | jq -r '.data[].id' 2>/dev/null); then
        record_error predelete-purge \
            "could not enumerate tenants from ${ATLAS_INGRESS_BASE}/api/tenants"
        return 1
    fi
    if [ -z "$ids" ]; then
        record_error predelete-purge \
            "no tenants returned; a PR env always owns >=1 tenant — refusing to report success"
        return 1
    fi

    local rc=0 id
    while IFS= read -r id; do
        [ -z "$id" ] && continue
        ATLAS_STEP=predelete-purge log info "purging tenant ${id}"
        if ! retry "$PURGE_DELETE_RETRIES" "$PURGE_DELETE_RETRY_SLEEP" delete_tenant_once "$id"; then
            ATLAS_STEP=predelete-purge log error \
                "purge tenant ${id} failed after ${PURGE_DELETE_RETRIES} attempts"
            rc=1
        fi
    done <<<"$ids"
    return $rc
}

# ----------------------------------------------------------------------------
# Orchestration — single phase.
# ----------------------------------------------------------------------------
ATLAS_PHASE_ERRORS=()
run_phase predelete-purge do_purge_tenants
summarize_phases 1
exit $?
