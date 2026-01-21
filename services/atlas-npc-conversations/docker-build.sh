#!/bin/bash
# Build from atlas root directory to include libs
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ATLAS_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SERVICE_NAME="${SCRIPT_DIR##*/}"

if [[ "$1" = "NO-CACHE" ]]
then
   docker build --no-cache -f "$SCRIPT_DIR/Dockerfile.dev" --tag "$SERVICE_NAME:latest" "$ATLAS_ROOT"
else
   docker build -f "$SCRIPT_DIR/Dockerfile.dev" --tag "$SERVICE_NAME:latest" "$ATLAS_ROOT"
fi
