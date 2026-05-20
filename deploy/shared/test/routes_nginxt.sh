#!/usr/bin/env bash
# Lightweight syntax check: run nginx in a container with the shared routes.conf
# included from a minimal http{}/server{} wrapper, and exit non-zero if
# `nginx -t` complains.
#
# Note: this does NOT test routing behavior. A docker-based upstream-stub
# regression suite is deferred per task-071 Task 15 scope guidance.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
ROUTES="$REPO_ROOT/deploy/shared/routes.conf"

if [[ ! -f "$ROUTES" ]]; then
  echo "routes.conf not found at $ROUTES" >&2
  exit 1
fi

# nginx requires a parent http{} block to validate location directives, and a
# valid resolver to allow runtime variable hostnames inside proxy_pass. The
# wrapper below provides both; we mount routes.conf untouched.
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

cat > "$TMPDIR/nginx.conf" <<'EOF'
events {}
http {
  # /etc/hosts-only resolver so `nginx -t` doesn't try real DNS for hostnames
  # like atlas-data, atlas-renders, minio referenced in proxy_pass lines.
  resolver 127.0.0.11 ipv6=off valid=30s;
  server {
    listen 80;
    server_name _;
    underscores_in_headers on;
    include /etc/nginx/conf.d/routes.conf;
  }
}
EOF

docker run --rm \
  -v "$TMPDIR/nginx.conf:/etc/nginx/nginx.conf:ro" \
  -v "$ROUTES:/etc/nginx/conf.d/routes.conf:ro" \
  nginx:alpine nginx -t
