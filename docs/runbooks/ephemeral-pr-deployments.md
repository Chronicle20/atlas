# Ephemeral per-PR Deployments — Runbook

Operational guide for the per-PR atlas environments. PRD / design /
implementation plan: `docs/tasks/task-063-ephemeral-pr-deployments/`.

## §9.1 First-time setup: canonical WZ PVC

Per-PR bootstrap mounts a ReadOnlyMany PVC named `atlas-wz-canonical-readonly`
containing `atlas.zip`. Create it once on the cluster, then load the canonical
WZ zip via a temporary writer pod:

```sh
# Apply the PVCs once (RWO source + ROX exposure).
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: atlas-wz-canonical
  namespace: longhorn-system
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 8Gi
  storageClassName: longhorn
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: atlas-wz-canonical-readonly
  namespace: argocd
spec:
  accessModes: [ReadOnlyMany]
  resources:
    requests:
      storage: 8Gi
  storageClassName: longhorn
EOF

# Mount into a temporary writer pod and copy the canonical zip.
kubectl run wz-uploader --image=alpine -i --tty --rm \
    --overrides='{"spec":{"volumes":[{"name":"wz","persistentVolumeClaim":{"claimName":"atlas-wz-canonical"}}],"containers":[{"name":"wz","image":"alpine","stdin":true,"tty":true,"volumeMounts":[{"name":"wz","mountPath":"/opt/wz"}]}]}}' \
    -- /bin/sh
# inside: scp atlas.zip into /opt/wz/atlas.zip
```

The atlas-pr-bootstrap container reads from `/opt/wz/atlas.zip` (env: `WZ_CANONICAL`).

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
