#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "bootstrap.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "bootstrap.sh fails without ATLAS_UI_BASE" {
    run env -u ATLAS_UI_BASE ATLAS_ENV=test bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_UI_BASE"* ]]
}

# --- task-098: baseline preflight ---------------------------------------

# Common env for a script run that should reach the preflight. TENANT_ID
# must be UUID-shaped or the earlier shape-check exits first.
prq_env() {
    echo ATLAS_ENV=test
    echo ATLAS_UI_BASE=http://atlas-ingress.test.svc.cluster.local
    echo TENANT_ID=00000000-0000-0000-0000-000000000001
    echo REGION=GMS
    echo MAJOR_VERSION=83
    echo MINOR_VERSION=1
    echo MINIO_ENDPOINT=http://minio.test:9000
    echo MINIO_PROBE_RETRIES=1
    echo MINIO_PROBE_SLEEP=0
    echo "CANONICAL_TENANT_JSON=$BATS_TEST_TMPDIR/tenant.json"
}

# Build a PATH dir containing a curl shim (emits $1 for every HEAD probe)
# and a kubectl shim (touches a sentinel so we can prove it never ran).
# Real jq is symlinked through so the script can still parse tenant.json.
make_shims() {
    local curl_code="$1"
    local dir="$BATS_TEST_TMPDIR/bin"
    mkdir -p "$dir"

    cat >"$dir/curl" <<EOF
#!/usr/bin/env bash
# baseline_object_status calls: curl -sS -o /dev/null -w '%{http_code}' -I <url>
if [ "$curl_code" = "000" ]; then echo 000; exit 7; fi
echo "$curl_code"
EOF

    cat >"$dir/kubectl" <<EOF
#!/usr/bin/env bash
touch "$BATS_TEST_TMPDIR/kubectl-ran"
exit 0
EOF

    ln -sf "$(command -v jq)" "$dir/jq"
    chmod +x "$dir/curl" "$dir/kubectl"
    echo "$dir"
}

write_fixture_tenant() {
    cat >"$BATS_TEST_TMPDIR/tenant.json" <<'EOF'
{"data":{"attributes":{"region":"GMS","majorVersion":83,"minorVersion":1}}}
EOF
}

@test "bootstrap.sh fails fast when no canonical baseline (404)" {
    command -v jq >/dev/null || skip "jq required"
    command -v timeout >/dev/null || skip "timeout required"
    write_fixture_tenant
    local bindir; bindir="$(make_shims 404)"
    run timeout 15 env $(prq_env) PATH="$bindir:$PATH" \
        bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"no canonical baseline"* ]]
    [[ "$output" == *"83.1"* ]]
    [[ "$output" == *"canonical-version-migration"* ]]
    # Preflight must run BEFORE any cluster mutation.
    [ ! -f "$BATS_TEST_TMPDIR/kubectl-ran" ]
}

@test "bootstrap.sh reports MinIO-unreachable distinctly (000)" {
    command -v jq >/dev/null || skip "jq required"
    command -v timeout >/dev/null || skip "timeout required"
    write_fixture_tenant
    local bindir; bindir="$(make_shims 000)"
    run timeout 15 env $(prq_env) PATH="$bindir:$PATH" \
        bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"MinIO unreachable"* ]]
    [[ "$output" != *"no canonical baseline"* ]]
    [ ! -f "$BATS_TEST_TMPDIR/kubectl-ran" ]
}

# --- sparse service-config upsert (task-243) -------------------------------
#
# bootstrap.sh's top-level code runs on source (require_env exits
# immediately), so these tests extract just the pure/guard-only function
# bodies with sed and eval them into the current shell — the same seam the
# two pre-existing service-config tests above use, extended here to
# actually invoke (not just grep) the extracted functions.

load_fn() {
    eval "$(sed -n "/^$1() {/,/^}/p" "$PROJECT_ROOT/scripts/bootstrap.sh")"
}

@test "sparse service table maps every SERVICE_ID-carrying deployment" {
    load_fn svc_table_lookup

    run svc_table_lookup atlas-login
    [ "$status" -eq 0 ]
    [ "$output" = "login-service login /atlas/canonical/services/login-service.json" ]

    run svc_table_lookup atlas-channel
    [ "$status" -eq 0 ]
    [ "$output" = "channel-service channel /atlas/canonical/services/channel-service.json" ]

    run svc_table_lookup atlas-drops
    [ "$status" -eq 0 ]
    [ "$output" = "drops-service none /atlas/canonical/services/drops-service.json" ]

    # atlas-world / atlas-character-factory / atlas-drop-information have no
    # canonical template baked into this image today (only login-service,
    # channel-service and drops-service do — see canonical/services/), so
    # they are deliberately left unmapped rather than fabricated.
    run svc_table_lookup atlas-world
    [ "$status" -ne 0 ]
    run svc_table_lookup atlas-character-factory
    [ "$status" -ne 0 ]
    run svc_table_lookup atlas-drop-information
    [ "$status" -ne 0 ]
}

@test "service id env var name is derived from the service type" {
    load_fn svc_id_var_name
    [ "$(svc_id_var_name login-service)" = "SERVICE_ID_LOGIN_SERVICE" ]
    [ "$(svc_id_var_name drops-information-service)" = "SERVICE_ID_DROPS_INFORMATION_SERVICE" ]
    [ "$(svc_id_var_name character-factory)" = "SERVICE_ID_CHARACTER_FACTORY" ]
}

@test "svc_id_var_name does not assume or append a trailing newline" {
    # tools/derive-service-id.sh (task-243 Task 1) emits its id with NO
    # trailing newline. svc_id_var_name is the boundary that turns a
    # SERVICE_ID_<TYPE> lookup into the exact key the CI rendering wrote —
    # pin that its own output carries no stray newline either, so a caller
    # concatenating it (e.g. `${!svc_id_var}`) never silently picks up one.
    load_fn svc_id_var_name
    local out
    out=$(svc_id_var_name login-service | wc -c)
    # "SERVICE_ID_LOGIN_SERVICE" is 24 bytes; wc -c on a $()-captured,
    # already-newline-stripped string only equals 24 if printf (not echo)
    # produced it with nothing appended.
    [ "$out" -eq 24 ]
}

@test "sparse service-config step fails when the CI-rendered id is absent" {
    load_fn svc_table_lookup
    load_fn svc_id_var_name
    load_fn upsert_sparse_service_config
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"

    unset SERVICE_ID_LOGIN_SERVICE
    run upsert_sparse_service_config atlas-login
    [ "$status" -ne 0 ]
    [[ "$output" == *"no SERVICE_ID_"* ]]
}
