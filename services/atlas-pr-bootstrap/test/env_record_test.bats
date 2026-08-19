#!/usr/bin/env bats
#
# env_record_get / env_record_patch are the shared control-plane
# environment-record helpers extracted from cleanup.sh (task-242 Task 1) so
# bootstrap.sh can reuse them without the two callers drifting.

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    BOOTSTRAP="$PROJECT_ROOT/scripts/bootstrap.sh"
    # lib.sh first, mirroring bootstrap.sh/cleanup.sh's own order:
    # env-record.sh calls log() but does not source it.
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
    # shellcheck source=../scripts/env-record.sh
    . "$PROJECT_ROOT/scripts/env-record.sh"

    # bootstrap.sh is not sourceable — it runs its whole flow at top level.
    # Extract just record_environment_tenant into a file we can source, so
    # the assertions exercise the real definition rather than a copy.
    HELPERS="$BATS_TEST_TMPDIR/helpers.sh"
    sed -n '/^record_environment_tenant()/,/^}/p' "$BOOTSTRAP" >"$HELPERS"
    # shellcheck source=/dev/null
    . "$HELPERS"

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
    declare -F record_environment_tenant >/dev/null || {
        echo "record_environment_tenant not extracted from $BOOTSTRAP" >&2
        return 1
    }

    export ATLAS_UI_BASE="http://ui"
    export ATLAS_ENVIRONMENT="pr-1411"
    CURL_ARGS="$BATS_TEST_TMPDIR/curl-args"
}

# Shim curl as a shell function: records every call's argv (appended, since
# record_environment_tenant makes a GET then a PATCH and both argvs must be
# visible), then distinguishes the two by scanning for a literal "PATCH"
# argument. GET_BODY/GET_RC drive the GET; PATCH_RC drives the PATCH.
curl() {
    printf '%s\n' "$@" >>"$CURL_ARGS"
    for a in "$@"; do
        if [ "$a" = "PATCH" ]; then
            [ "${PATCH_RC:-0}" -eq 0 ] || return "$PATCH_RC"
            return 0
        fi
    done
    [ "${GET_RC:-0}" -eq 0 ] || return "$GET_RC"
    printf '%s' "$GET_BODY"
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

# env_record <phase> <tenant> — a full environments GET document.
env_record() {
    printf '{"data":{"type":"environments","id":"pr-1411","attributes":{"name":"pr-1411","baseline":"main","namespace":"atlas-pr-1411","tenant":"%s","overrides":{"atlas-login":"atlas-pr-1411","atlas-channel":"atlas-pr-1411"},"phase":"%s"}}}' "$2" "$1"
}

# --- env_record_get --------------------------------------------------------

@test "env_record_get GETs this environment's record with the ENVIRONMENT header" {
    GET_BODY='{"data":{"id":"pr-1411"}}'
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
    GET_RC=22
    run env_record_get
    [ "$status" -eq 22 ]
}

# --- env_record_patch -------------------------------------------------------

@test "env_record_patch sends all five attributes plus the record id" {
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
    run env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'
    [ "$status" -eq 0 ]
    grep -qF -- '-X' "$CURL_ARGS"
    grep -qF 'PATCH' "$CURL_ARGS"
    grep -qF 'ENVIRONMENT: pr-1411' "$CURL_ARGS"
    grep -qF 'http://ui/api/configurations/environments/pr-1411' "$CURL_ARGS"
}

@test "env_record_patch accepts an empty overrides object" {
    run env_record_patch PROVISIONING main atlas-main "" '{}'
    [ "$status" -eq 0 ]
    local payload
    payload="$(patch_payload)"
    [ "$(printf '%s' "$payload" | jq -c '.data.attributes.overrides')" = "{}" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.tenant')" = "" ]
}

@test "env_record_patch propagates a failing PATCH" {
    PATCH_RC=22
    run env_record_patch ACTIVE main atlas-main 8f14e45f-ceea-467a-9c2a-1b3f4c5d6e70 '{"atlas-login":"atlas-pr-1411"}'
    [ "$status" -eq 22 ]
}

# --- record_environment_tenant ----------------------------------------------

NEW_TENANT="6a5f0c1e-9d2b-4a77-8c31-0f2e5b7a9d40"

@test "record_environment_tenant PATCHes the new tenant onto the record" {
    GET_BODY="$(env_record ACTIVE '')"
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 0 ]
    [ "$(patch_payload | jq -r '.data.attributes.tenant')" = "$NEW_TENANT" ]
}

@test "record_environment_tenant carries the record's current phase, never an empty one" {
    GET_BODY="$(env_record ACTIVE '')"
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 0 ]
    [ "$(patch_payload | jq -r '.data.attributes.phase')" = "ACTIVE" ]
}

@test "record_environment_tenant carries a PROVISIONING phase through unchanged" {
    GET_BODY="$(env_record PROVISIONING '')"
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 0 ]
    [ "$(patch_payload | jq -r '.data.attributes.phase')" = "PROVISIONING" ]
}

@test "record_environment_tenant carries baseline, namespace and overrides through unchanged" {
    GET_BODY="$(env_record ACTIVE '')"
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 0 ]
    local payload
    payload="$(patch_payload)"
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.baseline')" = "main" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.namespace')" = "atlas-pr-1411" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.overrides["atlas-channel"]')" = "atlas-pr-1411" ]
}

@test "record_environment_tenant is a no-op-shaped same-phase PATCH when the tenant is already recorded" {
    GET_BODY="$(env_record ACTIVE "$NEW_TENANT")"
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 0 ]
    local payload
    payload="$(patch_payload)"
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.tenant')" = "$NEW_TENANT" ]
    [ "$(printf '%s' "$payload" | jq -r '.data.attributes.phase')" = "ACTIVE" ]
}

@test "record_environment_tenant fails when no environment record exists" {
    GET_RC=22
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"no control-plane environment record for pr-1411"* ]]
}

@test "record_environment_tenant fails when the record has no phase" {
    GET_BODY='{"data":{"type":"environments","id":"pr-1411","attributes":{}}}'
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -eq 1 ]
    [[ "$output" == *"no control-plane environment record for pr-1411"* ]]
}

@test "record_environment_tenant propagates a failing PATCH" {
    GET_BODY="$(env_record ACTIVE '')"
    PATCH_RC=22
    run record_environment_tenant "$NEW_TENANT"
    [ "$status" -ne 0 ]
}
