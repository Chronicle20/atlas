#!/usr/bin/env bash
# Regenerate services/atlas-map-actions/docs/map_script_schema.json from Go
# source: the condition-type and operator constants in libs/atlas-saga/
# validation.go, and the operation cases in atlas-map-actions'
# ExecuteOperation switch. task-290 design D4 / PRD FR-1.1, FR-1.2, FR-1.5,
# FR-3.0.
#
#   gen-map-action-schema.sh           rewrite the schema in place
#   gen-map-action-schema.sh --check   exit 1 with a diff on drift; writes nothing
#
# The generator is its own module and is deliberately outside go.work, so it
# is invoked with GOWORK=off (same posture as tools/gen-topics.sh).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT/tools/gen-map-action-schema"
GOWORK=off exec go run . "$@"
