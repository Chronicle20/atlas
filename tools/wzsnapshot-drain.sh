#!/usr/bin/env bash
# wzsnapshot-drain.sh — drain one tenant's skill/job id-sets from live
# atlas-data and print the raw snapshot JSON on stdout.
#
# The skills LIST endpoint (GET /api/data/skills) returns HTTP 400 in this
# baseline, so the id-set is derived by the jobs-union method documented in
# libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md: drain
# GET /api/data/jobs?page[size]=200 and take the union of every row's
# attributes.skills array, with the row ids as the job set.
#
# Pipe into mksnapshot to get a canonical, hash-pinned snapshot file:
#
#   tools/wzsnapshot-drain.sh <tenant-uuid> GMS 95 1 \
#     | (cd libs/atlas-constants/gen && go run ./wzsnapshot/cmd/mksnapshot) \
#     > libs/atlas-constants/gen/wzsnapshot/gms_95_1.json
#
# Requires: kubectl context with access to namespace atlas-main, and jq.
set -euo pipefail

if [ "$#" -lt 4 ]; then
    echo "usage: $0 <tenant-uuid> <REGION> <major> <minor> [pod]" >&2
    exit 2
fi

TENANT="$1"
REGION="$2"
MAJOR="$3"
MINOR="$4"
NAMESPACE="${NAMESPACE:-atlas-main}"
POD="${5:-}"

if [ -z "$POD" ]; then
    POD="$(kubectl -n "$NAMESPACE" get pods -l app=atlas-data \
        -o jsonpath='{.items[0].metadata.name}')"
fi
if [ -z "$POD" ]; then
    echo "wzsnapshot-drain: no atlas-data pod found in namespace $NAMESPACE" >&2
    exit 1
fi

raw="$(kubectl -n "$NAMESPACE" exec "$POD" -- wget -q -O- \
    --header "TENANT_ID: $TENANT" \
    --header "REGION: $REGION" \
    --header "MAJOR_VERSION: $MAJOR" \
    --header "MINOR_VERSION: $MINOR" \
    'http://localhost:8080/api/data/jobs?page[size]=200')"

# Fail loudly rather than emitting an empty id-set: mksnapshot also rejects
# empties, but the pod name / tenant id is only known here.
pages="$(printf '%s' "$raw" | jq -r '.meta.page.last // 1')"
if [ "$pages" != "1" ]; then
    echo "wzsnapshot-drain: tenant $TENANT returned $pages pages; page[size]=200 no longer covers one page. Add a pagination loop before trusting this drain." >&2
    exit 1
fi

printf '%s' "$raw" | jq \
    --arg region "$(printf '%s' "$REGION" | tr '[:upper:]' '[:lower:]')" \
    --argjson major "$MAJOR" \
    --argjson minor "$MINOR" \
    '{region: $region, major: $major, minor: $minor,
      skills: ([.data[].attributes.skills // [] | .[]] | unique),
      jobs:   ([.data[].id | tonumber] | unique)}'
