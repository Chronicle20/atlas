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

@test "sweep-orphans.sh: phase names appear in --list output" {
    # Mock infra commands to emit one fake resource each, so list mode
    # produces the canonical "phase resource" lines and APPLY=0 means none
    # of them get acted on.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/kafka-topics.sh" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--list*) echo "atlas-faketopic-ed86" ;;
    *--delete*) echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/kafka-consumer-groups.sh" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--list*) echo "Fake Group [ed86]" ;;
    *--delete*) echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--scan*) echo "ed86:fake-key" ;;
    *DEL*)    echo "FAIL: DEL invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
echo "atlas-fake-ed86"
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
# Empty results — easier than mocking the rich gh api jq path.
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1   # "Application not found" — drop-app-finalizers phase no-ops.
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env \
        DB_HOST=fake DB_PORT=1 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=fake REDIS_URL=fake:6379 \
        GHCR_TOKEN=fake ATLAS_SERVICES=atlas-fake \
        bash "$SCRIPT" 491

    [[ "$output" == *"drop-dbs atlas-fake-ed86"* ]]
    [[ "$output" == *"drop-topics atlas-faketopic-ed86"* ]]
    [[ "$output" == *"drop-groups Fake Group [ed86]"* ]]
    [[ "$output" == *"drop-redis ed86:fake-key"* ]]
    [[ "$output" != *"FAIL:"* ]]

    rm -rf "$SHIM_DIR"
}
