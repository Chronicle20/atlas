#!/usr/bin/env bash
# Cross-references a list of "no-bounds" map IDs (maps whose WZ layout
# extraction failed during atlas-data ingest) against the set of portal
# target-map IDs across all parsed maps. A no-bounds map ID that appears
# as ANY portal's targetMapId is potentially user-visible; one that
# appears nowhere is dead WZ data and can be ignored. See task-076 F11.
#
# Log signal (atlas-data Map worker, see services/atlas-data/atlas.com/data/data/workers/mapw.go):
#   - per-map (DEBUG level):   "extract layout map <id>"  with WithError(err)
#     where the wrapped error is "resolve bounds: ..." or "invalid bounds WxH"
#     produced by libs/atlas-wz/mapimage/layers.go.
#   - aggregate (INFO level):  "map assets: ... extractLayoutErrs=<N> ..."
#
# Inputs:
#   IDS_FILE   (default /tmp/no-bounds-ids.txt) — newline-separated map IDs
#              captured from atlas-data worker logs. Capture recipe (requires
#              DEBUG-level logs from the Map worker):
#                kubectl -n atlas-main logs -l app=atlas-data --tail=20000 \
#                  | grep -E 'extract layout map [0-9]+' \
#                  | grep -oE 'map [0-9]+' \
#                  | awk '{print $2}' \
#                  | sort -un > /tmp/no-bounds-ids.txt
#              If only INFO logs are available, re-ingest with --log-level=debug
#              or temporarily promote the per-map log in mapw.go from Debugf to
#              Warnf to surface the IDs.
#   DATA_URL   (default http://localhost:8080) — reachable atlas-data REST
#   TENANT_ID, REGION, MAJOR_VERSION, MINOR_VERSION — header context
#
# REST shape: GET /api/data/maps?include=portals returns a jsonapi document
# whose `included[]` array contains every portal as a separate resource. The
# portal target-map field is attributes.targetMapId (see
# services/atlas-data/atlas.com/data/map/portal/rest.go).
#
# Output:
#   docs/tasks/task-076-task071-followups/no-bounds-triage.json
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
IDS_FILE="${IDS_FILE:-/tmp/no-bounds-ids.txt}"
OUT="$REPO_ROOT/docs/tasks/task-076-task071-followups/no-bounds-triage.json"

if [[ ! -f "$IDS_FILE" ]]; then
  echo "missing $IDS_FILE — capture the no-bounds IDs from a recent ingest first" >&2
  echo "see the comment block at the top of this script for the capture recipe" >&2
  exit 1
fi

DATA_URL="${DATA_URL:-http://localhost:8080}"
TENANT_ID="${TENANT_ID:-ec876921-c363-4cc6-9c51-5bb8d57f9553}"
REGION="${REGION:-GMS}"
MAJOR_VERSION="${MAJOR_VERSION:-83}"
MINOR_VERSION="${MINOR_VERSION:-1}"

# Pull every map document with portals sideloaded, then extract
# attributes.targetMapId from each included portal resource.
curl -fsS \
  -H "TENANT_ID: $TENANT_ID" -H "REGION: $REGION" \
  -H "MAJOR_VERSION: $MAJOR_VERSION" -H "MINOR_VERSION: $MINOR_VERSION" \
  "$DATA_URL/api/data/maps?include=portals" \
| jq -r '.included[]? | select(.type=="portals") | .attributes.targetMapId' \
| sort -un > /tmp/portal-targets.txt

# Reachable subset: no-bounds IDs that appear as some other map's portal target.
comm -12 \
  <(sort -u "$IDS_FILE") \
  <(sort -u /tmp/portal-targets.txt) > /tmp/reachable-ids.txt

# Unreachable subset: no-bounds IDs with no portal pointing to them.
comm -23 \
  <(sort -u "$IDS_FILE") \
  <(sort -u /tmp/portal-targets.txt) > /tmp/unreachable-ids.txt

jq -n \
  --argjson reachable "$(jq -R 'tonumber? // empty' /tmp/reachable-ids.txt | jq -s .)" \
  --argjson unreachable "$(jq -R 'tonumber? // empty' /tmp/unreachable-ids.txt | jq -s .)" \
  --arg generated "$(date -u +%FT%TZ)" \
  '{
    generatedAt: $generated,
    reachable: $reachable,
    unreachable: $unreachable,
    counts: {
      reachable: ($reachable|length),
      unreachable: ($unreachable|length),
      total: (($reachable|length) + ($unreachable|length))
    }
  }' > "$OUT"

echo "wrote $OUT (reachable=$(wc -l < /tmp/reachable-ids.txt), unreachable=$(wc -l < /tmp/unreachable-ids.txt))"
