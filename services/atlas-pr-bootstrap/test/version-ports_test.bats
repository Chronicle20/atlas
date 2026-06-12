#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/version-ports.sh
    . "$PROJECT_ROOT/scripts/version-ports.sh"
}

@test "derive_login_port: 83 -> 8300" {
    [ "$(derive_login_port 83)" = "8300" ]
}

@test "derive_channel_port: 83 -> 8301" {
    [ "$(derive_channel_port 83)" = "8301" ]
}

@test "derive_login_port: 12 -> 1200 and 185 -> 18500" {
    [ "$(derive_login_port 12)" = "1200" ]
    [ "$(derive_login_port 185)" = "18500" ]
}

@test "derive_channel_port: 12 -> 1201 and 185 -> 18501" {
    [ "$(derive_channel_port 12)" = "1201" ]
    [ "$(derive_channel_port 185)" = "18501" ]
}

@test "derive_login_port: non-integer is rejected" {
    run derive_login_port "8x"
    [ "$status" -ne 0 ]
    [[ "$output" == *"not a non-negative integer"* ]]
}
