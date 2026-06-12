#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/service-config.sh
    . "$PROJECT_ROOT/scripts/service-config.sh"
    export TENANT_ID="11111111-1111-1111-1111-111111111111"
    export MAJOR_VERSION="84"
    export LB_IP="10.0.0.9"
    CHANNEL_TMPL="$PROJECT_ROOT/canonical/services/channel-service.json"
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
