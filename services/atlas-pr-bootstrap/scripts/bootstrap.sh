#!/usr/bin/env bash
# Atlas PR-env bootstrap. Idempotent — short-circuits each step that
# is already complete. Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   WZ_CANONICAL       — path to canonical zip (default /opt/wz/atlas.zip)
#   TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION — required for tenant headers

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

require_env ATLAS_ENV ATLAS_UI_BASE TENANT_ID REGION MAJOR_VERSION MINOR_VERSION
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"

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

# Polling helpers — return 0 when the target value is non-zero/non-null,
# 1 otherwise. Designed for use with retry().
extraction_done() {
    local count
    count=$(get_attr "$ATLAS_UI_BASE/api/wz/extractions" fileCount)
    [ -n "$count" ] && [ "$count" != "0" ] && [ "$count" != "null" ]
}

data_processing_done() {
    local count
    count=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
    [ -n "$count" ] && [ "$count" != "0" ] && [ "$count" != "null" ]
}

ATLAS_STEP=wait-ready log info "waiting for atlas-tenants, atlas-configurations, atlas-data, atlas-wz-extractor"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/tenants"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/configurations/services"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/status"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/wz/input"
retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/wz/extractions"

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

upsert_service_config() {
    local payload_path="$1"
    local rewrite_ip="$2"   # "yes" to substitute tenants[].ipAddress with LB_IP
    local svc_id
    svc_id=$(jq -r '.data.id' "$payload_path")
    if [ -z "$svc_id" ] || [ "$svc_id" = "null" ]; then
        log error "missing data.id in $payload_path"
        return 1
    fi

    # Rewrite tenants[].id to per-PR TENANT_ID.
    # For channel-service, also rewrite tenants[].ipAddress to LB_IP.
    #
    # Tenant-agnostic configs (drops-service) have no .data.attributes.tenants
    # — guarded with `has("tenants")` instead of `(.tenants? // [])` because
    # the latter is not a valid path expression on the LHS of `|=` and jq
    # errors out with "Invalid path expression with result []".
    local rewritten
    if [ "$rewrite_ip" = "yes" ]; then
        rewritten=$(jq --arg tid "$TENANT_ID" --arg ip "$LB_IP" \
            'if .data.attributes | has("tenants") then .data.attributes.tenants |= map(.id = $tid | (if has("ipAddress") then .ipAddress = $ip else . end)) else . end' \
            "$payload_path")
    else
        rewritten=$(jq --arg tid "$TENANT_ID" \
            'if .data.attributes | has("tenants") then .data.attributes.tenants |= map(.id = $tid) else . end' \
            "$payload_path")
    fi

    local existing
    existing=$(curl -fsS -H 'Accept: application/vnd.api+json' \
        "$ATLAS_UI_BASE/api/configurations/services/$svc_id" 2>/dev/null || true)

    if echo "$existing" | jq -e '.data.id' >/dev/null 2>&1; then
        log info "service config $svc_id exists; PATCH"
        curl -fsS -X PATCH \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            -d "$rewritten" \
            "$ATLAS_UI_BASE/api/configurations/services/$svc_id" >/dev/null
    else
        log info "service config $svc_id absent; POST"
        curl -fsS -X POST \
            -H 'Accept: application/vnd.api+json' \
            -H 'Content-Type: application/vnd.api+json' \
            -d "$rewritten" \
            "$ATLAS_UI_BASE/api/configurations/services" >/dev/null
    fi
}

# login-service: rewrite tenants[].id only (no ipAddress)
upsert_service_config /atlas/canonical/services/login-service.json no

# channel-service: rewrite tenants[].id AND tenants[].ipAddress = LB_IP
upsert_service_config /atlas/canonical/services/channel-service.json yes

# drops-service: tenant-agnostic (no tenants array). The jq map is a no-op
# in that case because (.tenants? // []) yields an empty array.
upsert_service_config /atlas/canonical/services/drops-service.json no

# Rolling restart for the 5 services that read SERVICE_ID at startup
# so they re-fetch the freshly-written config. login/channel especially.
ATLAS_STEP=service-restart
for d in atlas-login atlas-channel atlas-drops atlas-character-factory atlas-world; do
    kubectl rollout restart deployment/"$d" 2>/dev/null || log warn "could not restart $d"
done
for d in atlas-login atlas-channel atlas-drops atlas-character-factory atlas-world; do
    kubectl rollout status deployment/"$d" --timeout=180s 2>/dev/null || log warn "$d not ready"
done

# WZ upload: PATCH /api/wz/input
ATLAS_STEP=wz-upload
files=$(get_attr "$ATLAS_UI_BASE/api/wz/input" fileCount)
if [ "$files" = "0" ] || [ "$files" = "null" ]; then
    log info "uploading canonical WZ zip"
    curl -fsS -X PATCH \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -F "zip_file=@$WZ_CANONICAL" \
        "$ATLAS_UI_BASE/api/wz/input"
else
    log info "WZ already uploaded (fileCount=$files), skipping"
fi

# WZ extraction
ATLAS_STEP=wz-extract
extracted=$(get_attr "$ATLAS_UI_BASE/api/wz/extractions" fileCount)
if [ "$extracted" = "0" ] || [ "$extracted" = "null" ]; then
    log info "running WZ extraction"
    post "$ATLAS_UI_BASE/api/wz/extractions"
    retry 240 10 extraction_done
fi

# Data processing
ATLAS_STEP=data-process
docs=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
if [ "$docs" = "0" ] || [ "$docs" = "null" ]; then
    log info "running data processing"
    post "$ATLAS_UI_BASE/api/data/process"
    retry 240 10 data_processing_done
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
