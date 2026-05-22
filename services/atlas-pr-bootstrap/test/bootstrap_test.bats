#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "bootstrap.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "bootstrap.sh fails without ATLAS_UI_BASE" {
    run env -u ATLAS_UI_BASE ATLAS_ENV=test bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_UI_BASE"* ]]
}
