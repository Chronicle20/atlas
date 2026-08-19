#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # lib.sh first, mirroring bootstrap.sh's own order: service-config.sh
    # calls log() but does not source it, so without this the error paths
    # below die with "log: command not found" (127) instead of emitting the
    # message they are asserting.
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
    # shellcheck source=../scripts/service-config.sh
    . "$PROJECT_ROOT/scripts/service-config.sh"
    export TENANT_ID="11111111-1111-1111-1111-111111111111"
    export MAJOR_VERSION="84"
    export LB_IP="10.0.0.9"
    CHANNEL_TMPL="$PROJECT_ROOT/canonical/services/channel-service.json"
    CANONICAL="$PROJECT_ROOT/canonical/services"
}

@test "build_login_entry: derived port, given id" {
    run build_login_entry
    [ "$status" -eq 0 ]
    [ "$(echo "$output" | jq -r '.id')" = "$TENANT_ID" ]
    [ "$(echo "$output" | jq -r '.port')" = "8400" ]
}

@test "build_channel_entry: derived channel port, id, ipAddress, worlds shell preserved" {
    run build_channel_entry "$CHANNEL_TMPL"
    [ "$status" -eq 0 ]
    [ "$(echo "$output" | jq -r '.id')" = "$TENANT_ID" ]
    [ "$(echo "$output" | jq -r '.ipAddress')" = "$LB_IP" ]
    [ "$(echo "$output" | jq -r '.worlds[0].channels[0].port')" = "8401" ]
    [ "$(echo "$output" | jq -r '.worlds[0].id')" = "0" ]
}

@test "merge_tenant_entry: appends when id absent, preserves foreign entries verbatim" {
    live='{"type":"login-service","tenants":[{"id":"aaaa","port":8300}]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants | length')" = "2" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].id')" = "aaaa" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].port')" = "8300" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].id')" = "bbbb" ]
}

@test "merge_tenant_entry: replaces in place by id, preserving array order" {
    live='{"tenants":[{"id":"aaaa","port":8300},{"id":"bbbb","port":1}]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants | length')" = "2" ]
    [ "$(echo "$merged" | jq -r '.tenants[0].id')" = "aaaa" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].id')" = "bbbb" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].port')" = "8400" ]
}

@test "merge_tenant_entry: preserves a foreign channel entry's ipAddress" {
    live='{"tenants":[{"id":"aaaa","ipAddress":"9.9.9.9","worlds":[]},{"id":"bbbb","ipAddress":"1.1.1.1","worlds":[]}]}'
    entry='{"id":"bbbb","ipAddress":"10.0.0.9","worlds":[]}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -r '.tenants[0].ipAddress')" = "9.9.9.9" ]
    [ "$(echo "$merged" | jq -r '.tenants[1].ipAddress')" = "10.0.0.9" ]
}

@test "merge_tenant_entry: idempotent — second merge of same entry is byte-identical" {
    live='{"tenants":[{"id":"aaaa","port":8300}]}'
    entry='{"id":"bbbb","port":8400}'
    once="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    twice="$(printf '%s' "$once" | merge_tenant_entry "$entry")"
    [ "$once" = "$twice" ]
}

@test "merge_tenant_entry: tenant-agnostic config (no tenants key) is unchanged" {
    live='{"type":"drops-service","tasks":[]}'
    entry='{"id":"bbbb","port":8400}'
    merged="$(printf '%s' "$live" | merge_tenant_entry "$entry")"
    [ "$(echo "$merged" | jq -cS .)" = "$(echo "$live" | jq -cS .)" ]
}

@test "sparse mode never reads or writes the pinned main service row" {
    export ATLAS_MODE=sparse
    export TENANT_ID="$(uuidgen)"
    export MAJOR_VERSION=83
    export LB_IP=192.168.23.211
    export ATLAS_ENVIRONMENT="pr-999"

    run build_service_config login "$CANONICAL/login-service.json"
    [ "$status" -eq 0 ]

    # A fresh id, NOT the canonical pinned one.
    pinned=$(jq -r '.data.id' "$CANONICAL/login-service.json")
    got=$(echo "$output" | jq -r '.data.id')
    [ "$got" != "$pinned" ]
    [ "$got" != "null" ]

    # Exactly this environment's one tenant — never a merge of main's list.
    [ "$(echo "$output" | jq '.data.attributes.tenants | length')" -eq 1 ]
    [ "$(echo "$output" | jq -r '.data.attributes.tenants[0].id')" = "$TENANT_ID" ]

    # The environment is stamped so teardown and write-authorisation can scope it.
    [ "$(echo "$output" | jq -r '.data.attributes.environment')" = "$ATLAS_ENVIRONMENT" ]
}

@test "isolated mode still merges into the pinned row" {
    export ATLAS_MODE=isolated
    run build_service_config login "$CANONICAL/login-service.json"
    [ "$status" -eq 0 ]
    pinned=$(jq -r '.data.id' "$CANONICAL/login-service.json")
    [ "$(echo "$output" | jq -r '.data.id')" = "$pinned" ]
}

@test "isolated mode POST body replaces channel-service's seeded placeholder tenant, never appends beside it" {
    export ATLAS_MODE=isolated
    run build_service_config channel "$CANONICAL/channel-service.json"
    [ "$status" -eq 0 ]

    # Fixture sanity: channel-service.json ships a non-empty seeded tenants[]
    # (placeholder id ec876921-c363-4cc6-9c51-5bb8d57f9553, port 0) — unlike
    # login-service.json's tenants: []. A merge instead of a replace here
    # appends beside the placeholder instead of discarding it.
    seeded=$(jq -r '.data.attributes.tenants[0].id' "$CANONICAL/channel-service.json")
    [ "$seeded" = "ec876921-c363-4cc6-9c51-5bb8d57f9553" ]

    pinned=$(jq -r '.data.id' "$CANONICAL/channel-service.json")
    [ "$(echo "$output" | jq -r '.data.id')" = "$pinned" ]

    # Exactly one tenant — this environment's, the placeholder is gone.
    [ "$(echo "$output" | jq '.data.attributes.tenants | length')" -eq 1 ]
    [ "$(echo "$output" | jq -r '.data.attributes.tenants[0].id')" = "$TENANT_ID" ]
    [ "$(echo "$output" | jq -r '.data.attributes.tenants[0].worlds[0].channels[0].port')" = "8401" ]
}

# --- sparse service-row id generation ------------------------------------
#
# The sparse env shipped broken because `uuidgen` was absent from the image
# and `--arg id "$(uuidgen)"` turned that into an empty string. The
# pre-existing "sparse mode never reads or writes the pinned main service
# row" test above did not catch it: it asserts only `!= pinned` and
# `!= "null"`, and "" satisfies both. These assert the id is actually a
# well-formed UUID.

UUID_RE='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'

@test "new_uuid: emits a well-formed UUID" {
    run new_uuid
    [ "$status" -eq 0 ]
    [[ "$output" =~ $UUID_RE ]]
}

@test "new_uuid: emits distinct values across calls" {
    a="$(new_uuid)"
    b="$(new_uuid)"
    [ "$a" != "$b" ]
}

@test "new_uuid: falls back to the kernel source when uuidgen is absent" {
    [ -r /proc/sys/kernel/random/uuid ] || skip "no /proc uuid source on this host"
    local bindir="$BATS_TEST_TMPDIR/emptybin"
    mkdir -p "$bindir"
    # The function is already sourced by setup(); restrict PATH only around
    # the call itself. Re-sourcing under a bare PATH would instead fail in
    # the script's own `dirname` preamble and prove nothing about new_uuid.
    local oldpath="$PATH"
    PATH="$bindir"
    run new_uuid
    PATH="$oldpath"
    [ "$status" -eq 0 ]
    [[ "$output" =~ $UUID_RE ]]
}

@test "new_uuid: fails loudly when no UUID source is available" {
    local bindir="$BATS_TEST_TMPDIR/emptybin2"
    mkdir -p "$bindir"
    local oldpath="$PATH"
    PATH="$bindir"
    _SC_UUID_PROC="$BATS_TEST_TMPDIR/definitely-absent" run new_uuid
    PATH="$oldpath"
    [ "$status" -ne 0 ]
    [[ "$output" == *"could not generate a UUID"* ]]
}

# Shape assertion only: on a host that HAS uuidgen this also passed before
# the fix, so it is not the regression test — the "no UUID source available"
# case below is. It is still worth asserting, because "" and "null" both slip
# past the older test's `!= pinned` / `!= "null"` checks.
@test "build_service_config: sparse id is a well-formed UUID, not empty" {
    export ATLAS_MODE=sparse
    export ATLAS_ENVIRONMENT="pr-999"
    run build_service_config login "$CANONICAL/login-service.json"
    [ "$status" -eq 0 ]
    got=$(echo "$output" | jq -r '.data.id')
    [[ "$got" =~ $UUID_RE ]]
}

@test "build_service_config: sparse fails when no UUID source is available" {
    # jq is still reachable, so this fails on the UUID source specifically
    # rather than on a missing tool somewhere earlier.
    local bindir="$BATS_TEST_TMPDIR/emptybin3"
    mkdir -p "$bindir"
    ln -sf "$(command -v jq)" "$bindir/jq"
    export ATLAS_MODE=sparse
    export ATLAS_ENVIRONMENT="pr-999"
    local oldpath="$PATH"
    PATH="$bindir"
    _SC_UUID_PROC="$BATS_TEST_TMPDIR/definitely-absent" \
        run build_service_config login "$CANONICAL/login-service.json"
    PATH="$oldpath"
    [ "$status" -ne 0 ]
    [[ "$output" == *"could not generate a UUID"* ]]
}
