# PR-Env Teardown Fixes — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-19
---

## 1. Overview

The ephemeral per-PR deployment system shipped in `task-063-ephemeral-pr-deployments` is, in practice, leaking per-env state on every teardown. The May 18–19 close of PRs #491 (merged) and #522 (label-removed) left behind 59 Postgres databases, 264 Kafka topics, 42 consumer groups, 53 Redis keys, two Argo `Application` CRDs wedged in Terminating, and two `bot/pr-N-resolved` branches. None of this was caught by an alert; it was discovered only because the user asked "why doesn't Argo seem to be cleaning up?"

Root-cause analysis identified **four independent latent bugs**, all of which were silently failing every previous teardown — meaning the cluster has been accumulating orphaned state since the system went live. Manual recovery is documented in this task's `recovery-log.md` for the two known cases; this PRD addresses the structural fixes so the failure mode stops.

The work spans three planes:

- **Argo CD configuration** in the cluster-infra repo: change the PostDelete cleanup mechanism so it doesn't depend on a namespace Argo CD is also pruning; rotate the GitHub token the cleanup CronJob uses; remove the (currently dead) 24h grace window.
- **In-repo manifests and scripts**: change `services/atlas-pr-bootstrap/scripts/cleanup.sh` so it derives `ATLAS_ENV` from the PR number directly (defensive, in case the annotation drifts again), and adjust `deploy/k8s/overlays/pr/postdelete-cleanup.yaml` to match the new cleanup mechanism.
- **Investigation + regression coverage**: root-cause the env-hash annotation drift, and add an end-to-end smoke test that verifies cleanup actually runs and reclaims everything for a synthetic PR.

The success metric is: open and immediately close a synthetic PR, then assert that within N minutes the per-env namespace is gone, no per-env databases / topics / groups / Redis keys / ghcr tags / bot branches remain, and the Argo Application no longer exists.

## 2. Goals

### Primary goals

- Every PR teardown completes cleanup successfully, without wedging in Terminating.
- Cleanup targets the correct `ATLAS_ENV` hash for the PR being torn down, regardless of any annotation drift.
- The `bot/pr-N-resolved` branch is deleted from GitHub after the Application finalizes.
- The actual cleanup contract ("PR close ⇒ teardown begins immediately") matches what the runbook says.
- A CI smoke test catches future regressions in any of the above.
- Root cause of the `atlas.env` annotation drift is identified and either fixed or explicitly waived.
- The recovery sweep from May 19, 2026 (PR 491 + 522) is documented as a runbook procedure, so the next operator finding orphaned state has commands to run.

### Non-goals

- Re-architecting the data-isolation model (env hash, shared infra). That is task-063's domain.
- Adding TLS, auth, or a different DNS provider to PR envs.
- Per-PR copies of Postgres / Kafka / Redis pods.
- Moving cleanup off the in-cluster CronJob + PostDelete hook architecture (i.e., no migration to a custom operator). Stay with the existing Argo CD primitives.
- Honoring a 24h cleanup-grace window. The user does not use it; it will be removed, not implemented (see §4.2).
- Cleanup of pre-existing orphaned envs other than 491/522. The recovery sweep script is in scope; operationalizing it against the full cluster is the operator's call after this lands.

## 3. User Stories

- **As a developer**, I want my closed PR's environment to be fully reclaimed without manual intervention, so I don't have to file a cleanup ticket or remember to chase the operator.
- **As the cluster operator**, I want to be confident that closing a PR releases every per-env resource (DBs, topics, groups, Redis keys, image tags, DNS, bot branch), so I'm not paying a slow-burn capacity bill for ghost state.
- **As the cluster operator**, I want a single command to sweep the cluster for orphaned per-env state, so I can recover from the next undiagnosed wedge without writing ad-hoc psql/kafka/redis scripts.
- **As an on-caller**, when a teardown wedges, I want the failure mode to be loud (alert, condition on Application) instead of silent (Application slowly accumulates finalizers and eventually deadlocks), so I notice it before 30 envs leak.
- **As a CI maintainer**, I want a regression test that exercises the open-close lifecycle, so the next refactor of cleanup can't silently break teardown.

## 4. Functional Requirements

### 4.1 Fix the finalizer-ordering wedge (bug #1)

**Current behavior:** The ApplicationSet template syncs the PR overlay with `CreateNamespace=true`, making the destination namespace `atlas-pr-<N>` an Argo-managed resource. On deletion, Argo runs `resources-finalizer.argocd.argoproj.io` first, which prunes the namespace along with everything else. Then `post-delete-finalizer.argocd.argoproj.io/cleanup` tries to create the `atlas-pr-cleanup` Job inside `atlas-pr-<N>` and gets `DeletionError: namespaces "atlas-pr-<N>" not found`. The Application wedges; the cleanup script never runs.

**Required behavior:** The PostDelete cleanup Job must run before the destination namespace is pruned, OR run in a namespace that isn't going away.

**Acceptance criteria:**
- A PR close, end-to-end, results in:
  - The `atlas-pr-cleanup` Job completing successfully (`status.succeeded == 1`).
  - The Application CRD fully removed (no lingering finalizers).
  - The destination namespace `atlas-pr-<N>` fully removed.
- No `DeletionError: namespaces "atlas-pr-<N>" not found` appears on any Application's `status.conditions`.

**Implementation directions (non-prescriptive — design is for the design phase):**

Three plausible approaches, ordered by perceived cost:

- **(a)** Run the PostDelete Job in the long-lived `argocd` namespace instead of `atlas-pr-<N>`. The Job already inlines all the config it needs (no `envFrom` from per-env ConfigMaps); the only reason it lives in `atlas-pr-<N>` is convention. Requires an Argo CD `ServiceAccount`/`Role` in `argocd` that can read the per-env credentials (db-credentials, pihole-credentials, ghcr-pat) — but the Reflector controller already replicates those, so we might be able to mount them from `argocd` if Reflector targets it, or pivot to using cluster-wide secrets.
- **(b)** Drop `CreateNamespace=true` and have the ApplicationSet's PreSync hook or a bootstrap Job manage the namespace out-of-band. The namespace becomes unmanaged; resources-finalizer leaves it alone; PostDelete hook can create its Job, the Job runs, then a final teardown step deletes the namespace.
- **(c)** Move per-env teardown out of the Application's PostDelete hook entirely — into the cluster-infra `atlas-pr-cleanup` CronJob, which already runs every hour and already has the API access needed. The PostDelete hook would simply mark the env as "ready for sweep"; the CronJob does the actual work in its own namespace.

The design phase will pick one. (a) is the smallest change to the existing architecture; (c) is the cleanest separation but the biggest refactor.

### 4.2 Drop the dead 24h cleanup-grace window (bug #2)

**Current behavior:** The ApplicationSet template stamps `atlas.cleanup-grace: 24h` on every generated Application. The cluster-infra `atlas-pr-cleanup` CronJob reads this and the `atlas.cleanup-deadline` annotation it sets, intending to defer `kubectl delete application` until the grace expires. But the ApplicationSet's PullRequest generator deletes the generated Application immediately on PR close (no `applicationsSync: create-update` policy), so the grace logic never gets a chance to fire. The runbook (§9.4) and inline comments in `.github/workflows/pr-cleanup.yml` describe this 24h grace as the active contract.

**Decision (per user):** Remove the grace mechanism rather than wire it up. The user does not use the window.

**Required behavior:** The system documents and enforces "PR close ⇒ teardown begins immediately."

**Acceptance criteria:**
- `atlas.cleanup-grace` annotation removed from the ApplicationSet template.
- `atlas.cleanup-deadline` annotation no longer set anywhere.
- The cluster-infra `atlas-pr-cleanup` CronJob's deadline-tracking logic deleted. The CronJob's role narrows to: "verify no orphaned per-env state exists; alert if it does."
- `.github/workflows/pr-cleanup.yml` comments updated; the cross-reference to runbook §9.4 removed or rewritten.
- `docs/runbooks/ephemeral-pr-deployments.md` §9.4 rewritten:
  - Remove all 24h grace language.
  - Replace with the new immediate-teardown contract.
  - Keep the "wedged Application" recovery procedure (finalizer patch + branch delete + state sweep) under a new section heading, since that procedure is still useful.

### 4.3 Rotate the cleanup token to a purpose-specific secret (bug #3)

**Current behavior:** The cluster-infra `atlas-pr-cleanup` CronJob mounts `argocd-repo-creds-chronicle20-atlas` (Argo CD's source-repo PAT) and uses it to call `DELETE /repos/Chronicle20/atlas/git/refs/heads/bot/pr-<N>-resolved`. The token responds 403, so branch deletion silently fails on every run.

**Required behavior:** A dedicated secret with exactly the scopes the cleanup tasks need, separate from Argo CD's source-repo creds.

**Acceptance criteria:**
- New Kubernetes Secret `atlas-pr-cleanup-gh-token` in the `argocd` namespace, populated from a fresh GitHub fine-grained PAT (or GitHub App installation token).
- PAT scopes (least-privilege):
  - `Contents: write` on `Chronicle20/atlas` (for branch deletion).
  - `Packages: read + write` on `Chronicle20/*` (for ghcr image-tag deletion).
  - Read-only on PR metadata for the "is the PR still open?" check the cron currently does, OR drop that check entirely if §4.2 makes it irrelevant.
- CronJob updated to mount the new secret; Argo source-repo creds left untouched.
- The PostDelete hook's `ghcr-pat` Secret is also examined: if it's the same token, it's redirected to read from the new secret as well, to consolidate.
- `.github/workflows/pr-cleanup.yml`'s `GHCR_TOKEN` repository secret kept (workflow context can't reach in-cluster secrets), but cross-referenced in the runbook so an operator knows to rotate both when rotating the underlying PAT.
- Runbook §9.5 updated to describe the new secret and its rotation procedure.
- After the cron runs once with the new token, `bot/pr-N-resolved` branches for any future closed PR are deleted within one cron cycle.

### 4.4 Defensive fix + investigation for env-hash drift (bug #4)

**Current behavior:** The ApplicationSet template sets `atlas.env: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'` (the value applied to pods, DBs, topics, Redis keys, etc.). The live Application's `metadata.annotations["atlas.env"]` reads a *different* value than the formula produces (491: annotation `f78b`, formula → pod label `ed86`; 522: annotation `d496`, formula → pod label `a476`). The cleanup script `services/atlas-pr-bootstrap/scripts/cleanup.sh` is invoked with `ATLAS_ENV` derived from the annotation (via the inlined env in `deploy/k8s/overlays/pr/postdelete-cleanup.yaml`, which uses the bot-substituted `PLACEHOLDER_ATLAS_ENV` token from `pr-validation.yml`). If cleanup had run, it would have targeted the wrong env hash and orphaned everything.

The drift values (`f78b`, `d496`) do not match any obvious formula on PR number, head sha, branch name, env hash, or simple casing/encoding variations of those. Investigation in the spec phase was inconclusive.

**Required behavior, part A (defensive — must ship):** Cleanup MUST target the correct env hash regardless of any annotation drift.

**Acceptance criteria, part A:**
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` computes `ATLAS_ENV` from `PR_NUMBER` directly: `ATLAS_ENV=$(printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4)`. The inlined `ATLAS_ENV` value from `postdelete-cleanup.yaml` is removed; only `PR_NUMBER` is templated in.
- A unit / dry-run test in `services/atlas-pr-bootstrap/test/` exercises this computation for at least three PR numbers (e.g., 1, 491, 522) and asserts the result matches a known good table.
- The bootstrap script (`bootstrap.sh`) and any other consumer of `ATLAS_ENV` are audited; if any read it from env instead of computing from `PR_NUMBER`, decide per-call whether they get the same defensive computation (preferred) or stay as-is.
- After this change, even if every Application in the cluster has a wrong `atlas.env` annotation, cleanup still targets the correct DBs/topics/groups/keys for each PR.

**Required behavior, part B (investigation — best-effort):** Identify what is writing the wrong value to the annotation.

**Investigation tasks:**
- Compare `metadata.managedFields` on a live Application's `metadata.annotations` to see which controller(s) own the `atlas.env` field.
- Check whether any admission webhook in the cluster mutates `atlas.env`. Inspect `MutatingWebhookConfiguration`s.
- Check whether the ApplicationSet `goTemplate` actually emits `ed86` when the controller renders it — does the controller log the rendered template? Can we add a temporary debug annotation to the template that emits another computed value, and see if it lands in the Application?
- Check the Application's edit history (Argo CD revision history doesn't include CRD edits, but `kubectl get events -n argocd` and `kubectl describe` may).
- Check whether the drift values match anything else: a previous PR number's env hash, a stale annotation from an earlier ApplicationSet template revision, a Reflector-replicated value, etc.

**Investigation deliverables:**
- A short writeup (markdown file in the task folder) of what was checked and what was found.
- If a root cause is identified, a follow-up acceptance criterion to fix it (in this task if cheap, deferred to a new task if not).
- If root cause is not identified within a reasonable timebox (~half a day), the defensive fix from part A stands alone and the investigation is closed with an explicit note that the annotation may continue to drift but cannot cause damage.

### 4.5 End-to-end smoke test (regression coverage)

**Required behavior:** A CI job that opens-then-immediately-closes a synthetic PR and asserts every per-env artifact is reclaimed.

**Acceptance criteria:**
- A workflow in `.github/workflows/` (e.g., `pr-env-smoke.yml`) that:
  - Triggers on `workflow_dispatch` and on a nightly schedule.
  - Creates a synthetic PR against a throwaway branch in `Chronicle20/atlas` with `deploy-env` label applied. (Or: against a side-repo, if synthetic PRs against the real repo are too noisy.)
  - Waits for the Application to become Healthy (timeout: 20 min).
  - Removes the `deploy-env` label (preferred over closing the PR, since label-removal teardown is the exact path that broke for 522).
  - Polls every 30s, up to 15 minutes, for: Application gone, namespace gone, zero per-env DBs / topics / groups / Redis keys / ghcr tags, branch deleted.
  - Fails the workflow if any of the above remains after the timeout.
  - On failure, dumps `kubectl describe application` and the last 200 lines of `atlas-pr-cleanup` Job logs into the workflow artifacts.
  - Closes the synthetic PR at the end.
- The CronJob's "verify no orphaned per-env state" mode (from §4.2) emits a Prometheus metric `atlas_pr_orphan_envs_total` that's scrapeable and can drive a future alert (alert wiring is out of scope; the metric must exist).

### 4.6 Recovery sweep tooling

**Required behavior:** Codify the May 19 manual recovery as a runnable script, so the next operator finding orphaned state has commands instead of a thread to scroll through.

**Acceptance criteria:**
- A script at `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` that:
  - Takes one or more PR numbers as arguments.
  - For each, computes `ATLAS_ENV=$(printf "pr-%d" "$N" | sha256sum | cut -c1-4)`.
  - Lists (default) or deletes (with `--apply`) per-env DBs, Kafka topics + consumer groups, Redis keys, ghcr image tags, the bot branch, and the Application's stuck finalizers.
  - Reuses the existing `lib.sh` helpers where possible.
  - Is idempotent.
- A new runbook section (`docs/runbooks/ephemeral-pr-deployments.md` §9.X) documenting how to invoke it.
- The May 19, 2026 recovery (PRs 491 + 522) is documented in the task folder's `recovery-log.md` with exact counts, commands run, and remaining manual steps (the two bot branches that couldn't be deleted due to PAT scope).

## 5. API Surface

No Atlas service APIs change. The "API surfaces" are:

- **GitHub REST endpoints used by the cleanup CronJob:**
  - `GET /repos/Chronicle20/atlas/pulls/{N}` — PR state lookup (may go away per §4.2).
  - `DELETE /repos/Chronicle20/atlas/git/refs/heads/bot/pr-{N}-resolved` — branch deletion.
  - `GET /users/chronicle20/packages/container/{svc}%2F{svc}/versions` and `DELETE` — ghcr tag cleanup.

- **Kubernetes API:**
  - `Application` CRD: removal of `metadata.annotations["atlas.cleanup-grace"]` and `metadata.annotations["atlas.cleanup-deadline"]`.
  - New `Secret/argocd-pr-cleanup-gh-token` in `argocd` namespace.
  - Possible new `ServiceAccount` + `Role` + `RoleBinding` in `argocd` if the PostDelete Job moves there (per §4.1 option (a)).

## 6. Data Model

No persistent data-model changes. The relevant "data" is the per-env namespace-suffixed Postgres databases, Kafka topics, consumer groups, and Redis keys — the schema for which is owned by `task-063-ephemeral-pr-deployments`.

The cleanup script's computation of `ATLAS_ENV` from `PR_NUMBER` is a contract that must match the ApplicationSet template's computation exactly:

```
ATLAS_ENV = first_4_chars( hex( sha256( "pr-${PR_NUMBER}" ) ) )
```

A unit test guards this.

## 7. Service Impact

| Surface | Change |
|---|---|
| `deploy/k8s/overlays/pr/postdelete-cleanup.yaml` | Switch to PR-number-only templating (drop `PLACEHOLDER_ATLAS_ENV`). Possibly move out of namespace per §4.1. Annotation changes. |
| `deploy/k8s/overlays/pr/kustomization.yaml` | If §4.1 chooses option (b), remove `CreateNamespace=true` from syncOptions. |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | Compute `ATLAS_ENV` from `PR_NUMBER` instead of reading from env. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | New helper `compute_atlas_env` (single source of truth, also used by sweep script). |
| `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` | New, per §4.6. |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Audit; align with new `ATLAS_ENV` derivation if it currently reads from env. |
| `services/atlas-pr-bootstrap/test/` | New tests for `compute_atlas_env` and for sweep script's dry-run mode. |
| `.github/workflows/pr-cleanup.yml` | Comments refreshed; cross-references to §9.4 updated. |
| `.github/workflows/pr-env-smoke.yml` (new) | Smoke test per §4.5. |
| `.github/workflows/pr-validation.yml` | `update-pr-overlay` step that substitutes `PLACEHOLDER_ATLAS_ENV` is removed/simplified, since the overlay no longer needs it. |
| `docs/runbooks/ephemeral-pr-deployments.md` | §9.2, §9.4, §9.5 updated; new §9.6 for sweep tooling. |
| Cluster-infra repo (out-of-tree): ApplicationSet, CronJob, Secret manifests. | Coordinated changes; this PRD lists them but the actual yaml diffs land in the cluster-infra repo as a sibling PR. |

## 8. Non-Functional Requirements

- **Idempotency:** every cleanup operation (DROP DATABASE, topic delete, consumer-group delete, Redis DEL, branch delete, ghcr version delete, finalizer patch) must tolerate the resource being absent. The current script uses `IF EXISTS`, `xargs -r`, `|| true`; preserve that discipline.
- **Observability:**
  - The `atlas-pr-cleanup` Job logs every phase via the existing `log` helper. New steps follow the same `ATLAS_STEP=<phase> log info/error <msg>` pattern.
  - A Prometheus counter (or kube-state-metrics-derived) `atlas_pr_orphan_envs_total` is emitted by the CronJob per §4.5.
  - A `DeletionError` or `cleanup-failed` condition on any Application must remain visible in the runbook's `kubectl get application` summary (don't suppress it).
- **Security:**
  - The new `atlas-pr-cleanup-gh-token` is least-privilege (only the scopes listed in §4.3); does not get `repo` (write to entire repo) or `admin:org`.
  - The new Secret is in `argocd` namespace only; not replicated by Reflector unless the design phase determines a clear need.
  - PAT rotation procedure documented; expiry no longer than 90 days; calendar reminder for rotation is the operator's responsibility, but the runbook §9.5 mentions the expiry date.
- **Multi-tenancy:** N/A — this is cluster-level infra, single-tenant per-PR.
- **Reliability:** the cleanup chain (per-PR) completes within 10 minutes end-to-end under normal conditions; smoke test asserts within 15 minutes (with margin). Postgres `DROP DATABASE WITH (FORCE)` is the long pole; ~30 DBs × 1–2s each.
- **Backward compatibility:** existing in-flight PR envs MUST keep working through the change. Specifically, an Application that exists *before* this fix lands must still be cleanable after the fix lands. Mitigation: the sweep script (§4.6) is the manual escape hatch for any pre-existing wedge.

## 9. Open Questions

1. **Which §4.1 approach?** (a) move Job to `argocd`, (b) unmanage the namespace, or (c) shift teardown to the CronJob entirely. Resolved in design phase. Recommendation: pick (a) for v1; (b)/(c) only if (a) hits a real blocker (e.g., Reflector can't replicate db-credentials to `argocd`).
2. **GitHub App vs fine-grained PAT for the new cleanup token?** App is more durable (no 90-day rotation), but needs an org-level install + private key handling. PAT is faster to ship. Recommendation: PAT for v1, with a follow-up task to migrate to a GitHub App.
3. **Synthetic PR repo for §4.5 smoke test:** real `Chronicle20/atlas` repo (noisy, PR numbers consumed forever) vs. a sandbox repo (cleaner but tests a different ApplicationSet). Recommendation: real repo, with a `[smoke-test]` title prefix and immediate label removal + close in a single workflow run.
4. **Should the §4.6 sweep script include Pi-hole DNS cleanup?** The user manually deleted the Pi-hole entries for 491/522; the cleanup script already handles this in normal teardown. Recommendation: yes, include it in sweep for consistency.
5. **Bug #4 root cause:** if investigation is inconclusive within timebox, do we add a *defensive* annotation reconciler (a tiny controller that resets `atlas.env` to the computed value), or accept the drift since the defensive code path makes it harmless? Recommendation: accept the drift unless investigation reveals it's caused by something else also writing wrong values.
6. **`tools/task-numbers.sh next` is broken** (exits 1 because of the non-numeric `task-stubs-followup` branch). Out of scope here, but worth a separate one-line fix task.

## 10. Acceptance Criteria

A single end-to-end check, repeated three times back-to-back via the smoke test (§4.5):

1. Open a PR labeled `deploy-env`. Wait for Application Healthy.
2. Remove the label.
3. Within 15 minutes:
   - `kubectl get application atlas-pr-<N> -n argocd` returns NotFound.
   - `kubectl get ns atlas-pr-<N>` returns NotFound.
   - `psql ... -c "SELECT count(*) FROM pg_database WHERE datname ~ '-<env>\$';"` returns 0.
   - `kafka-topics.sh --list | grep -E -- '-<env>\$'` returns no lines.
   - `kafka-consumer-groups.sh --list | grep -E -- '\[<env>\]\$'` returns no lines.
   - `redis-cli --scan --pattern '<env>:*' | wc -l` returns 0.
   - `gh api /users/chronicle20/packages/container/atlas-account%2Fatlas-account/versions --jq '[.[] | select(.metadata.container.tags[]? | startswith("pr-<N>-"))] | length'` returns 0 (spot-check; same for any 5 random services).
   - `gh api /repos/Chronicle20/atlas/git/refs/heads/bot/pr-<N>-resolved` returns 404.
4. Closing the PR (after the label-removal teardown completes) is a no-op — nothing remains to clean.
5. `kubectl get application -n argocd` shows zero Applications with a `DeletionError` condition.
6. The runbook's "force-cleanup wedged Application" recipe (§9.4 of the updated doc) is exercised by deliberately patching out a Job, observing the wedge, and running the documented recovery — all steps succeed.

The May 19 recovery is captured in `recovery-log.md` with exact commands, outputs, and remaining tasks (bot branches).
