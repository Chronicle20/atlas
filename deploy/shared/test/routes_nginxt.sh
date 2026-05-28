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

# Character-render hash-length guard: the loadout hash is the first 16 hex
# chars of SHA-256(canonical). Both the UI producer (characterRender.service.ts
# loadoutHash → .slice(0,16)) and the Go consumer (renders/character/hash.go
# LoadoutHash → [:16]) emit exactly 16 chars. The character-render location's
# hash capture MUST accept 16 chars; the original {32,64} quantifier (task-071
# #544) never matched a real hash, so every character image request fell
# through to the generic-asset block and 404'd against MinIO instead of
# reaching atlas-renders. Guard the exact-16 contract here.
if ! grep -nE 'character/\(\?<hash>\[a-f0-9\]\{16\}\)' "$ROUTES" >/dev/null; then
  echo "error: character-render location must capture a 16-char hex hash" \
       "(/character/(?<hash>[a-f0-9]{16})). The loadout hash is" \
       "SHA-256(canonical)[:16]; a wider/narrower quantifier routes character" \
       "images to MinIO (404) instead of atlas-renders." >&2
  grep -nE 'character/\(\?<hash>' "$ROUTES" >&2 || true
  exit 1
fi
echo "routes.conf character-render hash-length check: OK"

# F18: confirm the generated k8s routes file is in sync with the canonical
# shared source. If shared/routes.conf changes, the committer MUST also run
# tools/gen-routes.sh and commit the resulting routes.conf.template.generated.
GEN="$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"
if [[ -f "$GEN" ]]; then
  # Re-run the generator (it writes the committed file in place). Then use
  # `git diff --quiet` on that file: a clean tree means the committed copy
  # already matches; any drift means shared/routes.conf was updated without
  # re-running the generator.
  bash "$REPO_ROOT/tools/gen-routes.sh" >/dev/null
  if ! git -C "$REPO_ROOT" diff --quiet -- deploy/k8s/base/routes.conf.template.generated; then
    echo "error: deploy/shared/routes.conf changed but routes.conf.template.generated is stale." >&2
    echo "       run tools/gen-routes.sh and commit the result." >&2
    git -C "$REPO_ROOT" --no-pager diff -- deploy/k8s/base/routes.conf.template.generated | head -40 >&2
    # Restore committed copy so a stale local checkout doesn't surprise the next run.
    git -C "$REPO_ROOT" checkout -- deploy/k8s/base/routes.conf.template.generated >/dev/null 2>&1 || true
    exit 1
  fi
  echo "routes drift check (shared vs k8s-generated): OK"
else
  echo "warn: $GEN does not exist; skipping F18 drift check (was Task 6 / F8 applied?)" >&2
fi
