#!/usr/bin/env bash
# Atlas PR-env cleanup. Each phase is idempotent and runs through the
# shared run_phase orchestrator (lib.sh) so a single phase's failure
# does not skip subsequent phases. The Job exits non-zero iff at
# least one phase failed; the summary line names which.
#
# Required env:
#   PR_NUMBER              — PR number; ATLAS_ENV is derived as sha256("pr-N")[:4]
#   DB_HOST/PORT/USER/PASS — Postgres connection details
#   ATLAS_DB_NAMES    — space-separated list of base DB names
#   BOOTSTRAP_SERVERS — kafka.home:9093
#   REDIS_URL         — redis.home:6379
#   PIHOLE_API_BASE_1, PIHOLE_TOKEN_1, PIHOLE_API_BASE_2, PIHOLE_TOKEN_2
#   GHCR_TOKEN        — for image-tag delete + bot-branch delete
#   ATLAS_SERVICES    — comma-separated list of service names for image cleanup

set -uo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

# Phase 0 Task 0.1 finding: db-credentials secret values carry trailing
# whitespace (literal space + CR + LF). Strip BEFORE require_env so an
# all-whitespace value is caught by the empty check.
DB_USER="$(printf '%s' "${DB_USER:-}" | tr -d ' \r\n')"
DB_PASSWORD="$(printf '%s' "${DB_PASSWORD:-}" | tr -d ' \r\n')"

require_env PR_NUMBER DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL

# Derive ATLAS_ENV from PR_NUMBER. Bug #4 (env-hash annotation drift): the
# Application's atlas.env annotation can disagree with the formula's actual
# output (observed on PRs 491/522, see task-070 recovery-log.md). Deriving
# here guarantees cleanup targets the correct hash regardless. lib.sh's
# compute_atlas_env is pinned by test/lib_test.bats against the formula
# used by .github/workflows/pr-validation.yml and the ApplicationSet.
ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
export ATLAS_ENV
ATLAS_STEP=init log info "derived ATLAS_ENV=${ATLAS_ENV} for PR ${PR_NUMBER}"

# Derive ATLAS_ENVIRONMENT from PR_NUMBER the same way, and for the same
# reason: "pr-<N>" is the wire identity every other consumer of this
# convention already uses (libs/atlas-env.SelfVar; the sparse overlay's
# atlas-env ConfigMap literal ATLAS_ENVIRONMENT=pr-PLACEHOLDER_PR_NUMBER;
# environment-record.yaml's `name: "pr-PLACEHOLDER_PR_NUMBER"`), so
# deriving it here needs no new manifest wiring — postdelete-cleanup.yaml
# already carries PR_NUMBER (task-48 fix round 1). Only backfilled when
# unset so an explicit override (tests, or a future caller with a reason
# to differ) still wins.
ATLAS_ENVIRONMENT="${ATLAS_ENVIRONMENT:-pr-${PR_NUMBER}}"
export ATLAS_ENVIRONMENT
ATLAS_STEP=init log info "derived ATLAS_ENVIRONMENT=${ATLAS_ENVIRONMENT} for PR ${PR_NUMBER}"

# gh CLI requires its own credentials even when an explicit `-H
# "Authorization: Bearer …"` header is passed on the request — without
# GH_TOKEN/GITHUB_TOKEN in env it prompts for `gh auth login` and exits
# non-zero, which historically broke drop-branch silently and masked
# leaks in drop-images (which `2>&1 ||`'d the same error). Export here
# once so every gh invocation downstream is authenticated.
if [ -n "${GHCR_TOKEN:-}" ]; then
    export GH_TOKEN="$GHCR_TOKEN"
fi

# ----------------------------------------------------------------------------
# Phase functions. Each returns 0 on success, non-zero on failure;
# run_phase (lib.sh) records the phase name once on non-zero. Detail
# log lines inside a phase use log warn / log error.
# ----------------------------------------------------------------------------

# do_deactivate — FR-5.5: routing for this environment MUST stop before any
# destructive phase runs. Two PATCHes, because every pod's registry
# projection (libs/atlas-env) updates independently off a Kafka push, not a
# poll: DEACTIVATING tells atlas-ingress to stop routing to this environment
# and tells the gate to stop accepting new ownership/work for it; once every
# pod has projected that (the settle window below), DELETED removes the
# record entirely so no pod — including one that missed the DEACTIVATING
# event outright — can observe this environment again (FR-5.7).
#
# ATLAS_DEACTIVATE_SETTLE_S is deliberately NOT env.StaleAfter
# (libs/atlas-env/registry.go:10 — 120s, four missed 30s heartbeats). That
# bound covers the CONSUMER going silent; this wait covers the PRODUCER's
# message reaching every consumer, which is push-based over Kafka and
# settles in about one heartbeat interval. 35s is one 30s heartbeat plus
# margin. If Task 55 measurement shows that insufficient, raise it there —
# do not guess higher here, and do not conflate it with StaleAfter by
# reusing that constant's value for a different guarantee.

# _dcp_env_phase — echoes this environment's current control-plane phase
# (PROVISIONING/ACTIVE/DEACTIVATING/DELETED), or nothing if no record
# exists (a 404 from the collection GET) or ATLAS_UI_BASE/ATLAS_ENVIRONMENT
# are unset.
#
# This is the live, self-healing gate do_deactivate and
# do_drop_control_plane both use instead of a build-time ATLAS_MODE flag
# (task-48 fix round 1): isolated-mode PRs never register a control-plane
# environment record in the first place — only sparse mode's
# environment-record.yaml Job (task-44/47) POSTs one — so "no record"
# means exactly "isolated mode, or a sparse PR torn down before it ever
# registered," and either way there is nothing for these two phases to do.
# Checking live data can't drift from what actually got deployed the way a
# manifest flag could, and it needs no ATLAS_MODE wiring at all.
_dcp_env_phase() {
    [ -z "${ATLAS_UI_BASE:-}" ] && return 0
    [ -z "${ATLAS_ENVIRONMENT:-}" ] && return 0
    curl -fsS -H 'Accept: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" 2>/dev/null \
        | jq -r '.data.attributes.phase // empty' 2>/dev/null
}

do_deactivate() {
    if [ -z "${ATLAS_UI_BASE:-}" ] || [ -z "${ATLAS_ENVIRONMENT:-}" ]; then
        ATLAS_STEP=deactivate log error "ATLAS_UI_BASE and ATLAS_ENVIRONMENT are required to deactivate; routing may still be live when destructive phases run"
        return 1
    fi
    local phase
    phase=$(_dcp_env_phase)
    if [ -z "$phase" ]; then
        ATLAS_STEP=deactivate log info "skipped — no control-plane environment record for $ATLAS_ENVIRONMENT (isolated mode never registers one; sparse mode does via environment-record.yaml)"
        return 0
    fi
    if [ "$phase" = "DELETED" ]; then
        ATLAS_STEP=deactivate log info "already deactivated (phase=DELETED); skipping"
        return 0
    fi
    ATLAS_STEP=deactivate log info "deactivating environment=$ATLAS_ENVIRONMENT before any destructive phase (FR-5.5)"
    if ! curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        -d "{\"data\":{\"type\":\"environments\",\"id\":\"$ATLAS_ENVIRONMENT\",\"attributes\":{\"phase\":\"DEACTIVATING\"}}}" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null; then
        ATLAS_STEP=deactivate log warn "PATCH phase=DEACTIVATING failed"
        return 1
    fi
    sleep "${ATLAS_DEACTIVATE_SETTLE_S:-35}"
    if ! curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        -d "{\"data\":{\"type\":\"environments\",\"id\":\"$ATLAS_ENVIRONMENT\",\"attributes\":{\"phase\":\"DELETED\"}}}" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null; then
        ATLAS_STEP=deactivate log warn "PATCH phase=DELETED failed"
        return 1
    fi
    return 0
}

# _dcp_reclaim <list_url> <label> — GETs the JSON:API collection at
# $list_url, filters to rows whose attributes.environment equals
# ATLAS_ENVIRONMENT (client-side: none of services/tenants/templates/
# atlas-tenants expose a server-side ?environment= filter today — only
# templates supports a query filter, and it is region/version, not
# environment), and DELETEs every matching id individually.
#
# NEVER filters by Type and NEVER deletes "everything that isn't main" —
# only an exact match on THIS environment's own id, so a foreign row
# (main's, or another PR's) is never touched even when it shares a Type
# with one of this environment's rows. This also means every row for this
# environment is collected and deleted, not just the one a Deployment's
# SERVICE_ID currently points at: create_service_config (task-47) is not
# idempotent across a retried bootstrap Job, so a crashed attempt can leave
# an orphaned duplicate services row (same Type, same Environment,
# different id) behind. A SERVICE_ID-driven delete would silently leak
# every such orphan; the environment-scoped GET+filter does not, because it
# has no notion of "the current one" — it deletes every row it finds.
#
# The server-side scope.AuthorizeWrite gate (atlas-configurations,
# atlas-tenants; services/atlas-configurations/atlas.com/configurations/scope/scope.go)
# independently 403s any DELETE whose ENVIRONMENT header doesn't match the
# target row's own environment column. That gate and this loop's
# client-side filter are two independent layers; neither alone is
# sufficient — the gate cannot stop a scan that legitimately targets this
# environment's own rows, and the filter alone would be one bug away from a
# bare table sweep if the gate were ever removed.
_dcp_reclaim() {
    local list_url="$1" label="$2" rc=0
    local body ids id
    body=$(curl -fsS -H 'Accept: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        "$list_url") || { ATLAS_STEP=drop-control-plane log warn "$label: list failed"; return 1; }
    ids=$(printf '%s' "$body" | jq -r --arg env "$ATLAS_ENVIRONMENT" \
        '.data[]? | select(.attributes.environment == $env) | .id') || return 1
    if [ -z "$ids" ]; then
        ATLAS_STEP=drop-control-plane log info "$label: no rows for environment=$ATLAS_ENVIRONMENT"
        return 0
    fi
    while IFS= read -r id; do
        [ -z "$id" ] && continue
        if curl -fsS -X DELETE \
            -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
            "$list_url/$id" >/dev/null; then
            ATLAS_STEP=drop-control-plane log info "$label: deleted $id (environment=$ATLAS_ENVIRONMENT)"
        else
            ATLAS_STEP=drop-control-plane log warn "$label: delete $id failed"
            rc=1
        fi
    done <<<"$ids"
    return $rc
}

# do_drop_control_plane — reclaims this environment's rows from the
# atlas-configurations (services/tenants/templates) and atlas-tenants
# databases. In sparse mode those databases are SHARED with main (D1) — a
# mis-scoped delete here destroys main's rows, not just pollutes them. Every
# delete is scoped by the environment column via _dcp_reclaim; see that
# function's comment for the two independent layers that enforce it.
#
# Gated by _dcp_env_phase, the same live existence-check do_deactivate uses
# (task-48 fix round 1) — NOT ATLAS_MODE. Isolated-mode PRs never register
# a control-plane environment record, so the GET 404s and this phase
# correctly no-ops: do_drop_dbs already DROPs the whole per-env Postgres
# database there (including this environment's private atlas-configurations
# instance) regardless of what runs here, so a REST-based reclaim would be
# redundant even where it could reach a real record. Even in the
# (currently impossible, since no record exists) case where this ran
# against an isolated PR's own private instance, every row there carries
# Environment="" (isolated bootstrap never sends an ENVIRONMENT header),
# never "pr-<N>" — the environment-column filter in _dcp_reclaim would
# still find nothing to delete. Two independent reasons this is safe, not
# one.
do_drop_control_plane() {
    if [ -z "${ATLAS_UI_BASE:-}" ] || [ -z "${ATLAS_ENVIRONMENT:-}" ]; then
        ATLAS_STEP=drop-control-plane log error "ATLAS_UI_BASE and ATLAS_ENVIRONMENT are required; control-plane rows for this environment were not reclaimed"
        return 1
    fi
    if [ -z "$(_dcp_env_phase)" ]; then
        ATLAS_STEP=drop-control-plane log info "skipped — no control-plane environment record for $ATLAS_ENVIRONMENT (isolated mode: do_drop_dbs already destroys this environment's atlas-configurations rows)"
        return 0
    fi
    ATLAS_STEP=drop-control-plane log info "reclaiming control-plane rows for environment=$ATLAS_ENVIRONMENT"
    local rc=0
    _dcp_reclaim "$ATLAS_UI_BASE/api/configurations/services" services || rc=1
    _dcp_reclaim "$ATLAS_UI_BASE/api/configurations/tenants" configuration-tenants || rc=1
    _dcp_reclaim "$ATLAS_UI_BASE/api/configurations/templates" templates || rc=1
    _dcp_reclaim "$ATLAS_UI_BASE/api/tenants" atlas-tenants || rc=1
    return $rc
}

do_drop_dbs() {
    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        # Sparse mode shares main's Postgres databases (D1) — there is
        # nothing per-env to DROP. Log "skipped (sparse)" rather than
        # silently returning 0: a phase that reports success without doing
        # anything is indistinguishable from one that failed to find its
        # target.
        ATLAS_STEP=drop-dbs log info "skipped (sparse) — databases are shared with main"
        return 0
    fi
    ATLAS_STEP=drop-dbs log info "dropping per-env Postgres databases"
    # Probe connectivity before the per-DB loop. Postgres unreachable
    # means cleanup-targeting is broken and no other phase can be
    # trusted to reason about per-env state, so this is a hard exit.
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "SELECT 1" >/dev/null 2>&1; then
        ATLAS_STEP=drop-dbs log error "Postgres unreachable at $DB_HOST:$DB_PORT; aborting cleanup"
        exit 1
    fi
    local -a dbs
    read -ra dbs <<< "$ATLAS_DB_NAMES"
    local rc=0
    local db full
    for db in "${dbs[@]}"; do
        full="${db}-${ATLAS_ENV}"
        if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"$full\";" >/dev/null 2>&1; then
            ATLAS_STEP=drop-dbs log warn "drop $full failed"
            rc=1
        fi
    done
    return $rc
}

do_drop_topics() {
    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        # Sparse mode never suffixes topic names with ATLAS_ENV (D1,
        # FR-4.8) — topics are shared with main. Nothing per-env to
        # delete; log the skip explicitly rather than silently returning
        # 0 from a suffix scan that would find nothing anyway.
        ATLAS_STEP=drop-topics log info "skipped (sparse) — topics are shared with main"
        return 0
    fi
    ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
    local topics
    topics=$(rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_TOPICS_JQ") || return 1
    local matched
    matched=$(printf '%s\n' "$topics" | { grep -E -- "-${ATLAS_ENV}\$" || true; })
    [ -z "$matched" ] && return 0
    printf '%s\n' "$matched" | xargs -r -n 1 rpk topic delete -X brokers="$BOOTSTRAP_SERVERS"
}

do_drop_groups() {
    ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
    local groups
    groups=$(rpk group list -X brokers="$BOOTSTRAP_SERVERS" \
        | rpk_group_names_awk) || return 1
    local matched
    matched=$(printf '%s\n' "$groups" | { grep -E -- "\\[${ATLAS_ENV}\\]\$" || true; })
    [ -z "$matched" ] && return 0
    # Group names contain spaces (e.g. `Channel Service - 7e3a-0a1b [a1b2]`).
    # Can't use `xargs -n 1` because BusyBox xargs splits on whitespace and
    # would chop the name; the GNU-only `-d '\n'` workaround isn't available
    # because the bootstrap image's alpine base ships only BusyBox xargs
    # (verified via "xargs: unrecognized option: d"). while-read preserves
    # the line intact. Mirrors sweep-orphans.sh::sweep_kafka.
    local rc=0
    while IFS= read -r g; do
        [ -z "$g" ] && continue
        if ! rpk group delete -X brokers="$BOOTSTRAP_SERVERS" "$g"; then
            ATLAS_STEP=drop-groups log warn "delete group failed: $g"
            rc=1
        fi
    done <<<"$matched"
    return $rc
}

do_drop_redis() {
    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        # Sparse mode never prefixes Redis keys with ATLAS_ENV — the
        # per-env key prefix is inert there (design §9,
        # computeKeyPrefix("") is the legacy/shared path). Nothing per-env
        # to delete; log the skip explicitly rather than silently
        # returning 0 from a scan that would find nothing anyway.
        ATLAS_STEP=drop-redis log info "skipped (sparse) — keys are shared with main"
        return 0
    fi
    ATLAS_STEP=drop-redis log info "deleting per-env Redis keys"
    redis-cli -u "redis://$REDIS_URL" --scan --pattern "${ATLAS_ENV}:*" \
        | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL
}

do_drop_images() {
    if [ -z "${ATLAS_SERVICES:-}" ] || [ -z "${GHCR_TOKEN:-}" ]; then
        ATLAS_STEP=drop-images log info "no ATLAS_SERVICES/GHCR_TOKEN; skipping"
        return 0
    fi
    ATLAS_STEP=drop-images log info "deleting per-PR ghcr image tags"
    local -a svcs
    IFS=',' read -ra svcs <<< "$ATLAS_SERVICES"
    local svc vid rc=0
    for svc in "${svcs[@]}"; do
        while read -r vid; do
            gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" \
                >/dev/null 2>&1 || ATLAS_STEP=drop-images log warn "delete ${svc}/${vid} failed"
        done < <(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${PR_NUMBER}-\")) | .id" \
            2>/dev/null) || rc=1
    done
    return $rc
}

do_drop_dns() {
    if [ -z "${PIHOLE_API_BASE_1:-}" ] || [ -z "${PIHOLE_TOKEN_1:-}" ]; then
        ATLAS_STEP=drop-dns log info "no Pi-hole creds; skipping"
        return 0
    fi
    ATLAS_STEP=drop-dns log info "removing Pi-hole A records"
    local host="${PR_NUMBER}.atlas.home"
    local rc=0
    local i base_var token_var base token sid entry encoded_entry
    for i in 1 2; do
        base_var="PIHOLE_API_BASE_$i"
        token_var="PIHOLE_TOKEN_$i"
        base="${!base_var:-}"
        token="${!token_var:-}"
        [ -z "$base" ] && continue
        [ -z "$token" ] && continue
        sid=$(curl -k -fsS -X POST "$base/api/auth" \
            -H "Content-Type: application/json" \
            -d "{\"password\":\"$token\"}" 2>/dev/null \
            | jq -r '.session.sid // empty')
        if [ -z "$sid" ]; then
            ATLAS_STEP=drop-dns log warn "Pi-hole $i: auth failed, skipping host removal"
            rc=1
            continue
        fi
        entry=$(curl -k -fsS -H "X-FTL-SID: $sid" "$base/api/config/dns/hosts" \
            | jq -r ".config.dns.hosts[]? | select(endswith(\" $host\"))" | head -1)
        if [ -n "$entry" ]; then
            encoded_entry=$(printf '%s' "$entry" | sed 's/ /%20/g')
            curl -k -fsS -X DELETE -H "X-FTL-SID: $sid" \
                "$base/api/config/dns/hosts/$encoded_entry" || {
                    ATLAS_STEP=drop-dns log warn "Pi-hole $i delete failed for $host"
                    rc=1
                }
        fi
    done
    return $rc
}

do_drop_branch() {
    if [ -z "${PR_NUMBER:-}" ] || [ -z "${GHCR_TOKEN:-}" ]; then
        ATLAS_STEP=drop-branch log info "no GHCR_TOKEN; skipping"
        return 0
    fi
    ATLAS_STEP=drop-branch log info "deleting bot/pr-${PR_NUMBER}-resolved"
    local err
    local branch_deleted=0
    if err=$(gh api --method DELETE \
        -H "Authorization: Bearer ${GHCR_TOKEN}" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
        2>&1); then
        branch_deleted=1
    else
        case "$err" in
            *"Reference does not exist"*|*"Branch not found"*|*"404"*)
                # Already gone; treat as success for the race below — the
                # Application targets a missing branch either way.
                branch_deleted=1
                ;;
            *)
                ATLAS_STEP=drop-branch log warn "branch delete: $err"
                return 1
                ;;
        esac
    fi

    # Once the bot branch is gone, Argo CD's post-delete-finalizer drain
    # for atlas-pr-${PR_NUMBER} CANNOT re-render the source manifest. Its
    # next reconcile will record `DeletionError: failed to generate
    # manifest ... unable to resolve 'bot/pr-${PR_NUMBER}-resolved' to a
    # commit SHA` and the finalizers stay attached forever — the
    # "Source-branch-missing scenario" in runbook §9.4. PR 522 hit this
    # on 2026-05-27 and sat Terminating for 10h until a manual
    # finalizer-patch.
    #
    # Pre-empt the race by patching the post-delete finalizers ourselves
    # NOW, while we still have the Application's identity (PR_NUMBER) and
    # the cleanup Job is still running with its argocd-ns RBAC. After
    # this, the Application can GC even if Argo's drain fails to render.
    # The resources-finalizer drain already ran (we're in PostDelete);
    # the per-env namespace is gone; this is just removing the
    # bookkeeping finalizers Argo would otherwise drop after its
    # final-render verification.
    if [ "$branch_deleted" = "1" ] && command -v kubectl >/dev/null 2>&1; then
        ATLAS_STEP=drop-branch log info \
            "pre-empting post-delete-finalizer drain on atlas-pr-${PR_NUMBER}"
        kubectl -n argocd patch application.argoproj.io "atlas-pr-${PR_NUMBER}" \
            --type=merge -p '{"metadata":{"finalizers":[]}}' >/dev/null 2>&1 \
            || ATLAS_STEP=drop-branch log warn \
                "finalizer patch failed; manual recovery may be required (see runbook §9.4)"
    fi
    return 0
}

# ----------------------------------------------------------------------------
# Orchestration. PHASES is interleaved <phase_name> <function_name>.
# ----------------------------------------------------------------------------
PHASES=(
    deactivate           do_deactivate          # FR-5.5 — must be first
    drop-control-plane   do_drop_control_plane
    drop-dbs             do_drop_dbs            # no-op in sparse mode
    drop-topics          do_drop_topics         # no-op in sparse mode
    drop-groups          do_drop_groups
    drop-redis           do_drop_redis          # no-op in sparse mode
    drop-images          do_drop_images
    drop-dns             do_drop_dns
    drop-branch          do_drop_branch
)
TOTAL=$(( ${#PHASES[@]} / 2 ))
ATLAS_PHASE_ERRORS=()
for ((i=0; i<${#PHASES[@]}; i+=2)); do
    run_phase "${PHASES[i]}" "${PHASES[i+1]}"
done
summarize_phases "$TOTAL"
exit $?
