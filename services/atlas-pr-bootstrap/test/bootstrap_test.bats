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

@test "svc_id_var_name itself carries no incidental trailing newline" {
    # NOT the Task 1 no-trailing-newline contract (svc_id_var_name builds an
    # env-VAR-NAME string like SERVICE_ID_LOGIN_SERVICE from a service-type
    # string — it never touches a derived UUID). This just pins that its own
    # output is exactly the expected bytes, via printf rather than echo. The
    # real Task 1 call boundary is svc_id="${!svc_id_var:-}" in
    # upsert_sparse_service_config, pinned separately below.
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

@test "a no-trailing-newline derived id flows through svc_id=\"\${!svc_id_var:-}\" to the GET URL intact" {
    # The actual Task 1 call boundary: bootstrap.sh's
    # svc_id="${!svc_id_var:-}" reads the id tools/derive-service-id.sh
    # derived and the CI rendering exported as SERVICE_ID_LOGIN_SERVICE, and
    # the very next line uses it unmodified as
    # "$ATLAS_UI_BASE/api/configurations/services/$svc_id" in the GET. Drive
    # the REAL id (no trailing newline, per derive-service-id.sh) through the
    # real assignment and the real GET, and capture the exact URL curl was
    # invoked with: a stray appended newline or a silently truncated id would
    # show up there byte-for-byte.
    #
    # The GET (not the POST) is the boundary to pin here: build_service_config
    # (the POST path) needs a real canonical template file on disk, which
    # only exists inside the built image (/atlas/...), not on a bats host.
    # The GET fires unconditionally, before any template is read, so it is
    # both the true first use of svc_id AND reachable without one.
    command -v python3 >/dev/null || skip "python3 required"
    local repo_root id
    repo_root="$(cd "$PROJECT_ROOT/../.." && pwd)"
    id=$("$repo_root/tools/derive-service-id.sh" login-service pr-test)
    [[ "$id" != *$'\n'* ]]

    load_fn svc_table_lookup
    load_fn svc_id_var_name
    load_fn upsert_sparse_service_config
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
    # shellcheck source=../scripts/version-ports.sh
    . "$PROJECT_ROOT/scripts/version-ports.sh"
    # shellcheck source=../scripts/service-config.sh
    . "$PROJECT_ROOT/scripts/service-config.sh"

    export ATLAS_MODE=sparse ATLAS_ENVIRONMENT=pr-test
    export ATLAS_UI_BASE=http://atlas-ingress.test.svc.cluster.local
    export TENANT_ID=11111111-1111-1111-1111-111111111111 MAJOR_VERSION=83
    export SERVICE_ID_LOGIN_SERVICE="$id"
    ENV_HEADER=(-H "ENVIRONMENT: $ATLAS_ENVIRONMENT")

    # curl shim: captures the GET's URL (its last argv) byte-for-byte, and
    # answers with an existing row whose tenants[] already equals what
    # build_login_entry would compute — so upsert_sparse_service_config takes
    # the "matches; skip PATCH" branch and issues no second curl call. $id
    # and $TENANT_ID are interpolated at script-generation time (this
    # heredoc is unquoted), not read from the shim's own environment.
    local dir="$BATS_TEST_TMPDIR/bin" url_capture="$BATS_TEST_TMPDIR/get-url.txt"
    mkdir -p "$dir"
    cat >"$dir/curl" <<EOF
#!/usr/bin/env bash
last="\${*: -1}"
printf '%s' "\$last" > "$url_capture"
printf '{"data":{"id":"%s","attributes":{"tenants":[{"id":"%s","port":8300}]}}}' "$id" "$TENANT_ID"
exit 0
EOF
    chmod +x "$dir/curl"
    ln -sf "$(command -v jq)" "$dir/jq"

    PATH="$dir:$PATH" run upsert_sparse_service_config atlas-login
    [ "$status" -eq 0 ]
    [[ "$output" == *"matches; skipping PATCH"* ]]
    [ -f "$url_capture" ]
    # wc -c on the raw captured file (not a $()-captured comparand) so a
    # stray trailing newline the boundary might append cannot be silently
    # stripped by command substitution before the check sees it.
    local expected="$ATLAS_UI_BASE/api/configurations/services/$id"
    [ "$(wc -c <"$url_capture")" -eq "${#expected}" ]
    [ "$(cat "$url_capture")" = "$expected" ]
}

# --- activation (task-243 FR-4.1/FR-4.2/D5) --------------------------------

@test "activation_decision: PROVISIONING activates" {
    load_fn activation_decision
    run activation_decision PROVISIONING
    [ "$status" -eq 0 ]
    [ "$output" = "activate" ]
}

@test "activation_decision: ACTIVE skips" {
    load_fn activation_decision
    run activation_decision ACTIVE
    [ "$status" -eq 0 ]
    [ "$output" = "skip" ]
}

@test "activation_decision: DEACTIVATING fails" {
    load_fn activation_decision
    run activation_decision DEACTIVATING
    [ "$status" -eq 0 ]
    [ "$output" = "fail" ]
}

@test "activation_decision: DELETED fails" {
    load_fn activation_decision
    run activation_decision DELETED
    [ "$status" -eq 0 ]
    [ "$output" = "fail" ]
}

@test "activation_decision: no record (empty phase) fails" {
    load_fn activation_decision
    run activation_decision ""
    [ "$status" -eq 0 ]
    [ "$output" = "fail" ]
}

@test "activation is sparse-only" {
    # bootstrap_test.bats has no existing seam that runs the script far
    # enough (past require_env / the network-hitting preflight steps) to
    # observe ATLAS_STEP=activate never being set under ATLAS_MODE=isolated
    # without a live cluster + atlas-ui-base to talk to. Absent that seam,
    # this pins the guard expression itself by grepping the script — a weak
    # test (it would not catch e.g. the condition being inverted and then
    # separately negated), but it is honest about what it checks rather
    # than inventing a black-box seam that doesn't exist yet.
    run grep -F '[ "${ATLAS_MODE:-isolated}" = "sparse" ]' "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -eq 0 ]
}
