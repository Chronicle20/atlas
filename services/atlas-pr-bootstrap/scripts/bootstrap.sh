#!/usr/bin/env bash
# Atlas PR-env bootstrap (task-098: baseline-only). Idempotent —
# short-circuits each step that is already complete. Data provisioning is
# baseline-restore ONLY: a read-only preflight hard-fails before any
# data-affecting work when no published canonical baseline exists for the
# env's version (cold-start a new version via the canonical-version-migration
# runbook). Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   MINIO_ENDPOINT     — http://minio.minio.svc.cluster.local:9000
#                        (for the baseline-presence HEAD probe)
#   TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION — required for tenant headers

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

# lib.sh resets options to `set -uo pipefail` (the shared sourcers need
# try-all semantics). bootstrap.sh wants strict-fail; restore -e here.
set -e

# shellcheck source=version-ports.sh
. "$(dirname "$0")/version-ports.sh"
# shellcheck source=service-config.sh
. "$(dirname "$0")/service-config.sh"
# shellcheck source=env-record.sh
. "$(dirname "$0")/env-record.sh"

require_env ATLAS_ENV ATLAS_UI_BASE TENANT_ID REGION MAJOR_VERSION MINOR_VERSION
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
# Canonical tenant descriptor baked into the image. Single source of truth
# for the (region, major, minor) the baseline preflight probes and the
# tenant-create step uses. Overridable so bats can point at a fixture.
CANONICAL_TENANT_JSON="${CANONICAL_TENANT_JSON:-/atlas/canonical/tenant.json}"
# Baseline-probe retry budget. Only transient (000) connection failures are
# retried; a 404 is decisive. Test seam: bats sets these to 1/0.
MINIO_PROBE_RETRIES="${MINIO_PROBE_RETRIES:-5}"
MINIO_PROBE_SLEEP="${MINIO_PROBE_SLEEP:-5}"

# ENV_HEADER is the control-plane environment tag every scoped request
# carries. It is an ARRAY, not a string: a string would word-split on the
# space in "ENVIRONMENT: pr-1411" and curl would see two broken arguments.
#
# Gating on ATLAS_MODE rather than on "does an environment record exist"
# (cleanup.sh:135-137's live check) is deliberate. Teardown must not trust a
# build-time flag because it is *reacting* to whatever got deployed;
# bootstrap is *establishing* the state and must send the header on its very
# first call, before any record could answer the question. ATLAS_MODE is
# also the signal the neighbouring upsert_sparse_service_config already keys on.
#
# In isolated mode the array is empty, so every curl argv below is
# byte-identical to what it was before this existed (FR-2.5) — one code
# path, not two.
env_header_init() {
    ENV_HEADER=()
    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        require_env ATLAS_ENVIRONMENT
        ENV_HEADER=(-H "ENVIRONMENT: $ATLAS_ENVIRONMENT")
    fi
}
env_header_init

# find_environment_tenant <region> <major> <minor> — echoes THIS
# environment's tenant id for that version triple, or nothing.
#
# The ENVIRONMENT header is the entire fix for the adopt-main's-tenant
# defect. atlas-tenants' getAll already applies scope.Strict to the caller's
# environment (tenant/provider.go:27-32); without the header the caller is
# the legacy "" environment and sees the unfiltered union, in which main's
# canonical tenant ALWAYS matches the canonical version triple — because a
# sparse tenant deliberately shares that triple and is distinguished only by
# its environment and its generated UUID.
find_environment_tenant() {
    curl -fsS -H 'Accept: application/vnd.api+json' \
        "${ENV_HEADER[@]}" \
        "$ATLAS_UI_BASE/api/tenants" \
        | jq -r --arg r "$1" --arg M "$2" --arg m "$3" \
            '.data[] | select(.attributes.region == $r and (.attributes.majorVersion|tostring) == $M and (.attributes.minorVersion|tostring) == $m) | .id' \
        | head -1
}

# create_environment_tenant — POSTs the canonical tenant payload and echoes
# the assigned id. Entity.Environment is server-owned from request context
# (tenant/entity.go:16), so the ENVIRONMENT header is the ONLY way to stamp
# the new row with this environment; the request body cannot carry it.
create_environment_tenant() {
    local created id
    created=$(curl -fsS -X POST \
        -H 'Accept: application/vnd.api+json' \
        -H 'Content-Type: application/vnd.api+json' \
        "${ENV_HEADER[@]}" \
        -d @"$CANONICAL_TENANT_JSON" \
        "$ATLAS_UI_BASE/api/tenants") || { log error "tenant POST failed"; return 1; }
    id=$(printf '%s' "$created" | jq -r '.data.id // empty')
    if [ -z "$id" ] || [ "$id" = "null" ]; then
        log error "tenant POST returned no id"
        return 1
    fi
    printf '%s' "$id"
}

# record_environment_tenant <tenant_id> — writes the tenant id onto this
# environment's control-plane record (FR-3), which is the ONLY thing
# cleanup.sh's sweep-tenant phase reads to know what to reclaim
# (cleanup.sh:352-358). Without it a sparse environment's gameplay rows
# survive teardown forever, silently.
#
# The PATCH must carry the record's CURRENT phase, not a chosen one:
# bootstrap runs while the environment is PROVISIONING and must not promote
# it, and a body with no phase is a 400 (UpdateByName validates phase before
# it backfills anything). A same-phase transition is explicitly legal, which
# is also what makes re-running this idempotent (FR-3.4).
record_environment_tenant() {
    local tenant="$1" body phase baseline namespace overrides
    body=$(env_record_get) || body=""
    phase=$(printf '%s' "${body:-}" | jq -r '.data.attributes.phase // empty' 2>/dev/null)
    if [ -z "$phase" ]; then
        log error "no control-plane environment record for ${ATLAS_ENVIRONMENT:-<unset>}; cannot record tenant=$tenant"
        return 1
    fi
    baseline=$(printf '%s' "$body" | jq -r '.data.attributes.baseline // ""' 2>/dev/null)
    namespace=$(printf '%s' "$body" | jq -r '.data.attributes.namespace // ""' 2>/dev/null)
    overrides=$(printf '%s' "$body" | jq -c '.data.attributes.overrides // {}' 2>/dev/null)
    env_record_patch "$phase" "$baseline" "$namespace" "$tenant" "$overrides"
}

# activation_decision <phase> — pure decision for whether the tail-end
# activation step (below) should activate, skip, or fail, echoing one of
# activate / skip / fail. Split out from the block itself so it is testable
# without a cluster (bootstrap_test.bats).
#
# PROVISIONING is the only phase activation is meant to advance FROM.
# ACTIVE is already the target phase, so re-running is a no-op (skip), not
# an error — that is what makes a re-sync of an already-ACTIVE environment
# idempotent. Every other phase (DEACTIVATING, DELETED, empty/no record) is
# a hard fail: activating an environment that is tearing down, gone, or has
# no control-plane record at all would resurrect or fabricate a phase it
# never legitimately reached.
activation_decision() {
    case "$1" in
        PROVISIONING) printf 'activate' ;;
        ACTIVE)       printf 'skip' ;;
        *)            printf 'fail' ;;
    esac
}

# Sanity-check TENANT_ID shape. The libs/atlas-rest middleware that
# tenant-aware endpoints route through (ParseTenant) requires the
# header to be UUID-parseable; a non-UUID value would return 400 from
# every wait-ready probe and the retry loop would exhaust before the
# operator could diagnose. The TENANT_ID supplied here is the *initial*
# value (the canonical tenant lookup may overwrite it later); the only
# requirement is that it parses as a UUID.
if ! printf '%s' "$TENANT_ID" | grep -Eq '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'; then
    log error "TENANT_ID '$TENANT_ID' is not UUID-shaped; tenant-aware probes will 400. Fix Phase 7's Helm chart to inject a UUID."
    exit 1
fi

post() {
    curl -fsS -X POST \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "Content-Type: application/json" \
        "$@" -d '{}'
}

# get_attr URL ATTR [extra curl args...]
#
# Note the pipeline: the exit status is jq's, not curl's, so a failed request
# is NOT reported here — it surfaces as empty output. Callers that act on the
# value must validate it (see document_count).
get_attr() {
    local url="$1" attr="$2"
    shift 2
    curl -fsS \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "Accept: application/vnd.api+json" \
        "$@" \
        "$url" | jq -r ".data.attributes.$attr"
}

# document_count URL [extra curl args...] — echoes /api/data/status's
# documentCount for one scope, or returns non-zero if the value did not come
# back as a plain integer.
#
# The validation is the point. get_attr cannot report a failed request, so an
# unreachable atlas-data yields "" and a JSON:API error body yields "null".
# Either one, treated as a count, reads as "no data here" — which is what
# decides whether to run a destructive restore. Fail loudly instead.
document_count() {
    local url="$1" n
    shift
    n=$(get_attr "$url" documentCount "$@")
    case "$n" in
        '' | *[!0-9]*) return 1 ;;
    esac
    printf '%s' "$n"
}

# Polling helper — returns 0 when /api/data/status has *stopped*
# reporting fresh writes, 1 otherwise. Designed for use with retry().
#
# Earlier version returned success as soon as the counter went non-zero
# — i.e. on the *first* written document. That race let the next
# bootstrap step start while processing was still streaming. atlas-data
# workers (MAP, MONSTER, the CHARACTER / EQUIPMENT worker) open WZ XML
# files in their `Init*` calls and bail with `return err` on ENOENT, so
# any worker whose XML had not yet been extracted wrote ZERO documents.
# On 2026-05-16 the cold-start of PR #461's env reproduced this exactly:
# atlas-data started MAP at 12:09:37.209, hit ENOENT on Map.img.xml at
# 12:09:37.242, and the extractor wrote that file 168 ms later at
# 12:09:37.410. Net loss: 5,261 MAP + 1,568 MONSTER + 4,334 EQUIPMENT
# = 11,163 documents (~23 % deficit vs. atlas-main on the same tenant).
#
# Fix: detect actual *completion*, not first progress. /api/data/status
# exposes `updatedAt` = MAX(updated_at) across underlying rows — it
# advances on every write and stops advancing when writes stop. Require
# the counter to be non-zero AND `updatedAt` to be unchanged for
# STABLE_REQUIRED consecutive polls before declaring done. With the
# `retry 60 5 …` call shape, STABLE_REQUIRED=3 gives a ≥ 10 s no-write
# window (the first match arms the counter, the next two confirm). That
# comfortably covers the worst inter-write gap observed in practice
# (sub-second between Map.wz IMGs, ~2 s between UI.wz IMGs) while still
# bounding overshoot at one stability window.
#
# State lives in globals — retry() invokes the helper in the current
# shell (not a subshell), so updates accumulate across calls.
#
# This stability check is used after a baseline restore to wait for
# atlas-data to finish writing the restored documents.

DATA_PROCESSING_LAST_UPDATED=""
DATA_PROCESSING_STABLE_COUNT=0
STABLE_REQUIRED=3

data_processing_done() {
    local count updated
    count=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
    updated=$(get_attr "$ATLAS_UI_BASE/api/data/status" updatedAt)
    if [ -z "$count" ] || [ "$count" = "0" ] || [ "$count" = "null" ]; then
        return 1
    fi
    if [ -z "$updated" ] || [ "$updated" = "null" ]; then
        return 1
    fi
    if [ "$updated" = "$DATA_PROCESSING_LAST_UPDATED" ]; then
        DATA_PROCESSING_STABLE_COUNT=$((DATA_PROCESSING_STABLE_COUNT + 1))
    else
        DATA_PROCESSING_LAST_UPDATED="$updated"
        DATA_PROCESSING_STABLE_COUNT=1
    fi
    [ "$DATA_PROCESSING_STABLE_COUNT" -ge "$STABLE_REQUIRED" ]
}

# baseline_object_status <url> → echo the HTTP status of an anonymous HEAD
# (e.g. 200/404); echo 000 on a connection-level failure. Anonymous read is
# enabled on the atlas-canonical bucket, so no credentials are needed.
baseline_object_status() {
    local url="$1" code
    code=$(curl -sS -o /dev/null -w '%{http_code}' -I "$url" 2>/dev/null) || code=000
    printf '%s' "$code"
}

# baseline_reachable <url> — retry()-friendly predicate. Sets the global
# BASELINE_PROBE_CODE to the HTTP status and returns 0 whenever MinIO
# answered at all (even 404); returns 1 ONLY on a 000 connection failure so
# retry() rides out a cold-start MinIO blip without masking a real 404.
BASELINE_PROBE_CODE=""
baseline_reachable() {
    BASELINE_PROBE_CODE=$(baseline_object_status "$1")
    [ "$BASELINE_PROBE_CODE" != "000" ]
}

# probe_baseline_object <url> — drive baseline_reachable through retry(). On
# success leaves the HTTP code in BASELINE_PROBE_CODE. If MinIO stays
# unreachable through the retry budget, log a DISTINCT "unreachable" error
# and exit non-zero (do NOT tell the operator to publish a baseline that may
# already exist). Called directly (never in $()) so its exit halts the script.
probe_baseline_object() {
    local url="$1"
    if ! retry "$MINIO_PROBE_RETRIES" "$MINIO_PROBE_SLEEP" baseline_reachable "$url"; then
        log error "MinIO unreachable at $MINIO_ENDPOINT ($url) — cannot verify canonical baseline; check MinIO, do not assume the baseline is missing"
        exit 1
    fi
}

# preflight_baseline — hard-gate the bootstrap on a published canonical
# baseline BEFORE any data-affecting work. Reads (region, major, minor) from
# CANONICAL_TENANT_JSON so the probe targets exactly the version the later
# restore requests (not the initial env-injected values). HEADs BOTH the
# documents.dump.sha256 sidecar AND the documents.dump object, so a
# half-published baseline fails here rather than breaking the restore later.
preflight_baseline() {
    local region major minor base sha_code dump_code
    region=$(jq -r '.data.attributes.region' "$CANONICAL_TENANT_JSON")
    major=$(jq -r '.data.attributes.majorVersion' "$CANONICAL_TENANT_JSON")
    minor=$(jq -r '.data.attributes.minorVersion' "$CANONICAL_TENANT_JSON")
    base="$MINIO_ENDPOINT/atlas-canonical/baseline/regions/$region/versions/$major.$minor"

    probe_baseline_object "$base/documents.dump.sha256"
    sha_code="$BASELINE_PROBE_CODE"
    probe_baseline_object "$base/documents.dump"
    dump_code="$BASELINE_PROBE_CODE"

    if [ "$sha_code" = "200" ] && [ "$dump_code" = "200" ]; then
        log info "canonical baseline present for $region $major.$minor"
        return 0
    fi
    log error "no canonical baseline for $region $major.$minor (documents.dump.sha256=$sha_code documents.dump=$dump_code) — publish one (see docs/runbooks/canonical-version-migration.md) before deploying this env"
    exit 1
}

# Fail fast, before any data-affecting work (tenant create, config clone,
# restarts, restore), when no canonical baseline exists for this version.
# A read-only MinIO probe with no dependency on atlas-data being up.
ATLAS_STEP=preflight-baseline preflight_baseline

# wait-ready: poll the ingress-fronted endpoints we'll actually call
# during bootstrap. atlas-renders is included as a rollout-status check
# because its /healthz isn't surfaced through atlas-ingress and its
# render routes require a fully-set-up tenant + asset path to probe
# meaningfully.
ATLAS_STEP=wait-ready log info "waiting for atlas-tenants, atlas-configurations, atlas-data, atlas-renders"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/tenants"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/configurations/services"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/status"
kubectl rollout status deployment/atlas-renders --timeout=180s 2>/dev/null \
    || log warn "atlas-renders rollout status check failed; continuing"

# Tenant: POST canonical payload, capture the assigned id, override
# downstream TENANT_ID for all subsequent calls. The atlas-tenants pitfall
# (duplicate rows on retry-after-Kafka-failure) is mitigated by checking
# whether a tenant with the canonical region+major+minor already exists.
ATLAS_STEP=tenant-create
canonical_region=$(jq -r '.data.attributes.region' "$CANONICAL_TENANT_JSON")
canonical_major=$(jq -r '.data.attributes.majorVersion' "$CANONICAL_TENANT_JSON")
canonical_minor=$(jq -r '.data.attributes.minorVersion' "$CANONICAL_TENANT_JSON")

existing=$(find_environment_tenant "$canonical_region" "$canonical_major" "$canonical_minor")

if [ -n "$existing" ] && [ "$existing" != "null" ]; then
    log info "tenant already present for environment=${ATLAS_ENVIRONMENT:-<legacy>}: $existing"
    TENANT_ID="$existing"
else
    log info "creating tenant for environment=${ATLAS_ENVIRONMENT:-<legacy>} ($canonical_region v$canonical_major.$canonical_minor)"
    TENANT_ID=$(create_environment_tenant) || exit 1

    # Wait for the tenant.status Kafka event to settle. atlas-tenants writes
    # the DB row before the emit; if Kafka is unreachable the emit fails and
    # the next caller would see a tenant via REST with no event published.
    # This mirrors the onboarding doc's pitfall #1.
    sleep 10
fi

REGION="$canonical_region"
MAJOR_VERSION="$canonical_major"
MINOR_VERSION="$canonical_minor"
log info "using TENANT_ID=$TENANT_ID for downstream calls"

# Record the tenant on the control-plane environment record BEFORE any
# tenant-keyed write happens. The reverse order leaks on every partial
# failure: a bootstrap that dies after the config clone but before the PATCH
# leaves rows under a tenant teardown has no way to name.
#
# Sparse-only, and gated on the same ATLAS_MODE fact env_header_init uses:
# an isolated environment registers no control-plane record at all
# (cleanup.sh:120-133), so there is nothing to PATCH and do_drop_dbs reclaims
# its rows by dropping the databases outright.
#
# `|| exit 1` is belt-and-braces for `set -e` (restored at line 22 after
# lib.sh relaxes it), stated at the call site for the same reason the
# upsert_sparse_service_config loop below does. An environment that provisions a
# tenant but never records it produces exactly the residue this exists to
# prevent (FR-3.5).
if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
    ATLAS_STEP=record-tenant record_environment_tenant "$TENANT_ID" || exit 1
    ATLAS_STEP=record-tenant log info "recorded tenant=$TENANT_ID on environment record $ATLAS_ENVIRONMENT"
fi

# Tenant configuration: clone the canonical template's attributes into a
# per-tenant row in atlas-configurations. Rest-equivalent of the UI's
# Templates → Clone flow (see services/atlas-ui/.../onboarding.service.ts
# and docs/onboarding.md). Without this, /api/configurations/tenants/{id}
# returns null and atlas-channel / atlas-world / atlas-character-factory
# log.Fatalf("tenant not configured") on startup.
#
# The template is a cluster-side bootstrap concern: every Atlas env is
# expected to have at least one v83.1 template seeded into
# atlas-configurations before any per-PR sync runs. If the GET below
# returns nothing, the cluster operator needs to seed a template (see
# docs/onboarding.md Step 1).
ATLAS_STEP=tenant-config

existing_code=$(curl -fsS -o /dev/null -w '%{http_code}' \
    -H 'Accept: application/vnd.api+json' \
    "$ATLAS_UI_BASE/api/configurations/tenants/$TENANT_ID" 2>/dev/null || true)
if [ "$existing_code" = "200" ]; then
    log info "tenant configuration $TENANT_ID already present; skipping"
else
    template=$(curl -fsS \
        -H 'Accept: application/vnd.api+json' \
        "$ATLAS_UI_BASE/api/configurations/templates?region=$REGION&majorVersion=$MAJOR_VERSION&minorVersion=$MINOR_VERSION")
    template_id=$(echo "$template" | jq -r '.data.id // empty')
    if [ -z "$template_id" ]; then
        log error "no template found for region=$REGION majorVersion=$MAJOR_VERSION minorVersion=$MINOR_VERSION"
        log error "cluster setup issue — atlas-configurations must have a v${MAJOR_VERSION}.${MINOR_VERSION} template seeded; see docs/onboarding.md Step 1"
        exit 1
    fi
    log info "cloning template $template_id into tenant configuration $TENANT_ID"

    # Pipe via stdin (-d @-) because the template attributes are ~76KB and
    # passing them as a curl argv arg exceeds the kernel argv size limit
    # ("Argument list too long").
    echo "$template" | jq --arg tid "$TENANT_ID" \
        '{data: {id: $tid, type: "tenants", attributes: .data.attributes}}' \
        | curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            --data-binary @- \
            "$ATLAS_UI_BASE/api/configurations/tenants" >/dev/null
    log info "tenant configuration $TENANT_ID created"
fi

# Discover the per-PR LB IP before writing service config, so the
# channel-service tenants[].ipAddress is correct on the first write and
# the subsequent rolling restart picks up the right host in one shot.
ATLAS_STEP=lb-discover
LB_IP=$(kubectl get svc atlas-channel-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -z "$LB_IP" ]; then
    log error "atlas-channel-lb has no allocated LoadBalancer IP — MetalLB pool exhausted?"
    exit 1
fi
log info "LB IP for channel service: $LB_IP"

# Service configs: atlas-configurations API is keyed by service UUID. Phase 0
# Task 0.7 captured three canonical payloads (login/channel/drops), one per
# pinned SERVICE_ID. We POST/PATCH each individually against
# /api/configurations/services/{serviceId}. See upsert_service_config below.
ATLAS_STEP=service-config

# Read the live services config, upsert this PR's canonical tenant entry
# (keyed by id), and write back the merged result. Preserves every other
# tenants[] entry so co-resident versions are never drained (task-084 FR-2).
# The tenant entry is built fresh from version-derived ports (build_login_entry
# / build_channel_entry), so we no longer string-substitute the canonical
# payload's tenants[].
#   $1 = canonical template path
#   $2 = shape: login | channel | none (tenant-agnostic, e.g. drops)
upsert_service_config() {
    local payload_path="$1" shape="$2" svc_id entry
    svc_id=$(jq -r '.data.id' "$payload_path")
    if [ -z "$svc_id" ] || [ "$svc_id" = "null" ]; then
        log error "missing data.id in $payload_path"
        return 1
    fi

    case "$shape" in
        login)   entry=$(build_login_entry) ;;
        channel) entry=$(build_channel_entry "$payload_path") ;;
        none)    entry="" ;;
        *)       log error "upsert_service_config: unknown shape '$shape'"; return 1 ;;
    esac

    local existing
    existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \
        "$ATLAS_UI_BASE/api/configurations/services/$svc_id" 2>/dev/null || true)

    if echo "$existing" | jq -e '.data.id' >/dev/null 2>&1; then
        # Merge this PR's tenant entry into the LIVE attributes (id-keyed),
        # preserving every foreign tenants[] entry. Tenant-agnostic configs
        # (shape=none) pass the live attributes through unchanged.
        local live_attrs new_attrs
        live_attrs=$(echo "$existing" | jq -c '.data.attributes')
        if [ -n "$entry" ]; then
            new_attrs=$(printf '%s' "$live_attrs" | merge_tenant_entry "$entry")
        else
            new_attrs="$live_attrs"
        fi

        # Skip the PATCH if the merged attributes already match what's live.
        # Idempotency, and it dodges atlas-configurations' PATCH handler panic
        # ("reflect: reflect.Value.Set using unaddressable value") on
        # tenant-agnostic configs (drops-service) — a no-op PATCH there would
        # crash the handler.
        if [ "$(printf '%s' "$live_attrs" | jq -cS .)" = "$(printf '%s' "$new_attrs" | jq -cS .)" ]; then
            log info "service config $svc_id matches; skipping PATCH"
        else
            log info "service config $svc_id exists; PATCH (merged)"
            local body
            body=$(echo "$existing" | jq -c --argjson a "$new_attrs" '.data.attributes = $a')
            curl -fsS -X PATCH \
                -H 'Accept: application/vnd.api+json' \
                -H 'Content-Type: application/vnd.api+json' \
                -d "$body" \
                "$ATLAS_UI_BASE/api/configurations/services/$svc_id" >/dev/null
        fi
    else
        # First write: seed tenants[] with just this PR's entry (or post the
        # canonical payload verbatim for tenant-agnostic configs).
        log info "service config $svc_id absent; POST"
        local body
        body=$(build_service_config "$shape" "$payload_path")
        curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            -d "$body" \
            "$ATLAS_UI_BASE/api/configurations/services" >/dev/null
    fi
}

# Sparse mode (ATLAS_MODE=sparse): upsert a services row scoped to this
# environment instead of merging into the canonical (main-owned) row. main's
# row is never read, written or merged into (G7/NG6) — the whole point of
# this function existing separately from upsert_service_config.
#
# svc_table_lookup <deployment> — echoes "type shape template" for a
# SERVICE_ID-carrying override deployment, or returns non-zero if unmapped.
# Only atlas-login/atlas-channel/atlas-drops are listed: they are the only
# override deployments with a canonical template baked into this image
# today (see canonical/services/ — no world-service.json,
# character-factory.json or drops-information-service.json exists yet). An
# override deployment with no template is deliberately left unmapped rather
# than fabricated — upsert_sparse_service_config's caller treats "unmapped"
# and "no template" identically (log info, skip); add a case here the day
# the template lands.
svc_table_lookup() {
    local type shape
    case "$1" in
        atlas-login)   type=login-service shape=login ;;
        atlas-channel) type=channel-service shape=channel ;;
        atlas-drops)   type=drops-service shape=none ;;
        *) return 1 ;;
    esac
    printf '%s %s /atlas/canonical/services/%s.json' "$type" "$shape" "$type"
}

# svc_id_var_name <service-type> — the SERVICE_ID_<TYPE> env var name the CI
# rendering step (tools/derive-service-id.sh) is expected to have exported
# before this Job runs. Uppercase, `-` → `_`.
svc_id_var_name() {
    printf 'SERVICE_ID_%s' "$(printf '%s' "$1" | tr '[:lower:]-' '[:upper:]_')"
}

# upsert_sparse_service_config <deployment> — GET/compare/PATCH-or-POST a
# services row keyed on the id the CI rendering derived via
# tools/derive-service-id.sh and exposed as SERVICE_ID_<TYPE>. That id is
# NEVER re-derived here — a second derivation site is exactly the D2 defect
# this design forecloses (the sparse row's id and the Deployment's
# SERVICE_ID env var would drift the moment either derivation changed).
# Unlike the removed create_service_config, this only creates a row for a
# service actually deployed in this environment (the override set) and
# never mutates main's pinned row (G7/NG6).
upsert_sparse_service_config() {
    local deployment="$1" resolved type shape tmpl svc_id_var svc_id existing
    resolved=$(svc_table_lookup "$deployment") || {
        log info "no service-config mapping for $deployment; skipping"
        return 0
    }
    read -r type shape tmpl <<<"$resolved"

    svc_id_var=$(svc_id_var_name "$type")
    svc_id="${!svc_id_var:-}"
    if [ -z "$svc_id" ]; then
        log error "no $svc_id_var in environment; the CI rendering did not run"
        return 1
    fi

    existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \
        "$ATLAS_UI_BASE/api/configurations/services/$svc_id" 2>/dev/null || true)

    if echo "$existing" | jq -e '.data.id' >/dev/null 2>&1; then
        # Merge this environment's tenant entry into the LIVE attributes
        # (id-keyed), preserving every foreign tenants[] entry. Tenant-agnostic
        # configs (shape=none) pass the live attributes through unchanged. Same
        # jq -cS comparison as upsert_service_config above — it dodges the
        # PATCH-handler panic on tenant-agnostic configs too.
        local entry live_attrs new_attrs
        case "$shape" in
            login)   entry=$(build_login_entry) ;;
            channel) entry=$(build_channel_entry "$tmpl") ;;
            none)    entry="" ;;
        esac
        live_attrs=$(echo "$existing" | jq -c '.data.attributes')
        if [ -n "$entry" ]; then
            new_attrs=$(printf '%s' "$live_attrs" | merge_tenant_entry "$entry")
        else
            new_attrs="$live_attrs"
        fi

        if [ "$(printf '%s' "$live_attrs" | jq -cS .)" = "$(printf '%s' "$new_attrs" | jq -cS .)" ]; then
            log info "sparse service config $svc_id matches; skipping PATCH"
        else
            log info "sparse service config $svc_id exists; PATCH (merged)"
            local body
            body=$(echo "$existing" | jq -c --argjson a "$new_attrs" '.data.attributes = $a')
            curl -fsS -X PATCH \
                -H 'Accept: application/vnd.api+json' \
                -H 'Content-Type: application/vnd.api+json' \
                -d "$body" \
                "$ATLAS_UI_BASE/api/configurations/services/$svc_id" >/dev/null \
                || { log error "upsert_sparse_service_config: PATCH failed (type=$type, id=$svc_id)"; return 1; }
        fi
    else
        # First write: build_service_config seeds tenants[] with just this
        # environment's entry (or posts the canonical payload verbatim for
        # tenant-agnostic configs). The ENVIRONMENT header is what stamps the
        # row's environment column — it is server-owned and the request
        # body's .data.attributes.environment is deliberately ignored
        # (configurations/services/administrator.go's INSERT takes
        # env.MustFromContext(ctx); processor.go notes "Environment is
        # server-owned ... the Entity column always wins"). Omitting it is
        # what put every sparse row in the legacy '' environment, invisible
        # to cleanup.sh's environment-scoped reclaim.
        local body
        body=$(build_service_config "$shape" "$tmpl" "$svc_id") || return 1
        log info "sparse service config $svc_id absent; POST (type=$type, environment=$ATLAS_ENVIRONMENT)"
        curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            "${ENV_HEADER[@]}" \
            -d "$body" \
            "$ATLAS_UI_BASE/api/configurations/services" >/dev/null \
            || { log error "upsert_sparse_service_config: POST failed (type=$type, id=$svc_id)"; return 1; }
    fi
}

if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
    # Sparse mode: fresh Id-keyed rows, never main's pinned rows (G7/NG6),
    # and only for a service actually deployed here — reading the override
    # set from the environment record this bootstrap run already fetched
    # (record_environment_tenant, above) is what removes the orphan rows a
    # fixed three-call list would otherwise create in every namespace that
    # does not override all three, and is why there is no longer a `kubectl
    # set env` here: the id now arrives pre-rendered in the manifest CI
    # produced, so Argo's selfHeal has nothing of ours to revert (FR-1.2).
    #
    # What actually makes a bad row fatal is upsert_sparse_service_config's
    # own guards returning non-zero — line 21 restores `set -e` after
    # lib.sh relaxes it, so a bare call already aborts. The `|| exit 1` is
    # belt-and-braces for that dependency: it keeps this loop fatal if
    # `set -e` is ever relaxed again (lib.sh has done exactly that once
    # already), and states the intent at the call site. Failing the PostSync
    # hook is the point — better a visible hook failure in Argo than
    # restarting Deployments whose SERVICE_ID is wrong or absent.
    overrides=$(env_record_get | jq -r '.data.attributes.overrides // {} | keys[]')
    for d in $overrides; do upsert_sparse_service_config "$d" || exit 1; done
else
    # login-service: version-derived login port, id-keyed merge.
    upsert_service_config /atlas/canonical/services/login-service.json login

    # channel-service: version-derived channel port + LB_IP, id-keyed merge.
    upsert_service_config /atlas/canonical/services/channel-service.json channel

    # drops-service: tenant-agnostic (no tenants array) — pass through unchanged.
    upsert_service_config /atlas/canonical/services/drops-service.json none
fi

# Rolling restart for services that still read SERVICE_ID synchronously at
# startup. atlas-login and atlas-channel were removed by task-032 — they
# subscribe to the configuration projection topics and apply service /
# tenant updates live without a restart. Keeping them in this list would
# defeat the whole point of the dynamic-config feature.
ATLAS_STEP=service-restart
restart_targets="atlas-drops atlas-character-factory atlas-world"
for d in $restart_targets; do
    kubectl rollout restart deployment/"$d" 2>/dev/null || log warn "could not restart $d"
done
for d in $restart_targets; do
    kubectl rollout status deployment/"$d" --timeout=180s 2>/dev/null || log warn "$d not ready"
done

# Data ingest: baseline-restore only. The preflight already proved the
# baseline exists for this version, so there is no "what if absent" branch.
#
# The guard mirrors atlas-data's READ semantics instead of asking only "does
# this tenant own rows?". document/storage.go falls back to the version-scoped
# canonical tenant (canonical.TenantId) when the caller's tenant has none, so a
# tenant provisioned after canonical ingestion reads the full dataset while
# owning zero rows. That is the normal steady state, not an empty environment.
#
# Restoring in that state is not merely redundant, it is destructive: the dump
# is replayed through baseline.Rewriter, which rewrites ONLY the tenant_id
# column and copies every other column — primary keys included — verbatim. A
# COPY into a database that already holds the canonical rows therefore fails on
# documents_pkey and rolls back to a wiped target tenant. Isolated environments
# never hit this because they get an empty database; the first sparse
# environment shares the baseline's atlas-data, which already held all ~49k
# canonical rows, and its bootstrap died here.
#
# Both counts zero means the data is genuinely unreachable — a fresh isolated
# database — and the restore is what populates it.
ATLAS_STEP=data-ingest
docs=$(document_count "$ATLAS_UI_BASE/api/data/status") || {
    log error "data-ingest: could not read the tenant document count"
    exit 1
}
# scope=shared reports the canonical tenant's count and is operator-gated
# (status.go's resolveStatusTenantId 403s without this header).
canon=$(document_count "$ATLAS_UI_BASE/api/data/status?scope=shared" -H "X-Atlas-Operator: 1") || {
    log error "data-ingest: could not read the canonical document count"
    exit 1
}
if [ "$docs" = "0" ] && [ "$canon" = "0" ]; then
    log info "restoring canonical baseline → POST /api/data/baseline/restore"
    restore_body=$(jq -cn \
        --arg r "$REGION" \
        --arg M "$MAJOR_VERSION" \
        --arg m "$MINOR_VERSION" \
        --arg t "$TENANT_ID" \
        '{data:{type:"baselineRestores",attributes:{region:$r,majorVersion:($M|tonumber),minorVersion:($m|tonumber),tenantId:$t}}}')
    curl -fsS -X POST \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "X-Atlas-Operator: 1" \
        -H "Content-Type: application/vnd.api+json" \
        -d "$restore_body" \
        "$ATLAS_UI_BASE/api/data/baseline/restore" >/dev/null
    retry 60 5 data_processing_done
else
    log info "data already reachable (tenant=$docs, canonical=$canon); skipping ingest"
fi

# Per-domain seeds, in parallel
ATLAS_STEP=seed
log info "seeding domain data"
endpoints=(
    /api/drops/seed
    /api/gachapons/seed
    /api/npcs/conversations/seed
    /api/quests/conversations/seed
    /api/items/conversations/seed
    /api/shops/seed
    /api/portals/scripts/seed
    /api/reactors/actions/seed
    /api/maps/actions/seed
    /api/events/definitions/seed
    /api/party-quests/definitions/seed
)
for ep in "${endpoints[@]}"; do
    ( post "$ATLAS_UI_BASE$ep" >/dev/null && log info "seeded $ep" ) &
done
wait

# --- Activation (task-243 FR-4.1/FR-4.2/D5) --------------------------------
#
# Sparse-only [FR-4.1]: an isolated environment has no shared control-plane
# phase to advance, and no override set to roll out — activation is
# meaningless outside the shared-cluster model this gates.
#
# Observed, not assumed (FR-4.2). Readiness comes from `kubectl rollout
# status`, never from an assumption that an earlier step succeeded.
# Consumer-group initialization comes from the wave-0 precreate Job's own
# exit code, which Argo enforces as a precondition of wave 10 (this Job)
# existing at all — FR-4.3/FR-4.4 are structural, not re-checked here.
#
# Composes with FR-1.5. With Task 7's readiness probe, a pod whose
# service-config row never arrived is never Ready, so `rollout status`
# times out and activation fails — the environment stays PROVISIONING
# rather than advertising a capability it does not have.
#
# Idempotent. env_record_patch/UpdateByName accepts a same-phase transition
# and rejects skips and reverts (activation_decision, above), so a re-sync
# of an already-ACTIVE environment is a no-op and a re-sync during teardown
# cannot resurrect it.
if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
    ATLAS_STEP=activate

    # Same override-set jq read as the sparse service-config loop above:
    # only roll-check deployments this environment actually overrides.
    overrides=$(env_record_get | jq -r '.data.attributes.overrides // {} | keys[]')
    login_rolled=0
    channel_rolled=0
    for d in $overrides; do
        kubectl rollout status deployment/"$d" --timeout="${ACTIVATION_ROLLOUT_TIMEOUT:-300s}" \
            || { log error "activation: $d not ready"; exit 1; }
        case "$d" in
            atlas-login)   login_rolled=1 ;;
            atlas-channel) channel_rolled=1 ;;
        esac
    done

    # FR-4.2 mandatory sockets: activation requires both atlas-login and
    # atlas-channel to be part of the override set (and thus rolled out
    # above) — a sparse environment missing either is not activatable.
    if [ "$login_rolled" -ne 1 ] || [ "$channel_rolled" -ne 1 ]; then
        log error "activation requires atlas-login and atlas-channel in the override set (login=$login_rolled channel=$channel_rolled)"
        exit 1
    fi

    body=$(env_record_get) || body=""
    phase=$(printf '%s' "$body" | jq -r '.data.attributes.phase // empty')
    case "$(activation_decision "$phase")" in
        skip)
            log info "environment $ATLAS_ENVIRONMENT already ACTIVE"
            ;;
        fail)
            log error "environment $ATLAS_ENVIRONMENT is in phase '${phase:-<none>}'; refusing to activate"
            exit 1
            ;;
        activate)
            baseline=$(printf '%s' "$body" | jq -r '.data.attributes.baseline // ""')
            namespace=$(printf '%s' "$body" | jq -r '.data.attributes.namespace // ""')
            tenant=$(printf '%s' "$body" | jq -r '.data.attributes.tenant // ""')
            overrides=$(printf '%s' "$body" | jq -c '.data.attributes.overrides // {}')
            env_record_patch ACTIVE "$baseline" "$namespace" "$tenant" "$overrides" \
                || { log error "activation PATCH failed"; exit 1; }
            log info "environment $ATLAS_ENVIRONMENT is ACTIVE"
            ;;
    esac
fi

ATLAS_STEP=done log info "bootstrap complete"
