#!/usr/bin/env bash
# Shared helpers for bootstrap.sh and cleanup.sh.

set -euo pipefail

log() {
    local level="$1"; shift
    local step="${ATLAS_STEP:-init}"
    if command -v jq >/dev/null 2>&1; then
        printf '{"ts":"%s","level":"%s","atlas.env":"%s","atlas.step":"%s","msg":%s}\n' \
            "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "${ATLAS_ENV:-}" "$step" \
            "$(printf '%s' "$*" | jq -Rs .)"
    else
        # Fallback for environments without jq (e.g., bats hosts without
        # the bootstrap image installed). Emits the same raw message
        # without JSON encoding so test assertions still match.
        printf '[%s] %s\n' "$level" "$*" >&2
    fi
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

http_ok_tenant() {
    local url=$1
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' \
        -H "TENANT_ID: $TENANT_ID" -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" -H "MINOR_VERSION: $MINOR_VERSION" \
        "$url" || echo 000)
    [ "$status" = "200" ] || [ "$status" = "204" ]
}
