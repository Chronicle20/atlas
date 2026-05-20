# PR-Env Teardown Fixes — Context

Companion to `plan.md`. Captures decisions, key files, sibling-PR requirements, and known gaps so an execution agent doesn't re-derive them.

---

## Key decisions (from `design.md`, resolved)

| PRD § | Question | Decision |
|---|---|---|
| 4.1 | Finalizer-ordering fix: move Job vs unmanage namespace vs offload to CronJob | **(a)** Move Job to `argocd` namespace. Smallest delta, no contract change for the ApplicationSet. |
| 4.3 | Cleanup token: GitHub App vs fine-grained PAT | Fine-grained PAT (`atlas-pr-cleanup-gh-token`) for v1. App migration deferred. |
| 4.4-B | Reconciler if drift root cause unidentified | No reconciler. Defensive `compute_atlas_env` in `cleanup.sh` makes drift harmless. |
| 4.5 | Smoke-test target repo | Real `Chronicle20/atlas`. Title prefix `[smoke-test]`; close + branch delete in the same workflow run. |
| 4.6 | Sweep script includes Pi-hole? | Yes — for parity with `cleanup.sh`. |

## Formula contract (locked, pinned by tests)

```
ATLAS_ENV = first 4 hex chars of sha256("pr-<PR_NUMBER>")
```

Must agree across three sites:

1. cluster-infra `ApplicationSet(atlas-pr)` template: `{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}`.
2. `.github/workflows/pr-validation.yml` line 273: `printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4`.
3. `services/atlas-pr-bootstrap/scripts/lib.sh::compute_atlas_env` (added by plan Task 1).

Test `services/atlas-pr-bootstrap/test/lib_test.bats` asserts the contract via recovery-log oracles:
- PR 491 → `ed86`
- PR 522 → `a476`
- PR 1 → computed at test-write time and pinned literally.

If the formula ever changes, all three sites must change together and the oracle table updated.

## Key files (this repo)

| File | Role |
|---|---|
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Shared bash helpers. Gains `compute_atlas_env`. |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | PostDelete Job's entry point. Now derives `ATLAS_ENV` from `PR_NUMBER`; gains a `drop-branch` phase. |
| `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` | NEW. Operator-runnable orphan sweep (list/apply). |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Audited only. Reads `ATLAS_ENV` from the `atlas-env` ConfigMap built at PreSync time; no change. |
| `services/atlas-pr-bootstrap/test/{lib,cleanup,sweep}_test.bats` | bats coverage. |
| `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` | Job spec — moved from `pr/` during Design A overlay split. Lives in `argocd` namespace, has `serviceAccountName: atlas-pr-cleanup`, references the new `atlas-pr-cleanup-gh-token` Secret, no `ATLAS_ENV` env entry. |
| `deploy/k8s/overlays/pr-cleanup/kustomization.yaml` | NEW. Minimal overlay with no namespace directive so the Job's `namespace: argocd` survives kustomize-build. Mirrors `commonLabels` from `pr/` (`atlas.env`, `atlas.pr-number`, plus `app: atlas-pr-cleanup`). |
| `deploy/k8s/overlays/pr/kustomization.yaml` | Drops `postdelete-cleanup.yaml` from `resources:` (moved to `pr-cleanup/`). Per-PR namespace directive unchanged. |
| `.github/workflows/pr-cleanup.yml` | Comments + step output refreshed. No structural change. |
| `.github/workflows/pr-validation.yml` | `update-pr-overlay` step now sed-substitutes placeholders in BOTH `pr/` and `pr-cleanup/` overlays and stages both for the bot-branch force-push. |
| `.github/workflows/pr-env-smoke.yml` | NEW. Nightly + manual end-to-end teardown regression. |
| `docs/runbooks/ephemeral-pr-deployments.md` | §9.2 / §9.4 / §9.5 rewritten; new §9.11 (sweep). |
| `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md` | NEW. Best-effort writeup of bug #4 Part B. |

## Bootstrap audit (PRD §4.4 Part A asks)

`bootstrap.sh` reads `ATLAS_ENV` from `require_env ATLAS_ENV ...` (line 14). That env var comes from the kustomize `atlas-env` ConfigMap, which is built at create time from already-substituted `PLACEHOLDER_ATLAS_ENV` tokens. The substitution is performed by `pr-validation.yml`'s `update-pr-overlay` step using the canonical formula. So bootstrap's `ATLAS_ENV` is correct at create time even when the Application's annotation drifts later.

**No defensive derivation needed in bootstrap.** The drift only matters at teardown, when the ConfigMap is already gone and `cleanup.sh` previously read `ATLAS_ENV` from the drifted annotation.

## Sibling PR (cluster-infra repo)

These changes MUST land alongside this branch's PR. Do not merge either side alone.

| Component | Change |
|---|---|
| `ApplicationSet(atlas-pr)` template | **Switch to Argo CD multi-source.** Replace the single `spec.template.spec.source.path: deploy/k8s/overlays/pr` with a `spec.template.spec.sources:` list containing two entries: `deploy/k8s/overlays/pr` and `deploy/k8s/overlays/pr-cleanup`, both pinned to `targetRevision: bot/pr-{{.number}}-resolved`. Requires Argo CD ≥ 2.6 (GA in 2.7); verify the live cluster version before merging the template change. Also drop `metadata.annotations["atlas.cleanup-grace"]` and any logic setting `atlas.cleanup-deadline`. |
| New `ServiceAccount` `atlas-pr-cleanup` in `argocd` | Permissions: `get`/`list`/`patch` on `secrets` in `argocd` (Reflector replicas the Job reads); `get`/`list`/`patch`/`delete` on `applications.argoproj.io` in `argocd` (finalizer patch fallback). |
| New `Role` + `RoleBinding` | Bound to the new ServiceAccount in `argocd`. |
| Reflector source-Secret annotations | `db-credentials` (source in `atlas-main`) — extend `reflection-allowed-namespaces` and `reflection-auto-namespaces` from `atlas-pr-.*` to `atlas-pr-.*\|argocd` so the PostDelete Job in `argocd` can read it. `pihole-credentials` source already lives in `argocd`, so no annotation change needed (Job reads source directly). |
| New `Secret` `atlas-pr-cleanup-gh-token` in `argocd` | Key `GHCR_TOKEN`. Sourced from a fine-grained PAT minted under the `Chronicle20` user with: Repository permissions → **Contents: Read and write** + **Metadata: Read-only** (auto) on `Chronicle20/atlas`; Account permissions → **Packages: Read and write**. Expiry ≤ 90 days. See `docs/runbooks/ephemeral-pr-deployments.md` §9.5 for the full picker walkthrough + classic-PAT fallback. |
| `atlas-pr-cleanup` CronJob | Narrow to "orphan-sweep + metric emission" mode. Remove the deadline-tracking code path. Remove the GitHub-API branch-delete code (now owned by the PostDelete Job). |
| Prometheus metric | `atlas_pr_orphan_envs_total` counter, labels `pr_number`, `kind ∈ {application,database,topic,consumer_group,redis_key,image_tag,bot_branch}`. Implementation in the CronJob's orphan-sweep mode. |
| Repo secret `GHCR_TOKEN` (in this repo) | Rotated to the same underlying PAT, so `.github/workflows/pr-cleanup.yml`'s image-delete job and the PostDelete Job both work. |

Coordination protocol:

1. Land cluster-infra Secret + SA + Role first (no-op for current envs).
2. Land this branch (creates `pr-cleanup/` overlay; existing single-source ApplicationSet ignores it, so currently-deployed PR envs are unaffected).
3. Land the cluster-infra ApplicationSet template change (multi-source flip + drop grace + narrow CronJob). The flip is what actually activates the new cleanup Job path. Verify with a synthetic PR via `pr-env-smoke.yml`.

### Pre-flight verification for the multi-source flip

Before merging the ApplicationSet template change, confirm:

- `kubectl -n argocd get pods -l app.kubernetes.io/name=argocd-application-controller -o jsonpath='{.items[*].spec.containers[*].image}'` shows Argo CD ≥ 2.6 (multi-source GA in 2.7; functional in 2.6 with a feature flag).
- A handcrafted test Application using multi-source against `bot/pr-<some-existing>-resolved` reconciles cleanly. Argo CD applies each source's manifests independently and merges hook resources — both `pr/`'s PostSync hooks and `pr-cleanup/`'s PostDelete hook must register correctly.
- Argo CD's `destination` field: multi-source Applications use the destination's namespace as a fallback for resources without `metadata.namespace`. All resources in `pr/` set their namespace via the overlay-level directive (`atlas-pr-<N>`), and the `pr-cleanup/` Job sets `namespace: argocd` explicitly — so the destination namespace is informational. Set it to `atlas-pr-<N>` to match the historical contract and not surprise observability tooling.

## Migration / rollout

1. Land cluster-infra Secret + ServiceAccount/Role first.
2. Merge this branch (task-070). The `pr-cleanup/` overlay exists but is not yet referenced by any live ApplicationSet — currently-deployed PR envs continue to use the (broken-by-design but not actively wedged) single-source flow.
3. Run `sweep-orphans.sh --apply` against any currently-wedged Applications (today: none, per `recovery-log.md`).
4. Land the cluster-infra ApplicationSet template change (multi-source flip + drop cleanup-grace) and CronJob narrowing. This is the cutover commit — generated Applications now apply both overlays.
5. Manually trigger `pr-env-smoke.yml` once via `workflow_dispatch` to validate the end-to-end teardown.

**Rollback:**

- Revert the cluster-infra ApplicationSet template change (back to single-source pointing at `pr/`). Generated Applications go back to the prior single-overlay form. The `pr-cleanup/` overlay in this repo sits unused.
- For an emergency stop on this repo alone: revert the overlay-split commit; the Job manifest goes back into `pr/` (still broken vs kustomize namespace, but no worse than before task-070).
- Cluster-infra ServiceAccount/Secret can sit unused without harm.

**Backwards compatibility:** PR Applications created BEFORE the multi-source flip have the old single-source spec baked into their PostDelete hook (it resolves at Application create time). For those, recovery is via the sweep script + manual finalizer patch. Once the multi-source ApplicationSet template lands, newly-opened PRs use the new two-overlay flow; existing in-flight Applications continue under the old contract until they are torn down.

## Known gaps / follow-ups

- **Self-hosted runner**: `pr-env-smoke.yml` is committed with `if: false` on the two cluster-touching jobs if no `self-hosted, atlas-cluster` runner exists at execution time. Flip both to the real `if:` conditions when the runner is provisioned. See plan Task 9.
- **GitHub App migration** for the cleanup token (PRD OQ 2). Defer to a follow-up task.
- **Alert wiring** for `atlas_pr_orphan_envs_total` — metric is emitted but no Alertmanager rule. Operator's call when to wire it up.
- **`tools/task-numbers.sh next`** is broken (PRD OQ 6). Out of scope; needs a separate one-line fix task.
- **Pre-existing orphan envs** other than 491/522 may exist with different env hashes. The sweep script is the tool, but enumerating them across the whole cluster is operator-driven post-merge.

## References

- PRD: `docs/tasks/task-070-pr-env-teardown-fixes/prd.md`
- Design: `docs/tasks/task-070-pr-env-teardown-fixes/design.md`
- Recovery log (May 19, 2026): `docs/tasks/task-070-pr-env-teardown-fixes/recovery-log.md`
- Investigation deliverable: `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md` (created during execution)
- Runbook: `docs/runbooks/ephemeral-pr-deployments.md`
- Cluster-infra repo: ApplicationSet, CronJob, Secret manifests (out of tree).
