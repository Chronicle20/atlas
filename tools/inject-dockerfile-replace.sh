#!/usr/bin/env bash
# One-shot helper used during the deploy-reorg PR to inject `go mod edit
# -replace=...` + `go mod tidy` into every service Dockerfile so cold compose
# builds resolve Chronicle20/atlas-* modules from the in-repo libs/ rather
# than github.com (which 404s without auth).
#
# Idempotent: skips Dockerfiles that already contain the marker comment.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MARKER="# Force local resolution of Chronicle20/atlas-\* modules"

for df in "$REPO"/services/atlas-*/Dockerfile; do
  if grep -qF "Force local resolution of Chronicle20/atlas-* modules" "$df"; then
    continue
  fi

  build_line=$(grep -nE '^RUN go build -C services/atlas-[a-z-]+/atlas\.com/[a-zA-Z-]+' "$df" || true)
  if [[ -z "$build_line" ]]; then
    echo "skip $df (no go build line found)" >&2
    continue
  fi

  service_path=$(echo "$build_line" | sed -E 's/^[0-9]+:RUN go build -C ([^ ]+) .*/\1/')
  libs=$(grep -E '^COPY libs/atlas-[a-z-]+ libs/atlas-[a-z-]+$' "$df" | awk '{print $2}' | sed 's|libs/||' | sort -u)
  if [[ -z "$libs" ]]; then
    echo "skip $df (no libs copied)" >&2
    continue
  fi

  injection_file=$(mktemp)
  {
    echo "# Force local resolution of Chronicle20/atlas-* modules (bypasses unreachable github.com publishes)."
    echo "RUN cd $service_path && \\"
    echo "    go mod edit \\"
    while IFS= read -r lib; do
      echo "      -replace=github.com/Chronicle20/$lib=/app/libs/$lib \\"
    done <<< "$libs"
    echo "    && go mod tidy"
    echo ""
  } > "$injection_file"

  awk -v inject_file="$injection_file" -v target_path="$service_path" '
    BEGIN {
      while ((getline line < inject_file) > 0) inject = inject line "\n"
      close(inject_file)
    }
    /^# Build$/ && !done {
      printf "%s", inject
      done = 1
    }
    /^RUN go build -C / && !done {
      printf "%s", inject
      done = 1
    }
    { print }
  ' "$df" > "$df.tmp"
  mv "$df.tmp" "$df"
  rm -f "$injection_file"
  echo "patched $df"
done
