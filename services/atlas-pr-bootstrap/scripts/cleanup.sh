#!/usr/bin/env bash
# Atlas PR-env cleanup. Each step is idempotent; failures stop the run
# and leave the env intact for inspection (ArgoCD Application stays in
# 'cleanup-failed' state).
#
# Required env:
#   ATLAS_ENV              — env hash
#   DB_HOST/PORT/USER/PASS — Postgres connection details
#   ATLAS_DB_NAMES    — space-separated list of base DB names
#   BOOTSTRAP_SERVERS — kafka.home:9093
#   REDIS_URL         — redis.home:6379
#   PIHOLE_API_BASE_1, PIHOLE_TOKEN_1, PIHOLE_API_BASE_2, PIHOLE_TOKEN_2
#   GHCR_TOKEN        — for image-tag delete
#   PR_NUMBER         — for image-tag prefix
#   ATLAS_SERVICES    — comma-separated list of service names for image cleanup

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

# Phase 0 Task 0.1 finding: db-credentials secret values carry trailing
# whitespace (literal space + CR + LF). Strip BEFORE require_env so an
# all-whitespace value is caught by the empty check.
DB_USER="$(printf '%s' "${DB_USER:-}" | tr -d ' \r\n')"
DB_PASSWORD="$(printf '%s' "${DB_PASSWORD:-}" | tr -d ' \r\n')"

require_env ATLAS_ENV DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL PR_NUMBER

ATLAS_STEP=drop-dbs log info "dropping per-env Postgres databases"
# ATLAS_DB_NAMES is space-separated (matches kustomization.yaml's atlas-db-names
# configMapGenerator and the create-dbs Job's for-loop). Use default IFS so the
# `read -ra` splits on whitespace.
read -ra dbs <<< "$ATLAS_DB_NAMES"
for db in "${dbs[@]}"; do
    full="${db}-${ATLAS_ENV}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
        -c "DROP DATABASE IF EXISTS \"$full\";" || {
            log error "failed to drop $full"
            exit 1
        }
done

ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list \
    | grep -E -- "-${ATLAS_ENV}\$" \
    | xargs -r -n 1 kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --topic

ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
# Atlas consumer-group names contain spaces (e.g. "Party Quest Service [1756]",
# "Channel Service - %s [1756]"). xargs's default delimiter is whitespace, which
# would word-split each group name into 3-5 separate `--group` invocations and
# nothing would match. -d '\n' restricts splitting to newlines, so each group
# is passed intact. Observed 2026-05-16 cleaning up atlas-pr-461's leftover
# 1756-suffixed groups after the PostDelete hook had previously failed.
kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list \
    | grep -E -- "\\[${ATLAS_ENV}\\]\$" \
    | xargs -r -d '\n' -n 1 kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --group

ATLAS_STEP=drop-redis log info "deleting per-env Redis keys"
redis-cli -u "redis://$REDIS_URL" --scan --pattern "${ATLAS_ENV}:*" \
    | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL

if [ -n "${ATLAS_SERVICES:-}" ] && [ -n "${GHCR_TOKEN:-}" ]; then
    ATLAS_STEP=drop-images log info "deleting per-PR ghcr image tags"
    IFS=',' read -ra svcs <<< "$ATLAS_SERVICES"
    for svc in "${svcs[@]}"; do
        gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${PR_NUMBER}-\")) | .id" \
            | while read -r vid; do
                gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                    "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" || true
            done
    done
fi

if [ -n "${PIHOLE_API_BASE_1:-}" ] && [ -n "${PIHOLE_TOKEN_1:-}" ]; then
    ATLAS_STEP=drop-dns log info "removing Pi-hole A records"
    for i in 1 2; do
        base_var="PIHOLE_API_BASE_$i"
        token_var="PIHOLE_TOKEN_$i"
        base="${!base_var:-}"
        token="${!token_var:-}"
        if [ -z "$base" ] || [ -z "$token" ]; then
            continue
        fi
        # Pi-hole v6: session-based auth + path-encoded literal entry. The host
        # entry shape is "IP hostname" — we don't know the IP at cleanup time,
        # so list and grep for the entry whose suffix matches our hostname.
        # See deploy/k8s/overlays/pr/postsync-pihole-add.yaml for the matching
        # register flow.
        host="${PR_NUMBER}.atlas.home"
        sid=$(curl -k -fsS -X POST "$base/api/auth" \
            -H "Content-Type: application/json" \
            -d "{\"password\":\"$token\"}" 2>/dev/null \
            | jq -r '.session.sid // empty')
        if [ -z "$sid" ]; then
            log warn "Pi-hole $i: auth failed, skipping host removal"
            continue
        fi
        entry=$(curl -k -fsS -H "X-FTL-SID: $sid" "$base/api/config/dns/hosts" \
            | jq -r ".config.dns.hosts[]? | select(endswith(\" $host\"))" | head -1)
        if [ -n "$entry" ]; then
            encoded_entry=$(printf '%s' "$entry" | sed 's/ /%20/g')
            curl -k -fsS -X DELETE -H "X-FTL-SID: $sid" \
                "$base/api/config/dns/hosts/$encoded_entry" || \
                log warn "Pi-hole $i delete failed for $host"
        fi
    done
fi

ATLAS_STEP=done log info "cleanup complete"
