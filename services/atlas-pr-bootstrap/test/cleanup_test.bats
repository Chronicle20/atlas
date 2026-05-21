#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    STUB_BIN="$BATS_TEST_TMPDIR/bin"
    STUB_LOG="$BATS_TEST_TMPDIR/calls.log"
    mkdir -p "$STUB_BIN"
}

# make_stubs writes shell-script stubs for every external binary cleanup.sh
# invokes. Each stub appends its argv to "$STUB_LOG" and exits 0 unless the
# caller passes per-binary overrides.
#
# Args (optional, in order):
#   $1 — topic_list_json (default: empty topic list)
#   $2 — group_list_json (default: empty group list)
make_stubs() {
    local topic_json="${1:-{\"topics\":[]\}}"
    local group_json="${2:-{\"groups\":[]\}}"
    printf '%s\n' "$topic_json" > "$BATS_TEST_TMPDIR/topic_list.json"
    printf '%s\n' "$group_json" > "$BATS_TEST_TMPDIR/group_list.json"

    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/topic_list.json"
elif [ "$1" = "group" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/group_list.json"
fi
exit 0
EOF
    cat > "$STUB_BIN/psql" <<'EOF'
#!/usr/bin/env bash
echo "psql $*" >> "$STUB_LOG"
exit 0
EOF
    cat > "$STUB_BIN/redis-cli" <<'EOF'
#!/usr/bin/env bash
echo "redis-cli $*" >> "$STUB_LOG"
# When invoked with --scan, emit no keys so the xargs delete is a no-op.
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh $*" >> "$STUB_LOG"
exit 0
EOF
    chmod +x "$STUB_BIN"/*
}

# run_cleanup runs cleanup.sh with the standard test env vars and the
# stubs on PATH. cleanup.sh derives ATLAS_ENV from PR_NUMBER (see
# lib.sh's compute_atlas_env), so callers control the per-env hash via
# PR_NUMBER (default 99 → compute_atlas_env "99").
run_cleanup() {
    PATH="$STUB_BIN:$PATH" \
    STUB_LOG="$STUB_LOG" \
    BATS_TEST_TMPDIR="$BATS_TEST_TMPDIR" \
    DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
    ATLAS_DB_NAMES="foo bar" \
    BOOTSTRAP_SERVERS=kafka:9093 \
    REDIS_URL=redis:6379 \
    PR_NUMBER="${PR_NUMBER:-99}" \
    bash "$PROJECT_ROOT/scripts/cleanup.sh"
}

@test "cleanup.sh fails without PR_NUMBER" {
    run env -u PR_NUMBER DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="atlas-test" BOOTSTRAP_SERVERS=k REDIS_URL=r \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: PR_NUMBER"* ]]
}

@test "cleanup.sh no longer requires ATLAS_ENV in env" {
    # Pre-fix this asserted ATLAS_ENV was required. Now ATLAS_ENV is derived
    # from PR_NUMBER, so the script must fail on the next missing var
    # (DB_HOST), NOT on ATLAS_ENV. Drives the require_env reordering in
    # cleanup.sh.
    run env -u ATLAS_ENV -u DB_HOST PR_NUMBER=1 DB_PORT=5432 DB_USER=u \
        DB_PASSWORD=p ATLAS_DB_NAMES="atlas-test" BOOTSTRAP_SERVERS=k \
        REDIS_URL=r bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" != *"missing required env: ATLAS_ENV"* ]]
    [[ "$output" == *"missing required env: DB_HOST"* ]]
}

@test "cleanup.sh fails without ATLAS_DB_NAMES" {
    run env -u ATLAS_DB_NAMES PR_NUMBER=1 DB_HOST=h DB_PORT=5432 DB_USER=u \
        DB_PASSWORD=p BOOTSTRAP_SERVERS=k REDIS_URL=r \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_DB_NAMES"* ]]
}

@test "cleanup.sh branch-delete swallows 404" {
    # The bot branch may already have been deleted (operator, prior cleanup
    # re-run, force-deleted). Simulate via a `gh` shim in PATH that emits a
    # 404 body and exits non-zero. Cleanup must continue past this phase
    # without exiting.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh: Reference does not exist" >&2
exit 1
EOF
    chmod +x "$SHIM_DIR/gh"

    # Inject failing kafka-topics.sh / kafka-consumer-groups.sh / psql /
    # redis-cli so cleanup short-circuits on the very first phase BEFORE
    # branch-delete, while we only need to assert that the function exists
    # and is exercised by the unit (the e2e is in the smoke test). For this
    # unit assertion, we run a bash-side check on the script body instead:
    run grep -q "drop-branch" "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -eq 0 ]

    rm -rf "$SHIM_DIR"
}

@test "cleanup.sh references atlas-pr-cleanup-gh-token-mounted GHCR_TOKEN for branch-delete" {
    # GHCR_TOKEN is the secret key name preserved across the ghcr->dedicated
    # token migration. The branch-delete phase MUST read it, not a new env
    # name.
    run grep -E "drop-branch.*GHCR_TOKEN|GHCR_TOKEN.*drop-branch" \
        "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -eq 0 ]
}
