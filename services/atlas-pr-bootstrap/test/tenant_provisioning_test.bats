#!/usr/bin/env bats
#
# find_environment_tenant is the fix for the adopt-main's-tenant defect: a
# sparse PR environment shares its canonical (region, major, minor) version
# triple with main, so the version triple alone can never distinguish which
# tenant belongs to which environment. Only the server's environment-scoped
# listing (the ENVIRONMENT header) may resolve a tenant id. Isolated mode
# must be byte-identical to the pre-existing curl argv — no header appended.

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    BOOTSTRAP="$PROJECT_ROOT/scripts/bootstrap.sh"

    # bootstrap.sh is not sourceable — it runs its whole flow at top level.
    # Extract just the three helpers under test into a file we can source, so
    # the assertions exercise the real definitions rather than a copy.
    HELPERS="$BATS_TEST_TMPDIR/helpers.sh"
    {
        sed -n '/^env_header_init()/,/^}/p' "$BOOTSTRAP"
        sed -n '/^find_environment_tenant()/,/^}/p' "$BOOTSTRAP"
        sed -n '/^create_environment_tenant()/,/^}/p' "$BOOTSTRAP"
    } >"$HELPERS"
    # shellcheck source=/dev/null
    . "$PROJECT_ROOT/scripts/lib.sh"
    . "$HELPERS"

    # Prove the extraction found all three. Without this, a missing
    # definition makes `run` exit 127 — which satisfies every "must fail"
    # assertion below and turns the whole suite green for the wrong reason.
    declare -F env_header_init >/dev/null || {
        echo "env_header_init not extracted from $BOOTSTRAP" >&2
        return 1
    }
    declare -F find_environment_tenant >/dev/null || {
        echo "find_environment_tenant not extracted from $BOOTSTRAP" >&2
        return 1
    }
    declare -F create_environment_tenant >/dev/null || {
        echo "create_environment_tenant not extracted from $BOOTSTRAP" >&2
        return 1
    }

    export ATLAS_UI_BASE=http://ui
    export ATLAS_ENVIRONMENT=pr-1411
    export CANONICAL_TENANT_JSON="$PROJECT_ROOT/canonical/tenant.json"
    CURL_ARGS="$BATS_TEST_TMPDIR/curl-args"
}

# Shim curl as a shell function: records its argv, then emits $CURL_BODY.
# CURL_RC drives the failure cases.
curl() {
    printf '%s\n' "$@" >"$CURL_ARGS"
    [ "${CURL_RC:-0}" -eq 0 ] || return "$CURL_RC"
    printf '%s' "$CURL_BODY"
}

# A tenant listing carrying exactly one GMS 83.1 row.
one_tenant() {
    printf '{"data":[{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}]}' "$1"
}
# Two GMS 83.1 rows, to pin "first wins".
two_tenants() {
    printf '{"data":[{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}},{"type":"tenants","id":"%s","attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}]}' "$1" "$2"
}

ENV_TENANT=6a5f0c1e-9d2b-4a77-8c31-0f2e5b7a9d40
OTHER_TENANT=1c9d3f42-7b60-4e18-9a5c-2d8f6e0b31a7

# --- env_header_init -------------------------------------------------------

@test "env_header_init builds the ENVIRONMENT header in sparse mode" {
    export ATLAS_MODE=sparse
    env_header_init
    [ "${#ENV_HEADER[@]}" -eq 2 ]
    [ "${ENV_HEADER[0]}" = "-H" ]
    [ "${ENV_HEADER[1]}" = "ENVIRONMENT: pr-1411" ]
}

@test "env_header_init leaves ENV_HEADER empty in isolated mode" {
    unset ATLAS_MODE
    env_header_init
    [ "${#ENV_HEADER[@]}" -eq 0 ]
}

@test "env_header_init leaves ENV_HEADER empty when ATLAS_MODE is explicitly isolated" {
    export ATLAS_MODE=isolated
    env_header_init
    [ "${#ENV_HEADER[@]}" -eq 0 ]
}

@test "env_header_init fails loudly when sparse mode has no ATLAS_ENVIRONMENT" {
    export ATLAS_MODE=sparse
    unset ATLAS_ENVIRONMENT
    run env_header_init
    [ "$status" -eq 1 ]
    [[ "$output" == *"missing required env: ATLAS_ENVIRONMENT"* ]]
}

# --- find_environment_tenant ------------------------------------------------

@test "find_environment_tenant scopes the listing with the ENVIRONMENT header in sparse mode" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY='{"data":[]}'
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
    grep -qx 'ENVIRONMENT: pr-1411' "$CURL_ARGS"
}

@test "find_environment_tenant sends no ENVIRONMENT header in isolated mode" {
    export ATLAS_MODE=isolated
    env_header_init
    CURL_BODY='{"data":[]}'
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    ! grep -q '^ENVIRONMENT:' "$CURL_ARGS"
    grep -qx 'http://ui/api/tenants' "$CURL_ARGS"
}

@test "find_environment_tenant echoes the environment's own tenant id" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY="$(one_tenant "$ENV_TENANT")"
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [ "$output" = "$ENV_TENANT" ]
}

@test "find_environment_tenant echoes the first match when several exist" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY="$(two_tenants "$ENV_TENANT" "$OTHER_TENANT")"
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [ "$output" = "$ENV_TENANT" ]
}

@test "find_environment_tenant does not adopt a tenant when the scoped listing is empty" {
    # The core regression: the version triple alone must never resolve a
    # tenant; only the server's environment-scoped listing may.
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY='{"data":[]}'
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "find_environment_tenant ignores a tenant on a different version triple" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY='{"data":[{"type":"tenants","id":"'"$OTHER_TENANT"'","attributes":{"region":"GMS","majorVersion":95,"minorVersion":1}}]}'
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "find_environment_tenant expands an empty ENV_HEADER without tripping set -u" {
    set -u
    export ATLAS_MODE=isolated
    env_header_init
    CURL_BODY='{"data":[]}'
    run find_environment_tenant GMS 83 1
    [ "$status" -eq 0 ]
    [[ "$output" != *"unbound variable"* ]]
}

# --- create_environment_tenant ----------------------------------------------

@test "create_environment_tenant POSTs the canonical payload with the ENVIRONMENT header" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY='{"data":{"type":"tenants","id":"'"$ENV_TENANT"'"}}'
    run create_environment_tenant
    [ "$status" -eq 0 ]
    [ "$output" = "$ENV_TENANT" ]
    grep -qx -- '-X' "$CURL_ARGS"
    grep -qx 'POST' "$CURL_ARGS"
    grep -qx 'ENVIRONMENT: pr-1411' "$CURL_ARGS"
    grep -qx "@$PROJECT_ROOT/canonical/tenant.json" "$CURL_ARGS"
    grep -qx 'http://ui/api/tenants' "$CURL_ARGS"
}

@test "create_environment_tenant sends no ENVIRONMENT header in isolated mode" {
    export ATLAS_MODE=isolated
    env_header_init
    CURL_BODY='{"data":{"type":"tenants","id":"'"$ENV_TENANT"'"}}'
    run create_environment_tenant
    [ "$status" -eq 0 ]
    ! grep -q '^ENVIRONMENT:' "$CURL_ARGS"
}

@test "create_environment_tenant fails when the POST returns no id" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_BODY='{"data":{"type":"tenants"}}'
    run create_environment_tenant
    [ "$status" -eq 1 ]
    [[ "$output" == *"tenant POST returned no id"* ]]
}

@test "create_environment_tenant fails when the POST itself fails" {
    export ATLAS_MODE=sparse
    env_header_init
    CURL_RC=22
    run create_environment_tenant
    [ "$status" -eq 1 ]
    [[ "$output" == *"tenant POST failed"* ]]
}
