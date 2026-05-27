# PR-Env Teardown Fixes — Design

Version: v1
Status: Draft
Created: 2026-05-19
PRD: `prd.md`

---

## 1. Scope and approach

This task fixes the four latent bugs catalogued in `prd.md` and the May-19
recovery log:

1. **Finalizer-ordering wedge** (PRD §4.1): PostDelete Job is created in a
   namespace that Argo CD is simultaneously pruning, so it never runs.
2. **Dead 24h `atlas.cleanup-grace` window** (PRD §4.2): annotation is set and
   read but the ApplicationSet deletes the Application before the grace logic
   can fire.
3. **`argocd-repo-creds-chronicle20-atlas` lacks `Contents: write`** (PRD
   §4.3): bot-branch deletion 403s every run.
4. **`atlas.env` annotation drift** (PRD §4.4): annotation reads a value that
   does not match the formula's output; cleanup keyed off the annotation
   targets the wrong resources.

Plus two cross-cutting deliverables:

5. **End-to-end smoke test** (PRD §4.5).
6. **Sweep script** to recover from future wedges (PRD §4.6).

The design's organising rule is **defence in depth**: every individual fix is
narrow enough to land independently and survive on its own, even if the others
slip. In particular, fix (4) (defensive `ATLAS_ENV` derivation) is the load-
bearing one — once it ships, every other failure mode becomes recoverable
without orphaning state.

### What's in this repo vs cluster-infra

The cluster-infra repo (out of tree for this task) owns:

- The `atlas-pr` `ApplicationSet` (PullRequest generator + template, including
  the `atlas.cleanup-grace` annotation that §4.2 removes).
- The hourly `atlas-pr-cleanup` `CronJob` in `argocd` (deadline tracking and
  bot-branch deletion).
- The `argocd-repo-creds-chronicle20-atlas` Secret it reuses.
- The new `atlas-pr-cleanup-gh-token` Secret (§4.3) and any new
  `ServiceAccount`/`Role`/`RoleBinding` required by (§4.1).
- The Reflector source Secrets `db-credentials`, `pihole-credentials`, and
  `ghcr-pat` whose replication targets are namespace patterns.

This repo (`Chronicle20/atlas`) owns:

- `deploy/k8s/overlays/pr/postdelete-cleanup.yaml` (the Job spec — its
  namespace, sa, env, command).
- `deploy/k8s/overlays/pr/kustomization.yaml` (sync options, common labels).
- `services/atlas-pr-bootstrap/scripts/{cleanup,bootstrap,lib}.sh` and tests.
- `.github/workflows/pr-cleanup.yml` and `pr-validation.yml`.
- `docs/runbooks/ephemeral-pr-deployments.md`.
- New: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`.
- New: `.github/workflows/pr-env-smoke.yml`.

The plan must explicitly cross-reference, for every cluster-infra-touching
change, the corresponding manifest in cluster-infra and call it out as a
"sibling PR" so neither side merges without the other.

---

## 2. Architecture

### 2.1 Before

```
PR closed (or `deploy-env` label removed)
   │
   ▼
ApplicationSet PullRequest generator drops the matching generator entry
   │
   ▼
Argo CD deletes the generated Application
   │
   ├─ resources-finalizer.argocd.argoproj.io
   │     ├─ deletes every managed resource in namespace atlas-pr-<N>
   │     └─ deletes the namespace atlas-pr-<N> itself        ◀── too eager
   │
   ▼
post-delete-finalizer.argocd.argoproj.io/cleanup
   ├─ attempts to create Job atlas-pr-cleanup in atlas-pr-<N>
   └─ ERROR: namespaces "atlas-pr-<N>" not found             ◀── wedge
   │
   ▼
Application stuck Terminating; cleanup never runs; state leaks
```

### 2.2 After (recommended option 4.1.a — move Job to `argocd` namespace)

```
PR closed (or `deploy-env` label removed)
   │
   ▼
ApplicationSet PullRequest generator drops the matching generator entry
   │
   ▼
Argo CD deletes the generated Application
   │
   ├─ resources-finalizer
   │     └─ deletes every managed resource in namespace atlas-pr-<N>
   │                                  (namespace ALSO pruned — still managed)
   │
   ▼
post-delete-finalizer/cleanup
   ├─ creates Job atlas-pr-cleanup in argocd namespace         ◀── lives here
   │     (long-lived; never pruned)
   ├─ Job runs cleanup.sh:
   │     │ deriving ATLAS_ENV from PR_NUMBER (defensive — §4.4)
   │     │ drops per-env DBs, topics, groups, Redis keys, ghcr tags, DNS
   │     │ deletes bot/pr-<N>-resolved branch (new GH token — §4.3)
   ├─ exits 0 → HookSucceeded GC removes the Job
   │
   ▼
post-delete-finalizer is drained
Application CRD fully removed
Namespace atlas-pr-<N> already gone from resources-finalizer step
```

### 2.3 Cron's narrowed role

After this lands, the cluster-infra `atlas-pr-cleanup` CronJob no longer:

- Reads `atlas.cleanup-grace` / `atlas.cleanup-deadline` annotations (gone).
- Calls `kubectl delete application` (Argo CD deletes Applications itself
  the moment the generator entry disappears).
- Deletes bot branches (the PostDelete Job now owns this).

The CronJob's role narrows to **orphan-sweep + metrics**: scan per-env
artifacts whose corresponding Application no longer exists, emit
`atlas_pr_orphan_envs_total`, and (optionally) alert. Implementation lives in
cluster-infra; this PRD specifies the metric contract (§7.4).

---

## 3. Detailed designs per bug

### 3.1 Bug #1 — Finalizer-ordering wedge (PRD §4.1)

**Decision:** option **(a)** — move the PostDelete Job to the `argocd`
namespace.

**Why this option:**

| Option | Cost | Risk | Verdict |
|---|---|---|---|
| (a) Move Job to `argocd` | Small. Job spec gains `namespace: argocd` + `serviceAccountName: atlas-pr-cleanup`. Reflector secrets need a new target pattern (or replacement). | Low — only one yaml file in-tree + ~3 manifests in cluster-infra. | **Chosen.** Smallest delta, no contract change for the ApplicationSet. |
| (b) Drop `CreateNamespace=true`, manage namespace out of band | Medium. Need a bootstrap Job in `argocd` (or a `Namespace` resource managed by a sibling `ApplicationSet`) plus an explicit cleanup hook to delete the namespace at the end. | Medium — racing namespace deletion with last PostDelete operation; tricky to make idempotent. | Rejected. |
| (c) Move all teardown into the hourly CronJob | Large. CronJob has to know about every PR's overlay parameters. Argo CD's PostDelete becomes a no-op marker. | High — loses the "PR close ⇒ immediate teardown" contract the PRD requires (§4.2). | Rejected. |

**Concrete changes (in this repo):**

`deploy/k8s/overlays/pr/postdelete-cleanup.yaml`:

```diff
 apiVersion: batch/v1
 kind: Job
 metadata:
   name: atlas-pr-cleanup
+  namespace: argocd
   annotations:
     argocd.argoproj.io/hook: PostDelete
     argocd.argoproj.io/hook-delete-policy: HookSucceeded
 spec:
   backoffLimit: 0
   template:
     spec:
       restartPolicy: Never
+      serviceAccountName: atlas-pr-cleanup
       containers:
         - name: cleanup
           image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
           command: ["/atlas/cleanup.sh"]
           envFrom:
             - secretRef:
                 name: db-credentials
             - secretRef:
                 name: pihole-credentials
             - secretRef:
-                name: ghcr-pat
+                name: atlas-pr-cleanup-gh-token
           env:
-            - name: ATLAS_ENV
-              value: "PLACEHOLDER_ATLAS_ENV"
             - name: PR_NUMBER
               value: "PLACEHOLDER_PR_NUMBER"
             # …rest unchanged…
```

`.github/workflows/pr-validation.yml`:

- Keep `PLACEHOLDER_PR_NUMBER` substitution (still needed by other manifests
  and by the cleanup Job).
- `PLACEHOLDER_ATLAS_ENV` keeps being substituted globally (still used by
  `kustomization.yaml`, `atlas-env-tokens.yaml`, `commonLabels`, etc.). Only
  `postdelete-cleanup.yaml` drops the `ATLAS_ENV` env entry.

**Cluster-infra prerequisites (sibling PR):**

1. New `ServiceAccount` `atlas-pr-cleanup` in `argocd`.
2. New `Role` granting it `get`/`list`/`patch` on `secrets` in `argocd`
   (the Reflector-replicated `db-credentials` / `pihole-credentials` lives
   there) and `get`/`list` on `configmaps` if anything else is needed at
   runtime. PostDelete Job does NOT need cluster-wide privileges.
3. Reflector source-Secret annotations updated so `db-credentials` and
   `pihole-credentials` replicate into `argocd` in addition to whatever
   per-`atlas-pr-*` pattern is already there. Same for the new
   `atlas-pr-cleanup-gh-token` (which is single-namespace, no replication
   needed).
4. Verify the cluster's `argocd-server` `ClusterRole` does NOT prevent
   creating Jobs in `argocd`. Default Argo CD installations allow this; if
   the cluster overrides it, add an explicit allow rule.

**Backwards compatibility for in-flight wedges:**

PR Applications created **before** this fix lands still have the old Job spec
embedded in their PostDelete hook (the hook resolves at Application create
time, so an existing in-flight Application is locked in to the old behavior).
For those, recovery is via the sweep script (§3.6) followed by a manual
finalizer patch. The cluster-infra ApplicationSet template change is what
makes new Applications use the new spec; the in-repo change only matters for
new PRs going forward.

### 3.2 Bug #2 — Drop the dead 24h cleanup-grace (PRD §4.2)

**Decision:** remove the mechanism entirely.

**Concrete changes (in this repo):**

`.github/workflows/pr-cleanup.yml`:

- The block of comments referencing `atlas.cleanup-grace: 24h` and the
  expected 24h delay (lines ~60–79 today) is rewritten. The
  `notify-argo` Job's echo lines lose the "Argo CD's hourly cleanup CronJob
  will set cleanup-deadline" wording. The job is retained as a clean audit
  trail of PR-close timestamps; the misleading text goes.

`docs/runbooks/ephemeral-pr-deployments.md`:

- §9.4 ("Re-running a failed PostDelete") rewritten:
  - Open with the new contract: "PR close (or `deploy-env` label removal)
    triggers teardown immediately. There is no grace window."
  - Move the "wedged Application" diagnostic + recovery procedure under
    a sub-heading "When PostDelete fails", since that procedure is still
    useful (and is what the sweep script in §3.6 codifies).
  - Cross-reference §9.6 (new sweep section).
- §9.5 ("Rotating credentials") rewritten for the new
  `atlas-pr-cleanup-gh-token` (§3.3 below).

**Cluster-infra prerequisites (sibling PR):**

1. ApplicationSet template: drop `metadata.annotations["atlas.cleanup-grace"]`
   and any logic that sets `atlas.cleanup-deadline`.
2. `atlas-pr-cleanup` CronJob: remove deadline-tracking code. Replace with
   "verify no orphan per-env state and emit metric" mode (see §7.4).

### 3.3 Bug #3 — Purpose-built cleanup token (PRD §4.3)

**Decision:** fine-grained Personal Access Token, for v1. GitHub App is
recommended as a follow-up (PRD Open Question 2) — out of scope here.

**Token scopes (least-privilege):**

| Scope | Resource | Why |
|---|---|---|
| `Contents: write` | `Chronicle20/atlas` (single repo) | Delete `bot/pr-N-resolved` refs via `DELETE /repos/.../git/refs/heads/...`. |
| `Packages: write` | `chronicle20/*` (user packages) | Delete per-PR ghcr image tags. The existing `pr-cleanup.yml` already uses an equivalent via the repo's `GHCR_TOKEN` secret; the PostDelete Job's path uses the same scope. |
| `Metadata: read` | `Chronicle20/atlas` | Implicit prerequisite for any other repo scope; not used directly. |

**Not granted:**

- `Administration`, `Workflows`, `Pull requests`, `Issues` — none of the
  cleanup script's calls need these.
- Org-level scopes (`admin:org`) — single-repo PAT is sufficient.

**Secret material:**

- New Kubernetes Secret `atlas-pr-cleanup-gh-token` in `argocd`, single key
  `GHCR_TOKEN` so the existing script needs no rename. The cleanup script
  already reads `GHCR_TOKEN` for ghcr deletion; it gains a second use site
  for branch deletion (currently in the cluster-infra CronJob; the
  PostDelete Job adopts that responsibility too — see §3.1).
- The Reflector source for `ghcr-pat` is unchanged; the PostDelete Job stops
  using it (because it now uses `atlas-pr-cleanup-gh-token` instead). The
  existing `ghcr-pat` Reflector replica is kept for any other consumer; if
  none, it can be deleted in a follow-up.
- Repository secret `GHCR_TOKEN` (used by `.github/workflows/pr-cleanup.yml`)
  is rotated to the same underlying PAT. Workflow context can't read
  in-cluster secrets, so we keep both surfaces, derived from one PAT.

**Concrete changes (in this repo):**

`deploy/k8s/overlays/pr/postdelete-cleanup.yaml`: see diff in §3.1 — the
`envFrom` line swaps `ghcr-pat` for `atlas-pr-cleanup-gh-token`.

`services/atlas-pr-bootstrap/scripts/cleanup.sh`: a new phase added between
ghcr-tag deletion and Pi-hole cleanup:

```bash
if [ -n "${PR_NUMBER:-}" ] && [ -n "${GHCR_TOKEN:-}" ]; then
    ATLAS_STEP=drop-branch log info "deleting bot/pr-${PR_NUMBER}-resolved"
    # 404 is acceptable (branch already gone or never created); other
    # failures are surfaced. Use the new dedicated token (mounted via the
    # atlas-pr-cleanup-gh-token Secret); the variable is still named
    # GHCR_TOKEN so the existing key naming is preserved.
    status=$(gh api --method DELETE \
        -H "Authorization: Bearer ${GHCR_TOKEN}" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
        2>&1) || {
            case "$status" in
                *"Branch not found"*|*"Reference does not exist"*) ;;
                *) log warn "branch delete: $status" ;;
            esac
        }
fi
```

Rationale for adding branch deletion to the PostDelete Job rather than
leaving it in the cluster-infra CronJob: ownership locality. The Job already
knows it's tearing down PR #N; the CronJob has to discover it. Once §4.2 is
implemented, the CronJob loses every reason to talk to GitHub.

**Cluster-infra prerequisites (sibling PR):**

1. Create the `atlas-pr-cleanup-gh-token` Secret from the new PAT.
2. CronJob mount: removed (it no longer needs a GH token; orphan-sweep is
   in-cluster only).
3. Document the rotation procedure in runbook §9.5 (in-repo change).

**Rotation:**

- PAT expiry ≤ 90 days. Document the next-rotation date in §9.5 of the
  runbook (calendar reminder is operator's responsibility).
- Rotation procedure: generate new PAT → update repo secret `GHCR_TOKEN` →
  update cluster Secret `atlas-pr-cleanup-gh-token`. Both sites must move
  together; the smoke test (§3.5) will catch a missed rotation within 24h.

### 3.4 Bug #4 — Defensive `ATLAS_ENV` (PRD §4.4)

This is the load-bearing change.

**Part A — defensive computation (must ship).**

New helper `compute_atlas_env` in
`services/atlas-pr-bootstrap/scripts/lib.sh`:

```bash
# Single source of truth for the env-hash derivation. The formula MUST match
# .github/workflows/pr-validation.yml's update-pr-overlay step (which derives
# the same hash to substitute PLACEHOLDER_ATLAS_ENV) and the ApplicationSet
# template in cluster-infra. A unit test in test/lib_test.bats pins the
# expected outputs for PR 1, 491, 522.
compute_atlas_env() {
    local pr_number="$1"
    if [ -z "$pr_number" ]; then
        log error "compute_atlas_env: empty PR_NUMBER"
        return 1
    fi
    printf "pr-%d" "$pr_number" | sha256sum | cut -c1-4
}
```

`services/atlas-pr-bootstrap/scripts/cleanup.sh` changes:

- Remove `ATLAS_ENV` from `require_env`.
- Add immediately after `require_env`:
  ```bash
  ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
  ATLAS_STEP=init log info "derived ATLAS_ENV=${ATLAS_ENV} for PR ${PR_NUMBER}"
  ```
- All subsequent uses of `${ATLAS_ENV}` are unchanged.

`deploy/k8s/overlays/pr/postdelete-cleanup.yaml`: the `ATLAS_ENV` env entry
is removed (already covered in §3.1 diff). `PR_NUMBER` is the sole per-PR
substitution.

`services/atlas-pr-bootstrap/scripts/bootstrap.sh`: audit. Today bootstrap
gets `ATLAS_ENV` from the `atlas-env` ConfigMap at PreSync time — that
ConfigMap is built by kustomize from already-substituted placeholders, so
the value is correct. Bootstrap does NOT need defensive derivation (the
ConfigMap is the contract surface for the *normal* path; the issue only
occurs at teardown, when the ConfigMap is already gone). Decision: leave
bootstrap as-is. (Plan should explicitly assert this with a one-line audit
comment, not a code change.)

**Part B — investigation (best-effort, timeboxed).**

Tasks, in order, each with a clear exit condition:

1. **`managedFields` walk** — `kubectl get application atlas-pr-491 -n
   argocd -o yaml | yq '.metadata.managedFields[] |
   select(.fieldsV1.f:metadata.f:annotations.f:atlas.env)'` to identify
   which controller(s) own the field. Expected: ApplicationSet controller
   only.
2. **Webhook inventory** — `kubectl get mutatingwebhookconfigurations
   -A -o yaml` and grep for any rule matching
   `argoproj.io/Application`. Expected: none.
3. **Re-render dry-run** — temporarily add a debug annotation
   `atlas.env-debug: '{{ printf "%.4s" (sha256sum (printf "pr-%d"
   .number)) }}'` to the ApplicationSet template, force a regenerate, and
   compare `atlas.env` vs `atlas.env-debug` on the generated Application.
   If they match, the controller is fine and something downstream is
   rewriting; if they differ, the template's eval is the problem.
4. **Stale-template hypothesis** — check git history of the cluster-infra
   ApplicationSet for a previous formula that would emit `f78b`/`d496`.

Timebox: half a day. Deliverable: a markdown writeup in
`docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md`
with:

- Each step's command, output, and conclusion.
- A "root cause" or "inconclusive" line.
- If inconclusive: an explicit "the defensive fix in Part A makes this
  harmless; we accept the drift" closing paragraph.

If a root cause is found and is a one-line fix, fold it into this task. If
it's more, open a follow-up task and link from §10 of `prd.md`.

### 3.5 End-to-end smoke test (PRD §4.5)

`.github/workflows/pr-env-smoke.yml` (new):

**Triggers:** `workflow_dispatch` + nightly `schedule: cron: '17 4 * * *'`.

**Synthetic PR shape:**

- Branch: `smoke/pr-env-YYYYMMDD-HHMMSS` created from `main`.
- Single-file no-op commit (touch
  `docs/smoke-pr-env-touch.txt` with a timestamp; gitignored by the
  smoke-test path so subsequent runs don't accumulate).
- Title prefix `[smoke-test]`, body links to the workflow run.
- Label: `deploy-env` (applied at PR open).

**Phases:**

1. **Open + wait for Healthy.** Create PR, poll
   `gh api repos/.../pulls/<N>` and `kubectl get application atlas-pr-<N>
   -n argocd -o jsonpath='{.status.health.status}'` every 30s, up to 20 min.
   Fails fast if Application doesn't enter Healthy.
2. **Trigger label-removal teardown.** Remove the `deploy-env` label via
   `gh pr edit --remove-label deploy-env`. (Label removal is the path that
   broke for PR #522; cover it specifically.)
3. **Assert reclamation.** Poll every 30s, up to 15 min, for:
   - `kubectl get application atlas-pr-<N> -n argocd` → NotFound.
   - `kubectl get ns atlas-pr-<N>` → NotFound.
   - For ATLAS_ENV = compute_atlas_env(N):
     - psql: `SELECT count(*) FROM pg_database WHERE datname ~ '-<env>$';` → 0.
     - kafka-topics: list | grep `-<env>$` → empty.
     - kafka-consumer-groups: list | grep `\[<env>\]$` → empty.
     - redis: `--scan --pattern '<env>:*' | wc -l` → 0.
   - For each of 5 random services in `services.json`: `gh api` package
     versions with tag `pr-<N>-*` → 0.
   - `gh api /repos/.../git/refs/heads/bot/pr-<N>-resolved` → 404.
4. **Close PR.** Always (even on failure) close the PR and delete the
   smoke branch.
5. **Artifacts on failure.** `kubectl describe application
   atlas-pr-<N> -n argocd` and last 200 lines of `atlas-pr-cleanup` Job
   logs uploaded to `actions/upload-artifact`.

**Where the workflow runs:**

The runner needs kubectl + psql + kafka-consumer-groups.sh + redis-cli +
gh access to the cluster. Options:

| Option | Pro | Con |
|---|---|---|
| Self-hosted runner inside cluster | Direct cluster access; cheap. | Need to maintain a runner pod. |
| GitHub-hosted + WireGuard/Tailscale to cluster | No self-hosted dep. | Tunnel setup is non-trivial; secret sprawl. |
| GitHub-hosted + a "smoke driver" Job in cluster | The workflow only triggers the in-cluster Job and polls its result. | Two-step orchestration; failure attribution is harder. |

**Decision:** **self-hosted runner** in `argocd` namespace, single replica,
gated on the cluster operator labeling a node with
`atlas.smoke-runner: "true"`. Tagged `runs-on: [self-hosted, atlas-cluster]`.

If no self-hosted runner is available when the plan executes, the workflow
file is still written and committed, but `runs-on` is left as a TODO with
an explicit `if: false` so the workflow can be turned on later. Plan must
flag this as a known gap.

**Repository hygiene:**

- PR numbers are consumed even on success; this is unavoidable but
  acceptable (Chronicle20/atlas already uses ~600 PRs).
- Smoke PRs are titled `[smoke-test]` so they're filterable.

### 3.6 Sweep script (PRD §4.6)

`services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` (new):

```
Usage:
  sweep-orphans.sh [--apply] [--list] PR_NUMBER [PR_NUMBER ...]

Without --apply (default): enumerates per-env state that would be deleted.
With --apply: deletes it. Idempotent — running twice is safe.

For each PR number:
  1. Compute ATLAS_ENV = compute_atlas_env(N)
  2. Postgres: list/drop DBs matching '-<env>$'
  3. Kafka: list/delete topics matching '-<env>$'
  4. Kafka: list/delete consumer groups matching '\[<env>\]$'
  5. Redis: list/delete keys matching '<env>:*'
  6. ghcr: list/delete image tags matching 'pr-<N>-*' across services.json
  7. Pi-hole: list/delete '<N>.atlas.home' host entries from both servers
  8. Application: patch out finalizers if 'kubectl get application
     atlas-pr-<N> -n argocd' still returns something.
  9. Bot branch: DELETE /repos/.../bot/pr-<N>-resolved
```

**Design points:**

- Reuses `lib.sh`'s `log` / `require_env` / new `compute_atlas_env`.
- All connection details (DB host, Kafka bootstrap, Redis, Pi-hole APIs,
  GH token) come from env vars, defaulted to the same names cleanup.sh
  uses. Operator invokes via `kubectl exec` into a one-shot pod (recipe
  documented in §9.6 runbook) or locally with port-forwards.
- `--list` (default) prints each resource that *would* be deleted, one
  per line, prefixed by step name. Suitable for piping through diff for
  visual review before re-running with `--apply`.
- The Application-finalizer patch is the same `metadata.finalizers: []`
  merge patch used in the May-19 recovery.
- Idempotent on absent resources (same `IF EXISTS`, `xargs -r`, `|| true`
  discipline as `cleanup.sh`). A re-run after a clean sweep must exit 0
  with all enumerations empty.

**Tests:**

`services/atlas-pr-bootstrap/test/sweep_test.bats`:

- Argument parsing (missing PR number, multiple PRs).
- `--list` mode emits a stable, parseable line shape per resource.
- `compute_atlas_env(491)` == `ed86` (recovery-log oracle).
- A "smoke" test that boots ephemeral postgres + kafka + redis containers
  in CI, seeds the per-env tables/topics/keys, runs the sweep with
  `--apply`, and asserts everything is gone. (Same pattern as
  cleanup_test.bats but exercising sweep instead.)

---

## 4. Data contract

The hash formula:

```
ATLAS_ENV = sha256( "pr-<N>" )[:4]
```

This must be identical in:

1. The ApplicationSet template in cluster-infra (goTemplate
   `{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}`).
2. `.github/workflows/pr-validation.yml`'s `update-pr-overlay` step (today:
   `printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4`).
3. `services/atlas-pr-bootstrap/scripts/lib.sh::compute_atlas_env` (new).

**Test:** `test/lib_test.bats::compute_atlas_env_matches_oracle` asserts
the table:

| PR | ATLAS_ENV |
|---|---|
| 1 | (computed) |
| 491 | `ed86` |
| 522 | `a476` |

If the ApplicationSet's `goTemplate` formula ever changes, that breaks
both the workflow and `compute_atlas_env`. The test pins the contract so
the next refactor can't drift silently.

---

## 5. Failure modes & defence-in-depth

| Failure | Pre-fix outcome | Post-fix outcome |
|---|---|---|
| Argo CD prunes namespace before PostDelete fires | Wedge; nothing cleaned. | Job in `argocd` (long-lived) — runs normally. |
| `atlas.env` annotation drifts to a wrong value | Cleanup keys off annotation → wrong DBs/topics/groups/keys deleted (or, today, also wedge). | Cleanup derives from `PR_NUMBER` — annotation drift is harmless. |
| GitHub PAT lacks `Contents: write` | Branch never deleted; silent 403. | Dedicated PAT with `Contents: write`; branch deleted on first PostDelete. |
| Job exits non-zero mid-run | Hook stuck; namespace already gone; manual recovery. | Same Job failure → same hook stuck condition. Recovery path: sweep script + finalizer patch (now documented in §9.4/§9.6). |
| Image registry transient failure during ghcr deletion | Job exits 1; entire teardown stuck. | Ghcr step still uses `\|\| true`; transient failures are logged but don't fail the Job. Operator can re-run via sweep. |
| Bot branch missing (race with `pr-validation.yml`) | 404; gh exits non-zero; Job fails. | New code handles 404 as success (branch-not-found case). |
| `update-pr-overlay` substitutes a wrong `ATLAS_ENV` into pod env | Pod uses wrong env hash; orphans state on bootstrap. | Out of scope for this task — bootstrap reads `ATLAS_ENV` via ConfigMap and the substitution path is the source of truth at create time. (Spec PR #4.4 Part B is the investigation.) |

---

## 6. Migration plan

Per the PRD's backward-compatibility requirement (§8), in-flight PR
Applications must not be orphaned by this change.

**Sequencing:**

1. **Land cluster-infra Secret + ServiceAccount first.** Just creating
   `atlas-pr-cleanup-gh-token` Secret and the `atlas-pr-cleanup` SA/Role
   is a no-op for current Applications.
2. **Land this repo's PR (the task-070 branch).** New PRs opened after
   this point use the new `postdelete-cleanup.yaml`. Existing PRs are
   still wedge-prone if they close before step 4.
3. **Run the sweep script** against any currently-wedged Applications
   (today's snapshot shows zero, per recovery log §"Pre-existing orphan
   inventory at end of session"). If any reappear between steps 2 and 4,
   sweep them.
4. **Land cluster-infra ApplicationSet change** (drop
   `atlas.cleanup-grace`; remove deadline tracking from cron). The
   ApplicationSet PR's diff is reviewable independently.
5. **Run smoke test** manually via `workflow_dispatch` to validate
   end-to-end. Schedule nightly thereafter.

**Rollback:**

If step 2 lands and a structural problem emerges, revert the
`postdelete-cleanup.yaml` diff (single file) and re-deploy. PRs opened
during the bad window can be cleaned via the sweep script. Reverting does
not break any cluster-infra Secrets/ServiceAccounts (they sit unused).

---

## 7. Non-functional requirements

### 7.1 Idempotency

Every step in `cleanup.sh` and `sweep-orphans.sh` must be safe to run
twice. Today's `cleanup.sh` already preserves `IF EXISTS`, `xargs -r`,
`|| true`. The new branch-delete step handles 404 explicitly. Sweep
inherits the same discipline.

### 7.2 Observability

- `log info/warn/error` per phase, with `ATLAS_STEP=<phase>` and the
  derived `ATLAS_ENV`. Existing pattern.
- A new `init` log line emits `ATLAS_ENV=<computed> for PR <N>` so the
  Job's logs unambiguously show which env hash was targeted.
- `kubectl describe application atlas-pr-<N>` continues to surface
  PostDelete failures via `status.conditions`. No suppression.

### 7.3 Security

- `atlas-pr-cleanup-gh-token` PAT scopes are exactly those listed in
  §3.3.
- Secret lives in `argocd` only; no Reflector replication needed (the
  Job now runs there).
- The cluster-infra `argocd-repo-creds-chronicle20-atlas` Secret is
  untouched.
- Runbook §9.5 documents the rotation procedure and expiry date for the
  PAT.

### 7.4 Metric: `atlas_pr_orphan_envs_total`

(Implementation in cluster-infra; contract specified here.)

- **Type:** Prometheus Counter (monotonically increasing per scrape) or
  Gauge (current orphan count). Counter chosen so an alert can fire on
  rate > 0 without resetting per scrape.
- **Labels:** `pr_number` (string), `kind` (one of: `application`,
  `database`, `topic`, `consumer_group`, `redis_key`, `image_tag`,
  `bot_branch`).
- **Source:** the orphan-sweep mode of the cluster-infra CronJob (§3.2)
  scans state and emits one metric line per orphan discovered.
- **Alert wiring is out of scope.** The metric must be scrapeable from
  the CronJob's Prometheus exporter sidecar; runbook §9.6 includes the
  PromQL the operator can paste into Grafana.

### 7.5 Reliability

- PostDelete Job's wall-clock budget: 10 min normal (30 DBs × ~1–2s,
  150 topics × <1s, etc.). Smoke test allows 15 min as a margin.
- Job's `backoffLimit: 0` is preserved — a failed Job must surface
  loudly, not silently retry. (`HookSucceeded` GC removes the Pod only
  on success.)

### 7.6 Multi-tenancy

N/A — cluster-level infra.

---

## 8. Open questions resolved

| PRD § | Question | Decision |
|---|---|---|
| 4.1 | Which option for the finalizer-ordering fix? | (a) move Job to `argocd`. |
| 4.3 | GitHub App vs PAT for cleanup token? | Fine-grained PAT for v1. App is a follow-up task. |
| 4.5 | Synthetic-PR repo: real vs sandbox? | Real `Chronicle20/atlas`, titled `[smoke-test]`. |
| 4.6 | Pi-hole cleanup in sweep? | Yes. |
| 4.4-B | Reconciler if drift unidentified? | No reconciler. Defensive code makes drift harmless. |

`tools/task-numbers.sh next` brokenness (PRD §9-Q6) is explicitly out of
scope.

---

## 9. Test plan summary

| Surface | Test |
|---|---|
| `lib.sh::compute_atlas_env` | `test/lib_test.bats::compute_atlas_env_matches_oracle` (PR 1, 491, 522). |
| `cleanup.sh` ATLAS_ENV derivation | Existing `cleanup_test.bats` updated: no longer requires `ATLAS_ENV` in env; asserts the computed value gets logged. |
| `cleanup.sh` branch-delete phase | `test/cleanup_test.bats::branch_delete_handles_404` — mocks `gh api` to return a 404 body; expects Job to continue and log warn=no. |
| `sweep-orphans.sh` argument parsing | `test/sweep_test.bats::missing_pr_number`, `multiple_prs`. |
| `sweep-orphans.sh` end-to-end | `test/sweep_test.bats::full_sweep_in_ephemeral_infra` (boots postgres/kafka/redis in CI, seeds, sweeps, asserts empty). |
| Workflow | `.github/workflows/pr-env-smoke.yml` — manual + nightly. |
| Runbook | The "force-cleanup" recipe (§9.4) is exercised by deliberately patching out a Job and walking the recovery; documented as a test step in the runbook itself, not automated. |

---

## 10. Out of scope (deferred)

- GitHub App migration for the cleanup token (PRD Open Question 2).
- Alert wiring on `atlas_pr_orphan_envs_total` (PRD §4.5 — metric only,
  alert is the operator's call).
- Cluster-wide orphan sweep of envs that pre-date this task (PRD §2,
  non-goal). Sweep script is the tool; operationalizing it across the
  cluster is for the operator post-merge.
- Per-PR copies of Postgres/Kafka/Redis (PRD §2, non-goal).
- Fixing `tools/task-numbers.sh next` (PRD Open Question 6).
- A defensive `atlas.env` reconciler controller (PRD §4.4 Open
  Question 5 — rejected; defensive derivation suffices).

---

## 11. Cross-references

- PRD: `prd.md`.
- Recovery log: `recovery-log.md` (May 19, 2026 manual sweep of
  PR 491 + 522).
- Cluster-infra repo: ApplicationSet, CronJob, Secret manifests — sibling
  PR coordinated with this one.
- Runbook: `docs/runbooks/ephemeral-pr-deployments.md` §9.4, §9.5, new
  §9.6 (sweep).
- Investigation deliverable (best-effort):
  `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md`
  (created during execution).
