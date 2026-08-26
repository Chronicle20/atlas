# PR-Environment Teardown Sync Race — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26
---

## 1. Overview

Per-PR environment teardown can deadlock indefinitely. When an Argo CD sync
operation is in flight at the moment the `atlas-pr-<N>` Application is marked
for deletion, Argo neither completes nor terminates that operation. The
operation's Sync-hook Job keeps the runtime finalizer
`argocd.argoproj.io/hook-finalizer`, which only the controller removes as part
of completing or terminating the owning operation. The Job therefore never
deletes; `resources-finalizer.argocd.argoproj.io` blocks on it; and every other
namespaced resource — 63 Services, 9 ConfigMaps, the Ingress, ServiceAccounts,
Roles — is never even issued for deletion. The controller logs
`89 objects remaining for deletion` on every reconcile, forever, with zero
progress.

This was observed live on PR #1459 on 2026-08-26. An auto-sync had been running
for 24 minutes when the delete request landed, and it continued to create *new*
hook resources for four minutes afterward. Recovery required a manual
`kubectl patch` to strip the hook Job's finalizer. Runbook §9.4 does not cover
this failure mode: its recovery recipe drops the *Application's* finalizers,
which would have skipped the PostDelete cleanup hook entirely and leaked the
per-PR databases.

Three defects compound to produce the deadlock. The bootstrap Sync hook has no
wall-clock bound, so a sync can stay in flight long enough to overlap any
teardown. `.github/workflows/pr-cleanup.yml` eagerly deletes the per-PR GHCR
image tags on PR close, guaranteeing that any overlapping sync pulls tags that
no longer exist and hangs in `wait-ready` — and this job is a pure duplicate of
`cleanup.sh`'s existing `drop-images` PostDelete phase. And the cluster-infra
ApplicationSet deletes the Application without first terminating any in-flight
operation.

## 2. Goals

Primary goals:

- Make it impossible for a PR-environment teardown to block indefinitely on an
  in-flight sync operation.
- Bound the bootstrap Sync hook with a wall-clock deadline derived from observed
  production durations, so a failing sync fails fast instead of holding
  `operationState.phase = Running`.
- Stop deleting per-PR GHCR image tags before the environment that consumes them
  is gone.
- Terminate any in-flight operation before the ApplicationSet deletes the
  Application.
- Give operators a correct recovery recipe for the hook-finalizer deadlock — one
  that preserves PostDelete cleanup rather than skipping it.

Non-goals:

- Changing the PreDelete purge hook's failure tolerance. It is best-effort by
  design (`deploy/k8s/overlays/pr/predelete-purge.yaml:1-7`) with the
  `atlas-pr-sweep-orphans` CronJob as the documented backstop.
- Changing the orphan-sweep CronJobs (`atlas-pr-cleanup`, hourly;
  `atlas-pr-sweep-orphans`, every 6h).
- Reworking the multi-source ApplicationSet layout, the bot-branch lifecycle, or
  the sparse/isolated mode split.
- Introducing Prometheus instrumentation for the bootstrap Job. §9.6's metric is
  absent (see §9), but adding it is separate work.

## 3. User Stories

- As a PR author, I want my environment to tear down completely when I close the
  PR, so that I don't leave 90+ orphaned objects and a wedged Application behind.
- As an operator, I want a teardown that cannot hang forever, so that I am not
  paged to hand-patch finalizers on a Saturday.
- As an operator, when a teardown *does* wedge, I want a runbook recipe that
  unblocks it without skipping the PostDelete cleanup that reclaims databases.
- As a PR author, I want a slow-but-healthy bootstrap to still succeed, so that a
  new deadline does not flake environments that were previously fine.

## 4. Functional Requirements

### FR-1 — Bound the bootstrap Sync hook

**FR-1.1** `deploy/k8s/overlays/pr/sync-bootstrap.yaml` MUST set
`spec.activeDeadlineSeconds` on the `atlas-pr-bootstrap` Job.

**FR-1.2** `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` MUST set the same
value. Both overlays currently carry an identical `backoffLimit: 3` at line 88
and neither sets a deadline.

**FR-1.3** The value MUST be derived from observed bootstrap durations, not from
the declared retry budget. See §8 for the measured distribution and the proposed
value.

**FR-1.4** The deadline MUST apply to the Job as a whole (all `backoffLimit`
retries combined), so total hook lifetime is capped regardless of restarts.
`activeDeadlineSeconds` on `spec` has this semantic; a `spec.template.spec`
deadline does not.

**FR-1.5** When the deadline fires, the Job MUST reach a terminal `Failed` state
so Argo transitions the operation out of `Running` and reaps the hook resources.

### FR-2 — Stop eager GHCR tag deletion

**FR-2.1** The `delete-images` job MUST be removed from
`.github/workflows/pr-cleanup.yml`. It is functionally duplicated by
`cleanup.sh`'s `drop-images` phase
(`services/atlas-pr-bootstrap/scripts/cleanup.sh:465`, registered at line 592),
which runs PostDelete — after the namespace is pruned — and matches the same
`pr-<N>-*` tag prefix.

**FR-2.2** The `notify-argo` job's `needs: [delete-images]` edge MUST be removed
along with it.

**FR-2.3** Removal MUST be accompanied by a comment in `pr-cleanup.yml`
explaining why tag deletion belongs to the PostDelete phase, following the
precedent already set by the file's existing "Branch deletion intentionally NOT
done here" comment block, which documents the identical class of race for the
bot branch.

**FR-2.4** No change to `do_drop_images` behavior is required. It already skips
cleanly when `ATLAS_SERVICES` or `GHCR_TOKEN` are unset.

### FR-3 — Terminate in-flight operations before delete (cluster-infra)

**FR-3.1** The Application teardown path MUST terminate any in-flight sync
operation before the Application's finalizers begin draining.

**FR-3.2** The change lands in the cluster-infra repo (`argocd-atlas.yml`), which
owns the ApplicationSet and its `syncPolicy.automated` block (lines 159-162:
`selfHeal: true`, `prune: true`). Per runbook §9.13, cluster-infra changes land
BEFORE the consuming atlas PR.

**FR-3.3** A coordination artifact MUST be written under
`dev/cluster-infra-coordination/` specifying the required change, matching the
existing convention established by `atlas-pr-cleanup-env.example.yaml`.

**FR-3.4** Terminating the operation MUST NOT strip the Application's own
finalizers. `resources-finalizer.argocd.argoproj.io` and
`post-delete-finalizer.argocd.argoproj.io[/cleanup]` must remain so PostDelete
cleanup still reclaims per-env state.

### FR-4 — Runbook recovery recipe

**FR-4.1** Runbook §9.4 MUST gain a subsection covering the hook-finalizer
deadlock, distinct from the existing "drop the Application's finalizers" recipe.

**FR-4.2** The diagnostic signal MUST be documented: `operationState.phase =
Running` with `message: waiting for completion of hook batch/Job/<name>`,
combined with a repeating `N objects remaining for deletion` log line whose N
never decreases.

**FR-4.3** The recovery MUST be the narrow patch that removes only the hook Job's
finalizer, preserving the Application's own finalizers and therefore PostDelete
cleanup:

```sh
kubectl -n atlas-pr-<N> patch job <hook-job> \
    --type=merge -p '{"metadata":{"finalizers":null}}'
```

**FR-4.4** The runbook MUST state explicitly that `argocd app terminate-op` is
the correct first move while the operation still exists, and that clearing
`.operation` via `kubectl patch` is NOT equivalent — it removes the operation
spec without transitioning `operationState.phase`, leaving a zombie the
controller will never process, because the controller only processes an
operation when `app.Operation != nil`.

**FR-4.5** §9.6 MUST be corrected. Both of its queries are stale — see §9.

## 5. API Surface

No API changes. This task touches deployment manifests, a GitHub Actions
workflow, a cluster-infra coordination artifact, and runbook documentation.

## 6. Data Model

No data model changes. No entities, no migrations, no `tenant_id` scoping
implications.

## 7. Service Impact

No Go service is modified. Affected paths:

| Path | Change |
|---|---|
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | add `activeDeadlineSeconds` (FR-1.1) |
| `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` | add `activeDeadlineSeconds` (FR-1.2) |
| `.github/workflows/pr-cleanup.yml` | remove `delete-images` job + `needs` edge (FR-2) |
| `dev/cluster-infra-coordination/` | new terminate-op coordination artifact (FR-3.3) |
| `docs/runbooks/ephemeral-pr-deployments.md` | §9.4 recipe, §9.6 correction (FR-4) |
| cluster-infra `argocd-atlas.yml` | terminate-op before delete (FR-3.2, external repo) |

`services/atlas-pr-bootstrap/scripts/cleanup.sh` is deliberately unchanged —
`do_drop_images` already implements what FR-2 relies on.

## 8. Non-Functional Requirements

### Deriving the deadline

§9.6's PromQL cannot supply the value: `atlas_bootstrap_step_duration_ms_bucket`
does not exist in Prometheus. The instance carries 922 metric names and zero
match `atlas` or `bootstrap`; the query returns an empty vector at every
quantile.

The distribution was measured from Loki instead, over a 30-day window, selecting
`{container="bootstrap"}` and computing per-pod `max(ts) - min(ts)`. 150 runs
were captured; 104 carried a terminal `"atlas.step":"done"` success line:

| statistic | duration |
|---|---|
| min | 6s |
| median | 68s |
| p95 | 153s |
| max | 542s (PR 1337) |

**Caveat:** the query was capped at 5000 log lines, so this is a recent-window
sample rather than the full 30-day population. The design phase should re-run it
with pagination before pinning the final number.

**Proposed value: `activeDeadlineSeconds: 900`** (15 minutes) — 1.66× the
observed 542s maximum, leaving substantial headroom above p95 (153s) while
capping hook lifetime far below the >20-minute window that wedged PR #1459.

### Observability

- The deadline firing MUST be diagnosable from the Job's status
  (`DeadlineExceeded`) without cluster-admin access.
- No new metrics are required by this task, though §9 notes the absent
  instrumentation as a follow-up.

### Multi-tenancy

Not applicable — no tenant-scoped data is touched. The PreDelete purge that
handles per-tenant reclamation is explicitly out of scope.

### Safety

- No change may cause an Application to be deleted without its PostDelete
  cleanup running; that hook reclaims DBs, topics, consumer groups, Redis keys,
  GHCR tags, DNS records, and the bot branch.
- The sparse and isolated overlays must stay in parity.

## 9. Open Questions

1. **Exact deadline value.** 900s is proposed from a 5000-line-capped sample. Do
   we re-measure with pagination in the design phase, or accept the proposal?
2. **Terminate-op mechanism.** Argo CD offers no declarative
   "terminate-before-delete" field on an ApplicationSet template. The design
   phase must choose between a pre-delete automation in cluster-infra, an Argo CD
   version upgrade if newer releases terminate operations on delete natively, or
   accepting FR-1 as the sole mitigation and treating FR-3 as defense in depth.
   **This is the largest unresolved design question in the task.**
3. **Missing bootstrap instrumentation.** `atlas_bootstrap_step_duration_ms` is
   documented in §9.6 but emitted nowhere. Should restoring it be folded into
   this task or filed separately? Without it, future deadline tuning depends on
   Loki retention.
4. **§9.6 LogQL correction.** The documented selector
   `{atlas_env="a3f7", job=~"atlas-pr-bootstrap"}` does not match Loki's actual
   label set (`container`, `namespace`, `pod`, with
   `job="loki.source.kubernetes.pod_logs"`). Correcting it is in scope under
   FR-4.5; confirm the replacement selector against a live query before
   committing it.
5. **Backfill.** Are there other wedged `atlas-pr-*` Applications in the same
   state right now? Only #1459 was observed. A sweep would confirm.

## 10. Acceptance Criteria

- [ ] `deploy/k8s/overlays/pr/sync-bootstrap.yaml` sets `spec.activeDeadlineSeconds`.
- [ ] `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` sets the identical value.
- [ ] The deadline is on `spec`, not `spec.template.spec`, so it bounds all
      `backoffLimit` retries in aggregate.
- [ ] The chosen value is justified in the design doc against a re-measured
      duration distribution.
- [ ] The `delete-images` job is gone from `.github/workflows/pr-cleanup.yml`.
- [ ] `notify-argo` no longer declares `needs: [delete-images]` and the workflow
      still parses (actionlint / CI green).
- [ ] A comment in `pr-cleanup.yml` explains why tag deletion is PostDelete-only.
- [ ] A coordination artifact exists under `dev/cluster-infra-coordination/`
      specifying the terminate-op requirement.
- [ ] Runbook §9.4 documents the hook-finalizer deadlock: its signal, the narrow
      Job-finalizer patch, and why it is preferred over dropping the
      Application's finalizers.
- [ ] Runbook §9.4 states that clearing `.operation` by patch is not equivalent
      to `argocd app terminate-op`.
- [ ] Runbook §9.6's PromQL is marked absent-or-fixed and its LogQL selector
      matches the live Loki label set.
- [ ] `tools/verify.sh` exits 0.
- [ ] A teardown of a PR environment whose bootstrap hook is failing completes
      without manual intervention, verified end-to-end on a real PR.
