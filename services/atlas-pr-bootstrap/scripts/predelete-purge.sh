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

# do_purge_tenants
#
# 1. Fetches the tenant list from atlas-tenants via the ingress.
# 2. Fails visibly if the list is empty — a live PR env always owns >=1 tenant.
# 3. Issues DELETE /api/data/tenants/{id} for each; any non-2xx → phase fails.
#
# curl uses -f (fail-on-4xx/5xx) for the GET so a non-200 response causes a
# non-zero exit. For the DELETE we capture %{http_code} explicitly so we can
# log the status code and fail on anything outside 2xx without swallowing it.
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

    local rc=0 id status
    while IFS= read -r id; do
        [ -z "$id" ] && continue
        ATLAS_STEP=predelete-purge log info "purging tenant ${id}"
        status=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
            -H 'X-Atlas-Operator: 1' \
            "${ATLAS_INGRESS_BASE}/api/data/tenants/${id}" 2>/dev/null || echo 000)
        case "$status" in
            2*)
                ATLAS_STEP=predelete-purge log info "purged tenant ${id} (status ${status})" ;;
            *)
                ATLAS_STEP=predelete-purge log error \
                    "purge tenant ${id} failed (status ${status})"
                rc=1 ;;
        esac
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
