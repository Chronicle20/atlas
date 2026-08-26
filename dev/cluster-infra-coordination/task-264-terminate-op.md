# cluster-infra coordination — task-264 terminate stuck sync operations

This repo's task-264 PR bounds the per-PR Argo hook Jobs with
`activeDeadlineSeconds` and corrects the teardown runbook. One piece — a
defense-in-depth reconciler — lives in the sibling `cluster-infra` repo.
**This repo's PR can land independently.** Nothing in this repo's diff
references the CronJob, and the deadline/workflow/runbook changes are
self-contained. The CronJob is additive and inert-safe in either order.

(Runbook §9.13's rule is that cluster-infra lands *before* the consuming
atlas PR, because an atlas change depending on a missing cluster-infra
object wedges. That rule does not bind here — there is no such dependency.
Same posture as task-045's PreDelete hook.)

## The problem it covers

When an Argo CD sync is in flight at the moment `atlas-pr-<N>` is marked
for deletion, the operation neither completes nor terminates. Its Sync-hook
Job keeps `argocd.argoproj.io/hook-finalizer`, which only the controller
removes as part of completing or terminating the owning operation — and the
operation's phase is driven by that same Job. `resources-finalizer` blocks
on the undeletable Job, so no other namespaced object is ever issued for
deletion. Observed on PR #1459, 2026-08-26: 89 objects stuck, the controller
logging `N objects remaining for deletion` on every reconcile with N never
decreasing.

There are two independent ways to break that cycle. This repo's PR takes the
first (force the hook Job terminal via `activeDeadlineSeconds`). This CronJob
is the second (force the operation terminal directly). They fail
independently — the deadline is inert for any future hook added without one,
and the CronJob is inert once `.operation` has been cleared — so both are
worth having.

## The CronJob — cluster-infra owned (singleton)

Apply `terminate-stuck-ops-cronjob.example.yaml` (in this folder) in
cluster-infra. It is a cluster-wide singleton in `argocd`, alongside the two
that already live there: `atlas-pr-cleanup` (`5 * * * *`) and
`atlas-pr-sweep-orphans` (`0 */6 * * *`). This is a third. Do NOT add it to
this repo's per-PR `overlays/pr-cleanup` — CI renders that once per PR, which
would produce one copy per open PR.

Each run lists `atlas-pr-*` Applications in `argocd`, selects those with a
`deletionTimestamp`, `status.operationState.phase == "Running"`, and a
non-nil `.operation`, and patches the phase to `Terminating`. It is
idempotent and self-limiting: the predicate is false for every healthy
Application and false again once the controller has moved the phase off
`Running`.

## Required RBAC changes: NONE

Unlike task-045's sweep CronJob, this needs no new grant. Two facts, both
verified live on 2026-08-26 against `quay.io/argoproj/argocd:v3.4.2`:

**1. The Application CRD declares no `status` subresource**, so `status` is
writable by an ordinary `patch` on the main resource — no
`--subresource=status`, no separate verb:

    $ kubectl get crd applications.argoproj.io -o json \
        | jq '.spec.versions[] | {name, subresources}'
    { "name": "v1alpha1", "subresources": {} }

**2. The `atlas-pr-cleanup` Role in `argocd` already grants `patch` on
`applications`:**

    $ kubectl -n argocd get role atlas-pr-cleanup -o json | jq -r .rules
    [{"apiGroups":["argoproj.io"],"resources":["applications"],
      "verbs":["get","list","delete","patch"]}]

So the coordination here is a manifest to apply, not a permissions
negotiation. If the Role is ever narrowed, this CronJob needs `get`, `list`,
and `patch` on `applications.argoproj.io` in `argocd`.

## What it must not do

It touches only `status.operationState.phase`. The Application's own
finalizers — `resources-finalizer.argocd.argoproj.io` and
`post-delete-finalizer.argocd.argoproj.io[/cleanup]` — are never modified, so
PostDelete cleanup still reclaims per-env state. Stripping them would skip
cleanup entirely and leak the per-PR databases.

It must also leave `.operation` in place. Clearing `.operation` without
transitioning the phase is what made PR #1459 permanently unrecoverable: the
controller only processes an operation when `app.Operation != nil`, so after
that patch nothing could ever transition the phase or reap the hook
finalizer. The manifest logs and skips Applications already in that state.

## Merge ordering

1. This repo's PR (hook deadlines, workflow, runbook) — land any time.
2. cluster-infra: the CronJob — land any time. No dependency in either
   direction.
