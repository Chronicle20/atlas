# task-264 — Implementation Context

Companion to `plan.md`. Read this before dispatching the first implementer.

## What this task is

Per-PR environment teardown can deadlock forever when an Argo CD sync is in
flight at the moment the `atlas-pr-<N>` Application is marked for deletion. The
sync's hook Job holds `argocd.argoproj.io/hook-finalizer`; only the controller
removes it, and only when the owning operation completes or terminates; the
operation's phase is driven by that same Job. Nothing advances.
`resources-finalizer.argocd.argoproj.io` blocks on the undeletable Job, so no
other namespaced object is ever issued for deletion.

Observed on PR #1459, 2026-08-26: 89 objects stuck, controller logging
`N objects remaining for deletion` with N never decreasing, for 11 hours.

Six tasks, none of them Go. Two manifest pairs, one GitHub Actions workflow,
two new coordination artifacts, two runbook sections.

## Key files

| Path | Why it matters |
|---|---|
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | `spec:` at line 87, `backoffLimit: 3` at 88. Task 1 inserts `activeDeadlineSeconds: 900` between them. |
| `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` | Byte-identical mirror. Same line numbers. |
| `deploy/k8s/overlays/pr/postsync-pihole-add.yaml` | `spec:` at line 9, `template:` at 10. No `backoffLimit`. Task 2 inserts `activeDeadlineSeconds: 300`. |
| `deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml` | Byte-identical mirror. |
| `.github/workflows/pr-cleanup.yml` | Two jobs today: `delete-images` (lines 20-56) and `notify-argo` (line 78+, `needs: [delete-images]` at 81). The `# Branch deletion intentionally NOT done here.` block at 58-76 is the comment shape Task 3 copies. |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | **Read-only.** `do_drop_images` at line 465, registered at 592. This is why the workflow job is redundant. PRD FR-2.4 forbids changing it. |
| `dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml` | The example-manifest shape Task 4 copies: `# EXAMPLE — owned by the sibling cluster-infra repo` header, singleton warning, `namespace: argocd`, `serviceAccountName: atlas-pr-cleanup`. |
| `dev/cluster-infra-coordination/task-045-teardown.md` | The coordination-note structure Task 4 copies. |
| `docs/runbooks/ephemeral-pr-deployments.md` | §9.4 at 172 (`### Diagnose` 182, `### Recover` 221, `### Source-branch-missing scenario` 239), §9.5 at 245, §9.6 at 286-297, §9.7 at 299. **Task 5's insertion shifts everything below it** — Task 6 must re-locate its sections by `grep -n`, not by these numbers. |
| `tools/pr-sparse-mirror-guard.sh` | The `MIRRORS` array at lines 32-42 is the source of truth for pr ↔ pr-sparse pairing. Both files Tasks 1 and 2 touch are in it. |

## Decisions carried from the design, and what settles them

- **`activeDeadlineSeconds: 900`, on `spec` not `spec.template.spec`.** Derived
  from the full 30-day population — 120 × 6h Loki windows, 4663 lines, busiest
  window 153 (so not a capped sample). n=104 successful runs: p50 68s, p95
  162s, max 542s. 900 is 1.66× the max. `spec`-level bounds all `backoffLimit:
  3` retries in aggregate; pod-level would allow 4 × 900s.
- **The deadline is not inert against the case it targets.** The load-bearing
  worry was that a Job already under `deletionTimestamp` and pinned by a
  finalizer could never reach a terminal condition. Measured directly on this
  cluster (k3s v1.35.5+k3s1) in a throwaway namespace: it does. `FailureTarget`
  at t+60, `Failed` at t+91 — a 31s gap the finalizer introduces via the
  pod-finalizer sweep. Budget `deadline + ~30s`.
- **Pihole hooks bounded at 300s.** Widens the PRD's stated FR-1 scope; the
  user approved it during planning. The design surveyed six hook manifests and
  found **none** set a deadline. This closes two of the remaining five;
  `predelete-purge.yaml` (×2), `pr-cleanup/postdelete-cleanup.yaml`, and
  `base/atlas-minio-init.yaml` stay out and Task 2 has a step asserting they
  were not touched.
- **FR-3 is Option D — a CronJob in `argocd`, not a PreDelete hook in this
  repo.** Option C (PreDelete hook here) was seriously considered and fails on
  RBAC topology: the hook runs in `atlas-pr-<N>` under a namespace-local SA and
  must patch an Application in `argocd`, which needs a RoleBinding naming a
  subject in a namespace whose name is not known until the PR exists. The
  alternatives are binding the whole `system:serviceaccounts` group (privilege
  escalation for a teardown convenience) or cross-namespace resource creation
  that inverts the §9.13 ownership boundary. Option D needs no new grant at
  all.
- **FR-3 needs zero new RBAC.** Two live facts: the Application CRD declares no
  `status` subresource (`.spec.versions[].subresources` is `{}`), so `status`
  is writable by an ordinary patch; and the `atlas-pr-cleanup` Role in `argocd`
  already grants `get`/`list`/`delete`/`patch` on `applications`. Verified
  against `quay.io/argoproj/argocd:v3.4.2`.
- **The reconciler must leave `.operation` in place.** Clearing it without
  transitioning the phase is what made PR #1459 permanently unrecoverable — the
  controller only processes an operation when `app.Operation != nil`. The
  manifest logs and skips Applications already in that zombie state.
- **Merge ordering does not bind here.** §9.13's "cluster-infra lands first"
  rule exists because an atlas change depending on a missing cluster-infra
  object wedges. Nothing in this diff references the CronJob, so this repo's PR
  can land independently. The artifact says so in as many words.

## Dependencies between tasks

Tasks 1, 2, 3, and 4 are independent and can run in any order.

- **Task 6 depends on Task 3.** Task 6 deletes runbook §9.5's `gh secret set
  GHCR_TOKEN` rotation step, which is only obsolete once Task 3 has removed the
  workflow's last `GHCR_TOKEN` reference.
- **Tasks 5 and 6 edit the same file** at different sections. Task 5's
  insertion shifts §9.5–§9.16 down; run Task 5 first and have Task 6 re-locate
  by `grep -n '^## §9\.'`.
- **Tasks 5 and 6 consume literals from Tasks 1, 2, and 4** — the values `900`
  and `300`, and the CronJob name `atlas-pr-terminate-stuck-ops`. If any of
  those change, the runbook prose changes with them.

## Why there is no test file

This repo has no unit-test home for `deploy/k8s/` manifest field assertions.
`services/atlas-pr-bootstrap/test/*.bats` covers the shell scripts, and
`tools/verify.sh` gates that suite on a change to that service — which this
task does not make. The TDD cycle in Tasks 1–3 is therefore an explicit `yq` /
`grep` **assertion command** run before the edit (must fail) and after (must
pass), plus `tools/pr-sparse-mirror-guard.sh` and `actionlint`. Do not invent a
new bats file or guard script to satisfy the form of TDD; that would be
scaffolding no one maintains.

`yq` and `actionlint` are both present in this environment; `yamllint` is not.

## Task sizing

No task is oversized. Largest by file count is Task 4 at two new files; Tasks 1
and 2 touch two files each but the second is a byte-identical mirror of the
first, which is why they are not split. `plan-lint.sh`'s F4 warning should not
fire on any task.

## Live-cluster action already taken during planning

`atlas-pr-1459` was still wedged in the unrecoverable zombie state at plan
time. At the user's direction the FR-4.3 recipe was applied by hand:

```sh
kubectl -n atlas-pr-1459 patch job atlas-pr-bootstrap \
    --type=merge -p '{"metadata":{"finalizers":null}}'
```

Outcome, within about a minute: the namespace went from 89 objects to
`NotFound`, `resources-finalizer.argocd.argoproj.io` was reaped, the
Application retained
`post-delete-finalizer.argocd.argoproj.io[/cleanup]` and
`pre-delete-finalizer.argocd.argoproj.io/cleanup`, and `atlas-pr-cleanup-1459`
started in `argocd`. **PostDelete cleanup ran** — which is the entire reason
the new §9.4 subsection prefers this patch over the existing recipe's
"drop the Application's finalizers."

That execution is the evidence quoted in Task 5's runbook text. Every command
in the new subsection has now actually been run against a real wedged
environment, which is the bar design §5 sets for a runbook recipe.

## What still cannot be verified from the repo

The Argo half of FR-1 — that a `Failed/DeadlineExceeded` hook Job actually
drives the operation terminal and releases `hook-finalizer`. The Kubernetes
half is measured; this half needs a real PR env. The plan's final gate spells
out the post-merge exercise. Do not let a green `tools/verify.sh` be read as
having covered it.
