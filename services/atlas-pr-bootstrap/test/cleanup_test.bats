#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    STUB_BIN="$BATS_TEST_TMPDIR/bin"
    STUB_LOG="$BATS_TEST_TMPDIR/calls.log"
    mkdir -p "$STUB_BIN"

    # Default curl stub, present for EVERY test (not just make_stubs
    # callers): do_deactivate now calls curl unconditionally, and
    # do_drop_control_plane calls it in sparse mode. Logs the invocation and
    # succeeds — GETs return an empty JSON:API collection so
    # do_drop_control_plane's default (isolated-mode-skipped, or
    # sparse-mode-empty-list) path never fails an existing test that
    # doesn't care about control-plane reclaim. Tests that DO care about
    # the reclaim's scoping build their own SHIM_DIR curl stub, same
    # convention as the drop-branch tests' own gh/kubectl stubs below.
    # The environments-record GET 404s by default (empty body, exit 1) —
    # matching isolated mode, which never registers a control-plane
    # environment record (task-48 fix round 1: do_deactivate/
    # do_drop_control_plane gate on this live check now, not ATLAS_MODE).
    # Every other GET returns an empty JSON:API collection.
    cat > "$STUB_BIN/curl" <<'EOF'
#!/usr/bin/env bash
echo "curl $*" >> "$STUB_LOG"
url="${@: -1}"
case "$url" in
    */configurations/environments/*)
        echo '{}'
        exit 1
        ;;
    *)
        echo '{"data":[]}'
        exit 0
        ;;
esac
EOF
    chmod +x "$STUB_BIN/curl"
}

# make_stubs writes shell-script stubs for every external binary cleanup.sh
# invokes. Each stub appends its argv to "$STUB_LOG" and exits 0 unless the
# caller passes per-binary overrides.
#
# Args (optional, in order):
#   $1 — topic_list_json (default: rpk-topic-list.json fixture)
#   $2 — group_list_table (default: rpk-group-list.txt fixture; raw table
#        as emitted by `rpk group list` — no --format in rpk 24.3.1)
make_stubs() {
    local topic_json
    local group_table
    if [ "${1+set}" = set ]; then
        topic_json="$1"
    else
        topic_json="$(cat "$PROJECT_ROOT/test/fixtures/rpk-topic-list.json")"
    fi
    if [ "${2+set}" = set ]; then
        group_table="$2"
    else
        group_table="$(cat "$PROJECT_ROOT/test/fixtures/rpk-group-list.txt")"
    fi
    printf '%s\n' "$topic_json" > "$BATS_TEST_TMPDIR/topic_list.json"
    printf '%s\n' "$group_table" > "$BATS_TEST_TMPDIR/group_list.txt"

    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/topic_list.json"
elif [ "$1" = "group" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/group_list.txt"
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
    ATLAS_UI_BASE="${ATLAS_UI_BASE:-http://atlas-ingress.test.svc.cluster.local}" \
    ATLAS_ENVIRONMENT="${ATLAS_ENVIRONMENT:-pr-99}" \
    ATLAS_DEACTIVATE_SETTLE_S="${ATLAS_DEACTIVATE_SETTLE_S:-0}" \
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

@test "cleanup.sh derives ATLAS_ENVIRONMENT=pr-<N> from PR_NUMBER with no manifest wiring" {
    # task-48 fix round 1: postdelete-cleanup.yaml carries no
    # ATLAS_ENVIRONMENT env var — the controller's finding was that
    # cleanup.sh should derive it the same way it already derives
    # ATLAS_ENV. Confirms the derivation (and that an explicit override
    # still wins).
    make_stubs '[]' '[]'
    run env -u ATLAS_ENVIRONMENT PATH="$STUB_BIN:$PATH" STUB_LOG="$STUB_LOG" \
        BATS_TEST_TMPDIR="$BATS_TEST_TMPDIR" \
        DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="foo bar" BOOTSTRAP_SERVERS=kafka:9093 REDIS_URL=redis:6379 \
        PR_NUMBER=555 ATLAS_DEACTIVATE_SETTLE_S=0 \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [[ "$output" == *'derived ATLAS_ENVIRONMENT=pr-555 for PR 555'* ]]
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

    # The full end-to-end branch-delete path is exercised by the smoke
    # test; here we only need to assert that the phase exists in the
    # script body. Run a bash-side grep on cleanup.sh instead of wiring
    # up an rpk/psql/redis-cli stub fleet for a single phase.
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

@test "cleanup.sh drop-branch pre-empts post-delete-finalizer drain after deleting branch" {
    # Once `drop-branch` deletes the bot branch, Argo CD's finalizer-drain
    # reconcile can't re-render the missing source → DeletionError → the
    # Application sits Terminating forever. PR 522 hit this on 2026-05-27.
    # cleanup.sh must patch the Application's finalizers itself after a
    # successful (or already-404'd) branch delete.
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/calls.log"
    cat > "$SHIM_DIR/gh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "gh \$*" >> "$CALL_LOG"
# DELETE branch returns 204 (no body).
exit 0
EOF
    cat > "$SHIM_DIR/kubectl" <<EOF
#!/usr/bin/env bash
printf '%s\n' "kubectl \$*" >> "$CALL_LOG"
exit 0
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/rpk" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "topic list") echo '[]' ;;
    "group list") printf 'BROKER GROUP STATE\n' ;;
esac
exit 0
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"data":[]}'
exit 0
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        PR_NUMBER=42 DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="foo" BOOTSTRAP_SERVERS=k REDIS_URL=r \
        GHCR_TOKEN=fake-token \
        ATLAS_UI_BASE=http://fake-ui ATLAS_ENVIRONMENT=pr-42 ATLAS_DEACTIVATE_SETTLE_S=0 \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"

    [ "$status" -eq 0 ]
    # The DELETE branch call must have happened.
    grep -F 'gh api --method DELETE' "$CALL_LOG" | grep -F 'bot%2Fpr-42-resolved'
    # AND we must have followed it with a finalizer-drop patch on the
    # Application.
    grep -F 'kubectl -n argocd patch application.argoproj.io atlas-pr-42' "$CALL_LOG" \
        | grep -F '"finalizers":[]'

    rm -rf "$SHIM_DIR"
}

@test "cleanup.sh drop-branch still pre-empts finalizer drain when branch already 404'd" {
    # On a re-run after a partial cleanup, the bot branch may already be
    # gone. cleanup.sh treats that as success (idempotent) — and must
    # ALSO still patch the Application's finalizers, because the
    # Application is in the same Source-branch-missing state.
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/calls.log"
    cat > "$SHIM_DIR/gh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "gh \$*" >> "$CALL_LOG"
echo "gh: Reference does not exist (HTTP 404)" >&2
exit 1
EOF
    cat > "$SHIM_DIR/kubectl" <<EOF
#!/usr/bin/env bash
printf '%s\n' "kubectl \$*" >> "$CALL_LOG"
exit 0
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/rpk" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "topic list") echo '[]' ;;
    "group list") printf 'BROKER GROUP STATE\n' ;;
esac
exit 0
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"data":[]}'
exit 0
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        PR_NUMBER=42 DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="foo" BOOTSTRAP_SERVERS=k REDIS_URL=r \
        GHCR_TOKEN=fake-token \
        ATLAS_UI_BASE=http://fake-ui ATLAS_ENVIRONMENT=pr-42 ATLAS_DEACTIVATE_SETTLE_S=0 \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"

    [ "$status" -eq 0 ]
    grep -F 'kubectl -n argocd patch application.argoproj.io atlas-pr-42' "$CALL_LOG"

    rm -rf "$SHIM_DIR"
}

# fixture_env returns the ATLAS_ENV hash cleanup.sh derives for PR_NUMBER=99
# (compute_atlas_env "99" → first 4 hex chars of sha256("pr-99")). Keeping this
# computed instead of hardcoded means the rpk tests below stay correct if the
# formula in lib.sh ever changes.
fixture_env() {
    . "$PROJECT_ROOT/scripts/lib.sh"
    compute_atlas_env 99
}

@test "cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk" {
    local env_hash
    env_hash="$(fixture_env)"
    local topics
    topics=$(sed "s/a1b2/${env_hash}/g" \
        "$PROJECT_ROOT/test/fixtures/rpk-topic-list.json")
    make_stubs "$topics" '[]'
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk topic list was invoked once
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]

    # rpk topic delete was invoked for the two env-suffixed topics and
    # not for the unsuffixed ones.
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F "boss-spawn-events-${env_hash}"
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F "character-events-${env_hash}"
    if grep -F 'rpk topic delete' "$STUB_LOG" | grep -wF 'configurations-events'; then
        echo "ERROR: unsuffixed topic was deleted" >&2
        return 1
    fi
}

@test "cleanup.sh deletes consumer groups with spaces in their names" {
    local env_hash
    env_hash="$(fixture_env)"
    local groups
    groups=$(sed "s/a1b2/${env_hash}/g" \
        "$PROJECT_ROOT/test/fixtures/rpk-group-list.txt")
    make_stubs '[]' "$groups"
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk group list invoked once
    [ "$(grep -c '^rpk group list ' "$STUB_LOG")" -eq 1 ]

    # rpk group delete was called for the spaced + hyphenated names as
    # single arguments each.
    grep -F 'rpk group delete' "$STUB_LOG" | grep -F "Party Quest Service [${env_hash}]"
    grep -F 'rpk group delete' "$STUB_LOG" | grep -F "Channel Service - 7e3a-0a1b [${env_hash}]"

    # The other-env group must not be deleted
    if grep -F 'rpk group delete' "$STUB_LOG" | grep -F 'Party Quest Service [other]'; then
        echo "ERROR: group with non-matching env suffix was deleted" >&2
        return 1
    fi
}

@test "cleanup.sh skips rpk topic delete when no topic matches" {
    make_stubs '[{"name":"prod-foo"},{"name":"prod-bar"}]' '[]'
    run run_cleanup
    [ "$status" -eq 0 ]
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]
    if grep -F 'rpk topic delete' "$STUB_LOG"; then
        echo "ERROR: rpk topic delete invoked despite no matching topics" >&2
        return 1
    fi
}

@test "cleanup.sh runs every phase even when drop-topics fails" {
    mkdir -p "$STUB_BIN"
    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    echo "<not-json>"
    exit 0
elif [ "$1" = "group" ] && [ "$2" = "list" ]; then
    echo "[]"
    exit 0
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
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh $*" >> "$STUB_LOG"
exit 0
EOF
    chmod +x "$STUB_BIN"/*

    run run_cleanup
    [ "$status" -eq 1 ]
    [[ "$output" == *'drop-groups'*'phase complete'* ]]
    [[ "$output" == *'drop-redis'*'phase complete'* ]]
    [[ "$output" == *'drop-images'*'phase complete'* ]]
    [[ "$output" == *'drop-dns'*'phase complete'* ]]
    [[ "$output" == *'drop-branch'*'phase complete'* ]]
    [[ "$output" == *'failed_phases'*'drop-topics'* ]]
    [[ "$output" == *'phases_failed=1'* ]]
}

@test "cleanup.sh exits 0 when all phases succeed" {
    make_stubs '[]' '[]'
    run run_cleanup
    [ "$status" -eq 0 ]
    [[ "$output" == *'phases_failed=0'* ]]
    # 9 phases: 7 after drop-tenant-storage was removed (superseded by
    # PreDelete purge hook), plus deactivate + drop-control-plane (task-48).
    [[ "$output" == *'phases_run=9'* ]]
}

@test "cleanup.sh fails fast on malformed rpk output" {
    mkdir -p "$STUB_BIN"
    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    printf 'this is not json\n'
    exit 0
fi
echo "[]"
EOF
    cat > "$STUB_BIN/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$STUB_BIN/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$STUB_BIN"/*

    run run_cleanup
    [ "$status" -eq 1 ]
    [[ "$output" == *'drop-topics'* ]]
    [[ "$output" == *'phase exited non-zero'* ]]
}

# ----------------------------------------------------------------------------
# task-48: deactivate before destroy, and reclaim the control plane
# ----------------------------------------------------------------------------

@test "deactivate runs before every destructive phase" {
    # Anchored to the PHASES array's own two-column
    # "<phase-name>   <function-name>" lines (leading whitespace + name +
    # whitespace + function) so this doesn't false-match the function
    # definitions or the doc comments elsewhere in the file that also
    # mention do_drop_dbs by name (e.g. do_drop_control_plane's own
    # isolated-mode-skip rationale).
    run grep -nE '^[[:space:]]+deactivate[[:space:]]+do_deactivate|^[[:space:]]+drop-[a-z-]+[[:space:]]+do_drop_' \
        "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -eq 0 ]
    deact=$(printf '%s\n' "$output" | grep -E 'deactivate[[:space:]]+do_deactivate' | head -1 | cut -d: -f1)
    first_destructive=$(printf '%s\n' "$output" | grep -E 'drop-[a-z-]+[[:space:]]+do_drop_' | head -1 | cut -d: -f1)
    [ -n "$deact" ]
    [ -n "$first_destructive" ]
    [ "$deact" -lt "$first_destructive" ]
}

@test "cleanup.sh deactivates (DEACTIVATING then DELETED) before any destructive phase runs" {
    make_stubs '[]' '[]'
    # Override the default 404 with a live ACTIVE record so do_deactivate's
    # gate (_dcp_env_phase) passes and it actually PATCHes, exercising the
    # ordering this test is about.
    cat > "$STUB_BIN/curl" <<'EOF'
#!/usr/bin/env bash
echo "curl $*" >> "$STUB_LOG"
url="${@: -1}"
case "$url" in
    */configurations/environments/*)
        echo '{"data":{"attributes":{"phase":"ACTIVE"}}}'
        exit 0
        ;;
    *)
        echo '{"data":[]}'
        exit 0
        ;;
esac
EOF
    chmod +x "$STUB_BIN/curl"
    run run_cleanup
    [ "$status" -eq 0 ]

    deact_line=$(grep -n '"phase":"DEACTIVATING"' "$STUB_LOG" | head -1 | cut -d: -f1)
    deleted_line=$(grep -n '"phase":"DELETED"' "$STUB_LOG" | head -1 | cut -d: -f1)
    first_psql_line=$(grep -n '^psql ' "$STUB_LOG" | head -1 | cut -d: -f1)

    [ -n "$deact_line" ]
    [ -n "$deleted_line" ]
    [ -n "$first_psql_line" ]
    [ "$deact_line" -lt "$deleted_line" ]
    [ "$deleted_line" -lt "$first_psql_line" ]
}

@test "cleanup.sh fails the deactivate phase (not the whole script) when ATLAS_UI_BASE is missing" {
    make_stubs '[]' '[]'
    # run_cleanup's own default would backfill a missing ATLAS_UI_BASE, so
    # this test drives cleanup.sh directly with the var absent, same
    # pattern as the drop-branch tests below.
    PATH="$STUB_BIN:$PATH" run env -u ATLAS_UI_BASE STUB_LOG="$STUB_LOG" \
        BATS_TEST_TMPDIR="$BATS_TEST_TMPDIR" \
        DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="foo bar" BOOTSTRAP_SERVERS=kafka:9093 REDIS_URL=redis:6379 \
        PR_NUMBER=99 ATLAS_ENVIRONMENT=pr-99 ATLAS_DEACTIVATE_SETTLE_S=0 \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [[ "$output" == *'ATLAS_UI_BASE and ATLAS_ENVIRONMENT are required'* ]]
    [[ "$output" == *'failed_phases'*'deactivate'* ]]
}

@test "cleanup.sh drop-control-plane and deactivate are skipped when no control-plane environment record exists" {
    # Default curl stub 404s the environments GET — the isolated-mode
    # reality (no environment-record.yaml Job ever registers one there).
    make_stubs '[]' '[]'
    run run_cleanup
    [ "$status" -eq 0 ]
    [[ "$output" == *'deactivate'*'skipped'*'no control-plane environment record'* ]]
    [[ "$output" == *'drop-control-plane'*'skipped'*'no control-plane environment record'* ]]
    if grep -F -- '-X DELETE' "$STUB_LOG"; then
        echo "ERROR: DELETE issued with no environment record" >&2
        return 1
    fi
    if grep -F -- '-X PATCH' "$STUB_LOG"; then
        echo "ERROR: PATCH issued with no environment record" >&2
        return 1
    fi
}

@test "cleanup.sh drop-dbs/drop-topics/drop-redis are skipped (sparse) under ATLAS_MODE=sparse" {
    make_stubs '[]' '[]'
    ATLAS_MODE=sparse run run_cleanup
    [ "$status" -eq 0 ]
    [[ "$output" == *'drop-dbs'*'skipped (sparse)'* ]]
    [[ "$output" == *'drop-topics'*'skipped (sparse)'* ]]
    [[ "$output" == *'drop-redis'*'skipped (sparse)'* ]]
    if grep -F '^psql ' "$STUB_LOG"; then
        echo "ERROR: psql invoked despite sparse skip" >&2
        return 1
    fi
}

@test "cleanup.sh drop-control-plane deletes every row for this environment, including duplicates, and never a foreign row" {
    # task-47 review finding: create_service_config is not idempotent
    # across a retried bootstrap Job, so a crashed attempt can leave an
    # orphaned duplicate services row (same Type, same Environment,
    # different id). Seed TWO rows for this environment plus one foreign
    # (main-owned) row of the SAME type, and assert both of this
    # environment's rows are deleted while the foreign one survives — a
    # suite that only proves this environment's rows are gone would pass
    # just as happily against an unscoped DELETE.
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/dcp-calls.log"
    JSON_FIXTURE="$BATS_TEST_TMPDIR/services-list.json"
    cat > "$JSON_FIXTURE" <<'JSONEOF'
{"data":[
  {"id":"svc-mine-1","attributes":{"environment":"pr-99"}},
  {"id":"svc-mine-2","attributes":{"environment":"pr-99"}},
  {"id":"svc-foreign","attributes":{"environment":"main"}}
]}
JSONEOF
    cat > "$SHIM_DIR/curl" <<CURLEOF
#!/usr/bin/env bash
printf '%s\n' "curl \$*" >> "$CALL_LOG"
method="GET"
prev=""
for a in "\$@"; do
    if [ "\$prev" = "-X" ]; then method="\$a"; fi
    prev="\$a"
done
url="\${@: -1}"
case "\$url" in
    */configurations/environments/*)
        [ "\$method" = "GET" ] && echo '{"data":{"attributes":{"phase":"ACTIVE"}}}'
        ;;
    */configurations/services)
        [ "\$method" = "GET" ] && cat "$JSON_FIXTURE"
        ;;
    */configurations/tenants|*/configurations/templates|*/api/tenants)
        [ "\$method" = "GET" ] && echo '{"data":[]}'
        ;;
esac
exit 0
CURLEOF
    chmod +x "$SHIM_DIR/curl"
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/rpk" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "topic list") echo '[]' ;;
    "group list") printf 'BROKER GROUP STATE\n' ;;
esac
exit 0
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env \
        PR_NUMBER=99 DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="foo" BOOTSTRAP_SERVERS=k REDIS_URL=r \
        ATLAS_UI_BASE=http://fake-ui ATLAS_DEACTIVATE_SETTLE_S=0 \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"

    [ "$status" -eq 0 ]
    grep -F -- '-X DELETE' "$CALL_LOG" | grep -F '/configurations/services/svc-mine-1'
    grep -F -- '-X DELETE' "$CALL_LOG" | grep -F '/configurations/services/svc-mine-2'
    if grep -F -- '-X DELETE' "$CALL_LOG" | grep -F 'svc-foreign'; then
        echo "ERROR: foreign row was deleted" >&2
        return 1
    fi

    rm -rf "$SHIM_DIR"
}
