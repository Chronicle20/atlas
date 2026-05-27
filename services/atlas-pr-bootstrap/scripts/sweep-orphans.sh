#!/usr/bin/env bash
# Atlas PR-env orphan sweep. Codifies the May-19 recovery: enumerate (and
# optionally delete) per-env state for one or more PR numbers.
#
# Usage:
#   sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]
#   sweep-orphans.sh --minio [--apply]
#
# Without --apply (default): lists everything that would be deleted.
# With --apply: deletes it. Idempotent — safe to re-run after a partial sweep.
#
# --minio scans MinIO buckets atlas-wz/atlas-assets/atlas-renders for
# orphan per-tenant prefixes (UUIDs not present in atlas-main and aged
# past the safety window). See `sweep_minio` for details and issue #596.
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
# Required env for --minio:
#   MINIO_ENDPOINT                    — host:port (NOT a URL)
#   MINIO_ACCESS_KEY, MINIO_SECRET_KEY — MinIO credentials with delete
#                                       access on the per-tenant prefixes
#   ATLAS_MAIN_TENANTS_URL            — REST endpoint for atlas-main's
#                                       atlas-tenants (default:
#                                       http://atlas-tenants.atlas-main.svc.cluster.local:8080/api/tenants)
#   MINIO_TENANT_SAFETY_WINDOW_SEC    — seconds; UUIDs touched within
#                                       this window are skipped (default 7200)
#
# DRY_RUN_NO_INFRA=1 short-circuits external-command phases (testing only).

set -uo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

APPLY=0
MINIO_MODE=0
PR_NUMBERS=()

usage() {
    cat <<'EOF'
Usage: sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]
       sweep-orphans.sh --minio [--apply]

  --apply        Actually delete state. Without this flag, sweep is list-only.
  --minio        Scan MinIO atlas-wz/atlas-assets/atlas-renders for orphan
                 per-tenant prefixes (no PR_NUMBER needed). Cross-references
                 against atlas-main tenants and skips UUIDs touched within
                 MINIO_TENANT_SAFETY_WINDOW_SEC (default 7200s).
  PR_NUMBER      One or more positive integers.

Without --apply, all phases print what they would do, one resource per line,
prefixed with the phase name (drop-dbs, drop-topics, drop-groups,
drop-redis, drop-images, drop-dns, drop-app-finalizers, drop-branch,
drop-minio).
Suitable for piping through `tee` or `diff` for visual review before re-running
with --apply.
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --apply) APPLY=1 ; shift ;;
        --list)  APPLY=0 ; shift ;;     # explicit form, same as default
        --minio) MINIO_MODE=1 ; shift ;;
        -h|--help) usage ; exit 0 ;;
        --) shift ; break ;;
        -*) echo "unknown flag: $1" >&2 ; usage >&2 ; exit 2 ;;
        *)  PR_NUMBERS+=("$1") ; shift ;;
    esac
done

if [ "$MINIO_MODE" = "1" ]; then
    # --minio takes no PR_NUMBER args.
    if [ "${#PR_NUMBERS[@]}" -gt 0 ]; then
        echo "--minio takes no PR_NUMBER args (got ${PR_NUMBERS[*]})" >&2
        usage >&2
        exit 2
    fi
elif [ "${#PR_NUMBERS[@]}" -eq 0 ]; then
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

# gh CLI requires its own credentials even when an explicit `-H
# "Authorization: Bearer …"` header is passed on the request — without
# GH_TOKEN/GITHUB_TOKEN in env it prompts for `gh auth login` and exits
# non-zero. Mirror cleanup.sh: export GH_TOKEN once so every gh
# invocation in sweep_ghcr / sweep_branch is authenticated.
if [ -n "${GHCR_TOKEN:-}" ]; then
    export GH_TOKEN="$GHCR_TOKEN"
fi

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
    if ! command -v rpk >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "rpk not on PATH; skipping"
        return 0
    fi

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log info "scanning Kafka topics"
    local topics
    topics=$(rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_TOPICS_JQ" \
        | { grep -E -- "-${env_hash}\$" || true; })
    while IFS= read -r t; do
        [ -z "$t" ] && continue
        echo "drop-topics ${t}"
        if [ "$APPLY" = "1" ]; then
            rpk topic delete -X brokers="$BOOTSTRAP_SERVERS" "$t" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "delete topic $t failed"
        fi
    done <<<"$topics"

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log info "scanning Kafka consumer groups"
    local groups
    groups=$(rpk group list -X brokers="$BOOTSTRAP_SERVERS" \
        | rpk_group_names_awk \
        | { grep -E -- "\\[${env_hash}\\]\$" || true; })
    while IFS= read -r g; do
        [ -z "$g" ] && continue
        echo "drop-groups ${g}"
        if [ "$APPLY" = "1" ]; then
            rpk group delete -X brokers="$BOOTSTRAP_SERVERS" "$g" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log warn "delete group $g failed"
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

# sweep_minio enumerates per-tenant UUID prefixes under MinIO buckets
# (atlas-wz, atlas-assets, atlas-renders) and deletes any UUID that
# is BOTH:
#   - not present in atlas-main's atlas-tenants list (the long-lived
#     env's tenant UUIDs — these must never be touched), AND
#   - aged past MINIO_TENANT_SAFETY_WINDOW_SEC (default 7200s / 2h)
#     to protect bringups in progress whose data-ingest is still
#     writing per-tenant data.
#
# We don't enumerate per-PR atlas-tenants services because by the
# time this sweep runs (cron'd hourly, or after a wedged teardown),
# the PR env namespace is being deleted; querying its atlas-tenants
# would race with namespace termination. The age window covers
# bringups in flight; orphans older than the window cannot belong to
# an active env.
#
# Cleanup target: 13 GiB of leaked tenant data observed on 2026-05-26
# (see issue #596). Reclaims storage until atlas-data's per-tenant
# DELETE endpoint is wired into cleanup.sh (the cleaner long-term
# fix; see runbook §9.11 known follow-ups).
sweep_minio() {
    [ -z "${MINIO_ENDPOINT:-}" ] && {
        ATLAS_STEP=drop-minio log info "MINIO_ENDPOINT not set; skipping"
        return 0
    }
    [ -z "${MINIO_ACCESS_KEY:-}" ] && [ -z "${MINIO_SECRET_KEY:-}" ] && {
        ATLAS_STEP=drop-minio log info "MinIO credentials not set; skipping"
        return 0
    }
    if ! command -v mc >/dev/null 2>&1; then
        ATLAS_STEP=drop-minio log warn "mc not on PATH; skipping"
        return 0
    fi

    local main_url="${ATLAS_MAIN_TENANTS_URL:-http://atlas-tenants.atlas-main.svc.cluster.local:8080/api/tenants}"
    local safety_sec="${MINIO_TENANT_SAFETY_WINDOW_SEC:-7200}"

    ATLAS_STEP=drop-minio log info \
        "scanning MinIO for orphan tenants (main=${main_url}, safety_window_sec=${safety_sec})"

    # Fetch active tenant UUIDs from atlas-main. Treat fetch failure
    # as a hard stop — without the protected list we'd risk deleting
    # main's tenants.
    local active_uuids
    if ! active_uuids=$(curl -fsS -H 'Accept: application/vnd.api+json' "$main_url" \
        | jq -r '.data[].id' 2>/dev/null); then
        ATLAS_STEP=drop-minio log warn \
            "fetch active tenants from ${main_url} failed; aborting MinIO sweep"
        return 1
    fi
    if [ -z "$active_uuids" ]; then
        ATLAS_STEP=drop-minio log warn \
            "no active tenants returned by ${main_url}; aborting MinIO sweep (refusing to operate on empty allowlist)"
        return 1
    fi

    # Configure mc alias under a private CONFIG_DIR so the host's mc
    # config (if any) isn't touched. http vs https is keyed off the
    # endpoint scheme; default to http for in-cluster MinIO.
    export MC_CONFIG_DIR="${MC_CONFIG_DIR:-/tmp/.mc-sweep}"
    mkdir -p "$MC_CONFIG_DIR"
    local mc_endpoint="$MINIO_ENDPOINT"
    case "$mc_endpoint" in
        http://*|https://*) ;;
        *) mc_endpoint="http://${mc_endpoint}" ;;
    esac
    mc alias set bee "$mc_endpoint" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null 2>&1 || {
        ATLAS_STEP=drop-minio log warn "mc alias set failed; aborting MinIO sweep"
        return 1
    }

    local now_epoch
    now_epoch=$(date -u +%s)

    local rc=0
    for bucket in atlas-wz atlas-assets atlas-renders; do
        # Enumerate UUID prefixes; tolerate empty/missing bucket.
        local uuid_lines
        uuid_lines=$(mc ls "bee/${bucket}/tenants/" 2>/dev/null \
            | awk '{print $NF}' \
            | tr -d '/' \
            | grep -E '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' \
            || true)
        while IFS= read -r uuid; do
            [ -z "$uuid" ] && continue

            # Skip if this UUID is an active main tenant.
            if printf '%s\n' "$active_uuids" | grep -qFx "$uuid"; then
                continue
            fi

            # Skip if the prefix was touched within the safety window.
            # `mc stat` on the prefix may not exist; sample by listing
            # one nested object and reading its Last-Modified.
            local last_mod_iso last_mod_epoch age
            last_mod_iso=$(mc ls --recursive --json "bee/${bucket}/tenants/${uuid}/" 2>/dev/null \
                | jq -r 'select(.type == "file") | .lastModified' \
                | sort -r | head -1)
            if [ -n "$last_mod_iso" ]; then
                last_mod_epoch=$(date -u -d "$last_mod_iso" +%s 2>/dev/null || echo 0)
                age=$(( now_epoch - last_mod_epoch ))
                if [ "$age" -lt "$safety_sec" ]; then
                    ATLAS_STEP=drop-minio log info \
                        "skip ${bucket}/tenants/${uuid} (age=${age}s < ${safety_sec}s safety window)"
                    continue
                fi
            fi

            echo "drop-minio ${bucket}/tenants/${uuid}/"
            if [ "$APPLY" = "1" ]; then
                if ! mc rm --recursive --force "bee/${bucket}/tenants/${uuid}/" >/dev/null 2>&1; then
                    ATLAS_STEP=drop-minio log warn \
                        "rm ${bucket}/tenants/${uuid}/ failed"
                    rc=1
                fi
            fi
        done <<<"$uuid_lines"
    done
    return $rc
}

if [ "$MINIO_MODE" = "1" ]; then
    if [ -n "${DRY_RUN_NO_INFRA:-}" ]; then
        ATLAS_STEP=init log info "MinIO sweep apply=${APPLY} (dry-run; skipping)"
    else
        ATLAS_STEP=init log info "MinIO sweep apply=${APPLY}"
        sweep_minio
    fi
    ATLAS_STEP=done log info "sweep complete"
    exit 0
fi

for n in "${PR_NUMBERS[@]}"; do
    sweep_pr "$n"
done

ATLAS_STEP=done log info "sweep complete"
