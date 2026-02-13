#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ASSETS_DIR="${PROJECT_ROOT}/tmp/assets"
IMAGE_NAME="atlas-assets"
CONTAINER_NAME="atlas-assets-dev"
PORT="${1:-8090}"

if [ ! -d "$ASSETS_DIR" ]; then
  echo "Assets directory not found: $ASSETS_DIR"
  echo "Run atlas-wz-extractor first to generate assets."
  exit 1
fi

# Stop any existing container
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# Build the image
echo "Building ${IMAGE_NAME}..."
docker build -t "$IMAGE_NAME" "${PROJECT_ROOT}/services/atlas-assets"

# Run with local assets mounted
echo "Starting ${CONTAINER_NAME} on port ${PORT}..."
docker run --rm \
  --name "$CONTAINER_NAME" \
  -p "${PORT}:8080" \
  -v "${ASSETS_DIR}:/usr/assets:ro" \
  "$IMAGE_NAME"
