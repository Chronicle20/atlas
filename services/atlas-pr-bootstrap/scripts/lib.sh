#!/usr/bin/env bash
# Shared helpers for bootstrap.sh and cleanup.sh.

set -euo pipefail

log() {
    local level="$1"; shift
    local step="${ATLAS_STEP:-init}"
    printf '{"ts":"%s","level":"%s","atlas.env":"%s","atlas.cleanup-step":"%s","msg":%s}\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "${ATLAS_ENV:-}" "$step" \
        "$(printf '%s' "$*" | jq -Rs .)"
}

require_env() {
    for v in "$@"; do
        if [ -z "${!v:-}" ]; then
            log error "missing required env: $v"
            exit 1
        fi
    done
}

retry() {
    local max=$1; shift
    local sleep_s=$1; shift
    local n=0
    while ! "$@"; do
        n=$((n+1))
        if [ "$n" -ge "$max" ]; then
            log error "retry exhausted after $n attempts: $*"
            return 1
        fi
        sleep "$sleep_s"
    done
}

http_ok() {
    local url=$1
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' "$url" || echo 000)
    [ "$status" = "200" ] || [ "$status" = "204" ]
}
