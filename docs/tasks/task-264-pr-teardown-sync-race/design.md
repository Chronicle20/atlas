# PR-Environment Teardown Sync Race — Design

Version: v1
Status: Draft
Created: 2026-08-26
Inputs: `prd.md` (v1), `incident-pr-1459.md`
---

## 0. What changed since the PRD

The PRD left three questions open and proposed a deadline value from a capped
sample. All of them were resolved against the live cluster and Loki on
2026-08-26 between 12:50Z and 13:10Z. Two findings change the shape of the
design:

1. **`activeDeadlineSeconds` fires on a Job that is already marked for deletion
   and pinned by a finalizer.** This was not obvious — the Job controller stops
   creating pods once `deletionTimestamp` is set, so it was plausible that a
   deleted-but-finalized Job could never reach a terminal condition and FR-1
   would be inert against exactly the case it targets. It was measured (§2.1)
   and it is not inert. FR-1 is therefore a *guarantee*, not a mitigation, and
   it is sufficient on its own to bound the deadlock.
2. **The Application CRD has no `status` subresource, and the existing
   `atlas-pr-cleanup` Role already grants `patch` on `applications`.** A
   terminate-op reconciler needs *zero new RBAC*. This collapses FR-3 from "the
   largest unresolved design question" into a small, cheap, cluster-infra-owned
   CronJob.

Everything below is grounded in those measurements. Nothing is inferred from
remembered Argo CD or Kubernetes behavior.

---

## 1. Failure model

The deadlock is a three-party interaction, and it is worth naming each party
precisely because the fix for each is different.

**Party 1 — the hook Job that cannot terminate.** `Job/atlas-pr-bootstrap`
carries `argocd.argoproj.io/hook-finalizer`, applied by the Argo application
controller. Only the controller removes it, and only as part of completing or
terminating the owning operation.

**Party 2 — the operation that cannot complete.** The operation's phase is
driven by its hook batch. `status.operationState.message` on the live wedged
Application reads exactly:

```
waiting for completion of hook batch/Job/atlas-pr-bootstrap
```

Confirmed live at 12:52Z, still wedged 11h after the delete request:

```json
{
  "finalizers": ["resources-finalizer.argocd.argoproj.io",
                 "post-delete-finalizer.argocd.argoproj.io/cleanup",
                 "pre-delete-finalizer.argocd.argoproj.io/cleanup",
                 "post-delete-finalizer.argocd.argoproj.io"],
  "hasOperation": false,
  "opPhase": "Running",
  "opMsg": "waiting for completion of hook batch/Job/atlas-pr-bootstrap",
  "startedAt": "2026-08-26T11:48:57Z"
}
```

**Party 3 — the deletion that cannot proceed.** `resources-finalizer` blocks on
the undeletable Job, so the remaining 89 namespaced objects are never issued for
deletion.

The cycle closes because the *only* edge that can break it — the controller
reaping the hook finalizer — is gated on the operation reaching a terminal
phase, and the operation's terminal phase is gated on the hook Job reaching a
terminal condition. **Break either gate and the whole thing unwinds.** The two
independent ways to break it are:

- **(A)** force the hook Job terminal → operation transitions → finalizer reaped.
  This is FR-1.
- **(B)** force the operation terminal directly → controller reaps hook
  finalizers as part of termination. This is FR-3.

FR-2 is a different class: it removes the *precipitating* cause (the images the
hook waits on being deleted out from under it) rather than the deadlock
mechanism. All three are worth doing; only (A) and (B) bound the failure.

---

## 2. Decisions

### 2.1 FR-1 — `activeDeadlineSeconds` (DECIDED: 900s, on `spec`)

#### The load-bearing question, and its measurement

The incident timeline shows the hook Job created at 12:16:24 and marked for
deletion at 12:26:27 — 603 seconds later. Any deadline longer than 603s would
*not* have fired before Argo issued the delete. If a Job under
`deletionTimestamp` can no longer reach a terminal condition, then a 900s
deadline would have been useless on the very incident that motivated this task,
and the correct value would have to be shorter than the delete-issuance delay —
an unstable number we do not control.

This was tested directly on the live cluster (k3s, server v1.35.5+k3s1), in a
throwaway `t264-probe` namespace, reproducing the exact shape: a Job with a
blocking finalizer, deleted while its pod was still running, with the deadline
scheduled to fire *after* the deletion.

```yaml
# activeDeadlineSeconds: 60, backoffLimit: 3, finalizer: example.com/probe-finalizer
# container: sleep 3600   (only the deadline can terminate it)
```

Observed:

```
--- t+20 deleting job ---
job.batch "probe-deadline" deleted
t+30 = active=1 failed= del=2026-08-26T12:57:08Z
t+40 = active=1 failed= del=2026-08-26T12:57:08Z
t+50 = active=1 failed= del=2026-08-26T12:57:08Z
t+60 FailureTarget=DeadlineExceeded active= failed= del=2026-08-26T12:57:08Z
```

and the final object:

```json
"cond": [
  {"type":"FailureTarget","reason":"DeadlineExceeded","status":"True",
   "lastTransitionTime":"2026-08-26T12:57:39Z",
   "message":"Job was active longer than specified deadline"},
  {"type":"Failed","reason":"DeadlineExceeded","status":"True",
   "lastTransitionTime":"2026-08-26T12:58:10Z",
   "message":"Job was active longer than specified deadline"}
]
```

**Conclusion: the deadline fires and the Job reaches terminal `Failed` even
though it was deleted 40 seconds earlier and is still pinned by a finalizer.**
FR-1.5 is satisfiable. Note the 31s gap between `FailureTarget` and `Failed`
(12:57:39 → 12:58:10) — the finalizer delays the pod-finalizer sweep that
promotes the condition. Budget for it: the operation transitions roughly
`deadline + ~30s` after Job start, not at `deadline` exactly.

This is the single most important fact in the design, and it is why FR-1 is
listed as the primary fix rather than as one of three co-equal mitigations.

#### Deriving the value

The PRD's distribution was re-measured with pagination, as §9 asked. Loki was
port-forwarded and `{container="bootstrap"}` was queried in 120 consecutive
6-hour windows over 30 days, each with `limit=5000`; per-pod duration is
`max(ts) - min(ts)`.

Total lines returned across all 120 windows: **4663**. No single window came
close to the 5000 cap (busiest was 153 lines). **The PRD's "5000-line-capped
sample" caveat is retired — this is the full population, not a sample.**

All 150 bootstrap pods:

| stat | value |
|---|---|
| min | 0s |
| p50 | 56s |
| p90 | 253s |
| p95 | 362s |
| p99 | 382s |
| max | 542s |

The 104 pods carrying a terminal `"atlas.step":"done"` line (successful runs —
the population the deadline must not truncate):

| stat | value |
|---|---|
| n | 104 |
| min | 6s |
| p50 | 68s |
| p90 | 115s |
| p95 | 162s |
| p99 | 542s |
| max | 542s |

Top of the successful tail: `…-kdstp` 252.8s, `…-2tctt` 382.2s, `…-55c29`
541.7s, `…-9ds65` 542.4s. Everything below `…-kpxht` (176.4s) is under three
minutes.

**Caveat on the measurement, stated because it moves the number:** this is the
span of the container's *log* stream. It excludes image-pull time before the
first line and any scheduling delay. The Job's `activeDeadlineSeconds` clock
starts earlier than the first log line, so the true Job wall-clock exceeds these
figures by the pull time. Headroom must absorb that.

**Decision: `activeDeadlineSeconds: 900`.** Confirming the PRD's proposal, now
against the full population rather than a capped sample:

- 1.66× the observed successful maximum (542s), leaving 358s of absolute
  headroom for image pull and scheduling on the slowest run ever recorded.
- 5.5× p95 (162s). 97% of successful runs finish in under 4 minutes; they are
  nowhere near this bound.
- Caps worst-case teardown block at 900s + ~30s ≈ 15.5 minutes, against the
  runbook §9.4 contract of "~10 minutes" for the full reclaim. It is the same
  order of magnitude, not a new class of delay.

**Rejected: 600s.** Only 1.1× the observed max, with zero room for image pull.
`…-55c29` and `…-9ds65` would both be at real risk of being killed while
healthy, which violates the PRD's fourth user story ("a slow-but-healthy
bootstrap should still succeed").

**Rejected: shorter-than-delete-issuance (~500s).** This was the value the
design would have been forced into had the probe come out the other way. The
probe removed the constraint, so there is no reason to pay its flake cost.

#### Placement

`spec.activeDeadlineSeconds`, not `spec.template.spec.activeDeadlineSeconds`
(FR-1.4). The Job-level field bounds the aggregate lifetime across all
`backoffLimit: 3` retries; the pod-level field bounds each attempt
independently and would permit 4 × deadline total — precisely the semantics the
PRD rules out.

**Accepted trade-off:** with `backoffLimit: 3` retained (the PRD does not scope
changing it) and a 900s aggregate cap, a run at the 542s tail that fails once
will be killed partway through its retry rather than getting a full second
attempt. This is deliberate: a bootstrap that has already burned nine minutes
and exhausted its 60-attempt `wait-ready` loop is not usually rescued by a
retry — the PR-1459 pod failed with `retry exhausted after 60 attempts` and no
retry would have found the deleted images. Bounding total hook lifetime is worth
more than preserving retry depth for the slowest 2% of runs.

#### Scope note — other hooks are equally unbounded

Six hook manifests exist in `deploy/`; **none** sets `activeDeadlineSeconds`:

```
deploy/k8s/base/atlas-minio-init.yaml                    (PreSync)
deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml   (PostDelete)
deploy/k8s/overlays/pr/postsync-pihole-add.yaml          (PostSync)
deploy/k8s/overlays/pr/predelete-purge.yaml              (PreDelete, backoffLimit: 0)
deploy/k8s/overlays/pr-sparse/postsync-pihole-add.yaml   (PostSync)
deploy/k8s/overlays/pr-sparse/predelete-purge.yaml       (PreDelete, backoffLimit: 0)
```

The deadlock mechanism is a property of *any* Argo hook Job, not of bootstrap
specifically. The PRD scopes FR-1 to the two `sync-bootstrap.yaml` files, and
this design honors that scope — but the exposure is wider, and closing only the
observed hole leaves five open.

**Recommendation, flagged for the user rather than assumed:** bound the two
`postsync-pihole-add.yaml` hooks in the same change (they are a DNS API call —
a 300s deadline is generous) and leave the PreDelete/PostDelete hooks alone
(PreDelete is `backoffLimit: 0` and best-effort by design per the PRD's
non-goals; PostDelete runs in `argocd` and is covered by
`ttlSecondsAfterFinished` plus §9.4). `atlas-minio-init.yaml` is in `base/`, so
touching it affects `atlas-main` and is genuinely out of scope for a PR-teardown
task. **If the user does not want the pihole hooks in scope, FR-1 as written
still stands on its own.**

### 2.2 FR-2 — remove `delete-images` (DECIDED: straight removal)

No design tension here; the PRD's analysis holds. `delete-images` in
`.github/workflows/pr-cleanup.yml` and `cleanup.sh::do_drop_images` match the
same `pr-<N>-*` tag prefix over the same `services.json`-derived service list.
One runs on PR close, racing every in-flight sync; the other runs PostDelete,
after the namespace is pruned. The PostDelete one is correct and the workflow
one is a duplicate that actively causes the incident's proximate failure:

```
Failed to pull image "ghcr.io/chronicle20/atlas-data/atlas-data:pr-1459-99281705": ... not found
```

Delete the `delete-images` job (FR-2.1) and `notify-argo`'s `needs:
[delete-images]` (FR-2.2). With `delete-images` gone, `notify-argo` has no
remaining dependency and becomes the workflow's only job.

The replacement comment (FR-2.3) sits where `delete-images` was and mirrors the
structure of the existing "Branch deletion intentionally NOT done here" block
immediately below it — same failure class (workflow deletes an artifact the
Argo teardown still needs), same explanation shape, cross-referenced to §9.4.
That block is the precedent the PRD names, and matching it keeps the file
readable as a single "here is everything we deliberately do *not* do on PR
close" section.

No change to `do_drop_images` (FR-2.4).

**Note for the plan phase:** `.github/workflows/pr-cleanup.yml` will still
reference `secrets.GHCR_TOKEN` nowhere after this change. Runbook §9.5's
rotation procedure step 3 (`gh secret set GHCR_TOKEN --repo Chronicle20/atlas`)
becomes obsolete and must be dropped in the same runbook pass as FR-4, or the
next operator half-rotates a secret nothing reads. This is a real consequence of
FR-2 that the PRD's §7 table does not list.

### 2.3 FR-3 — terminate in-flight operations (DECIDED: reconciler CronJob in `argocd`)

The PRD calls this "the largest unresolved design question" and offers three
branches. It is now resolvable.

#### What terminate-op actually does, and the trap in reproducing it

The incident recorded an operator error that is the key to getting this right:

```sh
kubectl -n argocd patch application atlas-pr-1459 --type=merge -p '{"operation":null}'
```

This made the deadlock **permanent**. Clearing `.operation` removes the
operation spec without transitioning `status.operationState.phase`, which stayed
`Running`. The controller only processes an operation when `app.Operation !=
nil`, so after that patch nothing could ever transition the phase or reap the
hook finalizer. The live Application still shows `hasOperation: false` with
`opPhase: "Running"` — 11 hours later, unrecoverable by any operation-level
action.

The correct semantic — what `argocd app terminate-op` performs — is the
opposite: set `status.operationState.phase = "Terminating"` while **leaving
`.operation` in place**. The controller, still processing because `Operation !=
nil`, then runs its termination path: it terminates the sync, reaps hook
resources and their finalizers, sets the phase terminal, and clears
`.operation`. Any mechanism that reproduces terminate-op must preserve that
invariant, and must be inert when `.operation` is already nil (patching a
zombie's phase to `Terminating` produces another zombie, not a recovery).

#### Feasibility — measured

```
$ kubectl get crd applications.argoproj.io -o json | jq '.spec.versions[] | {name, subresources}'
{ "name": "v1alpha1", "subresources": {} }

$ kubectl -n argocd get role atlas-pr-cleanup -o json | jq -r .rules
[{"apiGroups":["argoproj.io"],"resources":["applications"],
  "verbs":["get","list","delete","patch"]}]
```

The CRD declares **no status subresource**, so `status` is writable by an
ordinary `patch` on the main resource — no `--subresource=status`, no separate
verb. And the `atlas-pr-cleanup` ServiceAccount in `argocd` (created
2026-05-14, live now) already holds `get`, `list`, and `patch` on
`applications`. **The terminate-op reconciler requires no new RBAC whatsoever.**

Argo CD in this cluster is `quay.io/argoproj/argocd:v3.4.2`
(`statefulset/argocd-application-controller`), which pins the version any
behavioral claim is made against.

#### Options considered

**Option A — declarative field on the ApplicationSet template.** Rejected: Argo
CD exposes no "terminate-before-delete" field on an Application or an
ApplicationSet template. The PRD says this; it is confirmed by the CRD schema
having no such property. There is nothing to configure.

**Option B — Argo CD version upgrade.** Rejected as the *primary* mechanism.
The cluster is already on v3.4.2, a current release, and it exhibits the
behavior. An upgrade is not a fix for a defect present in the newest version we
run, and gating a teardown fix on a control-plane upgrade is a much larger
blast radius than the problem warrants.

**Option C — a PreDelete hook in this repo that terminates the operation.**
Seriously considered and rejected on RBAC topology. It is attractive because it
would keep FR-3 entirely inside the atlas repo, and the evidence says PreDelete
hooks *do* run during a wedged teardown — `atlas-pr-predelete-purge` executed
at 12:26:26 and its Job carries no finalizer today, so its hook completed and
was reaped even while the sync operation was stuck. Ordering also works:
predelete ran at 12:26:26 and the bootstrap Job was marked for deletion at
12:26:27, i.e. PreDelete completes immediately *before* the resources-finalizer
phase that then blocks. A terminate-op there would land at the right moment.

It fails on identity. The hook runs in `atlas-pr-<N>` under a namespace-local
ServiceAccount and must patch an Application in `argocd`. That requires a
RoleBinding in `argocd` naming a subject in a namespace whose name is not known
until the PR exists. RBAC has no wildcard for
`system:serviceaccounts:atlas-pr-*`; the alternatives are binding the whole
`system:serviceaccounts` group (a cluster-wide privilege escalation for a
teardown convenience — not acceptable) or having the per-PR overlay create a
RoleBinding into `argocd` from an Application whose destination namespace is
`atlas-pr-<N>` (cross-namespace resource creation that inverts the ownership
boundary §9.13 establishes). Neither is worth it when Option D needs no new
grant at all.

**Option D — a reconciler CronJob in `argocd`, cluster-infra owned. CHOSEN.**

A single cluster-wide CronJob, `atlas-pr-terminate-stuck-ops`, in the `argocd`
namespace, running as the existing `atlas-pr-cleanup` ServiceAccount on a
`* * * * *` schedule. Each run:

1. Lists Applications in `argocd` matching `atlas-pr-*`.
2. Selects those where `metadata.deletionTimestamp != null` **and**
   `status.operationState.phase == "Running"` **and** `.operation != null`.
3. For each, patches `status.operationState.phase` to `"Terminating"`, leaving
   `.operation` untouched.
4. Logs and skips any Application where `.operation == null` — the zombie state.
   That case is unrecoverable at the operation level and belongs to the §9.4
   Job-finalizer recipe (FR-4.3), not to this job.

Properties that make this the right shape:

- **Zero new RBAC.** Verified above. The coordination artifact is a manifest to
  apply, not a permissions negotiation.
- **Idempotent and self-limiting.** The predicate is false for every healthy
  Application, and false again once the controller has moved the phase off
  `Running`. A stuck run costs one list and zero patches.
- **Reuses an established pattern.** `argocd` already hosts two cluster-wide
  singleton CronJobs for this exact lifecycle — `atlas-pr-cleanup` (`5 * * * *`)
  and `atlas-pr-sweep-orphans` (`0 */6 * * *`), both confirmed live. This is a
  third, and it must carry the same singleton warning task-045's artifact
  records: it belongs in cluster-infra, never in this repo's per-PR overlays,
  which CI renders once per PR into N copies.
- **Respects FR-3.4.** It touches only `status.operationState.phase`. The
  Application's own finalizers are never modified, so PostDelete cleanup still
  reclaims DBs, topics, groups, Redis keys, GHCR tags, DNS, and the bot branch.

**Why every-minute rather than folding into `atlas-pr-cleanup`'s hourly run:**
the whole point is bounding wedge duration. An hourly reconciler bounds it at
60 minutes, which is worse than FR-1's 15. A minutely one bounds it at ~1 minute
and makes FR-3 the *tighter* of the two bounds, which is what defense in depth
should look like.

#### Standing of FR-3 after the FR-1 probe

The PRD's third branch — "accept FR-1 as the sole mitigation and treat FR-3 as
defense in depth" — is now the accurate description of the risk, but it is not a
reason to drop FR-3. FR-1 is proven sufficient for the observed failure, and
FR-3 costs one manifest and no new grant. Keep both. They fail independently:
FR-1 is inert if a future hook is added without a deadline (five such hooks
exist today, §2.1), and FR-3 is inert if `.operation` has already been cleared
by an operator. Neither covers the other's gap.

#### Coordination artifact (FR-3.3)

`dev/cluster-infra-coordination/terminate-stuck-ops-cronjob.example.yaml`, plus
a `task-264-terminate-op.md` note alongside `task-045-teardown.md`, following
that file's structure: what lives where, what RBAC is required (here: none, and
say so explicitly with the Role dump as evidence), the singleton warning, and
merge ordering.

**Merge ordering.** §9.13's rule is that cluster-infra lands *before* the
consuming atlas PR, because an atlas change that depends on a missing
cluster-infra object wedges. That rule does not bind here: nothing in this
repo's diff references the CronJob, and FR-1/FR-2/FR-4 are self-contained. The
CronJob is additive and inert-safe in either order. **This repo's PR can land
independently** — same posture as task-045's PreDelete hook, and the artifact
should say so in as many words rather than leaving an operator to infer a
blocking dependency that does not exist.

### 2.4 FR-4 — runbook (DECIDED: new §9.4 subsection + §9.6 rewrite)

#### §9.4

A new subsection under §9.4, placed **after** the existing "Recover" block and
before "Source-branch-missing scenario", titled for the signal an operator will
actually have in hand — the repeating non-decreasing object count.

It must be visually distinct from the existing recipe, because the existing
recipe is *wrong for this failure* in a way that costs real state: §9.4 step 2
drops the Application's finalizers, which would strip
`post-delete-finalizer.argocd.argoproj.io/cleanup` and skip PostDelete cleanup
entirely, leaking the per-PR databases. The new subsection leads with that
contrast rather than burying it.

Content, per FR-4.2 through FR-4.4:

- **Signal.** `status.operationState.phase == Running` with `message: waiting
  for completion of hook batch/Job/<name>`, alongside a controller log line
  `N objects remaining for deletion` whose N never decreases across reconciles.
  Both quoted from the live incident so an operator can string-match them.
- **First move, while `.operation` still exists:** `argocd app terminate-op`.
- **The trap:** `kubectl patch ... -p '{"operation":null}'` is **not**
  equivalent and makes the wedge permanent — it removes the operation spec
  without transitioning `status.operationState.phase`, and the controller only
  processes an operation when `app.Operation != nil`. Record that this was
  attempted on PR #1459 and that the Application remains unrecoverable at the
  operation level 11 hours later.
- **Recovery once `.operation` is gone**, the narrow patch that preserves
  PostDelete:

  ```sh
  kubectl -n atlas-pr-<N> patch job <hook-job> \
      --type=merge -p '{"metadata":{"finalizers":null}}'
  ```

  Find the Job by its retained finalizer rather than by name, since the wedging
  hook need not be bootstrap:

  ```sh
  kubectl -n atlas-pr-<N> get jobs -o custom-columns=\
  'NAME:.metadata.name,DEL:.metadata.deletionTimestamp,FIN:.metadata.finalizers'
  ```

  which on the live incident prints:

  ```
  NAME                       DEL                    FIN
  atlas-minio-init           <none>                 <none>
  atlas-pr-bootstrap         2026-08-26T12:26:27Z   [argocd.argoproj.io/hook-finalizer]
  atlas-pr-predelete-purge   <none>                 <none>
  ```

- **Forward reference** to the FR-1 deadline and the FR-3 CronJob, so a future
  operator seeing this in a *bounded* environment knows the wedge should
  self-clear in ~15 minutes and that hand-patching is now an escalation rather
  than the routine path.

#### §9.6

Both queries are stale; both are corrected in place rather than deleted,
because §9.6 is the section a future deadline-tuning pass will open first.

**PromQL.** `atlas_bootstrap_step_duration_ms_bucket` does not exist. The PRD
verified the metric is absent from Prometheus (922 metric names, zero matching
`atlas` or `bootstrap`). Mark the query **not implemented**, state that the
instrumentation was never emitted, and point at the Loki method below as the
working substitute. Do not silently delete it — leaving a labeled dead query
documents the gap; deleting it loses the fact that someone intended this metric.

**LogQL.** The documented selector `{atlas_env="a3f7", job=~"atlas-pr-bootstrap"}`
matches nothing. Loki's live label set is:

```
__stream_shard__, container, instance, job, level, namespace, pod, service, service_name
```

with `job` having exactly one value, `loki.source.kubernetes.pod_logs`. There is
no `atlas_env` label — `atlas.env` is a *field inside the JSON payload*, which
is why it must be reached through `| json` and not through a stream selector.

The replacement was executed live and returns data:

```logql
{container="bootstrap", namespace="atlas-pr-<N>"} | json | atlas_step != ""
```

```json
{"ts":"2026-08-26T12:21:34Z","level":"info","atlas.env":"2a03",
 "atlas.step":"wait-ready",
 "msg":"waiting for atlas-tenants, atlas-configurations, atlas-data, atlas-renders"}
```

Add the duration-distribution recipe used in §2.1 — 6-hour windowed
`query_range` over `{container="bootstrap"}`, per-pod `max(ts) - min(ts)`,
filtered to pods carrying `"atlas.step":"done"` — since that is the method any
future retune of `activeDeadlineSeconds` must repeat, and it is not obvious.

**Retention caveat, and it matters.** The earliest bootstrap log in Loki is
`2026-08-12T10:36:26Z` — **14 days, not the 30 the query window assumed.** The
30-day query silently returned 14 days of data. Any future retune has at most a
fortnight of history, and if PR volume drops the population shrinks with it.
Record this in §9.6; it is the strongest argument for the missing metric
(PRD §9 open question 3) and it is not visible from the query results
themselves.

---

## 3. Open questions, resolved

| # | PRD question | Resolution |
|---|---|---|
| 1 | Exact deadline value; re-measure with pagination? | **Re-measured.** 120 × 6h windows, 4663 lines, no window near the 5000 cap — full population, not a sample. n=104 successful: p50 68s, p95 162s, max 542s. **900s confirmed** (§2.1). |
| 2 | Terminate-op mechanism — automation, upgrade, or drop FR-3? | **Resolved: Option D**, a minutely reconciler CronJob in `argocd` on the existing `atlas-pr-cleanup` SA. No new RBAC (CRD has no status subresource; Role already grants `patch`). Options A/B/C rejected with reasons (§2.3). |
| 3 | Fold missing `atlas_bootstrap_step_duration_ms` into this task? | **Recommend separate**, per the PRD's own non-goal. But §9.6 must record the 14-day Loki retention that makes the metric matter — see §4 for the one open call. |
| 4 | §9.6 LogQL correction — confirm against a live query. | **Confirmed live.** `{container="bootstrap", namespace="atlas-pr-<N>"} \| json \| atlas_step != ""` returns data; the label set has no `atlas_env` (§2.4). |
| 5 | Backfill — other wedged Applications? | **Swept.** Four Applications exist cluster-wide: `atlas-main` (Succeeded/Synced), `atlas-pr-1459` (**wedged**, deletionTimestamp 12:12:50Z, phase Running), `atlas-pr-1461` (no operation), `myfleet-main` (Succeeded/Synced). **#1459 is the only one.** |

---

## 4. What still needs the user

Two calls, neither blocking the bulk of the work:

1. **Scope of FR-1 — the other unbounded hooks (§2.1).** Five further hook Jobs
   have no deadline. Recommend adding a 300s bound to the two
   `postsync-pihole-add.yaml` files in this change and leaving the rest. This
   widens the PRD's stated scope, so it is the user's call, not mine. FR-1 as
   written is complete without it.
2. **`atlas-pr-1459` is still wedged right now.** It has been in the
   unrecoverable zombie state (`.operation` cleared, phase `Running`) for 11
   hours as of 12:52Z. It will not self-clear, and no change in this task
   retroactively fixes an Application that has already lost its `.operation`.
   Clearing it needs the FR-4.3 Job-finalizer patch applied by hand. That is an
   operational action against the live cluster, outside the deliverable — say
   the word and it is one command, or it can wait for the runbook to land and be
   used as the recipe's first live exercise.

Everything else in the PRD is decided and ready to plan.

---

## 5. Verification strategy

Beyond `tools/verify.sh` exiting 0:

- **FR-1** is already empirically validated at the Kubernetes layer (§2.1). The
  remaining question is the Argo layer — that a `Failed/DeadlineExceeded` hook
  Job actually drives the operation terminal and releases `hook-finalizer`. This
  cannot be proven from the repo and must be exercised on a real PR env, which
  is what acceptance criterion "verified end-to-end on a real PR" asks for.
  Plan it as: open a throwaway PR, break its bootstrap deliberately (a bad
  `ATLAS_UI_BASE`), close the PR mid-sync, and confirm teardown completes
  unattended within ~16 minutes.
- **FR-2** is `actionlint` on `.github/workflows/pr-cleanup.yml` plus confirming
  `notify-argo` still parses with no `needs`.
- **FR-3**'s reconciler predicate should be dry-run against the live cluster
  before the artifact is committed: the list-and-filter half is read-only and
  can be executed as-is to confirm it selects `atlas-pr-1459` and nothing else.
- **FR-4** — every command in the new §9.4 subsection is copied from a command
  that was actually run against the live incident, not composed. Keep it that
  way; a runbook recipe that has never been executed is a hypothesis.
