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

# String-level guard: any cross-namespace upstream MUST be fully qualified
# with `.svc.cluster.local`. The bare-hostname pattern (e.g. `minio:9000`)
# resolved to the ingress pod's OWN namespace and broke every UI image
# request on PR-544 — see commit history. nginx -t doesn't catch this
# because it's about routing intent, not syntax.
if grep -nE 'set \$u +"minio:[0-9]+"' "$ROUTES" >/dev/null; then
  echo "error: routes.conf uses bare \`minio:<port>\` upstream — MinIO is in the \`minio\` namespace, not the ingress's namespace. Use \`minio.minio.svc.cluster.local:<port>\`." >&2
  grep -nE 'set \$u +"minio:[0-9]+"' "$ROUTES" >&2
  exit 1
fi
echo "routes.conf MinIO upstream cross-namespace check: OK"

# String-level guard: any nginx location that proxies to atlas-renders MUST set
# the four tenant headers (TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION). The
# atlas-renders tenant middleware (services/atlas-renders/.../main.go) returns
# 400 when any are missing. The character-render block at routes.conf:190-204
# sets them; @maprender_miss originally did not — every cache-miss map render
# 400'd on PR-544 (see finish-line.md Bug A).
python3 - "$ROUTES" <<'PY' || exit 1
import re, sys, pathlib
text = pathlib.Path(sys.argv[1]).read_text()
# Find every location block (named or path) that proxies to atlas-renders.
# Lookahead-bounded block extraction: from `location ... {` to matching `}`.
def blocks(src):
    i = 0
    while True:
        m = re.search(r'\blocation\s+[^{]*\{', src[i:])
        if not m:
            return
        start = i + m.start()
        # Naive brace match (no embedded {}) — routes.conf locations are flat.
        depth = 0
        for j in range(start, len(src)):
            c = src[j]
            if c == '{': depth += 1
            elif c == '}':
                depth -= 1
                if depth == 0:
                    yield src[start:j+1]
                    i = j + 1
                    break
        else:
            return
required = ['TENANT_ID', 'REGION', 'MAJOR_VERSION', 'MINOR_VERSION']
fail = False
for b in blocks(text):
    if 'atlas-renders:' not in b:
        continue
    missing = [h for h in required if f'proxy_set_header {h}' not in b]
    if missing:
        fail = True
        header_line = b.splitlines()[0]
        print(f"error: location block proxying to atlas-renders missing headers {missing}:", file=sys.stderr)
        print(f"  {header_line}", file=sys.stderr)
if fail:
    sys.exit(1)
print("routes.conf atlas-renders tenant header check: OK")
PY
