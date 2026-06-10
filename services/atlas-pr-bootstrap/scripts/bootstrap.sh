#!/usr/bin/env bash
# Atlas PR-env bootstrap (task-071: MinIO-backed ingest). Idempotent —
# short-circuits each step that is already complete. Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   BOOTSTRAP_MODE     — auto|baseline|full (default auto)
#     baseline — restore from canonical baseline in MinIO (fast: ~60s).
#     full     — upload WZ zip, run ingest (~10min).
#     auto     — probe canonical baseline; fall back to full on absence.
#   WZ_CANONICAL       — path to canonical zip (default /opt/wz/atlas.zip,
#                        only used in full mode)
#   MINIO_ENDPOINT     — http://minio.minio.svc.cluster.local:9000
#                        (for baseline-detect HEAD)
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

require_env ATLAS_ENV ATLAS_UI_BASE TENANT_ID REGION MAJOR_VERSION MINOR_VERSION
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"
BOOTSTRAP_MODE="${BOOTSTRAP_MODE:-auto}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"

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

get_attr() {
    curl -fsS \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "Accept: application/vnd.api+json" \
        "$1" | jq -r ".data.attributes.$2"
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
# existing `retry 240 10 …` call shape, STABLE_REQUIRED=3 gives a
# ≥ 20 s no-write window (the first match arms the counter, the next
# two confirm). That comfortably covers the worst inter-write gap
# observed in practice (sub-second between Map.wz IMGs, ~2 s between
# UI.wz IMGs) while still bounding overshoot at one stability window.
#
# State lives in globals — retry() invokes the helper in the current
# shell (not a subshell), so updates accumulate across calls.
#
# Note: as of task-071 the WZ-extraction step is gone — atlas-data's
# /api/data/process call invokes WZ ingest directly. Only the
# data_processing stability check remains.

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

# Probe whether a canonical baseline exists for (region, major.minor).
# Returns 0 = present, 1 = absent. Reads MinIO directly via anonymous
# HEAD — the bucket is anonymous-read by `atlas-minio-init`, so no
# credentials are required.
canonical_baseline_exists() {
    local sha_url="$MINIO_ENDPOINT/atlas-canonical/baseline/regions/$REGION/versions/$MAJOR_VERSION.$MINOR_VERSION/documents.dump.sha256"
    local code
    code=$(curl -sS -o /dev/null -w '%{http_code}' -I "$sha_url" 2>/dev/null || echo 000)
    [ "$code" = "200" ]
}

# Resolve BOOTSTRAP_MODE=auto → baseline|full by probing MinIO; echo the
# chosen mode (and log a WARN on fallback). For explicit modes, just
# echo the value back after validation.
resolve_mode() {
    case "$BOOTSTRAP_MODE" in
        baseline|full)
            echo "$BOOTSTRAP_MODE"
            ;;
        auto)
            if canonical_baseline_exists; then
                echo baseline
            else
                log warn "no canonical baseline at $MINIO_ENDPOINT/atlas-canonical/baseline/regions/$REGION/versions/$MAJOR_VERSION.$MINOR_VERSION/; falling back to full"
                echo full
            fi
            ;;
        *)
            log error "BOOTSTRAP_MODE='$BOOTSTRAP_MODE' invalid; expected auto|baseline|full"
            exit 1
            ;;
    esac
}

# wait-ready: poll the ingress-fronted endpoints we'll actually call
# during bootstrap. atlas-renders is included as a rollout-status check
# because its /healthz isn't surfaced through atlas-ingress and its
# render routes require a fully-set-up tenant + asset path to probe
# meaningfully.
ATLAS_STEP=wait-ready log info "waiting for atlas-tenants, atlas-configurations, atlas-data, atlas-renders"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/tenants"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/configurations/services"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/status"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/wz"
kubectl rollout status deployment/atlas-renders --timeout=180s 2>/dev/null \
    || log warn "atlas-renders rollout status check failed; continuing"

# Tenant: POST canonical payload, capture the assigned id, override
# downstream TENANT_ID for all subsequent calls. The atlas-tenants pitfall
# (duplicate rows on retry-after-Kafka-failure) is mitigated by checking
# whether a tenant with the canonical region+major+minor already exists.
ATLAS_STEP=tenant-create
canonical_region=$(jq -r '.data.attributes.region' /atlas/canonical/tenant.json)
canonical_major=$(jq -r '.data.attributes.majorVersion' /atlas/canonical/tenant.json)
canonical_minor=$(jq -r '.data.attributes.minorVersion' /atlas/canonical/tenant.json)

existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \
    "$ATLAS_UI_BASE/api/tenants" \
    | jq -r --arg r "$canonical_region" --arg M "$canonical_major" --arg m "$canonical_minor" \
        '.data[] | select(.attributes.region == $r and (.attributes.majorVersion|tostring) == $M and (.attributes.minorVersion|tostring) == $m) | .id' \
    | head -1)

if [ -n "$existing" ] && [ "$existing" != "null" ]; then
    log info "canonical tenant already present: $existing"
    TENANT_ID="$existing"
else
    log info "creating canonical tenant ($canonical_region v$canonical_major.$canonical_minor)"
    created=$(curl -fsS -X POST \
        -H 'Accept: application/vnd.api+json' \
        -H 'Content-Type: application/vnd.api+json' \
        -d @/atlas/canonical/tenant.json \
        "$ATLAS_UI_BASE/api/tenants")
    TENANT_ID=$(echo "$created" | jq -r '.data.id')
    if [ -z "$TENANT_ID" ] || [ "$TENANT_ID" = "null" ]; then
        log error "tenant POST returned no id"
        exit 1
    fi

    # Wait for tenant.status Kafka event to settle. Atlas-tenants writes
    # the DB row before the emit; if Kafka is unreachable, the emit fails
    # and the next caller would see a tenant via REST but no event was
    # published. We poll the GET endpoint until the tenant is present —
    # which it already is post-POST — and additionally wait a short window
    # for downstream services to reconcile via the Kafka event. This mirrors
    # the onboarding doc pitfall #1.
    sleep 10
fi

REGION="$canonical_region"
MAJOR_VERSION="$canonical_major"
MINOR_VERSION="$canonical_minor"
log info "using TENANT_ID=$TENANT_ID for downstream calls"

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
        if [ -n "$entry" ]; then
            body=$(jq -c --argjson entry "$entry" '.data.attributes.tenants = [$entry]' "$payload_path")
        else
            body=$(cat "$payload_path")
        fi
        curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            -d "$body" \
            "$ATLAS_UI_BASE/api/configurations/services" >/dev/null
    fi
}

# login-service: version-derived login port, id-keyed merge.
upsert_service_config /atlas/canonical/services/login-service.json login

# channel-service: version-derived channel port + LB_IP, id-keyed merge.
upsert_service_config /atlas/canonical/services/channel-service.json channel

# drops-service: tenant-agnostic (no tenants array) — pass through unchanged.
upsert_service_config /atlas/canonical/services/drops-service.json none

# Rolling restart for services that still read SERVICE_ID synchronously at
# startup. atlas-login and atlas-channel were removed by task-032 — they
# subscribe to the configuration projection topics and apply service /
# tenant updates live without a restart. Keeping them in this list would
# defeat the whole point of the dynamic-config feature.
ATLAS_STEP=service-restart
for d in atlas-drops atlas-character-factory atlas-world; do
    kubectl rollout restart deployment/"$d" 2>/dev/null || log warn "could not restart $d"
done
for d in atlas-drops atlas-character-factory atlas-world; do
    kubectl rollout status deployment/"$d" --timeout=180s 2>/dev/null || log warn "$d not ready"
done

# Data ingest: branch on resolved BOOTSTRAP_MODE.
#   baseline → POST /api/data/baseline/restore (fast, ~60s).
#   full     → PATCH /api/data/wz upload + POST /api/data/process
#              (~10min; ingest now runs inside atlas-data, no separate
#              WZ-extraction step).
ATLAS_STEP=data-ingest
mode=$(resolve_mode)
log info "bootstrap mode: $mode"

docs=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
if [ "$docs" = "0" ] || [ "$docs" = "null" ]; then
    case "$mode" in
        baseline)
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
            ;;
        full)
            log info "uploading canonical WZ zip → PATCH /api/data/wz"
            curl -fsS -X PATCH \
                -H "TENANT_ID: $TENANT_ID" \
                -H "REGION: $REGION" \
                -H "MAJOR_VERSION: $MAJOR_VERSION" \
                -H "MINOR_VERSION: $MINOR_VERSION" \
                -F "zip_file=@$WZ_CANONICAL" \
                "$ATLAS_UI_BASE/api/data/wz" >/dev/null
            log info "running data processing → POST /api/data/process"
            post "$ATLAS_UI_BASE/api/data/process"
            retry 240 10 data_processing_done
            ;;
    esac
else
    log info "data already processed (documentCount=$docs); skipping ingest"
fi

# Per-domain seeds, in parallel
ATLAS_STEP=seed
log info "seeding domain data"
endpoints=(
    /api/drops/seed
    /api/gachapons/seed
    /api/npcs/conversations/seed
    /api/quests/conversations/seed
    /api/shops/seed
    /api/portals/scripts/seed
    /api/reactors/actions/seed
    /api/maps/actions/seed
)
for ep in "${endpoints[@]}"; do
    ( post "$ATLAS_UI_BASE$ep" >/dev/null && log info "seeded $ep" ) &
done
wait

ATLAS_STEP=done log info "bootstrap complete"
