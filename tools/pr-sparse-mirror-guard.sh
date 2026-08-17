#!/usr/bin/env bash
# Verifies deploy/k8s/overlays/pr-sparse's hand-synced copies of
# overlays/pr files are still byte-identical to their originals.
#
# WHY copies, not references: kustomize's default load restrictor rejects
# any resource/patch path that escapes the overlay's own directory tree
# (`security; file '...' is not in or below '.../pr-sparse'`), so
# pr-sparse/kustomization.yaml cannot reference ../pr/*.yaml directly. A
# shared `components:` directory (e.g. a hypothetical
# deploy/k8s/overlays/_shared/) hits the identical restrictor — verified
# not viable during task-232 Task 44's review (fix round 1) — so this
# guard, not a `components:` restructure, is what closes the drift hazard.
# Promoting these files into deploy/k8s/base/ (a true common ancestor, the
# same shape as deploy/k8s/base/components/seed-catalog/) would also close
# it, but requires touching overlays/pr, which is live in `main` today and
# out of this task's scope. This guard does not block that follow-up; it
# just makes today's duplication safe in the meantime.
#
# The MIRRORS array below is the single source of truth for which files
# mirror which — nothing else (README, comments) should re-enumerate this
# list; point at this script instead.
#
#   pr-sparse-mirror-guard.sh   diff every pair; exit 1 and name the
#                                diverged file(s) on any mismatch.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
PR_DIR="$REPO_ROOT/deploy/k8s/overlays/pr"
SPARSE_DIR="$REPO_ROOT/deploy/k8s/overlays/pr-sparse"

# Path relative to each overlay's own root; identical on both sides.
MIRRORS=(
  atlas-env-tokens.yaml
  ingress-route.yaml
  sync-bootstrap.yaml
  predelete-purge.yaml
  postsync-pihole-add.yaml
  patches/ingress-host.yaml
  patches/consumer-group-env.yaml
  patches/lb-allocate.yaml
  patches/seed-catalog-ref.yaml
)

status=0
for f in "${MIRRORS[@]}"; do
  src="$PR_DIR/$f"
  dst="$SPARSE_DIR/$f"
  if [[ ! -f "$src" ]]; then
    echo "pr-sparse-mirror-guard: missing source deploy/k8s/overlays/pr/$f" >&2
    status=1
    continue
  fi
  if [[ ! -f "$dst" ]]; then
    echo "pr-sparse-mirror-guard: missing mirror deploy/k8s/overlays/pr-sparse/$f" >&2
    status=1
    continue
  fi
  if ! diff -q "$src" "$dst" >/dev/null; then
    echo "pr-sparse-mirror-guard: deploy/k8s/overlays/pr-sparse/$f has drifted from deploy/k8s/overlays/pr/$f" >&2
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo "pr-sparse-mirror-guard: one or more mirrored files are stale; re-copy from overlays/pr/ and commit" >&2
  exit 1
fi
echo "pr-sparse-mirror-guard: up to date"
