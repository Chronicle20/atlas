#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/reclaim-main-bare-keys.sh"
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
args="$*"
if [[ "$args" == *"--scan"* ]]; then
    pat=""
    while [ $# -gt 0 ]; do
        if [ "$1" = "--pattern" ]; then pat="$2"; fi
        shift
    done
    case "$pat" in
        atlas:*|*:atlas:*) ;;                # must never be scanned
        "channel:tenants"|"drops:all"|"reactors:all"|"coordinator:active"|"invite:active-tenants"|"transport:instances"|"transport:characters")
            echo "$pat" ;;
        *) echo "${pat%\*}fake" ;;            # keyed families -> one fake key
    esac
    exit 0
fi
# Skip connection flags (-h host -p port / -u uri) before checking command.
while [ $# -gt 0 ]; do
    case "$1" in
        -h|-p|-u|-a|-n) shift 2 ;;
        DEL)
            shift
            printf '%s\n' "$@" >> "$SHIM_DIR/deleted.txt"
            exit 0 ;;
        *) break ;;
    esac
done
exit 0
EOF
    chmod +x "$SHIM_DIR/redis-cli"
    export PATH="$SHIM_DIR:$PATH"
    export SHIM_DIR
}

@test "reclaim: list mode (default) deletes nothing" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [ ! -f "$SHIM_DIR/deleted.txt" ]
}

@test "reclaim: --apply deletes only allowlisted bare keys, never atlas:*" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT" --apply
    [ "$status" -eq 0 ]
    [ -f "$SHIM_DIR/deleted.txt" ]
    run grep -E '(^atlas:|:atlas:)' "$SHIM_DIR/deleted.txt"
    [ "$status" -ne 0 ]
    grep -qx "channel:tenants" "$SHIM_DIR/deleted.txt"
}

@test "reclaim: idempotent re-run" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT" --apply
    [ "$status" -eq 0 ]
}
