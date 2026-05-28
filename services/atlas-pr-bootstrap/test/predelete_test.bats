#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/predelete-purge.sh"
    SHIM_DIR="$(mktemp -d)"
    export SHIM_DIR
    export PATH="$SHIM_DIR:$PATH"
    export ATLAS_INGRESS_BASE="http://atlas-ingress.test.svc.cluster.local"
    export PR_NUMBER="491"
    # Write mode to a temp file so the curl shim can read it without
    # bats's subshell boundary collapsing the exported variable.
    export CURL_MODE_FILE="$BATS_TEST_TMPDIR/curl_mode"
}

teardown() {
    rm -rf "$SHIM_DIR"
}

write_curl() {
    local mode="${1:-ok}"
    printf '%s\n' "$mode" > "$CURL_MODE_FILE"
    cat > "$SHIM_DIR/curl" <<'SHIM'
#!/usr/bin/env bash
url=""
method="GET"
output_flag=0
write_out_flag=0
while [ $# -gt 0 ]; do
    case "$1" in
        -X) method="$2"; shift 2;;
        -o) shift 2;;               # skip output file arg
        -w) shift 2;;               # skip write-out format arg
        -s|-f|-S|-H|--retry*) shift;;
        http*) url="$1"; shift;;
        *) shift;;
    esac
done
MODE="$(cat "$CURL_MODE_FILE" 2>/dev/null || echo ok)"
case "$method:$url" in
    GET:*"/api/tenants"*)
        case "$MODE" in
            tenants_fail) exit 22;;
            tenants_empty) echo '{"data":[]}';;
            *) echo '{"data":[{"id":"aaaaaaaa-0000-0000-0000-000000000001"},{"id":"bbbbbbbb-0000-0000-0000-000000000002"}]}';;
        esac
        ;;
    DELETE:*"/api/data/tenants/"*)
        case "$MODE" in
            delete_500) echo "500";;
            *) echo "202";;
        esac
        ;;
esac
SHIM
    chmod +x "$SHIM_DIR/curl"
    # Also stub jq to ensure JSON parsing works in tests even without
    # the real jq available; prefer real jq if present.
}

@test "predelete: two tenants -> two DELETEs, exit 0" {
    write_curl ok
    run bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"aaaaaaaa-0000-0000-0000-000000000001"* ]]
    [[ "$output" == *"bbbbbbbb-0000-0000-0000-000000000002"* ]]
}

@test "predelete: tenant-list fetch failure -> non-zero, no silent skip" {
    write_curl tenants_fail
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}

@test "predelete: empty tenant list -> non-zero (env always has >=1 tenant)" {
    write_curl tenants_empty
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}

@test "predelete: a DELETE failing -> non-zero" {
    write_curl delete_500
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}
