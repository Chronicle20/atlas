#!/usr/bin/env bash
# Start the Atlas compose stack. Does NOT build images — run ./build.sh first,
# or pass --build to rebuild in-place (parallel; may OOM on constrained hosts).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACK="${1:-core}"
if [[ $# -gt 0 ]]; then shift; fi

case "$STACK" in
  core)   FILES=(-f docker-compose.yml -f docker-compose.core.yml) ;;
  socket) FILES=(-f docker-compose.yml -f docker-compose.socket.yml) ;;
  all)    FILES=(-f docker-compose.yml -f docker-compose.core.yml -f docker-compose.socket.yml) ;;
  *)
    echo "usage: $(basename "$0") {core|socket|all} [docker compose up args...]" >&2
    exit 2
    ;;
esac

if [[ ! -f "$SCRIPT_DIR/.env" ]]; then
  echo "error: $SCRIPT_DIR/.env not found. Copy .env.example and edit." >&2
  exit 1
fi

cd "$SCRIPT_DIR"
exec docker compose --env-file "$SCRIPT_DIR/.env" --project-name atlas "${FILES[@]}" up "$@"
