#!/usr/bin/env bash
# Atlas PR-env orphan sweep. Codifies the May-19 recovery: enumerate (and
# optionally delete) per-env state for one or more PR numbers.
#
# Usage:
#   sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]
#
# Without --apply (default): lists everything that would be deleted.
# With --apply: deletes it. Idempotent — safe to re-run after a partial sweep.
#
# Required env (same names cleanup.sh uses; defaults match cluster reality):
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
#   ATLAS_DB_NAMES                    — space-separated base DB names
#   BOOTSTRAP_SERVERS                 — Kafka bootstrap
#   REDIS_URL                         — host:port (NOT a URL)
#   GHCR_TOKEN                        — GitHub PAT (Contents+Packages write)
#   ATLAS_SERVICES                    — comma-separated service names
#   PIHOLE_API_BASE_1 / PIHOLE_TOKEN_1 / PIHOLE_API_BASE_2 / PIHOLE_TOKEN_2
#
# DRY_RUN_NO_INFRA=1 short-circuits external-command phases (testing only).

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

APPLY=0
PR_NUMBERS=()

usage() {
    cat <<'EOF'
Usage: sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]

  --apply        Actually delete state. Without this flag, sweep is list-only.
  PR_NUMBER      One or more positive integers.

Without --apply, all phases print what they would do, one resource per line,
prefixed with the phase name (drop-dbs, drop-topics, drop-groups,
drop-redis, drop-images, drop-dns, drop-app-finalizers, drop-branch).
Suitable for piping through `tee` or `diff` for visual review before re-running
with --apply.
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --apply) APPLY=1 ; shift ;;
        --list)  APPLY=0 ; shift ;;     # explicit form, same as default
        -h|--help) usage ; exit 0 ;;
        --) shift ; break ;;
        -*) echo "unknown flag: $1" >&2 ; usage >&2 ; exit 2 ;;
        *)  PR_NUMBERS+=("$1") ; shift ;;
    esac
done

if [ "${#PR_NUMBERS[@]}" -eq 0 ]; then
    usage >&2
    exit 2
fi

for n in "${PR_NUMBERS[@]}"; do
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
        echo "PR number '$n' is not a number" >&2
        usage >&2
        exit 2
    fi
done

sweep_pr() {
    local pr_number="$1"
    local env_hash
    env_hash="$(compute_atlas_env "$pr_number")"
    ATLAS_ENV="$env_hash" ATLAS_STEP=init log info \
        "sweeping PR ${pr_number} (ATLAS_ENV=${env_hash}) apply=${APPLY}"

    # Phase implementations are added in Task 7. Each phase MUST:
    #   - read APPLY (0 = list-only, 1 = delete)
    #   - prefix each enumerated resource with its phase name
    #   - tolerate missing resources (idempotent)
    #   - skip the phase entirely if its required env vars are unset
    #
    # In DRY_RUN_NO_INFRA mode (testing), skip every infra call but still
    # emit the env_hash line above so the harness can grep for it.
    if [ -n "${DRY_RUN_NO_INFRA:-}" ]; then
        return 0
    fi

    sweep_pg "$pr_number" "$env_hash"
    sweep_kafka "$pr_number" "$env_hash"
    sweep_redis "$pr_number" "$env_hash"
    sweep_ghcr "$pr_number" "$env_hash"
    sweep_pihole "$pr_number" "$env_hash"
    sweep_app_finalizer "$pr_number" "$env_hash"
    sweep_branch "$pr_number" "$env_hash"
}

# Phase implementations (Task 7).

sweep_pg() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${DB_HOST:-}" ] && return 0
    DB_USER="$(printf '%s' "${DB_USER:-}" | tr -d ' \r\n')"
    DB_PASSWORD="$(printf '%s' "${DB_PASSWORD:-}" | tr -d ' \r\n')"
    [ -z "$DB_USER" ] && return 0
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dbs log info "scanning Postgres for orphans (PR $pr_number)"
    local dbs
    dbs=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc \
        "SELECT datname FROM pg_database WHERE datname ~ '-${env_hash}\$';") || return 0
    while IFS= read -r db; do
        [ -z "$db" ] && continue
        echo "drop-dbs ${db}"
        if [ "$APPLY" = "1" ]; then
            PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
                -c "DROP DATABASE IF EXISTS \"$db\" WITH (FORCE);" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dbs log warn "drop $db failed"
        fi
    done <<<"$dbs"
}

sweep_kafka() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${BOOTSTRAP_SERVERS:-}" ] && return 0
    if ! command -v kafka-topics.sh >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "kafka-topics.sh not on PATH; skipping"
        return 0
    fi
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log info "scanning Kafka topics"
    local topics
    topics=$(kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
        | grep -E -- "-${env_hash}\$" || true)
    while IFS= read -r t; do
        [ -z "$t" ] && continue
        echo "drop-topics ${t}"
        if [ "$APPLY" = "1" ]; then
            kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --topic "$t" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "delete topic $t failed"
        fi
    done <<<"$topics"

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log info "scanning Kafka consumer groups"
    local groups
    groups=$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
        | grep -E -- "\\[${env_hash}\\]\$" || true)
    while IFS= read -r g; do
        [ -z "$g" ] && continue
        echo "drop-groups ${g}"
        if [ "$APPLY" = "1" ]; then
            kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --group "$g" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log warn "delete group failed"
        fi
    done <<<"$groups"
}

sweep_redis() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${REDIS_URL:-}" ] && return 0
    if ! command -v redis-cli >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log warn "redis-cli not on PATH; skipping"
        return 0
    fi
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log info "scanning Redis"
    local keys
    keys=$(redis-cli -u "redis://$REDIS_URL" --scan --pattern "${env_hash}:*" || true)
    while IFS= read -r k; do
        [ -z "$k" ] && continue
        echo "drop-redis ${k}"
    done <<<"$keys"
    if [ "$APPLY" = "1" ] && [ -n "$keys" ]; then
        printf '%s\n' "$keys" | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL >/dev/null || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log warn "DEL failed"
    fi
}

sweep_ghcr() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${ATLAS_SERVICES:-}" ] && return 0
    [ -z "${GHCR_TOKEN:-}" ] && return 0
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-images log info "scanning ghcr tags pr-${pr_number}-*"
    local svcs
    IFS=',' read -ra svcs <<<"$ATLAS_SERVICES"
    for svc in "${svcs[@]}"; do
        local vids
        vids=$(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${pr_number}-\")) | [.id, (.metadata.container.tags|join(\",\"))] | @tsv" \
            2>/dev/null) || continue
        while IFS=$'\t' read -r vid tags; do
            [ -z "$vid" ] && continue
            echo "drop-images ${svc}/${vid} (${tags})"
            if [ "$APPLY" = "1" ]; then
                gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                    "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" \
                    >/dev/null 2>&1 || \
                    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-images log warn "delete ${svc}/${vid} failed"
            fi
        done <<<"$vids"
    done
}

sweep_pihole() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${PIHOLE_API_BASE_1:-}" ] && return 0
    [ -z "${PIHOLE_TOKEN_1:-}" ] && return 0
    local host="${pr_number}.atlas.home"
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log info "scanning Pi-hole hosts for ${host}"
    for i in 1 2; do
        local base_var="PIHOLE_API_BASE_$i"
        local token_var="PIHOLE_TOKEN_$i"
        local base="${!base_var:-}"
        local token="${!token_var:-}"
        [ -z "$base" ] && continue
        [ -z "$token" ] && continue
        local sid
        sid=$(curl -k -fsS -X POST "$base/api/auth" \
            -H "Content-Type: application/json" \
            -d "{\"password\":\"$token\"}" 2>/dev/null \
            | jq -r '.session.sid // empty')
        [ -z "$sid" ] && { ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log warn "pihole $i auth failed"; continue; }
        local entry
        entry=$(curl -k -fsS -H "X-FTL-SID: $sid" "$base/api/config/dns/hosts" \
            | jq -r ".config.dns.hosts[]? | select(endswith(\" $host\"))" | head -1)
        [ -z "$entry" ] && continue
        echo "drop-dns pihole-${i} ${entry}"
        if [ "$APPLY" = "1" ]; then
            local enc
            enc=$(printf '%s' "$entry" | sed 's/ /%20/g')
            curl -k -fsS -X DELETE -H "X-FTL-SID: $sid" \
                "$base/api/config/dns/hosts/$enc" >/dev/null 2>&1 || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log warn "pihole $i delete failed"
        fi
    done
}

sweep_app_finalizer() {
    local pr_number="$1"
    local env_hash="$2"
    command -v kubectl >/dev/null 2>&1 || return 0
    if ! kubectl -n argocd get application.argoproj.io "atlas-pr-${pr_number}" \
        >/dev/null 2>&1; then
        return 0
    fi
    echo "drop-app-finalizers atlas-pr-${pr_number}"
    if [ "$APPLY" = "1" ]; then
        kubectl -n argocd patch application.argoproj.io "atlas-pr-${pr_number}" \
            --type=merge -p '{"metadata":{"finalizers":[]}}' >/dev/null 2>&1 || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-app-finalizers log warn \
                "patch atlas-pr-${pr_number} failed"
    fi
}

sweep_branch() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${GHCR_TOKEN:-}" ] && return 0
    # Check existence first so list mode reports honestly.
    local status
    status=$(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${pr_number}-resolved" \
        --jq '.ref // empty' 2>/dev/null) || status=""
    [ -z "$status" ] && return 0
    echo "drop-branch bot/pr-${pr_number}-resolved"
    if [ "$APPLY" = "1" ]; then
        gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
            "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${pr_number}-resolved" \
            >/dev/null 2>&1 || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-branch log warn \
                "delete bot/pr-${pr_number}-resolved failed"
    fi
}

for n in "${PR_NUMBERS[@]}"; do
    sweep_pr "$n"
done

ATLAS_STEP=done log info "sweep complete"
