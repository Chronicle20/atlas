# cluster-infra coordination — task-045 teardown leak fixes

This repo's task-045 PR changes the per-PR teardown. Two pieces live in the
sibling `cluster-infra` repo. **This repo's PR can land independently** — the
PreDelete hook is self-contained and needs nothing from cluster-infra; the
sweep CronJob simply gains the live-env allowlist once the image bump
propagates.

## 1. PreDelete hook — no RBAC change
`deploy/k8s/overlays/pr/predelete-purge.yaml` runs a Job in the per-PR
namespace using the default namespace ServiceAccount. It only needs in-cluster
networking to reach `atlas-ingress` (no Kubernetes API). **Confirm** the default
SA is acceptable; no Role/RoleBinding required.

## 2. Sweep CronJob — cluster-infra owned (singleton)
Apply `sweep-orphans-cronjob.example.yaml` (in this folder) in cluster-infra.
It is a cluster-wide singleton in `argocd`; do NOT add it to this repo's
per-PR `overlays/pr-cleanup` (CI renders that once per PR → N copies).

### Required SA changes for `atlas-pr-cleanup`
The sweep's live-PR-env allowlist (task-045 §4.3) enumerates PR namespaces and
queries each namespace's `atlas-tenants`. Grant the `atlas-pr-cleanup` SA:
- ClusterRole: `list` on `namespaces`.
- Network egress to cross-namespace `atlas-tenants.<ns>.svc:8080`.
Without these, the sweep **fails closed** (aborts rather than deleting with a
partial allowlist) — safe but it won't reclaim anything.

## 3. Confirm reflected secrets/configmaps remain in `argocd`
- `minio-root-creds` (reflected from `minio` ns) — still consumed by the sweep
  CronJob (PostDelete no longer uses it; that envFrom was removed this PR).
- `atlas-pr-cleanup-env` — `MINIO_ENDPOINT`, `ATLAS_MAIN_TENANTS_URL`, etc.

## Merge ordering
1. This repo's PR (image/scripts/PreDelete manifest/sweep logic) — land first.
2. cluster-infra: SA RBAC + CronJob — land any time after the image bump
   propagates. The CronJob is inert-safe before then (fails closed).
