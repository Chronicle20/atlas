#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
}

@test "compute_atlas_env: PR 1" {
    run compute_atlas_env 1
    [ "$status" -eq 0 ]
    [ "$output" = "1a52" ]
}

@test "compute_atlas_env: PR 491 (recovery-log oracle)" {
    run compute_atlas_env 491
    [ "$status" -eq 0 ]
    [ "$output" = "ed86" ]
}

@test "compute_atlas_env: PR 522 (recovery-log oracle)" {
    run compute_atlas_env 522
    [ "$status" -eq 0 ]
    [ "$output" = "a476" ]
}

@test "compute_atlas_env: empty PR_NUMBER fails" {
    run compute_atlas_env ""
    [ "$status" -ne 0 ]
}
