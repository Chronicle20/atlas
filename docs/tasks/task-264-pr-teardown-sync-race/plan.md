# PR-Environment Teardown Sync Race — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make per-PR environment teardown unable to deadlock on an in-flight Argo CD sync operation, and give operators a recovery recipe that preserves PostDelete cleanup.

**Architecture:** Four independent changes, none of which is Go code. (1) Bound every in-scope Argo hook Job with `spec.activeDeadlineSeconds` so a wedged hook reaches terminal `Failed` and releases `argocd.argoproj.io/hook-finalizer`. (2) Delete the `delete-images` GitHub Actions job that races the sync by deleting the GHCR tags the bootstrap hook is still pulling. (3) Ship a cluster-infra coordination artifact for a minutely reconciler CronJob that patches `status.operationState.phase` to `Terminating` on Applications wedged mid-delete. (4) Correct the runbook — a new §9.4 subsection for the hook-finalizer deadlock, plus §9.5/§9.6 fixes.

**Tech Stack:** Kubernetes manifests (kustomize overlays), GitHub Actions YAML, Markdown runbook. No Go, no TypeScript. Verification is `yq`, `actionlint`, `tools/pr-sparse-mirror-guard.sh`, and `tools/verify.sh`.

**Spec:** `docs/tasks/task-264-pr-teardown-sync-race/design.md` (PRD at `prd.md`, incident record at `incident-pr-1459.md`)

## Global Constraints

- **`activeDeadlineSeconds` goes on `spec`, never on `spec.template.spec`.** The Job-level field bounds aggregate lifetime across all `backoffLimit` retries; the pod-level field bounds each attempt independently and would permit 4 × deadline total. Design §2.1 "Placement"; PRD FR-1.4.
- **Bootstrap deadline value is exactly `900`.** Derived in design §2.1 from the full 30-day population (n=104 successful runs, p50 68s, p95 162s, max 542s). Do not round, retune, or substitute.
- **Pihole deadline value is exactly `300`.** Design §2.1 scope note; approved by the user during planning.
- **`deploy/k8s/overlays/pr/` and `deploy/k8s/overlays/pr-sparse/` copies must stay byte-identical.** `tools/pr-sparse-mirror-guard.sh` enumerates the mirrored files in its `MIRRORS` array — `sync-bootstrap.yaml` and `postsync-pihole-add.yaml` are both in it. Every edit to one side must be applied identically to the other, in the same commit.
- **Never strip an Application's own finalizers.** `resources-finalizer.argocd.argoproj.io` and `post-delete-finalizer.argocd.argoproj.io[/cleanup]` must survive any recovery this task documents, or PostDelete cleanup is skipped and the per-PR databases leak. PRD FR-3.4.
- **The FR-3 CronJob is a cluster-wide singleton and is NOT applied from this repo.** It goes in `dev/cluster-infra-coordination/` as an example artifact only. Placing it under `deploy/k8s/overlays/pr-cleanup/` would make CI render one copy per PR.
- **Repo-relative paths only in committed files.** No literal home or absolute paths (enforced under `docs/` by `tools/verify.sh`).
- **Preserve existing line endings.** Do not normalize CRLF→LF as a side effect.

## File Structure

| Path | Task | Responsibility |
|---|---|---|
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | 1 | add `activeDeadlineSeconds: 900` |
| `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` | 1 | byte-identical mirror |
| `deploy/k8s/overlays/pr/postsync-pihole-add.yaml` | 2 | add `activeDeadlineSeconds: 300` |
| `deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml` | 2 | byte-identical mirror |
| `.github/workflows/pr-cleanup.yml` | 3 | remove `delete-images` job + `needs` edge, add explanatory comment |
| `dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml` | 4 | new file — the reconciler CronJob manifest |
| `dev/cluster-infra-coordination/task-264-terminate-op.md` | 4 | new file — coordination note |
| `docs/runbooks/ephemeral-pr-deployments.md` | 5, 6 | §9.4 new subsection (Task 5); §9.5 + §9.6 corrections (Task 6) |

**No Go or TypeScript file is touched.** `services/atlas-pr-bootstrap/scripts/cleanup.sh` is deliberately unchanged (PRD FR-2.4): `do_drop_images` already implements what FR-2 relies on.

**A note on "write the failing test."** This repo has no unit-test home for `deploy/k8s/` manifest field assertions — the `services/atlas-pr-bootstrap/test/*.bats` suite covers the shell scripts, and `tools/verify.sh` only gates it on a change to that service. The TDD cycle in Tasks 1–3 is therefore expressed as an explicit **assertion command** run before and after the edit: it must fail first with the stated output, then pass. Do not invent a new bats file or guard script; the assertion commands plus `tools/pr-sparse-mirror-guard.sh` are the verification surface. This is recorded in `context.md`.

---

### Task 1: Bound the bootstrap Sync hook (FR-1.1, FR-1.2)

**Files:**
- Modify: `deploy/k8s/overlays/pr/sync-bootstrap.yaml:87-88` (`spec:` at line 87, `backoffLimit: 3` at line 88)
- Modify: `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml:87-88` (identical mirror; same line numbers)

### Files

- `deploy/k8s/overlays/pr/sync-bootstrap.yaml` — add `activeDeadlineSeconds: 900` under `spec:`
- `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` — apply the identical edit; the mirror guard diffs these two byte-for-byte
- `tools/pr-sparse-mirror-guard.sh` — read-only; the guard that enforces the mirror. Its `MIRRORS` array is the source of truth for which files pair up.

Patterns to copy: the file's existing block comment above `argocd.argoproj.io/hook-delete-policy` (`deploy/k8s/overlays/pr/sync-bootstrap.yaml:76-86`) — a `WHY`-shaped comment explaining a non-obvious Argo interaction, which is the shape the new comment should match.

No module root; these are YAML manifests, all commands run from the worktree root.

**Interfaces:**
- Consumes: nothing.
- Produces: the deadline that Task 5's runbook subsection forward-references ("the wedge should self-clear in ~15 minutes") and that Task 6's §9.6 retune recipe exists to re-derive. The value `900` appears in Tasks 5 and 6 as prose; it must match.

- [ ] **Step 1: Write the failing assertion**

Run both, from the worktree root:

```bash
yq -e '.spec.activeDeadlineSeconds == 900' deploy/k8s/overlays/pr/sync-bootstrap.yaml
yq -e '.spec.activeDeadlineSeconds == 900' deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml
```

- [ ] **Step 2: Run them to verify they fail**

Expected: both exit non-zero. `yq -e` on a `null` result exits 1 and prints `Error: no matches found` (or evaluates the comparison to `false` and exits 1). Either failure shape is correct — what matters is a non-zero exit before the edit.

Also confirm the pod-level field is absent and stays absent:

```bash
yq '.spec.template.spec.activeDeadlineSeconds' deploy/k8s/overlays/pr/sync-bootstrap.yaml
```

Expected now and after the edit: `null`.

- [ ] **Step 3: Make the edit**

In `deploy/k8s/overlays/pr/sync-bootstrap.yaml`, the current lines 87-89 are:

```yaml
spec:
  backoffLimit: 3
  template:
```

Replace with:

```yaml
spec:
  # Wall-clock bound on the whole hook, all `backoffLimit` retries combined.
  #
  # WHY this exists: an Argo Sync-hook Job carries the runtime finalizer
  # `argocd.argoproj.io/hook-finalizer`, which only the application
  # controller removes, and only when the owning operation completes or
  # terminates. The operation's phase is in turn driven by this Job. If the
  # Job never reaches a terminal condition, the operation stays `Running`,
  # `resources-finalizer.argocd.argoproj.io` blocks on the undeletable Job,
  # and the entire namespace teardown deadlocks — 89 objects stuck with the
  # controller logging `N objects remaining for deletion` forever. Observed
  # on PR #1459, 2026-08-26; recovery required a manual finalizer patch
  # (see `docs/runbooks/ephemeral-pr-deployments.md` §9.4).
  #
  # A deadline breaks that cycle: the Job reaches terminal `Failed` even
  # when it is already under `deletionTimestamp` and pinned by the
  # finalizer, so the operation transitions and the controller reaps the
  # hook. Measured directly on this cluster (k3s v1.35.5+k3s1), not assumed.
  #
  # WHY 900 and not the retry budget: derived from 30 days of observed
  # bootstrap durations (n=104 successful runs — p50 68s, p95 162s, max
  # 542s). 900s is 1.66x the slowest successful run ever recorded, leaving
  # headroom for image pull and scheduling, which precede the first log
  # line and so are not in those figures. Re-derive with the LogQL recipe
  # in runbook §9.6 before changing it.
  #
  # WHY on `spec` and not `spec.template.spec`: the Job-level field bounds
  # the aggregate lifetime across all `backoffLimit: 3` retries. The
  # pod-level field bounds each attempt independently and would permit
  # 4 x 900s of total hook lifetime, which defeats the purpose.
  activeDeadlineSeconds: 900
  backoffLimit: 3
  template:
```

Then apply the **byte-identical** edit to `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml`.

- [ ] **Step 4: Run the assertions to verify they pass**

```bash
yq -e '.spec.activeDeadlineSeconds == 900' deploy/k8s/overlays/pr/sync-bootstrap.yaml
yq -e '.spec.activeDeadlineSeconds == 900' deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml
yq -e '.spec.backoffLimit == 3' deploy/k8s/overlays/pr/sync-bootstrap.yaml
yq '.spec.template.spec.activeDeadlineSeconds' deploy/k8s/overlays/pr/sync-bootstrap.yaml
```

Expected: first three exit 0 printing `true`; the fourth prints `null`.

- [ ] **Step 5: Run the mirror guard**

```bash
tools/pr-sparse-mirror-guard.sh && echo "MIRROR OK"
```

Expected: exit 0, prints `MIRROR OK`. If it prints `pr-sparse-mirror-guard: deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml has drifted from deploy/k8s/overlays/pr/sync-bootstrap.yaml`, the two edits are not byte-identical — diff them and fix, do not "fix" the guard.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/overlays/pr/sync-bootstrap.yaml deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml
git commit -m "fix(pr-env): bound the bootstrap Sync hook with activeDeadlineSeconds: 900"
```

---

### Task 2: Bound the PostSync pihole hooks (design §2.1 scope note)

**Files:**
- Modify: `deploy/k8s/overlays/pr/postsync-pihole-add.yaml:9-10` (`spec:` at line 9, `template:` at line 10 — this Job sets no `backoffLimit`)
- Modify: `deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml:9-10` (identical mirror)

### Files

- `deploy/k8s/overlays/pr/postsync-pihole-add.yaml` — add `activeDeadlineSeconds: 300` under `spec:`
- `deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml` — apply the identical edit
- `tools/pr-sparse-mirror-guard.sh` — read-only; `postsync-pihole-add.yaml` is in its `MIRRORS` array

Patterns to copy: `deploy/k8s/overlays/pr/sync-bootstrap.yaml` as edited by Task 1 — same field, same placement under `spec:`, shorter comment.

**Scope note:** the design surveyed six hook manifests and found **none** set a deadline. This task closes two of the remaining five; the user approved this widening of the PRD's scope during planning. Deliberately **out** of scope: `deploy/k8s/overlays/pr/predelete-purge.yaml` and its pr-sparse mirror (`backoffLimit: 0`, best-effort by design per PRD non-goals), `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` (runs in `argocd`, covered by `ttlSecondsAfterFinished: 86400` plus runbook §9.4), and `deploy/k8s/base/atlas-minio-init.yaml` (in `base/`, so touching it affects `atlas-main`). Do not edit those four.

**Interfaces:**
- Consumes: the comment shape established by Task 1.
- Produces: nothing consumed by a later task.

- [ ] **Step 1: Write the failing assertion**

```bash
yq -e '.spec.activeDeadlineSeconds == 300' deploy/k8s/overlays/pr/postsync-pihole-add.yaml
yq -e '.spec.activeDeadlineSeconds == 300' deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml
```

- [ ] **Step 2: Run them to verify they fail**

Expected: both exit non-zero — the field does not exist yet.

- [ ] **Step 3: Make the edit**

Current lines 9-10 of `deploy/k8s/overlays/pr/postsync-pihole-add.yaml`:

```yaml
spec:
  template:
```

Replace with:

```yaml
spec:
  # Wall-clock bound on the hook. Same deadlock class as
  # `sync-bootstrap.yaml` — an Argo hook Job that never reaches a terminal
  # condition holds `argocd.argoproj.io/hook-finalizer`, which wedges the
  # owning operation and blocks the whole namespace teardown. This hook is
  # two authenticated HTTP calls to a Pi-hole admin API, so 300s is
  # generous; it exists to bound the failure, not to bound normal work.
  activeDeadlineSeconds: 300
  template:
```

Then apply the **byte-identical** edit to `deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml`.

- [ ] **Step 4: Run the assertions to verify they pass**

```bash
yq -e '.spec.activeDeadlineSeconds == 300' deploy/k8s/overlays/pr/postsync-pihole-add.yaml
yq -e '.spec.activeDeadlineSeconds == 300' deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml
yq '.spec.template.spec.activeDeadlineSeconds' deploy/k8s/overlays/pr/postsync-pihole-add.yaml
```

Expected: first two exit 0 printing `true`; the third prints `null`.

- [ ] **Step 5: Confirm the out-of-scope hooks were not touched**

```bash
git status --porcelain deploy/k8s/
```

Expected: exactly two modified paths, both `postsync-pihole-add.yaml` (Task 1's files are already committed). If `predelete-purge.yaml`, `postdelete-cleanup.yaml`, or `base/atlas-minio-init.yaml` appear, revert them.

- [ ] **Step 6: Run the mirror guard**

```bash
tools/pr-sparse-mirror-guard.sh && echo "MIRROR OK"
```

Expected: exit 0.

- [ ] **Step 7: Commit**

```bash
git add deploy/k8s/overlays/pr/postsync-pihole-add.yaml deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml
git commit -m "fix(pr-env): bound the PostSync pihole hook with activeDeadlineSeconds: 300"
```

---

### Task 3: Remove the eager GHCR tag deletion (FR-2.1, FR-2.2, FR-2.3)

**Files:**
- Modify: `.github/workflows/pr-cleanup.yml` — delete the `delete-images` job (currently lines 20-56, from `  delete-images:` through the closing `          done`), replace it with an explanatory comment, and drop `needs: [delete-images]` from `notify-argo`

### Files

- `.github/workflows/pr-cleanup.yml` — the only file this task edits
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` — read-only; `do_drop_images` at line 465, registered in the phase list at line 592. This is the PostDelete implementation that makes the workflow job redundant. **Do not edit it** (PRD FR-2.4).
- `.github/config/services.json` — read-only; the service list both implementations derive from

Patterns to copy: the file's existing `# Branch deletion intentionally NOT done here.` comment block (`.github/workflows/pr-cleanup.yml:58-76`) — same failure class (a workflow deleting an artifact the Argo teardown still needs), same explanation shape, same cross-reference to the runbook. The new comment sits immediately above it so the file reads as one contiguous "here is everything we deliberately do *not* do on PR close" section.

**Interfaces:**
- Consumes: nothing.
- Produces: the fact that `.github/workflows/pr-cleanup.yml` no longer references `secrets.GHCR_TOKEN`, which Task 6 acts on by deleting the now-obsolete runbook §9.5 rotation step 3.

- [ ] **Step 1: Write the failing assertions**

```bash
# The delete-images job must be gone.
yq -e '.jobs | has("delete-images") | not' .github/workflows/pr-cleanup.yml
# notify-argo must have no `needs`.
yq -e '.jobs.notify-argo | has("needs") | not' .github/workflows/pr-cleanup.yml
# The workflow must no longer reference the GHCR_TOKEN repo secret.
! grep -q 'GHCR_TOKEN' .github/workflows/pr-cleanup.yml
```

- [ ] **Step 2: Run them to verify they fail**

Expected: all three exit non-zero today — `delete-images` exists (`has("delete-images")` is `true`, so the negation is `false` and `yq -e` exits 1), `notify-argo` has `needs: [delete-images]`, and `GHCR_TOKEN` appears at line 37.

Record the baseline so the diff is checkable:

```bash
yq -o=json '.jobs | keys' .github/workflows/pr-cleanup.yml
```

Expected before: `["delete-images","notify-argo"]`. Expected after: `["notify-argo"]`.

- [ ] **Step 3: Delete the job and add the comment**

Remove the entire `delete-images:` job — everything from `  delete-images:` down to and including the final `          done` of the `Delete tags` step, i.e. current lines 20 through 56. In its place, immediately above the existing `  # Branch deletion intentionally NOT done here.` block, insert:

```yaml
  # Image-tag deletion intentionally NOT done here.
  #
  # Deleting the per-PR `pr-<N>-*` ghcr tags on PR close races the Argo CD
  # teardown. Argo deletes the Application immediately on close, but an
  # auto-sync already in flight keeps running — on PR #1459 it ran for four
  # more minutes and kept creating hook resources. That sync's
  # `atlas-pr-bootstrap` hook then pulls the tags this job just deleted and
  # fails with `Failed to pull image
  # "ghcr.io/chronicle20/atlas-data/atlas-data:pr-1459-99281705": ... not
  # found`, hangs in `wait-ready`, and never reaches a terminal condition —
  # which holds `argocd.argoproj.io/hook-finalizer` and deadlocks the whole
  # namespace teardown. Reproduced 2026-08-26 on PR #1459.
  #
  # This job was also a pure duplicate. `cleanup.sh::do_drop_images`
  # (`services/atlas-pr-bootstrap/scripts/cleanup.sh:465`, registered in
  # the phase list at line 592) matches the same `pr-<N>-*` prefix over the
  # same `.github/config/services.json`-derived service list, using the
  # in-cluster `atlas-pr-cleanup-gh-token` PAT. It runs PostDelete — after
  # the namespace is pruned — which is the only point at which no consumer
  # of those tags can still exist.
  #
  # Tag lifetime is now coupled to PostDelete Job lifetime — see
  # `docs/runbooks/ephemeral-pr-deployments.md` §9.4.

```

Then delete the `    needs: [delete-images]` line from the `notify-argo` job. With `delete-images` gone, `notify-argo` has no remaining dependency and becomes the workflow's only job.

- [ ] **Step 4: Run the assertions to verify they pass**

```bash
yq -e '.jobs | has("delete-images") | not' .github/workflows/pr-cleanup.yml
yq -e '.jobs.notify-argo | has("needs") | not' .github/workflows/pr-cleanup.yml
! grep -q 'GHCR_TOKEN' .github/workflows/pr-cleanup.yml
yq -o=json '.jobs | keys' .github/workflows/pr-cleanup.yml
```

Expected: first three exit 0; the fourth prints `["notify-argo"]`.

- [ ] **Step 5: Lint the workflow**

```bash
actionlint .github/workflows/pr-cleanup.yml && echo "ACTIONLINT OK"
```

Expected: exit 0, no output before `ACTIONLINT OK`. `actionlint` is present in this environment; if it is missing, install it rather than skipping — a workflow with a dangling `needs` edge is exactly what it catches.

- [ ] **Step 6: Confirm cleanup.sh is untouched**

```bash
git status --porcelain services/atlas-pr-bootstrap/
```

Expected: empty. `do_drop_images` must not change (PRD FR-2.4).

- [ ] **Step 7: Commit**

```bash
git add .github/workflows/pr-cleanup.yml
git commit -m "fix(pr-env): stop deleting per-PR ghcr tags on PR close"
```

---

### Task 4: Cluster-infra coordination artifact for terminate-op (FR-3.3)

**Files:**
- Create: `dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml` (new file)
- Create: `dev/cluster-infra-coordination/task-264-terminate-op.md` (new file)

### Files

- `dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml` — **new file**; the CronJob manifest cluster-infra applies
- `dev/cluster-infra-coordination/task-264-terminate-op.md` — **new file**; the coordination note
- `dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml` — read-only; the manifest whose header, singleton warning, and SA reuse this file copies
- `dev/cluster-infra-coordination/task-045-teardown.md` — read-only; the note whose structure (what lives where / RBAC / singleton warning / merge ordering) this file copies

Patterns to copy: `dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml:1-27` — the `# EXAMPLE — owned by the sibling cluster-infra repo, NOT applied from this repo.` header, the singleton warning, `namespace: argocd`, `concurrencyPolicy: Forbid`, `serviceAccountName: atlas-pr-cleanup`, and `backoffLimit: 0`. For the container image and shell, copy the **live** `atlas-pr-cleanup` CronJob's shape: `alpine/k8s:1.30.0` with `command: ["/bin/bash", "-c"]` (confirmed live during design; it is the image the sibling minutely/hourly reconcilers already use, and it carries both `kubectl` and `jq`).

**Nothing in this repo's build or deploy references either new file.** They are documentation-class artifacts under `dev/`.

**Interfaces:**
- Consumes: nothing.
- Produces: the CronJob name `atlas-pr-terminate-stuck-ops`, which Task 5's runbook forward-reference names.

- [ ] **Step 1: Write the failing assertion**

```bash
test -f dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
test -f dev/cluster-infra-coordination/task-264-terminate-op.md
```

- [ ] **Step 2: Run it to verify it fails**

Expected: both exit 1 — neither file exists.

- [ ] **Step 3: Write the CronJob manifest**

Create `dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml` with exactly this content:

```yaml
# EXAMPLE — owned by the sibling cluster-infra repo, NOT applied from this repo.
# A cluster-wide SINGLETON CronJob (do not place under overlays/pr-cleanup,
# which CI renders once per PR -> N copies). Reproduces `argocd app
# terminate-op` for Applications that wedged mid-delete. See task-264
# design.md §2.3 (Option D) and task-264-terminate-op.md in this folder.
apiVersion: batch/v1
kind: CronJob
metadata:
  name: atlas-pr-terminate-stuck-ops
  namespace: argocd
spec:
  # Every minute. The whole point is bounding wedge duration: folding this
  # into atlas-pr-cleanup's hourly run would bound it at 60 minutes, which
  # is worse than the 15-minute bound the bootstrap hook's
  # activeDeadlineSeconds already gives. Minutely makes this the tighter of
  # the two bounds, which is what defense in depth should look like. The
  # predicate is false for every healthy Application, so a run costs one
  # list and zero patches.
  schedule: "* * * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 0
      activeDeadlineSeconds: 120
      template:
        spec:
          restartPolicy: Never
          # Reuses the existing PostDelete cleanup SA. NO NEW RBAC IS
          # REQUIRED — see task-264-terminate-op.md for the live Role dump.
          serviceAccountName: atlas-pr-cleanup
          containers:
            - name: terminate-stuck-ops
              image: alpine/k8s:1.30.0
              command: ["/bin/bash", "-c"]
              args:
                - |
                  set -euo pipefail

                  # Select Applications that are (a) marked for deletion,
                  # (b) still reporting an in-flight operation, and
                  # (c) still carrying an .operation spec.
                  #
                  # (c) is load-bearing. The Argo application controller
                  # only processes an operation when `app.Operation != nil`.
                  # An Application whose .operation has been cleared -- e.g.
                  # by `kubectl patch -p '{"operation":null}'`, which an
                  # operator attempted on PR #1459 -- is a zombie: its phase
                  # can never transition again, and patching it to
                  # "Terminating" produces another zombie rather than a
                  # recovery. Log and skip those; they belong to the
                  # Job-finalizer recipe in the atlas repo's
                  # docs/runbooks/ephemeral-pr-deployments.md §9.4.
                  apps=$(kubectl -n argocd get applications.argoproj.io -o json)

                  echo "$apps" | jq -r '
                    .items[]
                    | select(.metadata.name | startswith("atlas-pr-"))
                    | select(.metadata.deletionTimestamp != null)
                    | select(.status.operationState.phase == "Running")
                    | select(.operation == null)
                    | "SKIP \(.metadata.name): deletionTimestamp set and phase=Running, but .operation is already cleared (zombie; needs the hook-Job finalizer patch, runbook §9.4)"
                  '

                  targets=$(echo "$apps" | jq -r '
                    .items[]
                    | select(.metadata.name | startswith("atlas-pr-"))
                    | select(.metadata.deletionTimestamp != null)
                    | select(.status.operationState.phase == "Running")
                    | select(.operation != null)
                    | .metadata.name
                  ')

                  if [ -z "$targets" ]; then
                    echo "no stuck operations"
                    exit 0
                  fi

                  for app in $targets; do
                    echo "TERMINATE $app: patching status.operationState.phase -> Terminating"
                    # NOTE: no --subresource=status. The Application CRD
                    # declares no status subresource (verified live:
                    # `.spec.versions[].subresources` is `{}`), so status is
                    # writable by an ordinary patch on the main resource,
                    # which the atlas-pr-cleanup Role already grants.
                    #
                    # .operation is deliberately left in place. That is the
                    # difference between this and the operator error on PR
                    # #1459: the controller, still processing because
                    # Operation != nil, runs its termination path -- it
                    # terminates the sync, reaps the hook resources and
                    # their finalizers, sets the phase terminal, and clears
                    # .operation itself.
                    kubectl -n argocd patch applications.argoproj.io "$app" \
                      --type=merge \
                      -p '{"status":{"operationState":{"phase":"Terminating"}}}'
                  done
                  # The Application's own finalizers are never touched, so
                  # post-delete-finalizer.argocd.argoproj.io[/cleanup] still
                  # runs and PostDelete cleanup still reclaims DBs, topics,
                  # groups, Redis keys, ghcr tags, DNS, and the bot branch.
```

- [ ] **Step 4: Write the coordination note**

Create `dev/cluster-infra-coordination/task-264-terminate-op.md` with exactly this content:

```markdown
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
```

- [ ] **Step 5: Verify the manifest parses and asserts the right shape**

```bash
yq -e '.kind == "CronJob"' dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
yq -e '.metadata.namespace == "argocd"' dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
yq -e '.spec.schedule == "* * * * *"' dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
yq -e '.spec.concurrencyPolicy == "Forbid"' dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
yq -e '.spec.jobTemplate.spec.template.spec.serviceAccountName == "atlas-pr-cleanup"' \
  dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
```

Expected: all five exit 0 printing `true`.

- [ ] **Step 6: Verify the embedded script's shell syntax**

The reconciler logic is a YAML block scalar, so `bash -n` cannot see it directly. Extract and check it:

```bash
yq -r '.spec.jobTemplate.spec.template.spec.containers[0].args[0]' \
  dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml | bash -n \
  && echo "SHELL SYNTAX OK"
```

Expected: exit 0, prints `SHELL SYNTAX OK`.

- [ ] **Step 7: Dry-run the selection predicate against the live cluster**

The list-and-filter half is read-only. Run it as-is (this is the same jq the manifest embeds, minus the patch):

```bash
kubectl -n argocd get applications.argoproj.io -o json | jq -r '
  .items[]
  | select(.metadata.name | startswith("atlas-pr-"))
  | select(.metadata.deletionTimestamp != null)
  | select(.status.operationState.phase == "Running")
  | "\(.metadata.name) hasOp=\(.operation != null)"
'
```

Expected: it selects only Applications that are genuinely mid-delete with a `Running` operation, and no healthy one. If the cluster currently has none, empty output is the correct result and the predicate is confirmed non-destructive. **Do not run the patch half.** If `kubectl` is unavailable in the executor's environment, note it and move on — this step is confirmatory, not blocking.

- [ ] **Step 8: Confirm no absolute paths leaked in**

```bash
grep -nE '/home/|/Users/' dev/cluster-infra-coordination/task-264-terminate-op.md \
  dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml
```

Expected: no matches (grep exits 1).

- [ ] **Step 9: Commit**

```bash
git add dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml \
        dev/cluster-infra-coordination/task-264-terminate-op.md
git commit -m "docs(cluster-infra): coordination artifact for terminate-stuck-ops CronJob"
```

---

### Task 5: Runbook §9.4 — the hook-finalizer deadlock recipe (FR-4.1 – FR-4.4)

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md` — insert a new `###` subsection under §9.4, **after** the existing `### Recover` block (which ends at line 238, the `gh api --method DELETE` command) and **before** `### Source-branch-missing scenario` (line 239)

### Files

- `docs/runbooks/ephemeral-pr-deployments.md` — the only file this task edits. §9.4 spans lines 172-243; `### Diagnose` at 182, `### Recover` at 221, `### Source-branch-missing scenario` at 239.
- `docs/tasks/task-264-pr-teardown-sync-race/incident-pr-1459.md` — read-only; the incident record the quoted signals come from
- `docs/tasks/task-264-pr-teardown-sync-race/design.md` — read-only; §2.4 specifies this subsection's content

Patterns to copy: §9.4's existing `### Recover` block (`docs/runbooks/ephemeral-pr-deployments.md:221-238`) — numbered `sh` steps with inline `#` commentary. The new subsection uses the same shape but must open by contrasting itself with that block, because that block is *wrong for this failure*.

**Critical:** the existing §9.4 `Recover` step 2 patches the **Application's** finalizers to `[]`. Doing that for this failure strips `post-delete-finalizer.argocd.argoproj.io/cleanup` and skips PostDelete cleanup entirely, leaking the per-PR databases. The new subsection must lead with that contrast rather than bury it.

**Interfaces:**
- Consumes: the value `900` from Task 1 and the CronJob name `atlas-pr-terminate-stuck-ops` from Task 4. Both appear in the forward-reference paragraph and must match.
- Produces: nothing consumed by a later task. Task 6 edits a different part of the same file; if both are in flight, re-read the file before editing.

- [ ] **Step 1: Write the failing assertion**

```bash
RB=docs/runbooks/ephemeral-pr-deployments.md
for pat in 'hook batch/Job/' 'hook-finalizer' 'terminate-op'; do
  grep -q "$pat" "$RB" && echo "PRESENT: $pat" || echo "ABSENT:  $pat"
done
```

- [ ] **Step 2: Run it to verify it fails**

Expected: all three print `ABSENT:` — the runbook does not currently mention the hook-finalizer deadlock at all. Confirm the insertion point is still where the plan says:

```bash
sed -n '236,241p' docs/runbooks/ephemeral-pr-deployments.md
```

Expected: the tail of the `gh api --method DELETE` step, a blank line, then `### Source-branch-missing scenario`. If it does not match, find the boundary with `grep -n '^### ' docs/runbooks/ephemeral-pr-deployments.md` and insert between `### Recover` and `### Source-branch-missing scenario` regardless of exact line numbers.

- [ ] **Step 3: Insert the new subsection**

Insert this verbatim between the end of `### Recover` and the `### Source-branch-missing scenario` heading:

````markdown
### Teardown wedged on a hook Job's finalizer (repeating, non-decreasing object count)

**Do not use the `Recover` recipe above for this failure.** Its step 2 drops
the *Application's* finalizers, which strips
`post-delete-finalizer.argocd.argoproj.io/cleanup` and skips PostDelete
cleanup entirely — the per-PR databases, topics, consumer groups, Redis keys,
ghcr tags, DNS entry, and bot branch all leak. The recipe below removes only
the **hook Job's** finalizer and leaves the Application's own finalizers
intact, so PostDelete still runs.

#### Signal

Two signs together, both of which an operator can string-match:

```sh
kubectl -n argocd get application atlas-pr-<N> -o json \
  | jq -c '{fin:.metadata.finalizers, del:.metadata.deletionTimestamp,
            hasOp:(.operation!=null), phase:.status.operationState.phase,
            msg:.status.operationState.message}'
```

prints an operation stuck in `Running` whose message names a hook batch:

```json
{"fin":["resources-finalizer.argocd.argoproj.io",
        "post-delete-finalizer.argocd.argoproj.io/cleanup",
        "pre-delete-finalizer.argocd.argoproj.io/cleanup",
        "post-delete-finalizer.argocd.argoproj.io"],
 "del":"2026-08-26T12:12:50Z","hasOp":false,"phase":"Running",
 "msg":"waiting for completion of hook batch/Job/atlas-pr-bootstrap"}
```

and the application controller logs `N objects remaining for deletion` on
every reconcile with **N never decreasing**:

```sh
kubectl -n argocd logs statefulset/argocd-application-controller --tail=200 \
  | grep 'objects remaining for deletion'
```

The mechanism: an Argo hook Job carries the runtime finalizer
`argocd.argoproj.io/hook-finalizer`, which only the controller removes, and
only as part of completing or terminating the owning operation. The
operation's phase is in turn driven by that Job. Neither can advance, so
`resources-finalizer.argocd.argoproj.io` blocks on the undeletable Job and no
other namespaced object is ever issued for deletion. On PR #1459 that was 89
objects — 63 Services, 9 ConfigMaps, the Ingress, ServiceAccounts, Roles —
stuck for 11 hours.

#### First move, while `.operation` still exists

If the `hasOp` field above is `true`:

```sh
argocd app terminate-op atlas-pr-<N>
```

This sets `status.operationState.phase` to `Terminating` while leaving
`.operation` in place. The controller, still processing the operation, runs
its termination path: it terminates the sync, reaps the hook resources and
their finalizers, sets the phase terminal, and clears `.operation` itself.

#### The trap — do not clear `.operation`

```sh
# WRONG. Do not run this.
kubectl -n argocd patch application atlas-pr-<N> --type=merge -p '{"operation":null}'
```

This is **not** equivalent to `terminate-op` and makes the wedge
**permanent**. It removes the operation spec without transitioning
`status.operationState.phase`, which stays `Running` — and the controller only
processes an operation when `app.Operation != nil`. After this patch nothing
can ever transition the phase or reap the hook finalizer. This was attempted
on PR #1459 on 2026-08-26; the Application then sat with `hasOp: false` and
`opPhase: "Running"` for 11 hours, unrecoverable by any operation-level
action.

#### Recovery once `.operation` is gone

Find the wedging Job by its retained finalizer rather than by name — the hook
need not be bootstrap:

```sh
kubectl -n atlas-pr-<N> get jobs -o custom-columns=\
'NAME:.metadata.name,DEL:.metadata.deletionTimestamp,FIN:.metadata.finalizers'
```

On PR #1459 this printed:

```
NAME                       DEL                    FIN
atlas-minio-init           <none>                 <none>
atlas-pr-bootstrap         2026-08-26T12:26:27Z   [argocd.argoproj.io/hook-finalizer]
atlas-pr-predelete-purge   <none>                 <none>
```

The wedging Job is the one with both a `DEL` timestamp and a retained `FIN`.
Drop only its finalizer:

```sh
kubectl -n atlas-pr-<N> patch job <hook-job> \
    --type=merge -p '{"metadata":{"finalizers":null}}'
```

Confirm the teardown resumed — the namespace should go and PostDelete should
fire:

```sh
kubectl get ns atlas-pr-<N>                       # expect: NotFound
kubectl -n argocd get jobs -l app=atlas-pr-cleanup,atlas.pr-number=<N>
```

Executed live on PR #1459 on 2026-08-26. Within about a minute the namespace
went from 89 objects to `NotFound`, `resources-finalizer.argocd.argoproj.io`
was reaped, the Application retained its
`post-delete-finalizer.argocd.argoproj.io[/cleanup]` finalizers, and
`atlas-pr-cleanup-1459` started in `argocd` — i.e. PostDelete cleanup ran,
which is the whole reason to prefer this patch over dropping the
Application's finalizers.

#### This should now be an escalation, not the routine path

Two bounds were added in task-264 and both should fire before an operator
does:

- `deploy/k8s/overlays/pr/sync-bootstrap.yaml` sets
  `spec.activeDeadlineSeconds: 900`, so a wedged bootstrap hook reaches
  terminal `Failed` on its own. Budget `deadline + ~30s`: the Job's
  finalizer delays the pod-finalizer sweep that promotes `FailureTarget` to
  `Failed` (measured at 31s). The wedge should self-clear in **~15.5
  minutes**. `postsync-pihole-add.yaml` is bounded at 300s the same way.
- A cluster-infra CronJob, `atlas-pr-terminate-stuck-ops` in `argocd`,
  reproduces `terminate-op` every minute for Applications that are
  mid-delete with a `Running` operation and a non-nil `.operation` — see
  `dev/cluster-infra-coordination/task-264-terminate-op.md`. It logs and
  skips the `.operation == null` zombie state, which is exactly the case the
  manual recipe above exists for.

If you are hand-patching a finalizer in an environment where both are live,
something else is wrong — capture the Application JSON and the controller log
before you patch.

````

- [ ] **Step 4: Run the assertions to verify they pass**

```bash
RB=docs/runbooks/ephemeral-pr-deployments.md
rc=0
for pat in 'hook batch/Job/' 'hook-finalizer' 'terminate-op' \
           'atlas-pr-terminate-stuck-ops' 'activeDeadlineSeconds: 900'; do
  grep -q "$pat" "$RB" || { echo "MISSING: $pat"; rc=1; }
done
[ "$rc" -eq 0 ] && echo "ALL PRESENT"
```

Expected: prints `ALL PRESENT` and nothing else.

- [ ] **Step 5: Verify placement and that no existing content was clobbered**

```bash
grep -n '^### ' docs/runbooks/ephemeral-pr-deployments.md | sed -n '1,20p'
```

Expected: within §9.4, the order is `### Diagnose`, `### Recover`, `### Teardown wedged on a hook Job's finalizer …`, `### Source-branch-missing scenario`. The existing three headings must all still be present.

```bash
git diff --stat docs/runbooks/ephemeral-pr-deployments.md
```

Expected: insertions only, zero deletions.

- [ ] **Step 6: Confirm no absolute paths leaked in**

```bash
grep -nE '/home/|/Users/' docs/runbooks/ephemeral-pr-deployments.md
```

Expected: no matches (grep exits 1). This is enforced under `docs/` by `tools/verify.sh`.

- [ ] **Step 7: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbook): add the hook-finalizer teardown deadlock recipe to §9.4"
```

---

### Task 6: Runbook §9.5 and §9.6 corrections (FR-4.5)

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md` — §9.5 rotation procedure (remove the now-obsolete step 3, currently near line 279) and §9.6 in full (currently lines 286-297)

### Files

- `docs/runbooks/ephemeral-pr-deployments.md` — the only file this task edits. §9.5 starts at line 245, §9.6 at line 286, §9.7 at line 299. **Line numbers shift by Task 5's insertion** — locate the sections with `grep -n '^## §9\.[567]' docs/runbooks/ephemeral-pr-deployments.md` rather than trusting these.
- `.github/workflows/pr-cleanup.yml` — read-only; Task 3 removed its last `GHCR_TOKEN` reference, which is what makes §9.5 step 3 obsolete
- `docs/tasks/task-264-pr-teardown-sync-race/design.md` — read-only; §2.4 "§9.6" specifies the replacement queries verbatim

Patterns to copy: none — this is prose and query correction in place.

**Interfaces:**
- Consumes: Task 3's removal of `GHCR_TOKEN` from `.github/workflows/pr-cleanup.yml`, and Task 1's value `900`. **Task 3 must land before this task**, or §9.5 step 3 would be deleted while a consumer still exists.
- Produces: nothing.

- [ ] **Step 1: Write the failing assertions**

```bash
RB=docs/runbooks/ephemeral-pr-deployments.md
# Must be GONE after the edit: the obsolete rotation step, the stale selector.
for pat in 'gh secret set GHCR_TOKEN' 'atlas_env="a3f7"'; do
  grep -q "$pat" "$RB" && echo "STILL PRESENT: $pat" || echo "GONE:    $pat"
done
# Must be ADDED by the edit: the corrected selector, the retention caveat.
for pat in 'container="bootstrap"' '14 days'; do
  grep -q "$pat" "$RB" && echo "PRESENT: $pat" || echo "ABSENT:  $pat"
done
```

- [ ] **Step 2: Run them to verify they fail**

Expected today: `STILL PRESENT` for both of the first pair, `ABSENT` for both of the second.

- [ ] **Step 3: Correct §9.5's rotation procedure**

In §9.5, under `atlas-pr-cleanup-gh-token`, the rotation block currently reads:

```sh
# 1. Mint a new PAT with the scope set above.
# 2. Update the cluster secret.
kubectl -n argocd edit secret atlas-pr-cleanup-gh-token   # set key GHCR_TOKEN
# 3. Update the repo secret used by .github/workflows/pr-cleanup.yml's image-delete step.
gh secret set GHCR_TOKEN --repo Chronicle20/atlas --body "$NEW_PAT"
```

Replace it with:

```sh
# 1. Mint a new PAT with the scope set above.
# 2. Update the cluster secret. This is the only consumer.
kubectl -n argocd edit secret atlas-pr-cleanup-gh-token   # set key GHCR_TOKEN
```

Immediately after that block, add:

```markdown
  **There is no repo-secret half to this rotation.** Until task-264,
  `.github/workflows/pr-cleanup.yml` carried a `delete-images` job that read a
  `GHCR_TOKEN` repository secret, and this procedure had a third step to
  rotate it. That job was removed — it raced the Argo teardown by deleting
  ghcr tags an in-flight sync was still pulling, and it duplicated
  `cleanup.sh::do_drop_images`, which does the same work PostDelete. Nothing
  in `.github/` reads `GHCR_TOKEN` now. If a `GHCR_TOKEN` repository secret
  still exists on `Chronicle20/atlas`, it is unreferenced and can be deleted.
```

Leave the sentence about the nightly smoke test (§4.5 / `pr-env-smoke.yml`) in place — it still catches a missed cluster-secret rotation.

- [ ] **Step 4: Replace §9.6 in full**

Replace the whole of §9.6 — from the `## §9.6 Bootstrap-duration metrics` heading down to (but not including) `## §9.7 Hash-collision resolution` — with:

````markdown
## §9.6 Bootstrap-duration metrics

### PromQL — this metric was never emitted

```promql
# DEAD QUERY. atlas_bootstrap_step_duration_ms_bucket does not exist in
# Prometheus; the bootstrap Job carries no instrumentation. Kept as a record
# of intent, not as a runnable query. Use the LogQL method below instead.
histogram_quantile(0.95,
  rate(atlas_bootstrap_step_duration_ms_bucket{atlas_env!="main"}[1h]))
```

The bootstrap Job was never instrumented. Verified 2026-08-26: the Prometheus
instance carries 922 metric names and **zero** matching `atlas` or
`bootstrap`. The query is left here rather than deleted because it documents
an intent — someone meant this metric to exist — and deleting it loses that.
Until it is emitted, use the Loki method below.

### LogQL — stepwise breakdown

The previously documented selector `{atlas_env="a3f7", job=~"atlas-pr-bootstrap"}`
matches nothing and always did. Loki's live label set is:

```
__stream_shard__, container, instance, job, level, namespace, pod, service, service_name
```

`job` has exactly one value, `loki.source.kubernetes.pod_logs`. There is no
`atlas_env` stream label — `atlas.env` is a **field inside the JSON payload**,
so it must be reached through `| json`, not through a stream selector. The
working query:

```logql
{container="bootstrap", namespace="atlas-pr-<N>"} | json | atlas_step != ""
```

which returns lines shaped like:

```json
{"ts":"2026-08-26T12:21:34Z","level":"info","atlas.env":"2a03",
 "atlas.step":"wait-ready",
 "msg":"waiting for atlas-tenants, atlas-configurations, atlas-data, atlas-renders"}
```

Drop the `namespace` matcher to query across every PR env at once.

### Re-deriving the bootstrap deadline

`deploy/k8s/overlays/pr/sync-bootstrap.yaml` sets
`spec.activeDeadlineSeconds: 900`. That value came from the distribution
below; any retune must repeat this method, because the shortcut of eyeballing
a few recent runs will miss the tail that sets the bound.

Method: port-forward Loki, then run `query_range` over `{container="bootstrap"}`
in **consecutive 6-hour windows** across the retention period, each with
`limit=5000`. Group the returned lines by pod and take
`max(ts) - min(ts)` as that pod's duration. Filter to pods carrying a terminal
`"atlas.step":"done"` line — those are the successful runs, and they are the
population the deadline must not truncate. Windowing matters: a single
30-day query silently truncates at the line limit and biases the result
toward recent runs.

Result on 2026-08-26 — 120 windows, 4663 lines total, busiest window 153
lines (nowhere near the cap, so this is the full population, not a sample):

| stat | all 150 pods | 104 successful pods |
|---|---|---|
| min | 0s | 6s |
| p50 | 56s | 68s |
| p90 | 253s | 115s |
| p95 | 362s | 162s |
| p99 | 382s | 542s |
| max | 542s | 542s |

**Caveat that moves the number:** this is the span of the container's *log*
stream. It excludes image-pull and scheduling time, both of which precede the
first log line. The Job's `activeDeadlineSeconds` clock starts earlier, so
true Job wall-clock exceeds these figures — headroom must absorb that. 900s is
1.66x the slowest successful run and 5.5x p95.

**Retention caveat.** The earliest bootstrap log in Loki on 2026-08-26 was
`2026-08-12T10:36:26Z` — **14 days, not 30.** A 30-day query window silently
returns 14 days of data. Any future retune has at most a fortnight of history,
and if PR volume drops the population shrinks with it. This is the strongest
argument for emitting the missing metric above: the log-derived method has a
hard horizon that a histogram would not.
````

- [ ] **Step 5: Run the assertions to verify they pass**

```bash
RB=docs/runbooks/ephemeral-pr-deployments.md
rc=0
for pat in 'gh secret set GHCR_TOKEN' 'atlas_env="a3f7"'; do
  grep -q "$pat" "$RB" && { echo "STILL PRESENT: $pat"; rc=1; }
done
for pat in 'container="bootstrap"' '14 days' 'never emitted'; do
  grep -q "$pat" "$RB" || { echo "MISSING: $pat"; rc=1; }
done
[ "$rc" -eq 0 ] && echo "SECTION 9.6 OK"
```

Expected: prints `SECTION 9.6 OK` and nothing else.

- [ ] **Step 6: Verify section boundaries survived**

```bash
grep -n '^## §9\.' docs/runbooks/ephemeral-pr-deployments.md
```

Expected: §9.1 through §9.16 all still present, in order, none duplicated and none missing. §9.6 must appear exactly once.

- [ ] **Step 7: Confirm no absolute paths leaked in**

```bash
grep -nE '/home/|/Users/' docs/runbooks/ephemeral-pr-deployments.md
```

Expected: no matches (grep exits 1).

- [ ] **Step 8: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbook): correct §9.6 queries and drop the obsolete GHCR_TOKEN rotation step"
```

---

## Final gate

After Task 6, before requesting review:

- [ ] `tools/pr-sparse-mirror-guard.sh` exits 0
- [ ] `actionlint .github/workflows/pr-cleanup.yml` exits 0
- [ ] The flagless `tools/verify.sh` exits 0. `--quick` / `--no-docker` do **not** count (CLAUDE.md: "Done means verified"). Dispatch `atlas-verifier` rather than running it inline.
- [ ] Code review before opening the PR — `atlas-reviewer` per task, then `superpowers:requesting-code-review`. No guideline reviewer applies: `task-facts.sh` reports `go_changed=false`, `ts_changed=false`, `backend_audit_families=none`, `frontend_review=false`.

**Acceptance criterion this plan cannot satisfy from the repo.** The PRD asks for FR-1 "verified end-to-end on a real PR." The Kubernetes half is already proven (design §2.1 measured a deadline firing on a Job under `deletionTimestamp` pinned by a finalizer, reaching terminal `Failed`). The Argo half — that a `Failed/DeadlineExceeded` hook Job actually drives the operation terminal and releases `hook-finalizer` — cannot be proven from source. Exercise it after merge: open a throwaway PR, break its bootstrap deliberately (a bad `ATLAS_UI_BASE`), close the PR mid-sync, and confirm the teardown completes unattended within ~16 minutes. Record the result in the task folder.
