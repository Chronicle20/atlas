#!/usr/bin/env bash
# Shared control-plane environment-record helpers, sourced by BOTH
# bootstrap.sh (which records the environment's tenant, FR-3) and
# cleanup.sh (which flips phase during teardown). Extracted from cleanup.sh
# so the two callers cannot drift on the one thing that is easy to get
# wrong: a PATCH body that omits an attribute.
#
# Sourced, never executed — no `set` here (lib.sh owns option state) and no
# executable bit (see test/dockerfile_test.bats).

# env_record_get — echoes this environment's environments record GET
# response body (the full JSON:API document), or nothing if no record exists
# (a 404 from the GET) or ATLAS_UI_BASE/ATLAS_ENVIRONMENT are unset. Exit
# status mirrors curl's.
env_record_get() {
    [ -z "${ATLAS_UI_BASE:-}" ] && return 1
    [ -z "${ATLAS_ENVIRONMENT:-}" ] && return 1
    curl -fsS -H 'Accept: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" 2>/dev/null
}

# env_record_patch <phase> <baseline> <namespace> <tenant> <overrides_json>
# — PATCHes the environments record, sending ALL five attributes.
# environments.RestModel's fields are non-pointer (environments/rest.go), so
# ParseInput unmarshals a PATCH body into a fresh zero-value struct first:
# any attribute omitted from the body is zeroed, not left alone
# (environments/administrator.go's update() doc comment is explicit about
# this). The processor now ALSO backfills omitted fields from the existing
# record (environments/processor.go:243-255), so the two layers disagree
# about who is responsible — sending everything is the move that is correct
# under both.
#
# <phase> must be non-empty: UpdateByName calls validatePhase BEFORE any
# backfill (processor.go:224-226) and phaseIndex("") is -1, so a
# phase-less body is a 400. A same-phase value is a legal transition
# (processor.go:82-88), which is what makes a tenant-only PATCH possible.
env_record_patch() {
    local phase="$1" baseline="$2" namespace="$3" tenant="$4" overrides="$5"
    local payload
    payload=$(jq -nc \
        --arg id "$ATLAS_ENVIRONMENT" \
        --arg baseline "$baseline" \
        --arg namespace "$namespace" \
        --arg tenant "$tenant" \
        --argjson overrides "$overrides" \
        --arg phase "$phase" \
        '{data:{type:"environments",id:$id,attributes:{baseline:$baseline,namespace:$namespace,tenant:$tenant,overrides:$overrides,phase:$phase}}}')
    curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -H "ENVIRONMENT: $ATLAS_ENVIRONMENT" \
        -d "$payload" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null
}
