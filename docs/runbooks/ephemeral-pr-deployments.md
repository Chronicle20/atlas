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

### Refreshing the canonical zip

A re-upload (`mc cp ... atlas.zip`) is picked up by every subsequent PR sync;
existing PR envs need a manual sync to pull the new zip (`argocd app sync
atlas-pr-<N>` or label-toggle the PR).

## §9.2 Force-cleanup of a PR env

Bypass the grace period:

```sh
kubectl delete application -n argocd atlas-pr-<N>
```

Argo's PostDelete hook fires immediately. Verify:

```sh
kubectl get jobs -n atlas-pr-<N>
kubectl logs -n atlas-pr-<N> job/atlas-pr-cleanup -f
```

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

## §9.4 Re-running a failed PostDelete

If the cleanup Job fails, the Application stays in `cleanup-failed`. Re-trigger by force-syncing then re-deleting:

```sh
argocd app sync atlas-pr-<N> --force
kubectl delete application -n argocd atlas-pr-<N>
```

Alternatively, manually re-create the cleanup Job from the rendered overlay (advanced; see plan §11.4).

## §9.5 Rotating credentials

All Argo CD-related Secrets live in the `argocd` namespace and are templated by `argocd-secrets.yml.example` in the cluster-infra repo. To rotate:

- **GitHub PAT for Argo:** generate a new fine-scoped PAT, then
  ```sh
  kubectl edit secret argocd-repo-creds-chronicle20-atlas -n argocd
  ```
  replace `password`, save. ApplicationSet picks up on next reconcile (~30s).
- **Pi-hole tokens:** `kubectl edit secret pihole-credentials -n argocd`. The PostSync register Job reads at run-time; rotation takes effect on the next PR sync.
- **ghcr PAT:** `kubectl edit secret ghcr-pat -n argocd`. Used by the PostDelete cleanup Job for image-tag deletion.

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
