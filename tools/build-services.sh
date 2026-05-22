#!/usr/bin/env bash
# Builds Atlas Go-service images via docker buildx bake against the
# repo-root docker-bake.hcl. Forwards any arguments through to bake so
# callers can target a subset:
#
#   tools/build-services.sh                        # all-go-services
#   tools/build-services.sh atlas-account          # one
#   tools/build-services.sh atlas-account atlas-ban  # subset
#
# Run from the repo root.
set -euo pipefail
exec docker buildx bake "$@"
