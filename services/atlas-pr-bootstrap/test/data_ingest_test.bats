#!/usr/bin/env bats
#
# The data-ingest guard decides whether to run a DESTRUCTIVE baseline restore.
# baseline.Rewriter rewrites only the tenant_id column and copies primary keys
# verbatim, so a restore into a database that already holds the canonical rows
# fails on documents_pkey and rolls the target tenant back to empty. Getting
# this guard wrong is not a redundant no-op; it wipes the tenant it targets.

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    BOOTSTRAP="$PROJECT_ROOT/scripts/bootstrap.sh"

    # bootstrap.sh is not sourceable — it runs its whole flow at top level.
    # Extract just the two helpers under test into a file we can source, so
    # the assertions exercise the real definitions rather than a copy.
    HELPERS="$BATS_TEST_TMPDIR/helpers.sh"
    {
        sed -n '/^get_attr()/,/^}/p' "$BOOTSTRAP"
        sed -n '/^document_count()/,/^}/p' "$BOOTSTRAP"
    } >"$HELPERS"
    # shellcheck source=/dev/null
    . "$HELPERS"

    # Prove the extraction found both helpers. Without this, a missing
    # definition makes `run` exit 127 — which satisfies every "must fail"
    # assertion below and turns the whole suite green for the wrong reason.
    declare -F get_attr >/dev/null || {
        echo "get_attr not extracted from $BOOTSTRAP" >&2
        return 1
    }
    declare -F document_count >/dev/null || {
        echo "document_count not extracted from $BOOTSTRAP" >&2
        return 1
    }

    export TENANT_ID="11111111-1111-1111-1111-111111111111"
    export REGION=GMS MAJOR_VERSION=83 MINOR_VERSION=1
    CURL_ARGS="$BATS_TEST_TMPDIR/curl-args"
}

# Shim curl as a shell function: records its argv, then emits $CURL_BODY.
# CURL_RC drives the failure cases — get_attr pipes curl into jq, so a failed
# request surfaces as empty stdout, never as a non-zero status.
curl() {
    printf '%s\n' "$@" >"$CURL_ARGS"
    [ "${CURL_RC:-0}" -eq 0 ] || return "$CURL_RC"
    printf '%s' "$CURL_BODY"
}

status_body() {
    printf '{"data":{"type":"dataStatus","id":"%s","attributes":{"documentCount":%s,"updatedAt":null}}}' \
        "$TENANT_ID" "$1"
}

# --- document_count -------------------------------------------------------

@test "document_count echoes the count from a well-formed response" {
    CURL_BODY="$(status_body 49049)"
    run document_count "http://ui/api/data/status"
    [ "$status" -eq 0 ]
    [ "$output" = "49049" ]
}

@test "document_count accepts a legitimate zero" {
    # Zero is the isolated-environment case and MUST succeed — it is what
    # allows the restore to run against a genuinely empty database.
    CURL_BODY="$(status_body 0)"
    run document_count "http://ui/api/data/status"
    [ "$status" -eq 0 ]
    [ "$output" = "0" ]
}

@test "document_count fails when the request fails" {
    # curl -f on an HTTP error writes nothing; jq then emits nothing. Read as
    # a count that is "no data", which would trigger a destructive restore.
    CURL_RC=22 CURL_BODY=""
    run document_count "http://ui/api/data/status"
    [ "$status" -eq 1 ]
}

@test "document_count fails on a JSON:API error body" {
    # Wrong shape -> jq -r prints the string "null", which is not a count.
    CURL_BODY='{"errors":[{"status":"403","title":"operator required"}]}'
    run document_count "http://ui/api/data/status"
    [ "$status" -eq 1 ]
    [ "$output" != "null" ]
}

@test "document_count fails on a non-numeric count" {
    CURL_BODY='{"data":{"attributes":{"documentCount":"lots"}}}'
    run document_count "http://ui/api/data/status"
    [ "$status" -eq 1 ]
}

# --- get_attr pass-through ------------------------------------------------

@test "get_attr forwards extra curl arguments and keeps the URL last" {
    CURL_BODY="$(status_body 7)"
    run get_attr "http://ui/api/data/status?scope=shared" documentCount -H "X-Atlas-Operator: 1"
    [ "$status" -eq 0 ]
    [ "$output" = "7" ]
    grep -qx 'X-Atlas-Operator: 1' "$CURL_ARGS"
    [ "$(tail -1 "$CURL_ARGS")" = "http://ui/api/data/status?scope=shared" ]
}

@test "get_attr still sends the tenant headers" {
    CURL_BODY="$(status_body 1)"
    run get_attr "http://ui/api/data/status" documentCount
    [ "$status" -eq 0 ]
    grep -qx "TENANT_ID: $TENANT_ID" "$CURL_ARGS"
    grep -qx "REGION: GMS" "$CURL_ARGS"
}

# --- the guard itself -----------------------------------------------------
#
# The guard is top-level script code, so these are structural assertions —
# the behavioural half is covered by the document_count cases above.

@test "restore runs only when BOTH the tenant and canonical counts are zero" {
    # A tenant with zero rows still READS the full dataset: atlas-data's
    # document/storage.go falls back to canonical.TenantId when the caller's
    # tenant owns none. Gating on the tenant count alone is what fired a
    # restore into the shared baseline database and collided on documents_pkey.
    local guard
    guard=$(grep -n 'if \[ "\$docs" = "0" \] && \[ "\$canon" = "0" \]; then' "$BOOTSTRAP")
    [ -n "$guard" ]
}

@test "the canonical count is read at scope=shared with the operator header" {
    # resolveStatusTenantId (atlas-data status.go) 403s scope=shared without
    # X-Atlas-Operator: 1, and document_count turns that 403 into a hard fail.
    local line
    line=$(grep -n 'canon=\$(document_count' "$BOOTSTRAP")
    [ -n "$line" ]
    [[ "$line" == *"scope=shared"* ]]
    [[ "$line" == *"X-Atlas-Operator: 1"* ]]
}

@test "an unreadable count aborts instead of falling through to the restore" {
    local body
    body=$(sed -n '/^docs=\$(document_count/,/^fi$/p' "$BOOTSTRAP")
    [ -n "$body" ]
    [[ "$body" == *"could not read the tenant document count"* ]]
    [[ "$body" == *"could not read the canonical document count"* ]]
    [ "$(printf '%s\n' "$body" | grep -c 'exit 1')" -eq 2 ]
}
