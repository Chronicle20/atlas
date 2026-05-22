#!/usr/bin/env bash
# Shared helpers for bootstrap.sh and cleanup.sh.

set -uo pipefail

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

# ----------------------------------------------------------------------------
# Phase-runner framework. See task-075/design.md §3.1.
#
# Used by cleanup.sh and sweep-orphans.sh to run every phase regardless
# of any single phase's outcome. ONLY run_phase appends to
# ATLAS_PHASE_ERRORS — phase functions emit detail logs via `log warn`
# / `log error` and return non-zero on failure; run_phase records the
# phase name exactly once.
# ----------------------------------------------------------------------------

# ATLAS_PHASE_ERRORS holds the names of phases that have failed. The
# caller initialises (or resets, for per-PR sweep loops) by assigning
# ATLAS_PHASE_ERRORS=() before run_phase is called.
declare -ga ATLAS_PHASE_ERRORS=()

# record_error <phase> <msg>
# Appends <phase> to ATLAS_PHASE_ERRORS and logs <msg> at level=error
# with ATLAS_STEP=<phase>.
record_error() {
    local phase="$1"; shift
    ATLAS_PHASE_ERRORS+=("$phase")
    ATLAS_STEP="$phase" log error "$*"
}

# run_phase <phase_name> <function_name>
# Emits a "phase start" info log, runs <function_name>, and either
# emits "phase complete" (on zero return) or appends <phase_name> to
# ATLAS_PHASE_ERRORS via record_error (on non-zero). Always returns 0
# so a caller without `set -e` continues running subsequent phases.
run_phase() {
    local phase="$1"; local fn="$2"
    ATLAS_STEP="$phase" log info "phase start"
    if "$fn"; then
        ATLAS_STEP="$phase" log info "phase complete"
    else
        record_error "$phase" "phase exited non-zero"
    fi
    return 0
}

# summarize_phases <total_phase_count>
# Emits one JSON summary line and returns 0 (success) or 1 (errors
# recorded). Callers typically `exit $?` after this.
summarize_phases() {
    local total="$1"
    local failed="${#ATLAS_PHASE_ERRORS[@]}"
    local failed_json
    if [ "$failed" -eq 0 ]; then
        ATLAS_STEP=done log info "cleanup complete phases_run=$total phases_failed=0"
        return 0
    fi
    failed_json=$(printf '%s\n' "${ATLAS_PHASE_ERRORS[@]}" \
        | jq -Rsc 'split("\n") | map(select(length>0))')
    ATLAS_STEP=done log error "cleanup completed with errors phases_run=$total phases_failed=$failed failed_phases=$failed_json"
    return 1
}

# ----------------------------------------------------------------------------
# rpk JSON-output schema constants. See test/fixtures/rpk-*.json.
#
# rpk 24.3.1 emits a flat array for both topic list and group list:
#   [{"name":"…","partitions":…}, …]
# The pre-fix queries (.topics[].name / .groups[].name) assumed an
# object wrapping the array and failed with "Cannot index array with
# string …" — see prd.md §1 / Bug 1.
#
# Bumping ARG RPK_VERSION in the Dockerfile invalidates the fixtures.
# Regenerate against the new rpk and re-run bats; the schema may move
# again.
# ----------------------------------------------------------------------------
readonly RPK_TOPICS_JQ='.[].name'
readonly RPK_GROUPS_JQ='.[].name'
