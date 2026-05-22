#!/usr/bin/env bash
# Shared helpers for bootstrap.sh and cleanup.sh.

set -euo pipefail

log() {
    # Logs are diagnostic output and MUST go to stderr. The fallback branch
    # already does; the jq branch did not, which caused subtle bugs when a
    # caller captured a function's stdout via $(): e.g. resolve_mode echoes
    # the resolved mode on stdout, but a `log warn` inside it would prepend
    # a JSON line into the captured value and break the subsequent `case`
    # match. PR-544 hit this — auto-mode resolution silently produced a
    # multi-line mode value, the case statement no-op'd, and the bootstrap
    # exited cleanly without running ingest. Fixed by redirecting both
    # branches to stderr.
    local level="$1"; shift
    local step="${ATLAS_STEP:-init}"
    if command -v jq >/dev/null 2>&1; then
        printf '{"ts":"%s","level":"%s","atlas.env":"%s","atlas.step":"%s","msg":%s}\n' \
            "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "${ATLAS_ENV:-}" "$step" \
            "$(printf '%s' "$*" | jq -Rs .)" >&2
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

# compute_atlas_env: derive the 4-hex-char per-env hash from a PR number.
# MUST stay in sync with .github/workflows/pr-validation.yml's update-pr-overlay
# step and the cluster-infra ApplicationSet template. test/lib_test.bats pins
# the contract via the PR 491 / 522 recovery-log oracles.
compute_atlas_env() {
    local pr_number="$1"
    if [ -z "$pr_number" ]; then
        log error "compute_atlas_env: empty PR_NUMBER"
        return 1
    fi
    printf "pr-%d" "$pr_number" | sha256sum | cut -c1-4
}
