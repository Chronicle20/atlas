#!/usr/bin/env bash
# Regenerate the `routes.conf` block-scalar key inside deploy/k8s/ingress.yaml
# from deploy/shared/routes.conf. With --check, exit non-zero if the rendered
# block differs from the committed file (drift detection).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SHARED="$REPO_ROOT/deploy/shared/routes.conf"
INGRESS="$REPO_ROOT/deploy/k8s/ingress.yaml"
INDENT="    "  # 4 spaces — block-scalar content depth inside `data:` (2) + 2.

CHECK=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK=1
fi

if [[ ! -f "$SHARED" ]]; then
  echo "error: $SHARED not found" >&2
  exit 1
fi
if [[ ! -f "$INGRESS" ]]; then
  echo "error: $INGRESS not found" >&2
  exit 1
fi

# Render the indented routes block for embedding in the YAML scalar.
rendered=$(mktemp)
trap 'rm -f "$rendered"' EXIT
awk -v ind="$INDENT" '{ if (length($0) == 0) print ""; else print ind $0 }' "$SHARED" > "$rendered"

# Splice: replace lines from `^  routes.conf: \|` through the next top-level
# `---` (or the next `data` sibling) with the new block.
out=$(mktemp)
trap 'rm -f "$rendered" "$out"' EXIT

awk -v block="$rendered" '
  BEGIN { in_block=0 }
  /^  routes\.conf: \|$/ {
    print
    while ((getline line < block) > 0) print line
    close(block)
    in_block=1
    next
  }
  in_block==1 {
    if ($0 ~ /^---$/) { in_block=0; print; next }
    if ($0 ~ /^[a-zA-Z]/) { in_block=0; print; next }
    next
  }
  { print }
' "$INGRESS" > "$out"

if [[ $CHECK -eq 1 ]]; then
  if ! diff -u "$INGRESS" "$out" > /dev/null; then
    echo "error: deploy/k8s/ingress.yaml is out of sync with deploy/shared/routes.conf" >&2
    diff -u "$INGRESS" "$out" >&2 || true
    exit 1
  fi
  exit 0
fi

if diff -q "$INGRESS" "$out" > /dev/null; then
  exit 0
fi
mv "$out" "$INGRESS"
echo "updated $INGRESS"
