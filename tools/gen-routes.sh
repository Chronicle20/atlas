#!/usr/bin/env bash
# Generates deploy/k8s/base/routes.conf.template.generated from
# deploy/shared/routes.conf by rewriting bare service hostnames to FQDNs
# templated on ${POD_NAMESPACE}. See task-076 F8.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
SRC="$REPO_ROOT/deploy/shared/routes.conf"
OUT="$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"

if [[ ! -f "$SRC" ]]; then
  echo "missing $SRC" >&2
  exit 1
fi

# Rewrite `set $u "atlas-XXX:8080"` and `proxy_pass http://atlas-XXX:8080...`
# to use `.${POD_NAMESPACE}.svc.cluster.local`. The `minio:9000` upstream is
# already namespace-qualified in the shared file (per the F8/F18 cross-ns
# guard), so we leave that untouched.
sed -E \
  -e 's|set \$u "(atlas-[a-z-]+):8080"|set $u "\1.${POD_NAMESPACE}.svc.cluster.local:8080"|g' \
  -e 's|proxy_pass http://(atlas-[a-z-]+):8080|proxy_pass http://\1.${POD_NAMESPACE}.svc.cluster.local:8080|g' \
  -e 's|set \$u "atlas-ui:80"|set $u "atlas-ui.${POD_NAMESPACE}.svc.cluster.local:80"|g' \
  "$SRC" > "$OUT"

echo "wrote $OUT ($(wc -l <"$OUT") lines)"
