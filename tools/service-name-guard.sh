#!/usr/bin/env bash
# tools/service-name-guard.sh — asserts every service Deployment carries a
# non-empty SERVICE_NAME, sourced from the Deployment's own `app` pod label
# via the downward API (never a hand-maintained per-service literal).
#
# Why this exists (task-232 Task 29A): libs/atlas-kafka/consumer/manager.go
# reads SERVICE_NAME into Consumer.service, and libs/atlas-env/registry.go
# MapRegistry.IsOwner keys off it. If a new service's base Deployment lands
# without SERVICE_NAME (or with a value that doesn't match the Deployment's
# own name — e.g. "monsters" instead of "atlas-monsters"), the ownership gate
# silently collapses to `Baseline == self`: the baseline permanently steals
# that service's overridden-environment traffic while every counter and test
# still looks correct (see task-29A-brief.md). That is a silent misroute, not
# a crash — exactly what a rendered-manifest assertion catches and a
# `go build`/`go vet` pass cannot.
#
# Method: render both the `main` and `pr` overlays with `kubectl kustomize`
# (no external `kustomize` binary required — kubectl ships on every
# GitHub-hosted Ubuntu runner) and assert every Deployment's non-sidecar
# container carries:
#   - name: SERVICE_NAME
#     valueFrom:
#       fieldRef:
#         fieldPath: metadata.labels['app']
#
# Sidecars that are not the service's own Go binary (currently only the
# seed-catalog component's `git-sync` container) are exempt — they never
# construct a libs/atlas-kafka Consumer and do not participate in the
# ownership gate. New sidecar container names must be added to
# SIDECAR_ALLOWLIST explicitly; nothing is exempted implicitly.
#
# NOT wired into tools/verify.sh yet. tools/verify.sh is being edited
# concurrently on this branch by another task; wiring this guard into its
# deploy/-path change-gate list is left to whichever task next touches that
# file, to avoid colliding with the concurrent edit. Run directly:
#   tools/service-name-guard.sh
#
# Exit 0 = every service Deployment carries a correct SERVICE_NAME.
# Exit 1 = violations found (listed, one per line).

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

fail=0

for overlay in deploy/k8s/overlays/main deploy/k8s/overlays/pr; do
    result="$(kubectl kustomize "$overlay" | python3 tools/service-name-guard-check.py "$overlay" || true)"
    if [ -n "$result" ]; then
        printf '%s\n' "$result"
        fail=1
    fi
done

echo ""
if [ "$fail" -ne 0 ]; then
    echo "service-name-guard: violations found (see above)"
    echo "See libs/atlas-env/registry.go MapRegistry.IsOwner and .superpowers/sdd/plan/task-29A-brief.md."
    exit 1
fi
echo "service-name-guard: clean (main + pr overlays checked)"
