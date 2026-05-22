# Ephemeral per-PR Deployments — Runbook

Operational guide for the per-PR atlas environments. PRD / design /
implementation plan: `docs/tasks/task-063-ephemeral-pr-deployments/`.

## §9.1 First-time setup: canonical WZ via MinIO

The PR bootstrap Job fetches `atlas.zip` from cluster-internal MinIO into an
emptyDir via an init container. PVCs can't cross namespaces; MinIO is the
single source of truth for the canonical zip. The Service is ClusterIP-only,
so the blob isn't reachable from outside the cluster.

### Stand up MinIO (one-time)

Apply the manifest from the cluster-infra repo, then wait for the Deployment:

```sh
kubectl apply -f <infra-repo>/minio.yml
kubectl rollout status -n minio deployment/minio --timeout=120s
```

### Create the bucket and upload `atlas.zip` (one-time per WZ revision)

The `atlas-canonical` bucket is created once and re-populated whenever the
canonical WZ revision changes (monthly at most). Anonymous-read policy on
the bucket means the per-PR init container needs no credentials.

Install the MinIO client (`mc`) first if you don't have it:
<https://min.io/download> (or `brew install minio/stable/mc` on macOS).

Run the block below as a single script — the `trap` cleans up the
port-forward only when the shell that set it exits, so copy-pasting line
by line into an interactive shell will leak the background process.

```sh
# Port-forward locally so `mc` can talk to the in-cluster MinIO.
kubectl port-forward -n minio svc/minio 9000:9000 &
PF_PID=$!
trap 'kill $PF_PID 2>/dev/null' EXIT

# Set up the mc alias using the root credentials from minio-root-creds.
# (MINIO_USER instead of USER to avoid clobbering the shell's $USER.)
MINIO_USER=$(kubectl -n minio get secret minio-root-creds -o jsonpath='{.data.MINIO_ROOT_USER}' | base64 -d)
MINIO_PASS=$(kubectl -n minio get secret minio-root-creds -o jsonpath='{.data.MINIO_ROOT_PASSWORD}' | base64 -d)
mc alias set bee "http://localhost:9000" "$MINIO_USER" "$MINIO_PASS"

# First time only: create the bucket and set anonymous-read.
mc mb --ignore-existing bee/atlas-canonical
mc anonymous set download bee/atlas-canonical

# Upload (or re-upload) the canonical zip.
mc cp /path/to/atlas.zip bee/atlas-canonical/atlas.zip
```

If the bootstrap Job later fails with `curl: (22) The requested URL
returned error: 404`, the bucket exists but the `atlas.zip` object is
missing — re-run the `mc cp` step.

The atlas-pr-bootstrap Job's init container `fetch-wz-canonical` runs:

```sh
curl -fsSL -o /opt/wz/atlas.zip \
    http://minio.minio.svc.cluster.local:9000/atlas-canonical/atlas.zip
```

The main bootstrap container then reads `/opt/wz/atlas.zip` via the
`WZ_CANONICAL` env (default unchanged).

### Bootstrap mode (task-071: MinIO-backed ingest)

As of task-071, `bootstrap.sh` runs WZ ingest entirely through atlas-data
— the donor `atlas-wz-extractor` step is gone, and WZ assets land in
MinIO (`atlas-canonical`, `atlas-wz`, `atlas-assets`, `atlas-renders`
buckets) instead of an extracted XML tree on disk.

The `BOOTSTRAP_MODE` env controls the data-ingest step:

- `baseline` — call `POST /api/data/baseline/restore`. atlas-data pulls
  a pre-built document dump from MinIO at
  `atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump`
  and replays it directly into Postgres. Fast (~60 s); requires the
  operator to have published a baseline first via `POST /api/data/baseline/publish`.
- `full` — `PATCH /api/data/wz` uploads the canonical zip to MinIO,
  then `POST /api/data/process` invokes WZ ingest. ~10 minutes; used
  when no baseline exists for the region/version pair.
- `auto` (default) — probes
  `HEAD $MINIO_ENDPOINT/atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump.sha256`
  and resolves to `baseline` on 200 or `full` on absence (with a WARN
  log so operators can correlate slow PR envs with missing baselines).

To force a particular mode, override `BOOTSTRAP_MODE` on the bootstrap
Job (Helm value or `kubectl set env`).

### Refreshing the canonical zip

A re-upload (`mc cp ... atlas.zip`) is picked up by every subsequent PR sync;
existing PR envs need a manual sync to pull the new zip (`argocd app sync
atlas-pr-<N>` or label-toggle the PR).

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

If something in that chain fails, the Application sits in `Terminating` with finalizers `post-delete-finalizer.argocd.argoproj.io/cleanup` and `resources-finalizer.argocd.argoproj.io` still present. Per-env state may be partially reclaimed.

### Diagnose

**Read the summary line first.** As of task-075, `cleanup.sh` runs every
phase regardless of any single phase's outcome. The final log line is
the authoritative status:

```
{"ts":…,"level":"info","atlas.env":"…","atlas.step":"done","msg":"cleanup complete phases_run=7 phases_failed=0"}
```

or, on partial failure:

```
{"ts":…,"level":"error","atlas.env":"…","atlas.step":"done","msg":"cleanup completed with errors phases_run=7 phases_failed=2 failed_phases=[\"drop-topics\",\"drop-redis\"]"}
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

### Source-branch-missing scenario

If the PostDelete render fails with `unable to resolve 'bot/pr-<N>-resolved' to a commit SHA`, the Application targets a branch that no longer exists. Diagnose: `kubectl -n argocd get application atlas-pr-<N> -o yaml | yq '.status.conditions[] | select(.message | contains("ComparisonError"))'`. Recovery is the same finalizer patch (step 2 above) followed by the sweep (step 1) — the branch is already gone so `drop-branch` is a no-op.

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
  # 2. Update the cluster secret.
  kubectl -n argocd edit secret atlas-pr-cleanup-gh-token   # set key GHCR_TOKEN
  # 3. Update the repo secret used by .github/workflows/pr-cleanup.yml's image-delete step.
  gh secret set GHCR_TOKEN --repo Chronicle20/atlas --body "$NEW_PAT"
  ```

  The nightly smoke test (§4.5 / `pr-env-smoke.yml`) will catch a missed half-rotation within 24h.

  **If your GitHub plan does not expose Account-level `Packages` on fine-grained PATs:** mint a classic PAT instead (*Tokens (classic) → Generate new token (classic)*) with the `repo` scope and `delete:packages` (which auto-selects `read:packages`). Classic PATs are broader (whole-user repo write) but reliably support GHCR package deletion. Document the choice when rotating so the next operator knows which type to renew.

- **GitHub PAT for Argo source-repo creds:** `kubectl edit secret argocd-repo-creds-chronicle20-atlas -n argocd`, replace `password`. ApplicationSet picks up on next reconcile (~30s). This token does NOT need `Contents: Read and write` (the cleanup PAT above owns branch deletion).

- **Pi-hole tokens:** `kubectl edit secret pihole-credentials -n argocd`. Source Secret lives in `argocd` and is Reflector-replicated to every `atlas-pr-*` namespace. The PostSync register Job (in `atlas-pr-<N>`) reads the replica; the PostDelete cleanup Job (in `argocd`) reads the source directly. Rotation takes effect on the next PR sync.

- **Database credentials (`db-credentials`):** source Secret lives in `atlas-main` and is Reflector-replicated to `atlas-pr-.*|argocd` (the per-PR namespaces AND `argocd` so the PostDelete cleanup Job can read it). `kubectl edit secret db-credentials -n atlas-main`; Reflector pushes the change to all replicas within seconds.

- **ghcr-pat (legacy).** No longer used by the PostDelete Job (replaced by `atlas-pr-cleanup-gh-token`). If no other consumer needs it, remove it in a cluster-infra follow-up.

## §9.6 Bootstrap-duration metrics

```promql
histogram_quantile(0.95,
  rate(atlas_bootstrap_step_duration_ms_bucket{atlas_env!="main"}[1h]))
```

Loki: filter the `atlas.step` field for stepwise breakdown:

```logql
{atlas_env="a3f7", job=~"atlas-pr-bootstrap"} | json | atlas_step != ""
```

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

- `cleanup.sh` does not currently invoke `DELETE /api/data/tenants/<id>` because `TENANT_ID` is not injected into the cleanup environment (cleanup keys off `ATLAS_ENV` for DB and Kafka drops; per-tenant MinIO cleanup needs the UUID). MinIO per-tenant prefixes (under `atlas-wz/tenants/<id>/`, `atlas-assets/tenants/<id>/`, `atlas-renders/tenants/<id>/`) therefore leak across PR teardowns. Postgres state continues to clear via the per-env database drop. Resolving this requires the cleanup Helm chart to inject `TENANT_ID`; until then operators may run `mc rm --recursive --force <alias>/<bucket>/tenants/<id>/` manually as needed. Extending `sweep-orphans.sh` (§9.11) with a MinIO phase would be the cleaner long-term fix.

## §9.12 Diagnosing partial-cleanup failure

As of task-075 the PostDelete Job runs every phase regardless of any
single phase's outcome. The summary line names which phases failed:

```
cleanup completed with errors phases_run=7 phases_failed=2 failed_phases=["drop-topics","drop-redis"]
```

Re-run only the failed phases via the §9.11 sweep-orphans path with
`--apply`, or manually:

| Phase | Manual re-run |
|---|---|
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
