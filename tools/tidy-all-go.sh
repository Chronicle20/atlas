#!/usr/bin/env bash
# tools/tidy-all-go.sh — `go mod tidy && go mod download` across every
# workspace module.
#
# These mutate GOMODCACHE, which is machine-global while worktrees are not.
# Concurrent sessions running this against the same cache is the one genuinely
# unsafe concurrency in the build system, so the whole sweep takes an exclusive
# lock — distinct from the counting build slots in tools/lib/build-slot.sh,
# which bound CPU and RAM rather than protecting a shared mutable store.
set -euo pipefail

LOCK="${ATLAS_GOMODCACHE_LOCK:-/var/tmp/atlas/gomodcache.lock}"
mkdir -p "$(dirname "$LOCK")"
exec 9>"$LOCK"
flock 9

mods=$(
  find ./services ./libs -name go.mod -print0 \
    | xargs -0 -n1 dirname \
    | sort -u
)

while IFS= read -r d; do
  echo "==> $d"
  (cd "$d" && go mod tidy && go mod download)
done <<< "$mods"

