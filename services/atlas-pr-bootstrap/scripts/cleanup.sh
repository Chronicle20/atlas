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
# shellcheck source=env-record.sh
. "$(dirname "$0")/env-record.sh"

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

# _dcp_env_get — echoes this environment's environments record GET response
# body (the full JSON:API document), or nothing if no record exists (a 404
# from the GET) or ATLAS_UI_BASE/ATLAS_ENVIRONMENT are unset. Exit status
# mirrors curl's.
#
# Body now lives in env-record.sh (env_record_get), shared with
# bootstrap.sh — see that file for the full rationale.
_dcp_env_get() {
    env_record_get
}

# _dcp_env_phase — echoes this environment's current control-plane phase
# (PROVISIONING/ACTIVE/DEACTIVATING/DELETED), or nothing if no record
# exists.
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
#
# This is a DIFFERENT question than ATLAS_MODE answers for do_drop_dbs/
# do_drop_topics/do_drop_redis below (task-48 fix round 2 Important 3):
# those three ask "is this environment's Postgres/Kafka/Redis footprint
# private or shared with main" — a build-time deployment-topology fact with
# no live signal to check (sparse mode's shared resources have no per-env
# marker to probe; the absence IS the fact). This gate asks "does a
# control-plane record for this environment currently exist" — a fact that
# IS live-checkable, and checking it live is strictly more robust than
# trusting a flag that could drift from what was actually deployed. Two
# different questions, answered two different ways; not two competing
# signals for the same decision. Proof that swapping ATLAS_MODE for this
# live check left isolated-mode teardown behaviour unchanged: no file under
# deploy/k8s/overlays/pr/ (isolated mode's overlay) mentions "environments"
# at all — only pr-sparse's environment-record.yaml POSTs one, and only to
# the baseline's atlas-configurations (ATLAS_UI_BASE always resolves to the
# baseline, never a PR's own instance — see postdelete-cleanup.yaml). An
# isolated PR's GET against baseline/environments/pr-<N> 404s exactly as it
# would have skipped under the old ATLAS_MODE=isolated gate; the pinned
# bats test "cleanup.sh drop-control-plane and deactivate are skipped when
# no control-plane environment record exists" exercises this via the
# suite's own 404-by-default curl stub.
_dcp_env_phase() {
    local body
    body=$(_dcp_env_get) || return 0
    printf '%s' "$body" | jq -r '.data.attributes.phase // empty' 2>/dev/null
}

# _dcp_patch_phase <new_phase> <baseline> <namespace> <tenant> <overrides_json>
# — PATCHes the environments record to <new_phase>, carrying the OTHER four
# attributes through unchanged. environments.RestModel's fields are
# non-pointer (environments/rest.go), so ParseInput unmarshals a PATCH body
# into a fresh zero-value struct first: any attribute omitted from the body
# is zeroed, not left alone (environments/administrator.go's update() doc
# comment is explicit about this). A phase-only body — the shape task-48's
# own plan Step 3 sample showed — would silently wipe baseline/namespace/
# tenant/overrides on every DEACTIVATING/DELETED transition (task-48 fix
# round 2 Critical 2). GET-then-PATCH-with-everything is the fix.
#
# Body now lives in env-record.sh (env_record_patch), shared with
# bootstrap.sh — see that file for the full rationale.
_dcp_patch_phase() {
    env_record_patch "$@"
}

do_deactivate() {
    if [ -z "${ATLAS_UI_BASE:-}" ] || [ -z "${ATLAS_ENVIRONMENT:-}" ]; then
        ATLAS_STEP=deactivate log error "ATLAS_UI_BASE and ATLAS_ENVIRONMENT are required to deactivate; routing may still be live when destructive phases run"
        return 1
    fi
    local body phase baseline namespace tenant overrides
    body=$(_dcp_env_get)
    phase=$(printf '%s' "${body:-}" | jq -r '.data.attributes.phase // empty' 2>/dev/null)
    if [ -z "$phase" ]; then
        ATLAS_STEP=deactivate log info "skipped — no control-plane environment record for $ATLAS_ENVIRONMENT (isolated mode never registers one; sparse mode does via environment-record.yaml)"
        return 0
    fi
    if [ "$phase" = "DELETED" ]; then
        ATLAS_STEP=deactivate log info "already deactivated (phase=DELETED); skipping"
        return 0
    fi
    baseline=$(printf '%s' "$body" | jq -r '.data.attributes.baseline // ""' 2>/dev/null)
    namespace=$(printf '%s' "$body" | jq -r '.data.attributes.namespace // ""' 2>/dev/null)
    tenant=$(printf '%s' "$body" | jq -r '.data.attributes.tenant // ""' 2>/dev/null)
    overrides=$(printf '%s' "$body" | jq -c '.data.attributes.overrides // {}' 2>/dev/null)
    ATLAS_STEP=deactivate log info "deactivating environment=$ATLAS_ENVIRONMENT before any destructive phase (FR-5.5)"
    if ! _dcp_patch_phase DEACTIVATING "$baseline" "$namespace" "$tenant" "$overrides"; then
        ATLAS_STEP=deactivate log warn "PATCH phase=DEACTIVATING failed"
        return 1
    fi
    sleep "${ATLAS_DEACTIVATE_SETTLE_S:-35}"
    if ! _dcp_patch_phase DELETED "$baseline" "$namespace" "$tenant" "$overrides"; then
        ATLAS_STEP=deactivate log warn "PATCH phase=DELETED failed"
        return 1
    fi
    return 0
}

# _dcp_reclaim <list_url> <label> — GETs the JSON:API collection at
# $list_url one page at a time (page[size]=250, paginate.MaxPageSize —
# libs/atlas-rest/server/paginate/params.go — vs. the 50-row
# paginate.DefaultPageSize every one of these four list endpoints falls
# back to unpaginated), filters each page to rows whose
# attributes.environment equals ATLAS_ENVIRONMENT (client-side: none of
# services/tenants/templates/atlas-tenants expose a server-side
# ?environment= filter today — only templates supports a query filter, and
# it is region/version, not environment), and DELETEs every matching id
# individually. Stops once meta.page.last is reached (task-48 fix round 2
# Important 1) — a single unpaginated GET would silently miss every row
# past page 1 once a resource's row count exceeds DefaultPageSize, which is
# realistic precisely because this task exists to clean up the duplicate-row
# leak task-47 left behind.
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
    local page=1 last=1 found_any=0
    local body ids id
    while :; do
        body=$(curl -fsSg -H 'Accept: application/vnd.api+json' \
            -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
            "${list_url}?page[size]=250&page[number]=${page}") \
            || { ATLAS_STEP=drop-control-plane log warn "$label: list failed (page=$page)"; return 1; }
        ids=$(printf '%s' "$body" | jq -r --arg env "$ATLAS_ENVIRONMENT" \
            '.data[]? | select(.attributes.environment == $env) | .id') || return 1
        if [ -n "$ids" ]; then
            found_any=1
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
        fi
        last=$(printf '%s' "$body" | jq -r '.meta.page.last // 1' 2>/dev/null)
        case "$last" in (''|*[!0-9]*) last=1 ;; esac
        [ "$page" -ge "$last" ] && break
        page=$((page + 1))
    done
    if [ "$found_any" -eq 0 ]; then
        ATLAS_STEP=drop-control-plane log info "$label: no rows for environment=$ATLAS_ENVIRONMENT"
    fi
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

# do_sweep_tenant — reclaims THIS environment's tenant's rows from the
# shared databases sparse mode never DROPs (do_drop_dbs skips entirely
# there, see below). Isolated mode's do_drop_dbs already destroys this
# environment's entire private database, tables and all, so there is
# nothing left for this phase to do there.
#
# Gated on the SAME live control-plane existence-check do_drop_control_plane
# and do_deactivate use (_dcp_env_phase, task-48 fix round 1), NOT
# ATLAS_MODE. This deliberately diverges from do_drop_dbs/do_drop_topics/
# do_drop_redis below, which stay ATLAS_MODE-gated (task-48 fix round 2
# Important 3: those three ask "is this environment's Postgres/Kafka/Redis
# footprint private or shared with main", answerable only from a manifest,
# not from anything live). do_sweep_tenant's question is different — "does
# a control-plane environment record exist for this environment" — and that
# IS live-checkable: only sparse mode's environment-record.yaml Job
# registers one at all (isolated bootstrap never does), so record-presence
# already implies sparse. Bug: the atlas-pr-cleanup Job (deploy/k8s/
# overlays/pr-cleanup/postdelete-cleanup.yaml) is one shared manifest for
# BOTH isolated and sparse PRs — unlike atlas-pr-bootstrap, which gets
# ATLAS_MODE from a pr-sparse-only kustomize patch, there is no per-mode
# overlay to patch it onto here. ATLAS_MODE therefore never reached this
# Job and always defaulted to "isolated", so every sparse teardown took
# the skip branch below and never reclaimed anything (confirmed verbatim
# in PR-1412's cleanup log: "sweep-tenant skipped (isolated)"). The live
# check does not have this gap: it needs no wiring, only the record this
# phase already GETs on the next line to read the tenant id.
do_sweep_tenant() {
    # No record means either isolated mode (do_drop_dbs already destroys
    # this environment's entire private database, tenant rows included) or
    # a sparse PR torn down before it ever registered a tenant
    # (environment-record.yaml never ran, or this is a re-run after a
    # prior teardown already deleted it) — nothing was ever written under
    # a tenant that doesn't exist yet, so this is "nothing to do", not a
    # failure.
    if [ -z "$(_dcp_env_phase)" ]; then
        ATLAS_STEP=sweep-tenant log info "skipped — no control-plane environment record for $ATLAS_ENVIRONMENT (nothing registered to reclaim)"
        return 0
    fi
    local body tenant
    body=$(_dcp_env_get)
    tenant=$(printf '%s' "${body:-}" | jq -r '.data.attributes.tenant // empty' 2>/dev/null)
    if [ -z "$tenant" ]; then
        ATLAS_STEP=sweep-tenant log warn "control-plane environment record for $ATLAS_ENVIRONMENT has no tenant attribute; cannot reclaim tenant-keyed rows"
        return 1
    fi
    ATLAS_STEP=sweep-tenant log info "reclaiming tenant-keyed rows for tenant=$tenant across shared databases"
    if ! "$(dirname "$0")/sweep-orphans.sh" --sweep-tenant "$tenant" --apply; then
        ATLAS_STEP=sweep-tenant log warn "sweep-orphans.sh --sweep-tenant reported failures"
        return 1
    fi
    return 0
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
        # Sparse mode's Redis keys are the BASELINE's: libs/atlas-redis
        # prefixes them with ATLAS_REDIS_ENV, which the sparse overlay sets
        # to the baseline's environment id, so they read `main:atlas:…`.
        # The scan below looks for `${ATLAS_ENV}:*` — this environment's own
        # prefix — and in sparse mode nothing is keyed that way. Skipping is
        # therefore not merely a no-op but load-bearing: a scan on the
        # baseline's prefix would delete main's live state.
        #
        # (This comment previously cited design §9's "the per-env prefix is
        # inert in sparse mode, computeKeyPrefix("") is the shared path".
        # That premise was retracted on 2026-08-20 — the overlay set
        # ATLAS_ENV per container regardless, and the baseline's prefix is
        # `main:atlas`, not the empty-env `atlas`. See docs/tasks/
        # task-232-sparse-ephemeral-environments/bug-sparse-baseline-scoping.md.
        # The skip was correct then and is correct now, for a better reason.)
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
    sweep-tenant         do_sweep_tenant        # no-op in isolated mode; sparse-only reclaim
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
