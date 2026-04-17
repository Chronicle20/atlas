#!/usr/bin/env bash
# Build Atlas service images one at a time to avoid OOM on constrained hosts (e.g. WSL).
# Parallel `docker compose build` spawns many Go compilers at once and can crash WSL;
# this script builds each service sequentially.
#
# Usage:
#   ./build.sh {core|socket|all} [service ...]
# Examples:
#   ./build.sh core                       # build every service in the core stack
#   ./build.sh all                        # build every service in core + socket
#   ./build.sh core atlas-account atlas-ban   # build only the named services from core
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACK="${1:-core}"
if [[ $# -gt 0 ]]; then shift; fi

case "$STACK" in
  core)   FILES=(-f docker-compose.yml -f docker-compose.core.yml) ;;
  socket) FILES=(-f docker-compose.yml -f docker-compose.socket.yml) ;;
  all)    FILES=(-f docker-compose.yml -f docker-compose.core.yml -f docker-compose.socket.yml) ;;
  *)
    echo "usage: $(basename "$0") {core|socket|all} [service ...]" >&2
    exit 2
    ;;
esac

if [[ ! -f "$SCRIPT_DIR/.env" ]]; then
  echo "error: $SCRIPT_DIR/.env not found. Copy .env.example and edit." >&2
  exit 1
fi

cd "$SCRIPT_DIR"

COMPOSE=(docker compose --env-file "$SCRIPT_DIR/.env" --project-name atlas "${FILES[@]}")

if [[ $# -gt 0 ]]; then
  SERVICES=("$@")
else
  # Only build services that declare a build: section. `docker compose config`
  # flattens the merged YAML; grep matches the two-space-indented service headers
  # whose block contains a build: line.
  mapfile -t SERVICES < <(
    "${COMPOSE[@]}" config |
    awk '
      /^[a-zA-Z_-]+:$/ { in_services = ($1 == "services:") ? 1 : 0; next }
      !in_services { next }
      /^  [a-zA-Z0-9_-]+:$/ { svc = $1; sub(":", "", svc); has_build = 0; next }
      /^    build:/ && svc != "" && !printed[svc] { print svc; printed[svc] = 1 }
    ' | sort
  )
fi

if (( ${#SERVICES[@]} == 0 )); then
  echo "error: no buildable services found." >&2
  exit 1
fi

echo "Building ${#SERVICES[@]} service(s) sequentially:"
printf '  - %s\n' "${SERVICES[@]}"
echo

FAILED=()
for svc in "${SERVICES[@]}"; do
  echo "==========================================================================="
  echo ">> Building $svc"
  echo "==========================================================================="
  if "${COMPOSE[@]}" build "$svc"; then
    echo ">> OK: $svc"
  else
    echo ">> FAIL: $svc" >&2
    FAILED+=("$svc")
  fi
  echo
done

if (( ${#FAILED[@]} > 0 )); then
  echo "Failed builds: ${FAILED[*]}" >&2
  exit 1
fi

echo "All builds succeeded. Run ./up.sh $STACK to start the stack without rebuilding."
