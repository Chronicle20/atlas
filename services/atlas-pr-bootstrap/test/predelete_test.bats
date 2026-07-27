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
    # Every curl invocation's full argument list is appended here so tests
    # can assert on headers sent (e.g. the tenant headers on the DELETE).
    export CURL_ARGS_FILE="$BATS_TEST_TMPDIR/curl_args"
    # Counter used by the delete_retry_then_ok MODE to flip the DELETE
    # response from 503 (1st call) to 202 (2nd+ call).
    export DELETE_COUNT_FILE="$BATS_TEST_TMPDIR/delete_count"
    # Keep the retry loop's sleep out of the test's wall-clock time.
    export PURGE_DELETE_RETRY_SLEEP="0"
}

teardown() {
    rm -rf "$SHIM_DIR"
}

write_curl() {
    local mode="${1:-ok}"
    printf '%s\n' "$mode" > "$CURL_MODE_FILE"
    cat > "$SHIM_DIR/curl" <<'SHIM'
#!/usr/bin/env bash
if [ -n "${CURL_ARGS_FILE:-}" ]; then
    printf '%s\n' "$*" >> "$CURL_ARGS_FILE"
fi
url=""
method="GET"
while [ $# -gt 0 ]; do
    case "$1" in
        -X) method="$2"; shift 2;;
        -o) shift 2;;               # skip output file arg
        -w) shift 2;;               # skip write-out format arg
        -H) shift 2;;               # skip header value too
        -s|-f|-S|--retry*) shift;;
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
            delete_retry_then_ok)
                count_file="${DELETE_COUNT_FILE:-/tmp/delete_count}"
                count=0
                [ -f "$count_file" ] && count="$(cat "$count_file")"
                count=$((count + 1))
                echo "$count" > "$count_file"
                if [ "$count" -eq 1 ]; then
                    echo "503"
                else
                    echo "202"
                fi
                ;;
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

@test "predelete: DELETE sends synthetic tenant headers" {
    write_curl ok
    run bash "$SCRIPT"
    [ "$status" -eq 0 ]
    delete_args="$(grep -F '/api/data/tenants/' "$CURL_ARGS_FILE")"
    [[ "$delete_args" == *"TENANT_ID: "* ]]
    [[ "$delete_args" == *"REGION: "* ]]
    [[ "$delete_args" == *"MAJOR_VERSION: "* ]]
    [[ "$delete_args" == *"MINOR_VERSION: "* ]]
}

@test "predelete: DELETE retries a transient failure then succeeds" {
    write_curl delete_retry_then_ok
    run bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"purged tenant"* ]]
}
