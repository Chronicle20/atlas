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

@test "record_error: appends phase to ATLAS_PHASE_ERRORS and logs error" {
    ATLAS_PHASE_ERRORS=()
    run bash -c 'command() { if [ "$1" = "-v" ] && [ "$2" = "jq" ]; then return 1; else builtin command "$@"; fi; }; . "'"$PROJECT_ROOT"'/scripts/lib.sh"; ATLAS_PHASE_ERRORS=(); record_error drop-topics "rpk failed"; printf "%s\n" "${ATLAS_PHASE_ERRORS[@]}"'
    [ "$status" -eq 0 ]
    [[ "$output" == *"drop-topics"* ]]
    [[ "$output" == *"rpk failed"* ]]
}

@test "run_phase: success path emits start + complete, no error appended" {
    run bash -c '
        command() { if [ "$1" = "-v" ] && [ "$2" = "jq" ]; then return 1; else builtin command "$@"; fi; }
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        ok_phase() { return 0; }
        run_phase good_phase ok_phase
        echo "errors=${#ATLAS_PHASE_ERRORS[@]}"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phase start"* ]]
    [[ "$output" == *"phase complete"* ]]
    [[ "$output" == *"errors=0"* ]]
}

@test "run_phase: failure path records phase and returns 0 (orchestration continues)" {
    run bash -c '
        command() { if [ "$1" = "-v" ] && [ "$2" = "jq" ]; then return 1; else builtin command "$@"; fi; }
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        bad_phase() { return 7; }
        run_phase bad_phase bad_phase
        echo "rc=$?"
        echo "errors=${ATLAS_PHASE_ERRORS[*]}"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"rc=0"* ]]
    [[ "$output" == *"errors=bad_phase"* ]]
}

@test "summarize_phases: phases_failed=0 success line, exit 0" {
    run bash -c '
        command() { if [ "$1" = "-v" ] && [ "$2" = "jq" ]; then return 1; else builtin command "$@"; fi; }
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        summarize_phases 7
        echo "rc=$?"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phases_run=7"* ]]
    [[ "$output" == *"phases_failed=0"* ]]
    [[ "$output" == *"rc=0"* ]]
}

@test "summarize_phases: error path lists failed phases as JSON array, exits 1" {
    run bash -c '
        command() { if [ "$1" = "-v" ] && [ "$2" = "jq" ]; then return 1; else builtin command "$@"; fi; }
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=(drop-topics drop-redis)
        summarize_phases 7
        echo "rc=$?"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phases_run=7"* ]]
    [[ "$output" == *"phases_failed=2"* ]]
    [[ "$output" == *'["drop-topics","drop-redis"]'* ]]
    [[ "$output" == *"rc=1"* ]]
}
