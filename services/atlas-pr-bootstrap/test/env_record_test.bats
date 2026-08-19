#!/usr/bin/env bats
#
# env_record_get / env_record_patch are the shared control-plane
# environment-record helpers extracted from cleanup.sh (task-242 Task 1) so
# bootstrap.sh can reuse them without the two callers drifting.

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # lib.sh first, mirroring bootstrap.sh/cleanup.sh's own order:
    # env-record.sh calls log() but does not source it.
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
    # shellcheck source=../scripts/env-record.sh
    . "$PROJECT_ROOT/scripts/env-record.sh"

    # Prove the extraction found both helpers. Without this, a missing
    # definition makes `run` exit 127 — which satisfies every "must fail"
    # assertion below and turns the whole suite green for the wrong reason.
    declare -F env_record_get >/dev/null || {
        echo "env_record_get not defined by $PROJECT_ROOT/scripts/env-record.sh" >&2
        return 1
    }
    declare -F env_record_patch >/dev/null || {
        echo "env_record_patch not defined by $PROJECT_ROOT/scripts/env-record.sh" >&2
        return 1
    }

    export ATLAS_UI_BASE="http://ui"
    export ATLAS_ENVIRONMENT="pr-1411"
    CURL_ARGS="$BATS_TEST_TMPDIR/curl-args"
}

# Shim curl as a shell function: records its argv, then emits $CURL_BODY.
# CURL_RC drives the failure cases.
curl() {
    printf '%s\n' "$@" >"$CURL_ARGS"
    [ "${CURL_RC:-0}" -eq 0 ] || return "$CURL_RC"
    printf '%s' "$CURL_BODY"
}

# patch_payload — echoes the recorded `-d` argument (the only argv line that
# parses as JSON). The curl shim records one argument per line.
patch_payload() {
    while IFS= read -r line; do
        if printf '%s' "$line" | jq -e . >/dev/null 2>&1; then
            printf '%s' "$line"
            return 0
        fi
    done <"$CURL_ARGS"
    return 1
}

# --- env_record_get --------------------------------------------------------

@test "env_record_get GETs this environment's record with the ENVIRONMENT header" {
    CURL_BODY='{"data":{"id":"pr-1411"}}'
    run env_record_get
    [ "$status" -eq 0 ]
    [ "$output" = '{"data":{"id":"pr-1411"}}' ]
    grep -qF 'ENVIRONMENT: pr-1411' "$CURL_ARGS"
    grep -qF 'http://ui/api/configurations/environments/pr-1411' "$CURL_ARGS"
}

@test "env_record_get fails when ATLAS_UI_BASE is unset" {
    unset ATLAS_UI_BASE
    run env_record_get
    [ "$status" -eq 1 ]
    [ ! -e "$CURL_ARGS" ]
}

@test "env_record_get fails when ATLAS_ENVIRONMENT is unset" {
    unset ATLAS_ENVIRONMENT
    run env_record_get
    [ "$status" -eq 1 ]
    [ ! -e "$CURL_ARGS" ]
}

@test "env_record_get mirrors curl's exit status on a 404" {
    CURL_RC=22
    run env_record_get
    [ "$status" -eq 22 ]
}

# --- env_record_patch -------------------------------------------------------

@test "env_record_patch sends all five attributes plus the record id" {
    CURL_BODY=""
    run env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'
    [ "$status" -eq 0 ]
    local payload
    payload="$(patch_payload)"
    [ "$(printf '%s' "$payload" | jq -r '.data.type')" = "environments" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.id')" = "pr-1411" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.phase')" = "ACTIVE" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.baseline')" = "main" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.namespace')" = "atlas-main" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.tenant')" = "8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.overrides["atlas-login"]')" = "atlas-pr-1411" ]
}

@test "env_record_patch targets the environments PATCH route with the ENVIRONMENT header" {
    CURL_BODY=""
    run env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'
    [ "$status" -eq 0 ]
    grep -qF -- '-X' "$CURL_ARGS"
    grep -qF 'PATCH' "$CURL_ARGS"
    grep -qF 'ENVIRONMENT: pr-1411' "$CURL_ARGS"
    grep -qF 'http://ui/api/configurations/environments/pr-1411' "$CURL_ARGS"
}

@test "env_record_patch accepts an empty overrides object" {
    CURL_BODY=""
    run env_record_patch PROVISIONING main atlas-main "" '{}'
    [ "$status" -eq 0 ]
    local payload
    payload="$(patch_payload)"
    [ "$(printf '%s' "$payload" | jq -c '.data.attributes.overrides')" = "{}" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.tenant')" = "" ]
}

@test "env_record_patch propagates a failing PATCH" {
    CURL_RC=22
    run env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'
    [ "$status" -eq 22 ]
}
