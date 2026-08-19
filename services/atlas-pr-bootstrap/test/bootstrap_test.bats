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

# --- sparse service-config creation (task-232 follow-up) ------------------

@test "create_service_config sends the ENVIRONMENT header on POST" {
    # The row's environment column is server-owned — atlas-configurations
    # stamps it from env.MustFromContext(ctx) and deliberately ignores the
    # body's .data.attributes.environment. Without this header every sparse
    # row landed with environment='' in the SHARED baseline, where
    # cleanup.sh's environment-scoped reclaim can never match it.
    local fn
    fn=$(sed -n '/^create_service_config()/,/^}/p' "$PROJECT_ROOT/scripts/bootstrap.sh")
    [ -n "$fn" ]
    [[ "$fn" == *'-H "ENVIRONMENT: $ATLAS_ENVIRONMENT"'* ]]
}

@test "create_service_config refuses an empty service id" {
    # `kubectl set env SERVICE_ID=` does not fail on an empty value — it
    # writes an env entry with no value, which panics the consumer at
    # startup. The guard must sit before that call.
    local fn
    fn=$(sed -n '/^create_service_config()/,/^}/p' "$PROJECT_ROOT/scripts/bootstrap.sh")
    local guard_line set_env_line
    guard_line=$(printf '%s\n' "$fn" | grep -n 'refusing to continue with empty service id' | cut -d: -f1 | head -1)
    # anchored: the phrase also appears in this function's comments
    set_env_line=$(printf '%s\n' "$fn" | grep -n '^ *kubectl set env' | cut -d: -f1 | head -1)
    [ -n "$guard_line" ]
    [ -n "$set_env_line" ]
    [ "$guard_line" -lt "$set_env_line" ]
}
