#!/usr/bin/env bash
# Builds Atlas Go-service images via docker buildx bake against the
# repo-root docker-bake.hcl. Forwards any arguments through to bake so
# callers can target a subset:
#
#   tools/build-services.sh                        # all-go-services
#   tools/build-services.sh atlas-account          # one
#   tools/build-services.sh atlas-account atlas-ban  # subset
#
# `--load` is mandatory here: the governed `atlas` builder
# (tools/buildx-bootstrap.sh) uses the docker-container driver, which — unlike
# the default `docker` driver — does not write built images to the local
# image store on its own. Without `--load` this script would "succeed" while
# producing no `<svc>:local` image for anything downstream to run.
#
# Run from the repo root.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
./tools/buildx-bootstrap.sh --check
exec docker buildx bake --builder "${ATLAS_BUILDER:-atlas}" --load "$@"
