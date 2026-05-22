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

# ----------------------------------------------------------------------------
# Phase functions. Each returns 0 on success, non-zero on failure;
# run_phase (lib.sh) records the phase name once on non-zero. Detail
# log lines inside a phase use log warn / log error.
# ----------------------------------------------------------------------------

do_drop_dbs() {
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
    groups=$(rpk group list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_GROUPS_JQ") || return 1
    local matched
    matched=$(printf '%s\n' "$groups" | { grep -E -- "\\[${ATLAS_ENV}\\]\$" || true; })
    [ -z "$matched" ] && return 0
    printf '%s\n' "$matched" | xargs -r -d '\n' -n 1 rpk group delete -X brokers="$BOOTSTRAP_SERVERS"
}

do_drop_redis() {
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
    if ! err=$(gh api --method DELETE \
        -H "Authorization: Bearer ${GHCR_TOKEN}" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
        2>&1); then
        case "$err" in
            *"Reference does not exist"*|*"Branch not found"*|*"404"*)
                return 0
                ;;
            *)
                ATLAS_STEP=drop-branch log warn "branch delete: $err"
                return 1
                ;;
        esac
    fi
    return 0
}

# ----------------------------------------------------------------------------
# Orchestration. PHASES is interleaved <phase_name> <function_name>.
# ----------------------------------------------------------------------------
PHASES=(
    drop-dbs     do_drop_dbs
    drop-topics  do_drop_topics
    drop-groups  do_drop_groups
    drop-redis   do_drop_redis
    drop-images  do_drop_images
    drop-dns     do_drop_dns
    drop-branch  do_drop_branch
)
TOTAL=$(( ${#PHASES[@]} / 2 ))
ATLAS_PHASE_ERRORS=()
for ((i=0; i<${#PHASES[@]}; i+=2)); do
    run_phase "${PHASES[i]}" "${PHASES[i+1]}"
done
summarize_phases "$TOTAL"
exit $?
