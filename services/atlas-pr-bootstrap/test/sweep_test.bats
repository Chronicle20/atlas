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
    cat > "$SHIM_DIR/rpk" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "topic list")
        echo '[{"name":"atlas-faketopic-ed86","partitions":1,"replicas":1}]'
        ;;
    "group list")
        # rpk 24.3.1 `group list` has no --format flag; emit the raw
        # table that lib.sh's rpk_group_names_awk parses.
        printf 'BROKER  GROUP             STATE\n1       Fake Group [ed86]  Stable\n'
        ;;
    "topic delete"|"group delete")
        echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
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
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1
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

@test "sweep-orphans.sh: --apply deletes spaced group names intact" {
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/rpk-calls.log"
    cat > "$SHIM_DIR/rpk" <<EOF
#!/usr/bin/env bash
printf '%s\n' "rpk \$*" >> "$CALL_LOG"
case "\$1 \$2" in
    "topic list") echo '[]' ;;
    "group list")
        # rpk 24.3.1 `group list` table — no --format flag.
        printf 'BROKER  GROUP                       STATE\n1       Party Quest Service [ed86]  Stable\n'
        ;;
esac
exit 0
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        DB_HOST=fake DB_PORT=1 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=fake REDIS_URL=fake:6379 \
        GHCR_TOKEN=fake ATLAS_SERVICES=atlas-fake \
        bash "$SCRIPT" --apply 491

    grep -F "rpk group delete -X brokers=fake Party Quest Service [ed86]" "$CALL_LOG"

    rm -rf "$SHIM_DIR"
}

@test "sweep-orphans.sh: --minio rejects PR_NUMBER arg" {
    run bash "$SCRIPT" --minio 491
    [ "$status" -ne 0 ]
    [[ "$output" == *"--minio takes no PR_NUMBER"* ]]
}

@test "sweep-orphans.sh: --minio (no args) skips when MINIO_ENDPOINT unset" {
    run env -u MINIO_ENDPOINT bash "$SCRIPT" --minio
    [ "$status" -eq 0 ]
    [[ "$output" == *"MINIO_ENDPOINT not set"* ]]
    [[ "$output" == *"sweep complete"* ]]
}

@test "sweep-orphans.sh: --minio aborts if active tenants fetch fails" {
    # mc stub never invoked because curl fails first; sweep should
    # refuse to operate on empty allowlist.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/mc" <<'EOF'
#!/usr/bin/env bash
echo "FAIL: mc should not be called when atlas-main tenants fetch fails" >&2
exit 99
EOF
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
# Simulate atlas-main unreachable.
exit 6
EOF
    # kubectl is not reached because curl fails first; shim exits cleanly
    # so it never masks a real kubectl-not-found error on this path.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
echo "FAIL: kubectl should not be called when main tenant fetch fails" >&2
exit 99
EOF
    chmod +x "$SHIM_DIR"/*
    PATH="$SHIM_DIR:$PATH" run env \
        MINIO_ENDPOINT=fake:9000 \
        MINIO_ACCESS_KEY=x MINIO_SECRET_KEY=y \
        bash "$SCRIPT" --minio
    [[ "$output" != *"FAIL:"* ]]
    [[ "$output" == *"aborting MinIO sweep"* ]]
    rm -rf "$SHIM_DIR"
}

@test "sweep-orphans.sh: --minio --apply deletes orphan UUIDs but not active ones" {
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/mc-calls.log"

    # atlas-main returns one active UUID (this must NOT be deleted).
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *"/api/tenants"*)
        echo '{"data":[{"type":"tenants","id":"ec876921-c363-4cc6-9c51-5bb8d57f9553"}]}'
        ;;
    *)
        exit 7 ;;
esac
EOF

    # kubectl returns empty: no live PR namespaces → no extra allowlist entries.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

    # mc lists three UUIDs per bucket: the active one + two orphans
    # (one fresh, one old). The fresh orphan must be skipped by the
    # safety window; the old orphan must be deleted.
    cat > "$SHIM_DIR/mc" <<EOF
#!/usr/bin/env bash
printf '%s\n' "mc \$*" >> "$CALL_LOG"
case "\$*" in
    "alias set"*) exit 0 ;;
    "ls bee/atlas-wz/tenants/"|"ls bee/atlas-assets/tenants/"|"ls bee/atlas-renders/tenants/")
        printf '[2026-05-27 00:00 UTC]     0B ec876921-c363-4cc6-9c51-5bb8d57f9553/\n'
        printf '[2026-05-27 00:00 UTC]     0B aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/\n'
        printf '[2026-05-27 00:00 UTC]     0B bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/\n'
        ;;
    "ls --recursive --json bee/atlas-wz/tenants/aaaaaaaa-"*|"ls --recursive --json bee/atlas-assets/tenants/aaaaaaaa-"*|"ls --recursive --json bee/atlas-renders/tenants/aaaaaaaa-"*)
        # Old orphan — last-modified > safety window ago.
        printf '{"type":"file","lastModified":"2024-01-01T00:00:00Z"}\n'
        ;;
    "ls --recursive --json bee/atlas-wz/tenants/bbbbbbbb-"*|"ls --recursive --json bee/atlas-assets/tenants/bbbbbbbb-"*|"ls --recursive --json bee/atlas-renders/tenants/bbbbbbbb-"*)
        # Fresh orphan — last-modified now (within safety window).
        printf '{"type":"file","lastModified":"%s"}\n' "\$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        ;;
    "rm --recursive --force"*) exit 0 ;;
    *) exit 0 ;;
esac
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        MINIO_ENDPOINT=fake:9000 \
        MINIO_ACCESS_KEY=x MINIO_SECRET_KEY=y \
        ATLAS_MAIN_TENANTS_URL=http://fake/api/tenants \
        bash "$SCRIPT" --minio --apply

    [ "$status" -eq 0 ]
    # Active main UUID must never be touched.
    if grep -F "rm --recursive --force bee/atlas-wz/tenants/ec876921" "$CALL_LOG"; then
        echo "ERROR: active main tenant was deleted" >&2
        return 1
    fi
    # Old orphan (aaa) must be deleted across all three buckets.
    grep -F "rm --recursive --force bee/atlas-wz/tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/" "$CALL_LOG"
    grep -F "rm --recursive --force bee/atlas-assets/tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/" "$CALL_LOG"
    grep -F "rm --recursive --force bee/atlas-renders/tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/" "$CALL_LOG"
    # Fresh orphan (bbb) must be skipped by the safety window.
    if grep -F "rm --recursive --force bee/atlas-wz/tenants/bbbbbbbb-" "$CALL_LOG"; then
        echo "ERROR: fresh orphan within safety window was deleted" >&2
        return 1
    fi
    # And the list output should announce "drop-minio" lines.
    [[ "$output" == *"drop-minio atlas-wz/tenants/aaaaaaaa-"* ]]

    rm -rf "$SHIM_DIR"
}

@test "sweep-orphans.sh: --minio --list (default) does not call rm" {
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/mc-calls.log"
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *"/api/tenants"*)
        echo '{"data":[{"type":"tenants","id":"ec876921-c363-4cc6-9c51-5bb8d57f9553"}]}'
        ;;
    *) exit 7 ;;
esac
EOF
    # kubectl returns empty: no live PR namespaces → no extra allowlist entries.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/mc" <<EOF
#!/usr/bin/env bash
printf '%s\n' "mc \$*" >> "$CALL_LOG"
case "\$*" in
    "alias set"*) exit 0 ;;
    "ls bee/atlas-wz/tenants/"|"ls bee/atlas-assets/tenants/"|"ls bee/atlas-renders/tenants/")
        printf '[2024-01-01 00:00 UTC]     0B aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/\n'
        ;;
    "ls --recursive --json"*)
        printf '{"type":"file","lastModified":"2024-01-01T00:00:00Z"}\n'
        ;;
    "rm --recursive --force"*) exit 0 ;;
    *) exit 0 ;;
esac
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        MINIO_ENDPOINT=fake:9000 \
        MINIO_ACCESS_KEY=x MINIO_SECRET_KEY=y \
        ATLAS_MAIN_TENANTS_URL=http://fake/api/tenants \
        bash "$SCRIPT" --minio

    [ "$status" -eq 0 ]
    [[ "$output" == *"drop-minio atlas-wz/tenants/aaaaaaaa-"* ]]
    if grep -F "rm --recursive --force" "$CALL_LOG"; then
        echo "ERROR: rm called in --list (default) mode" >&2
        return 1
    fi

    rm -rf "$SHIM_DIR"
}

@test "sweep_minio: protects a live PR-env tenant, deletes a true orphan" {
    SHIM_DIR="$(mktemp -d)"
    # kubectl returns one live PR namespace.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
echo "atlas-pr-700"
EOF
    # curl: atlas-main returns UUID 111...; atlas-pr-700 returns UUID 222...
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
for a in "$@"; do url="$a"; done
case "$url" in
    *atlas-main*)    echo '{"data":[{"id":"11111111-1111-1111-1111-111111111111"}]}';;
    *atlas-pr-700*)  echo '{"data":[{"id":"22222222-2222-2222-2222-222222222222"}]}';;
    *)               echo '{"data":[]}';;
esac
EOF
    # mc lists three UUIDs: one main-active, one PR-env-active, one true orphan.
    # Recursive JSON listings return an old timestamp so the orphan clears the
    # safety window (MINIO_TENANT_SAFETY_WINDOW_SEC=0 → age threshold is 0s,
    # so any non-zero age passes; returning empty also works since the safety
    # block is skipped when last_mod_iso is empty).
    cat > "$SHIM_DIR/mc" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    "alias set"*) exit 0;;
    "ls --recursive --json"*) exit 0;;
    "ls"*)
        printf '[2024-01-01 00:00 UTC]     0B 11111111-1111-1111-1111-111111111111/\n'
        printf '[2024-01-01 00:00 UTC]     0B 22222222-2222-2222-2222-222222222222/\n'
        printf '[2024-01-01 00:00 UTC]     0B 33333333-3333-3333-3333-333333333333/\n'
        ;;
    "rm"*) exit 0;;
esac
EOF
    chmod +x "$SHIM_DIR"/*
    run env PATH="$SHIM_DIR:$PATH" \
        MINIO_ENDPOINT="minio.test:9000" MINIO_ROOT_USER=u MINIO_ROOT_PASSWORD=p \
        MINIO_TENANT_SAFETY_WINDOW_SEC=0 \
        bash "$SCRIPT" --minio --apply
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    # True orphan must be announced for deletion.
    [[ "$output" == *"33333333-3333-3333-3333-333333333333"* ]]
    # Main-active and PR-env-active UUIDs must NOT be deleted.
    [[ "$output" != *"drop-minio"*"22222222-2222-2222-2222-222222222222"* ]]
    [[ "$output" != *"drop-minio"*"11111111-1111-1111-1111-111111111111"* ]]
    rm -rf "$SHIM_DIR"
}

@test "sweep_minio: namespace-enumeration failure aborts (fail-closed)" {
    SHIM_DIR="$(mktemp -d)"
    # kubectl fails → sweep must abort, never touch MinIO.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
    # curl succeeds for atlas-main so we get past the first allowlist fetch.
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"data":[{"id":"11111111-1111-1111-1111-111111111111"}]}'
EOF
    cat > "$SHIM_DIR/mc" <<'EOF'
#!/usr/bin/env bash
[ "$1 $2" = "alias set" ] && exit 0
exit 0
EOF
    chmod +x "$SHIM_DIR"/*
    run env PATH="$SHIM_DIR:$PATH" \
        MINIO_ENDPOINT="minio.test:9000" MINIO_ROOT_USER=u MINIO_ROOT_PASSWORD=p \
        bash "$SCRIPT" --minio --apply
    [ "$status" -ne 0 ]
    [[ "$output" == *"abort"* ]] || [[ "$output" == *"enumerat"* ]]
    rm -rf "$SHIM_DIR"
}
