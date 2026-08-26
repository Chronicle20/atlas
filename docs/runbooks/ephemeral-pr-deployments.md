# Ephemeral per-PR Deployments — Runbook

Operational guide for the per-PR atlas environments. PRD / design /
implementation plan: `docs/tasks/task-063-ephemeral-pr-deployments/`.

> **See also:** [canonical-version-migration.md](canonical-version-migration.md)
> for the one-time migration that provisions per-version canonical baselines consumed
> by the baseline-only bootstrap this runbook describes.

## §9.1 Data provisioning: baseline-only

Ephemeral PR envs provision game data **only** by restoring the published
canonical baseline for their `(region, major, minor)`. There is no WZ
re-ingest path in the bootstrap — `bootstrap.sh` calls
`POST /api/data/baseline/restore` and nothing else. This keeps PR envs fast
(~60s restore, no ~1 GB download, no ~10-min ingest) and guarantees no
ephemeral env ever writes a per-tenant WZ/asset tree into shared MinIO.

### Fail-fast on a missing baseline

Before any data-affecting work, `bootstrap.sh` runs a read-only preflight
that HEADs both the baseline dump and its sha256 sidecar in MinIO:

```
HEAD $MINIO_ENDPOINT/atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump
HEAD $MINIO_ENDPOINT/atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump.sha256
```

- **Both 200** → the bootstrap proceeds.
- **Either 404** → the Job exits non-zero with a single greppable line:
  `no canonical baseline for <region> <major>.<minor> … publish one … before deploying this env`.
  Argo CD surfaces the failed Job; the env does not come up half-seeded.
- **MinIO unreachable (000)** after a bounded retry → a *distinct* `MinIO unreachable`
  error (so a transient blip is never misread as "go publish a baseline").

Cold-starting a brand-new version therefore requires publishing its baseline
**first** — that is the [canonical-version-migration](canonical-version-migration.md)
runbook (its step 4, `POST /api/data/baseline/publish`). Publishing the
baseline is a prerequisite for any PR env on that version, not an optimization.

### Stand up MinIO (one-time)

MinIO is where the canonical baselines live. Apply the manifest from the
cluster-infra repo, then wait for the Deployment:

```sh
kubectl apply -f <infra-repo>/minio.yml
kubectl rollout status -n minio deployment/minio --timeout=120s
```

The `atlas-canonical` bucket has an anonymous-read policy (set by
`atlas-minio-init`), so the bootstrap's preflight needs no credentials. The
baselines themselves are produced and consumed by `baseline/publish` and
`baseline/restore` (atlas-data) — see the migration runbook for the publish
procedure. There is no `atlas.zip` to upload.

## §9.1b Cross-namespace Secret replication (Reflector)

Per-PR hook Jobs (`atlas-pr-create-dbs`, `atlas-pr-pihole-register`,
`atlas-pr-cleanup`) reference Secrets that live in `atlas-main` or `argocd`:

- `db-credentials` (in `atlas-main`) — Postgres user + password for the
  create-dbs Job and the cleanup Job's DB-drop step.
- `pihole-credentials` (in `argocd`) — Pi-hole API tokens for DNS
  registration / deregistration.
- `ghcr-pat` (in `argocd`) — GitHub PAT for deleting per-PR image tags
  on env teardown.

Kubernetes Secrets are namespace-scoped, so PR namespaces need their own
copies. [Reflector](https://github.com/emberstack/kubernetes-reflector) is
a small controller that watches annotated Secrets and auto-creates copies
in matching namespaces.

### Stand up Reflector (one-time)

From the cluster-infra repo:

```sh
kubectl apply -f <infra-repo>/reflector.yml
kubectl rollout status -n reflector deployment/reflector --timeout=120s
```

### Annotate the source Secrets (one-time)

The source Secrets need these annotations to enable replication into
`atlas-pr-*` namespaces:

```yaml
metadata:
  annotations:
    reflector.v1.k8s.emberstack.com/reflection-allowed: "true"
    reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces: "atlas-pr-.*"
    reflector.v1.k8s.emberstack.com/reflection-auto-enabled: "true"
    reflector.v1.k8s.emberstack.com/reflection-auto-namespaces: "atlas-pr-.*"
```

For the live cluster (one-shot kubectl annotate, since the source Secrets
were created out-of-band):

```sh
kubectl annotate secret db-credentials -n atlas-main \
    reflector.v1.k8s.emberstack.com/reflection-allowed=true \
    reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces=atlas-pr-.* \
    reflector.v1.k8s.emberstack.com/reflection-auto-enabled=true \
    reflector.v1.k8s.emberstack.com/reflection-auto-namespaces=atlas-pr-.*

kubectl annotate secret pihole-credentials -n argocd \
    reflector.v1.k8s.emberstack.com/reflection-allowed=true \
    reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces=atlas-pr-.* \
    reflector.v1.k8s.emberstack.com/reflection-auto-enabled=true \
    reflector.v1.k8s.emberstack.com/reflection-auto-namespaces=atlas-pr-.*

kubectl annotate secret ghcr-pat -n argocd \
    reflector.v1.k8s.emberstack.com/reflection-allowed=true \
    reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces=atlas-pr-.* \
    reflector.v1.k8s.emberstack.com/reflection-auto-enabled=true \
    reflector.v1.k8s.emberstack.com/reflection-auto-namespaces=atlas-pr-.*
```

Future re-applies of the source Secret manifests preserve these
annotations — see `deploy/k8s/base/secrets.example.yaml` (for
`db-credentials`) and `<infra-repo>/argocd-secrets.yml.example` (for
`pihole-credentials` + `ghcr-pat`).

### Verify replication

After annotations are in place, replicated copies should appear in any
existing `atlas-pr-*` namespace within seconds:

```sh
kubectl get secret -n atlas-pr-<N> db-credentials pihole-credentials ghcr-pat
```

For new PR namespaces, the copies appear as soon as the namespace is
created by Argo CD's ApplicationSet.

## §9.2 Force-cleanup of a PR env

Removing the `deploy-env` label or closing the PR triggers immediate teardown — there is **no grace window**. If a teardown wedges, see §9.4 for recovery and §9.11 for the orphan-sweep script.

To stop a running env without closing the PR:

```sh
gh pr edit <N> --remove-label deploy-env
```

The ApplicationSet drops its generator entry on the next reconcile (~30s), Argo CD deletes the Application, and the PostDelete Job in the `argocd` namespace reclaims per-env state within ~10 minutes.

Verify in-flight cleanup:

```sh
kubectl -n argocd get jobs -l app=atlas-pr-cleanup
kubectl -n argocd logs -l app=atlas-pr-cleanup --tail=200
```

If you specifically need to force-delete an Application that is stuck (i.e., the ApplicationSet's generator still points at it), see §9.4.

## §9.3 Inspecting a stuck env

```sh
argocd app get atlas-pr-<N>
kubectl get all,configmap,secret -n atlas-pr-<N>
kubectl logs -n atlas-pr-<N> job/atlas-pr-bootstrap
```

Loki query for env-scoped logs (`atlas.env=<token>`):

```logql
{atlas_env="a3f7"} |= ""
```

## §9.4 Recovery when teardown wedges

**Contract:** PR close (or `deploy-env` label removal) ⇒ Argo CD deletes the Application immediately ⇒ the PostDelete Job in `argocd` namespace runs `cleanup.sh` ⇒ all per-env state (DBs, topics, groups, Redis keys, ghcr tags, bot branch) is reclaimed within ~10 minutes.

The PostDelete Job is named `atlas-pr-cleanup-<N>` in the `argocd` namespace (per-PR suffix as of task-077). Pre-task-077 the Job was a literal `atlas-pr-cleanup` for every PR — combined with `hook-delete-policy: HookSucceeded` (a Failed Job is never garbage-collected) this caused cross-PR head-of-line blocking: one PR's Failed cleanup would wedge every subsequent PR's teardown with `SharedResourceWarning: Job/atlas-pr-cleanup is part of applications argocd/atlas-pr-<X> and atlas-pr-<Y>`. Look up Jobs by label rather than by name (`kubectl -n argocd get jobs -l app=atlas-pr-cleanup,atlas.pr-number=<N>`) so existing recipes survive future renames.

**Retention: 24 hours.** `hook-delete-policy: HookSucceeded` never reaps a `PostDelete` hook — Argo deletes hook resources during a reconciliation of the owning Application, but once the PostDelete hook succeeds Argo drops the Application's finalizer and the Application object is gone, so no later reconcile exists to collect the Job. The Job has no ownerReferences and lives in the long-lived `argocd` namespace, so neither cascading GC nor namespace deletion catches it either. Every PR therefore leaked one Job plus its pod into `argocd` permanently (~75 accumulated before this was noticed). The Job now sets `ttlSecondsAfterFinished: 86400`, so the k8s TTL-after-finished controller reaps it — Complete and Failed alike — 24h after it finishes. **Pull cleanup logs within that window**; after it, the Job and its pod are gone and the diagnose recipes below return `No resources found`. If you need the logs to outlive the TTL, grab them before the window closes.

If something in that chain fails, the Application sits in `Terminating` with finalizers `post-delete-finalizer.argocd.argoproj.io/cleanup` and `resources-finalizer.argocd.argoproj.io` still present. Per-env state may be partially reclaimed.

### Diagnose

**Read the summary line first.** As of task-075, `cleanup.sh` runs every
phase regardless of any single phase's outcome. As of task-48, the phase
list also includes `deactivate` (always first — FR-5.5: routing stops
before any destructive phase runs) and `drop-control-plane` (reclaims this
environment's `atlas-configurations` services/tenants/templates rows and
its `atlas-tenants` row; a no-op that logs "skipped (isolated)" outside
sparse mode, since `drop-dbs` already destroys the whole per-env database
there). The final log line is the authoritative status:

```
{"ts":…,"level":"info","atlas.env":"…","atlas.step":"done","msg":"cleanup complete phases_run=9 phases_failed=0"}
```

or, on partial failure:

```
{"ts":…,"level":"error","atlas.env":"…","atlas.step":"done","msg":"cleanup completed with errors phases_run=9 phases_failed=2 failed_phases=[\"drop-topics\",\"drop-redis\"]"}
```

Use the `failed_phases` array to scope your re-run — only the listed
phases need a manual recovery pass.  Every other phase ran to
completion (look for its `phase complete` log line). Pre-task-075
runbooks said "assume every phase after the failed one was skipped"; that
assumption no longer applies.

```sh
kubectl -n argocd get application atlas-pr-<N> -o yaml | yq '.status.conditions'
kubectl -n argocd get jobs -l app=atlas-pr-cleanup,atlas.pr-number=<N>
kubectl -n argocd logs -l app=atlas-pr-cleanup,atlas.pr-number=<N> --tail=500
```

Common signals:

- `DeletionError: namespaces "atlas-pr-<N>" not found` — should not happen post-task-070; if it does, the cluster-infra ApplicationSet was rolled back. File an incident.
- The PostDelete Job is `Failed` with logs showing a specific phase (e.g. `drop-topics`) erroring on a missing dep — fix the dep, re-run via the sweep (§9.11).
- `cleanup.sh` ran to completion but `kubectl get application` still shows the Application — finalizer wasn't drained because the Job container exited non-zero on a non-critical step. Patch the finalizers (below).

### Recover

```sh
# 1. (If state is suspected leaked.) Run the orphan sweep in list mode,
#    review output, then re-run with --apply. See §9.11.
sweep-orphans.sh <N>          # list
sweep-orphans.sh --apply <N>  # reclaim

# 2. Drop the Application's finalizers so the CRD can be removed.
kubectl -n argocd patch application.argoproj.io atlas-pr-<N> \
    --type=merge -p '{"metadata":{"finalizers":[]}}'

# 3. (If the bot branch survived.) The sweep script handles this, but the
#    manual command is:
gh api --method DELETE \
    /repos/Chronicle20/atlas/git/refs/heads/bot/pr-<N>-resolved
```

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

### Source-branch-missing scenario

If the PostDelete render fails with `unable to resolve 'bot/pr-<N>-resolved' to a commit SHA`, the Application targets a branch that no longer exists. Diagnose: `kubectl -n argocd get application atlas-pr-<N> -o yaml | yq '.status.conditions[] | select(.message | contains("ComparisonError"))'`. Recovery is the same finalizer patch (step 2 above) followed by the sweep (step 1) — the branch is already gone so `drop-branch` is a no-op.

**As of task-078, `cleanup.sh::do_drop_branch` pre-empts this race itself** — after a successful branch delete it patches the Application's `post-delete-finalizer.argocd.argoproj.io[/cleanup]` finalizers off so Argo CD can GC the Application without ever needing to re-render the now-missing source. The runbook step above is still the manual recovery for legacy Applications that were torn down before this fix, or for clusters where `atlas-pr-cleanup` SA was denied the patch permission. Observed first on PR 522 on 2026-05-27 — Application sat Terminating for 10h before manual finalizer-patch.

## §9.5 Rotating credentials

All Argo CD-related Secrets live in the `argocd` namespace and are templated by `argocd-secrets.yml.example` in the cluster-infra repo. To rotate:

- **`atlas-pr-cleanup-gh-token` (PR-env cleanup PAT).** Used by the PostDelete Job for bot-branch deletion and ghcr image-tag deletion. Fine-grained PAT minted under the `Chronicle20` user account.

  Mint the token at *Settings → Developer settings → Personal access tokens → Fine-grained tokens → Generate new token*:

  - **Resource owner:** `Chronicle20`.
  - **Repository access:** *Only selected repositories* → `Chronicle20/atlas`.
  - **Repository permissions:**
    - **Contents** → *Read and write* — enables `DELETE /repos/Chronicle20/atlas/git/refs/heads/bot/pr-<N>-resolved`.
    - **Metadata** → *Read-only* (mandatory; auto-selected).
    - everything else → *No access*.
  - **Account permissions:**
    - **Packages** → *Read and write* — enables `DELETE /users/chronicle20/packages/container/<svc>/versions/<vid>` against ghcr.
    - everything else → *No access*.
  - **Expiration:** ≤ 90 days; operator calendars the next rotation.

  Rotation procedure:

  ```sh
  # 1. Mint a new PAT with the scope set above.
  # 2. Update the cluster secret. This is the only consumer.
  kubectl -n argocd edit secret atlas-pr-cleanup-gh-token   # set key GHCR_TOKEN
  ```

  **There is no repo-secret half to this rotation.** Until task-264,
  `.github/workflows/pr-cleanup.yml` carried a `delete-images` job that read a
  `GHCR_TOKEN` repository secret, and this procedure had a third step to
  rotate it. That job was removed — it raced the Argo teardown by deleting
  ghcr tags an in-flight sync was still pulling, and it duplicated
  `cleanup.sh::do_drop_images`, which does the same work PostDelete.
  `pr-cleanup.yml` reads no `GHCR_TOKEN` secret now. **Do not delete the
  repository-level `GHCR_TOKEN` secret itself** — it is a distinct,
  still-live secret read by `ghcr-cleanup.yml`, `main-publish.yml`,
  `pr-env-smoke.yml`, and `pr-validation.yml` for unrelated image-publish and
  smoke-test work; only this rotation procedure's third step is gone.

  The nightly smoke test (§4.5 / `pr-env-smoke.yml`) will catch a missed half-rotation within 24h.

  **If your GitHub plan does not expose Account-level `Packages` on fine-grained PATs:** mint a classic PAT instead (*Tokens (classic) → Generate new token (classic)*) with the `repo` scope and `delete:packages` (which auto-selects `read:packages`). Classic PATs are broader (whole-user repo write) but reliably support GHCR package deletion. Document the choice when rotating so the next operator knows which type to renew.

- **GitHub PAT for Argo source-repo creds:** `kubectl edit secret argocd-repo-creds-chronicle20-atlas -n argocd`, replace `password`. ApplicationSet picks up on next reconcile (~30s). This token does NOT need `Contents: Read and write` (the cleanup PAT above owns branch deletion).

- **Pi-hole tokens:** `kubectl edit secret pihole-credentials -n argocd`. Source Secret lives in `argocd` and is Reflector-replicated to every `atlas-pr-*` namespace. The PostSync register Job (in `atlas-pr-<N>`) reads the replica; the PostDelete cleanup Job (in `argocd`) reads the source directly. Rotation takes effect on the next PR sync.

- **Database credentials (`db-credentials`):** source Secret lives in `atlas-main` and is Reflector-replicated to `atlas-pr-.*|argocd` (the per-PR namespaces AND `argocd` so the PostDelete cleanup Job can read it). `kubectl edit secret db-credentials -n atlas-main`; Reflector pushes the change to all replicas within seconds.

- **ghcr-pat (legacy).** No longer used by the PostDelete Job (replaced by `atlas-pr-cleanup-gh-token`). If no other consumer needs it, remove it in a cluster-infra follow-up.

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

## §9.7 Hash-collision resolution

Two open PRs hash to the same 4-hex `ATLAS_ENV` token. Symptom: the second PR's Application sync fails with a namespace conflict.

Workaround: close-and-reopen one PR — head SHA changes (or force-push to perturb the head). Long-term mitigation: bump the suffix to 6 hex by editing the `ApplicationSet(atlas-pr)` template (`printf "%.6s"` instead of `%.4s`).

## §9.8 main env cutover (one-time)

Pre-flight check that the rendered overlay matches the live cluster:

```sh
kustomize build deploy/k8s/overlays/main > /tmp/built.yaml
kubectl get -n atlas all,configmap -o yaml > /tmp/live.yaml
yq eval-all 'select(fileIndex == 0) - select(fileIndex == 1)' \
    /tmp/built.yaml /tmp/live.yaml
```

Expected: only Kustomize-injected labels are net new.

Cutover steps:

1. Apply Argo CD on the cluster per the cluster-infra repo's `argocd.yml` header comment (install upstream Argo CD, `--insecure` patch, IngressRoute, secrets, longhorn-pr StorageClass).
2. Apply `<infra-repo>/argocd-atlas.yml` — Argo creates `Application(atlas-main)` with `prune: false`. Wait for `Synced/Healthy` with zero diffs.
3. Drain the legacy `atlas` namespace, rename Postgres DBs from `atlas-<svc>` to `atlas-<svc>-main` in place, flush legacy Redis keys, optionally drop legacy Kafka topics, delete the `atlas` namespace.
4. Argo reconciles `atlas-main` from the rendered overlay against the renamed DBs.
5. Wait ~7 days of clean syncs. Edit the Application section of `<infra-repo>/argocd-atlas.yml` to set `prune: true`. Reapply.

## §9.9 Adding a service after cutover

Follow `deploy/k8s/README.md`'s "Adding a new service" section — the patch generators must be re-run so `consumer-group-env.yaml` and `db-name-suffix.yaml` include the new entry.

## §9.10 PR env doesn't get scheduled

ApplicationSet only generates an Application for PRs carrying the `deploy-env` label. To request an env for a PR:

```sh
gh pr edit <N> --add-label deploy-env
```

Within ~30s, Argo CD's PR generator polls GitHub and creates the Application.

To stop a PR's env without closing the PR: remove the label, then force-delete the Application (§9.2).

## §9.11 Orphan sweep

For PR-envs whose teardown wedged or pre-dated the task-070 fixes, `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` enumerates and (with `--apply`) deletes every leaked artifact.

### One-shot from a workstation

For one-off recovery you can run the image directly from a workstation
with cluster credentials (kubeconfig pointing at the prod cluster's
`argocd` namespace). The Job manifest form below mirrors the
PostDelete cleanup Job's shape (envFrom the cluster-infra-owned
ConfigMap; PR_NUMBER as the only per-invocation override). It is
preferred over `kubectl run --rm -i` — non-TTY pods don't always stream
logs reliably, and a Job leaves an inspectable record.

Apply this manifest (substitute `PR_NUMBER`):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  generateName: atlas-pr-cleanup-oneshot-
  namespace: argocd
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      serviceAccountName: atlas-pr-cleanup
      containers:
        - name: cleanup
          image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
          command: ["/atlas/cleanup.sh"]
          envFrom:
            - secretRef: { name: db-credentials }
            - secretRef: { name: pihole-credentials }
            - secretRef: { name: atlas-pr-cleanup-gh-token }
            - configMapRef: { name: atlas-pr-cleanup-env }
          env:
            - name: PR_NUMBER
              value: "<PR_NUMBER>"
```

Pipe through `kubectl -n argocd create -f -` (no `apply`; oneshot
Jobs use `generateName`). Tail logs with:

```bash
kubectl -n argocd logs -l app.kubernetes.io/part-of=atlas-pr-cleanup --tail=-1 -f
```

The workstation no longer needs to export `DB_HOST`, `BOOTSTRAP_SERVERS`,
`ATLAS_DB_NAMES`, `ATLAS_SERVICES`, etc. — those come from the
cluster-infra-owned `atlas-pr-cleanup-env` ConfigMap. `PR_NUMBER` is
the only value you supply.

### In-cluster (preferred for production cluster credentials)

`/atlas/sweep-orphans.sh` is part of the published bootstrap image as of
task-075. The legacy `kubectl create configmap` + script-mount workaround
is no longer needed.

```bash
kubectl -n argocd run sweep-orphans \
    --rm -i --restart=Never \
    --serviceaccount=atlas-pr-cleanup \
    --image=ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest \
    --overrides='{
      "spec": {
        "containers": [{
          "name": "sweep-orphans",
          "image": "ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest",
          "command": ["/atlas/sweep-orphans.sh", "--apply", "<PR_NUMBER>"],
          "envFrom": [
            {"secretRef": {"name": "db-credentials"}},
            {"secretRef": {"name": "pihole-credentials"}},
            {"secretRef": {"name": "atlas-pr-cleanup-gh-token"}},
            {"configMapRef": {"name": "atlas-pr-cleanup-env"}}
          ]
        }]
      }
    }'
```

Drop `--apply` (or pass `--list` explicitly) to enumerate without
deleting. The script's Kafka phases use rpk as of task-075; the previous
"kafka-topics.sh not on PATH; skipping" warning is gone.

Idempotent — re-running on an already-clean PR exits 0 with all enumerations empty. The script tolerates absent infrastructure (it skips any phase whose required env var is unset), so partial-credential invocations also work for diagnosing one subsystem at a time.

### Metric (cluster-infra)

The cluster-infra `atlas-pr-cleanup` CronJob's orphan-sweep mode emits `atlas_pr_orphan_envs_total{pr_number,kind}` (counter). Operator dashboard query:

```promql
sum by (kind) (atlas_pr_orphan_envs_total)
```

Alert wiring is out of scope for task-070 — this is observable but not paged.

### Known follow-ups (post task-071)

- ~~`cleanup.sh` does not currently invoke `DELETE /api/data/tenants/<id>` because `TENANT_ID` is not injected into the cleanup environment~~ — addressed by task-078 (#596): `cleanup.sh` now has a `drop-tenant-storage` phase that reads tenant UUIDs from atlas-data's still-alive Postgres database and `mc rm`'s the per-tenant MinIO prefixes BEFORE the `drop-dbs` phase tears the database down. See §9.14.

### Per-tenant cleanup architecture

#### Teardown-integrated (primary, post-#596)

The PostDelete cleanup Job (`atlas-pr-cleanup-<N>` in argocd ns) runs a new `drop-tenant-storage` phase BEFORE `drop-dbs`:

1. Reads tenant UUIDs from the still-alive `atlas-data-<env>` Postgres database via `SELECT DISTINCT tenant_id FROM tenant_baselines`. The DB is alive at PostDelete time because the same cleanup Job's `drop-dbs` phase is what eventually destroys it — ordering matters.
2. For each tenant, runs `mc rm --recursive --force bee/{atlas-wz,atlas-assets,atlas-renders}/tenants/<id>/`. These are the same MinIO buckets/prefixes that atlas-data's `tenantpurge.Purge` would clean if it were callable (atlas-data is gone at PostDelete time, but its DB isn't).
3. The Postgres half of `tenantpurge.Purge` (DELETE FROM per-tenant tables) is moot here — the next phase (`drop-dbs`) drops the entire `atlas-data-<env>` database.

Implementation notes:

- Required env on the cleanup Job: `MINIO_ENDPOINT` (added to the `atlas-pr-cleanup-env` ConfigMap — cluster-infra mirrors from `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`) and `MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` (from `minio-root-creds` Secret reflected from `minio` ns; `optional: true` on the envFrom so cleanup doesn't wedge in clusters where the reflection isn't wired up).
- Cleanup is one phase among eight; failure is logged + tolerated, not fatal (matches the rest of the script's idiom). `sweep-orphans.sh --minio` is the operator backstop for cases where the cleanup Job itself doesn't run (force-evicted before completing, manual finalizer-strip, etc.).
- No additional RBAC needed — the cleanup Job already has `db-credentials` to reach Postgres, and `mc` ships in the bootstrap image (also added in task-078).
- atlas-data does NOT need a SIGTERM handler, ClusterRole for `namespaces: get`, or extended `terminationGracePeriodSeconds`. Routine atlas-data pod restarts preserve tenants; only the PostDelete Job during env teardown wipes them.

#### `--minio` (backstop, also primary on pre-#596 leaks)

`sweep-orphans.sh --minio` (added task-078, issue #596) enumerates UUID-prefixed paths under `atlas-wz/tenants/`, `atlas-assets/tenants/`, `atlas-renders/tenants/` and deletes any UUID that is BOTH:

- not present in atlas-main's `atlas-tenants` REST listing (the long-lived env's tenant UUIDs — these are the allowlist), AND
- aged past `MINIO_TENANT_SAFETY_WINDOW_SEC` (default 7200s / 2h) — covers in-flight bringups whose data-ingest is still writing.

Refusal-to-act guards: an empty atlas-main tenant list aborts the sweep (refuses to operate on an empty allowlist). Missing `mc` aborts. Missing `MINIO_ENDPOINT` or credentials silently skips (matches the rest of the script's idiom).

```bash
# Dry-run from a workstation with cluster kubeconfig
kubectl -n argocd run sweep-minio --rm -i --restart=Never \
    --serviceaccount=atlas-pr-cleanup \
    --image=ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest \
    --overrides='{
      "spec": {
        "containers": [{
          "name": "sweep-minio",
          "image": "ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest",
          "command": ["/atlas/sweep-orphans.sh", "--minio"],
          "envFrom": [
            {"configMapRef": {"name": "atlas-pr-cleanup-env"}}
          ],
          "env": [
            {"name": "MINIO_ENDPOINT", "value": "minio.minio.svc.cluster.local:9000"},
            {"name": "MINIO_ACCESS_KEY", "valueFrom": {"secretKeyRef": {"name": "minio-root-creds", "key": "MINIO_ROOT_USER"}}},
            {"name": "MINIO_SECRET_KEY", "valueFrom": {"secretKeyRef": {"name": "minio-root-creds", "key": "MINIO_ROOT_PASSWORD"}}}
          ]
        }]
      }
    }'

# Replace `--minio` with `--minio --apply` once the list looks correct.
```

`minio-root-creds` is in the `minio` namespace; reflect it (or copy) to `argocd` first if running from there. Operators can also `mc rm --recursive --force <alias>/<bucket>/tenants/<id>/` per-UUID for surgical recovery.

## §9.12 Diagnosing partial-cleanup failure

As of task-075 the PostDelete Job runs every phase regardless of any
single phase's outcome. The summary line names which phases failed:

```
cleanup completed with errors phases_run=9 phases_failed=2 failed_phases=["drop-topics","drop-redis"]
```

Re-run only the failed phases via the §9.11 sweep-orphans path with
`--apply`, or manually:

| Phase | Manual re-run |
|---|---|
| `deactivate` | **Highest priority — routing may still be live.** `curl -X PATCH -H 'Content-Type: application/vnd.api+json' -H "ENVIRONMENT: <env>" -d '{"data":{"type":"environments","id":"<env>","attributes":{"baseline":"<baseline>","namespace":"<namespace>","tenant":"<tenant>","overrides":<overrides-json>,"phase":"DEACTIVATING"}}}' <baseline-ui-base>/api/configurations/environments/<env>`, wait ~35s, then repeat with `"phase":"DELETED"`. GET the record first (`.../api/configurations/environments/<env>`) to fill `<baseline>`/`<namespace>`/`<tenant>`/`<overrides-json>` from its current values — omitting them zeroes the record (§9.4 background). |
| `drop-control-plane` | Reclaims leaked `services`/`tenants`/`templates` (atlas-configurations) and tenant (atlas-tenants) rows for this environment. Re-run: `for res in configurations/services configurations/tenants configurations/templates tenants; do curl -H "ENVIRONMENT: <env>" "<baseline-ui-base>/api/$res?page[size]=250" \| jq -r --arg env "<env>" '.data[]? \| select(.attributes.environment == $env) \| .id' \| xargs -r -I{} curl -X DELETE -H "ENVIRONMENT: <env>" "<baseline-ui-base>/api/$res/{}"; done` (loop `page[number]` if `meta.page.last` > 1). Never delete a row whose `attributes.environment` isn't exactly `<env>` — that field is what keeps this scoped away from `main`. |
| `drop-dbs` | `psql -h postgres.home -U <user> -c 'DROP DATABASE IF EXISTS "atlas-<base>-<env>";'` (per leaked DB) |
| `drop-topics` | `rpk topic list -X brokers=kafka.home:9093 --format json \| jq -r '.[].name' \| grep -- '-<env>$' \| xargs -r -n1 rpk topic delete -X brokers=kafka.home:9093` |
| `drop-groups` | `rpk group list -X brokers=kafka.home:9093 --format json \| jq -r '.[].name' \| grep -- '\[<env>\]$' \| xargs -r -d '\n' -n1 rpk group delete -X brokers=kafka.home:9093` |
| `drop-redis` | `redis-cli -u redis://redis.home:6379 --scan --pattern '<env>:*' \| xargs -r -n 1000 redis-cli -u redis://redis.home:6379 DEL` |
| `drop-images` | See §9.5 GHCR token; the image-cleanup phase of `/atlas/sweep-orphans.sh --apply <PR>` is the canonical re-run path |
| `drop-dns` | Pi-hole admin UI on each replica; remove A records ending `… <PR_NUMBER>.atlas.home` |
| `drop-branch` | `gh api --method DELETE /repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-<PR>-resolved` |

The full re-run path (`/atlas/sweep-orphans.sh --apply <PR>`) is
idempotent and is the recommended recovery — it touches every phase
again with `WHERE NOT EXISTS`-equivalent semantics. The per-phase
recipes above are for cases where the operator wants to address a
single phase in isolation (e.g. the rpk broker is the only thing that
was unavailable during cleanup).

## §9.13 Coordination with cluster-infra

This repo (`Chronicle20/atlas`) deploys per-PR resources into
`atlas-pr-<N>` namespaces. Long-lived `argocd`-namespace dependencies
are owned by the cluster-infra repo. The atlas repo expects these to
already exist in `argocd`:

- `ServiceAccount atlas-pr-cleanup` + `Role` / `RoleBinding` granting
  the PostDelete Job permission to query+patch Applications.
- `Secret atlas-pr-cleanup-gh-token` (fine-grained PAT for GHCR + bot
  branch delete).
- `ConfigMap atlas-pr-cleanup-env` — shape mirrored from
  `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.

When a new service is added to `.github/config/services.json`,
`gen-cleanup-env.sh` regenerates the example artifact and CI fails
the PR until the artifact is committed. Once that PR merges,
cluster-infra mirrors the new shape into the live ConfigMap. Order
of merges matters: cluster-infra changes land BEFORE the consuming
atlas PR, otherwise the next PostDelete Job wedges with
`CreateContainerConfigError: configmap "atlas-pr-cleanup-env" not found`.

## §9.14 Adding (or removing) a game version

A version's login/channel ports are derived from its `majorVersion`
(`loginPort = major × 100`, `channelPort = loginPort + 1`) by one shared
formula (`services/atlas-pr-bootstrap/scripts/version-ports.sh`). Two places
that used to be hand-maintained are now generated from a single declared list.

**To expose a new version on the LoadBalancers:**

1. Add the version to `deploy/k8s/base/versions.json`:
   ```json
   { "region": "gms", "majorVersion": 84, "minorVersion": 1 }
   ```
   (Two versions may not share a `majorVersion` — they would collide on the
   same port; the generator rejects this.)
2. Regenerate the manifests:
   ```bash
   tools/gen-lb-ports.sh
   ```
   This rewrites the `# BEGIN/END generated:*` blocks in
   `deploy/k8s/base/atlas-{login,channel}.yaml`. Nothing outside the markers
   changes. CI (`gen-lb-ports --check`) fails any PR where these drift.
3. Commit both the `versions.json` edit and the regenerated manifests, then
   redeploy the base.

The tenant row and its per-tenant configuration still have to exist:
ephemeral envs get them from `atlas-pr-bootstrap`; persistent envs from the
UI Templates → Clone flow. The declared version set only controls **LB
exposure**.

**Additive bootstrap guarantee.** `atlas-pr-bootstrap` now upserts only its
canonical tenant into the live `services` config (keyed by tenant id) and
leaves every other tenant entry untouched. A second version added by hand in
an ephemeral env **survives every bootstrap re-run** — its socket listener
and its per-tenant Kafka consumers are no longer drained. (Previously the
bootstrap rebuilt `tenants[]` from a template and clobbered the second
version, leaving its consumers drained so clients logged in and hung.)

**Coexistence verification (manual repro).** With v83 (canonical) + v84
(hand-added) both present in the `services` config, re-run the bootstrap and
confirm in the login/channel logs:
- `projection.applied op=add` for **both** tenants, and
- **no** `projection.applied op=drain` for v84,
then connect a v84 client and confirm the login handshake completes (no hang).

## §9.15 Sparse vs. isolated mode, and the per-PR override labels

`tools/mode-select.sh` (task-232 FR-9.2–9.5) decides, per PR, whether the
`deploy-env`-labeled environment is:

- **sparse (default)** — only the changed services plus the mandatory floor
  (`atlas-login`, `atlas-channel` — FR-9.4/D6) are deployed as
  Deployments; everything else in the namespace is served by `main`
  (the shared baseline environment). This is what `mode-select.sh`'s
  no-trigger branch resolves to — the changed-file set mapping cleanly to a
  small, known service set is the common case, not the exception. Full
  operational detail (gate counters, the P0 leakage alert, the unmeasured
  fan-out cost, the MetalLB pool ceiling, the NetworkPolicy dependency) is
  in `docs/runbooks/sparse-environments.md`, not duplicated here.
- **isolated (escalation)** — every service is deployed into the PR's own
  namespace, nothing shared with `main`. This is the documented escalation
  path, triggered automatically whenever the change set touches something
  whose blast radius `mode-select.sh` cannot narrow to a specific service
  list — a shared library
  (`libs/atlas-kafka`, `libs/atlas-rest`, `libs/atlas-tenant`, `libs/atlas-redis`,
  `libs/atlas-env`, `libs/atlas-service`), `deploy/k8s/base/*`, a Kafka
  message contract, an `entity.go`/`migration*.go`, `atlas-configurations`,
  `atlas-tenants`, or any path outside the repo's known top-level roots
  (including `go.work`, `docker-bake.hcl`, and any unrecognized root file) —
  or explicitly via the `atlas:isolated` label below.

The `detect-changes` composite action (`.github/actions/detect-changes/action.yml`)
runs `mode-select.sh` once per PR validation run and posts a single, in-place-updated
PR comment (marked with `<!-- atlas:mode-report -->`) naming the mode, the reason,
and the override set — see the "Ephemeral Environment Mode Report" job in
`pr-validation.yml`.

**Overriding the computed mode.** Two labels force the mode explicitly, in
either direction (FR-9.5):

```sh
# Force sparse even on a change that would otherwise escalate to isolated.
# This is a deliberate, author-asserted risk: you are telling CI the
# change is safe to validate against the shared control plane (atlas-login,
# atlas-channel, and every other service still on `main`), not that the
# escalation table is wrong.
gh pr edit <N> --add-label atlas:sparse

# Force isolated even on a change that would otherwise compute sparse.
gh pr edit <N> --add-label atlas:isolated
```

Applying **both labels to the same PR is an error, not a precedence
rule** — `mode-select.sh` exits non-zero rather than silently picking one,
and the `detect-changes` job (and therefore the whole PR Validation run)
fails until one label is removed.

The nightly `pr-env-smoke.yml` run (FR-9.6) opens its synthetic PR with
`atlas:isolated` for exactly this reason: its change is a docs-only touch
(`docs/smoke/touch.txt`), which `mode-select.sh` would otherwise compute as
sparse, and the whole point of that run is to prove the **full isolated
stack** still deploys and tears down cleanly.

## §9.16 Sparse-mode tenant ownership and the environment boundary

**What a sparse environment owns.** `bootstrap.sh`'s `tenant-create` step
(`services/atlas-pr-bootstrap/scripts/bootstrap.sh:320`) mints exactly one
tenant per PR, scoped to that PR's own `ATLAS_ENVIRONMENT` and distinct from
the baseline's. `record-tenant` then patches the control-plane environment
record's `tenant` attribute with it and logs `recorded tenant=$TENANT_ID on
environment record $ATLAS_ENVIRONMENT`
(`services/atlas-pr-bootstrap/scripts/bootstrap.sh:363`). The minted tenant
deliberately shares the baseline's `(region, majorVersion, minorVersion)`
triple — `find_environment_tenant`/`create_environment_tenant` look up and
create against that same triple — and is distinguished from every other
tenant only by environment and a generated UUID, never by version.

**What it does *not* duplicate.** Correct the widespread assumption that
each sparse environment gets its own multi-thousand-document `atlas-data`
restore. It does not: the data-ingest guard
(`services/atlas-pr-bootstrap/scripts/bootstrap.sh:602`, merged in
`c5e88320a`) checks both the PR tenant's own document count and the
canonical (shared) tenant's count before restoring, and skips the restore
whenever the canonical count is already non-zero — the comment there notes
the shared `atlas-data` database already held all ~49k canonical rows the
first time a sparse environment hit this path. The sparse tenant instead
reads that shared corpus through `document/storage.go`'s canonical
fallback, which resolves to an id derived from the version triple, not the
environment. So expect the PR tenant to report **0 owned documents** at
`GET /api/data/status` and the shared/canonical scope
(`GET /api/data/status?scope=shared`) to report the full total. Mutable
gameplay state is what is actually tenant-keyed, and it is what
`sweep-orphans.sh --sweep-tenant` reclaims on teardown — not the read-only
canonical corpus.

**How to confirm the boundary held**, as a checklist an operator can run:

- The environment record carries a non-empty `tenant` that is not the
  baseline's `tenant`.
- `GET /api/tenants` against `<N>.atlas.home` returns exactly one tenant.
- `GET /api/tenants` against `dev.atlas.home` returns only main's tenant
  and does not list the PR's.
- The bootstrap Job's logs contain the line `recorded tenant=<id> on
  environment record pr-<N>`.
- `tools/overlay-env-guard.sh` is the automated counterpart to the above:
  it renders every overlay and pins the per-overlay `ATLAS_ENVIRONMENT`
  literal and the ingress `ATLAS_ENVIRONMENT_DEFAULT` wiring, so a
  regression that lets a PR overlay pick up `main`'s environment id (and
  therefore its tenant) fails `tools/verify.sh`'s `deploy/` block before it
  ever reaches a live cluster.

**The two metrics to watch, not one.** A stale registry produces
`atlas_kafka_gate_dropped_unresolvable_total`
(`libs/atlas-kafka/consumer/gate.go:38`), a *different* verdict from
`atlas_kafka_gate_skipped_not_owner_total`
(`libs/atlas-kafka/consumer/gate.go:30`). Both must stay flat for a
non-overridden service (e.g. `atlas-ban`) processing a PR-environment
operation. Name the heartbeat as the reason: with no `main` record and no
`ATLAS_ENVIRONMENT`, `environments.StartHeartbeat` published nothing and
every registry aged into `Stale()` — a rising `dropped_unresolvable`
counter on an unrelated service is that failure mode, not evidence the PR
environment leaked traffic to it.

**The `main` prerequisite and the bring-up window.** `overlays/main` now
deploys `environment-record.yaml`
(`deploy/k8s/overlays/main/environment-record.yaml`) at sync-wave 11 and
sets `ATLAS_ENVIRONMENT=main`. On a *fresh* `atlas-main` bring-up there is a
seconds-long window where atlas-ingress is Ready and stamping `main` before
the record exists, and every REST call 400s; it is bounded by Argo's wave
gating (wave 11 is gated on wave 10's health) and self-heals. Also record
the one-time operator step for the **existing** live cluster: after the
sync completes, run `kubectl rollout restart deployment/atlas-ingress -n
atlas-main`, because `atlas-ingress-configmap` is a plain resource (not
generated by kustomize) and editing `nginx.conf` does not roll the pods on
its own.

**A pointer to the regression pin.** Sparse bootstrap depends on
`templates` having a baseline-fallback scope rather than a strict one — a
tenant provisioned under `pr-<N>` with no `pr-<N>`-scoped template row must
still resolve `main`'s row for the same `(region, majorVersion,
minorVersion)`. The pin for this is `TestTemplatesFallBackToTheBaselineRow`
(`services/atlas-configurations/atlas.com/configurations/templates/overlay_test.go:65`).
A refactor that makes `templates` strictly scoped turns every sparse
bootstrap into `no template found … cluster setup issue`.
