#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/sweep-orphans.sh"
}

@test "sweep-orphans.sh: missing PR number prints usage and exits non-zero" {
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "sweep-orphans.sh: rejects non-numeric PR number" {
    run bash "$SCRIPT" abc
    [ "$status" -ne 0 ]
    [[ "$output" == *"not a number"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "sweep-orphans.sh: --list (default) on PR 491 prints derived ATLAS_ENV" {
    # No infra to talk to in unit tests; assert the script gets far enough
    # to print the computed env hash before any external command fails or
    # is no-op'd by being unreachable.
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491
    [[ "$output" == *"ed86"* ]]
}

@test "sweep-orphans.sh: --apply requires explicit confirmation flag" {
    # Idempotency / blast-radius: require the operator to type --apply.
    # Default behavior MUST be list-only.
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491
    [[ "$output" != *"DROP DATABASE"* ]]
    [[ "$output" != *"--delete --topic"* ]]
}

@test "sweep-orphans.sh: accepts multiple PR numbers" {
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491 522
    [[ "$output" == *"ed86"* ]]
    [[ "$output" == *"a476"* ]]
}
