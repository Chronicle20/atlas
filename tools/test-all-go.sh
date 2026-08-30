#!/usr/bin/env bash

# Build all modules
#
# A module directory found by the glob below may or may not be a go.work
# workspace member. libs/atlas-kafka/gen is deliberately excluded from
# go.work (its gen scanner loads every `use` directive, so a gen module
# inside the workspace would scan itself) while libs/atlas-constants/gen
# looks identical by name but IS a member. Ask `go work edit -json` for the
# authoritative `use` list rather than guessing from the directory name, and
# run non-members with GOWORK=off so `go test` resolves them as standalone
# modules instead of failing to find themselves in the workspace.
declare -A workspace_members
while IFS= read -r use_dir; do
  workspace_members["$(cd "$use_dir" && pwd)"]=1
done < <(go work edit -json | jq -r '.Use[].DiskPath')

while IFS= read -r moddir; do
  echo "==> $moddir"
  abs_moddir="$(cd "$moddir" && pwd)"
  if [[ -n "${workspace_members[$abs_moddir]:-}" ]]; then
    (cd "$moddir" && go test ./... )
  else
    (cd "$moddir" && GOWORK=off go test ./... )
  fi
done < <(find ./services ./libs -name go.mod -print0 | xargs -0 -n1 dirname | sort -u)
