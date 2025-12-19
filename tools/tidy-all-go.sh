#!/usr/bin/env bash
set -euo pipefail

mods=$(
  find ./services ./libs -name go.mod -print0 \
    | xargs -0 -n1 dirname \
    | sort -u
)

while IFS= read -r d; do
  echo "==> $d"
  (cd "$d" && go mod tidy && go mod download)
done <<< "$mods"

