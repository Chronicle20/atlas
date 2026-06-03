#!/usr/bin/env bash
# One-time reclamation of atlas-main's now-dead BARE Redis keys after the
# task-045 namespacing fix. After the fix, main writes prefixed keys
# (atlas:drops:all, …) and stops touching the bare forms. DELs only an explicit
# allowlist of bare namespaces; MUST NEVER match atlas:* or *:atlas:*.
# List-only by default; --apply to delete. Idempotent. REDIS_URL = host:port.
set -uo pipefail

. "$(dirname "$0")/lib.sh"

require_env REDIS_URL

APPLY=0
[ "${1:-}" = "--apply" ] && APPLY=1

# Split REDIS_URL (host:port) into separate flags so redis-cli sub-commands
# remain the first positional argument — required by the test shim and by
# redis-cli's own argument ordering for commands like DEL.
REDIS_HOST="${REDIS_URL%%:*}"
REDIS_PORT="${REDIS_URL##*:}"

EXACT_KEYS=(
    "channel:tenants"
    "drops:all"
    "reactors:all"
    "coordinator:active"
    "invite:active-tenants"
    "transport:instances"
    "transport:characters"
)
PREFIX_PATTERNS=(
    "coordinator:agreement:*"
    "coordinator:char:*"
    "transport:instance:*"
    "transport:route:*"
    "transport:channels:*"
    "drop:*"
    "reactor:*"
    "reactors:map:*"
    "drops:map:*"
    "reactor:cd:*"
    "reactor:spot:*"
    "reservation:*"
    "invlock:*"
    "atlas-data:ingest:*"
    "*:merchant:shop-visitors:*"
)

reclaim_pattern() {
    local pat="$1"
    case "$pat" in
        atlas:*|*:atlas:*)
            ATLAS_STEP=reclaim log error "refusing to scan prefixed pattern: $pat"
            return 1 ;;
    esac
    local keys
    keys=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" --scan --pattern "$pat") || return 1
    [ -z "$keys" ] && return 0
    while IFS= read -r k; do
        [ -z "$k" ] && continue
        case "$k" in
            atlas:*|*:atlas:*)
                ATLAS_STEP=reclaim log warn "skipping prefixed key: $k"
                continue ;;
        esac
        echo "reclaim DEL $k"
        if [ "$APPLY" = "1" ]; then
            redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" DEL "$k" >/dev/null 2>&1 \
                || ATLAS_STEP=reclaim log warn "DEL $k failed"
        fi
    done <<<"$keys"
}

ATLAS_STEP=reclaim log info "reclaim-main-bare-keys apply=${APPLY}"
rc=0
for k in "${EXACT_KEYS[@]}"; do
    reclaim_pattern "$k" || rc=1
done
for p in "${PREFIX_PATTERNS[@]}"; do
    reclaim_pattern "$p" || rc=1
done
ATLAS_STEP=done log info "reclaim complete"
exit $rc
