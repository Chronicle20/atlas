#!/usr/bin/env bash

# Build all modules
while IFS= read -r moddir; do
  echo "==> $moddir"
  (cd "$moddir" && go test ./... )
done < <(find ./services ./libs -name go.mod -print0 | xargs -0 -n1 dirname | sort -u)

