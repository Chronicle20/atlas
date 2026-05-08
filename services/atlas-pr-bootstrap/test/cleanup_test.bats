#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "cleanup.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "cleanup.sh fails without ATLAS_DB_NAMES" {
    run env ATLAS_ENV=test DB_HOST=h DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=k REDIS_URL=r PR_NUMBER=1 \
        -u ATLAS_DB_NAMES bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_DB_NAMES"* ]]
}
