# deploy/k8s/overlays/pr-sparse

The sparse PR overlay (task-232 Phase D). Where `overlays/pr` gives a PR its
own fully-isolated copy of every one of the base's 64 Deployments plus a
private database and topic namespace, `pr-sparse` deploys **only the
override set** — the handful of services the PR actually changed — and
routes every other request through to the baseline (`overlays/main`) via
the per-service `NS_*` variables Task 43 wired into every nginx upstream.

## What's here

| File | Role |
|---|---|
| `kustomization.yaml` | The overlay shell. `resources:` includes `../../base` plus the environment-record Job and the reused-unchanged `overlays/pr` Jobs (`sync-bootstrap`, `predelete-purge`, `postsync-pihole-add`, `ingress-route`, `atlas-env-tokens`). `patches:` deletes every base Deployment not in the override set (`PLACEHOLDER_DELETE_BLOCK`, filled by CI) and reuses `overlays/pr`'s replica/topology-spread/imagePullPolicy/lb-allocate/ingress-host/consumer-group-env/seed-catalog-ref/minio-delete patches unchanged. |
| `ns-overrides.yaml` | Strategic-merge patch onto `atlas-ingress`'s nginx container: overrides every `NS_*` variable **except** the override set's own to point at the baseline namespace (`PLACEHOLDER_NS_OVERRIDES`, filled by CI). |
| `environment-record.yaml` | Sync-wave-1 Job that POSTs the new environment's record (`phase: PROVISIONING`) to the baseline's `atlas-configurations`, idempotently (GET-then-POST) since `sync-options: Force=true,Replace=true` reruns it on every sync. |
| `README.md` | This file. |

## What it does NOT carry, relative to `overlays/pr` (the whole point of D1 — shared substrates)

| Removed | Why |
|---|---|
| `patches/db-name-suffix.yaml` | Shared databases — no per-environment `DB_NAME` suffix. |
| Topic suffixing in the `atlas-env` `configMapGenerator` | Sparse consumes the unsuffixed baseline topics (FR-4.8); the generator uses `behavior: merge`, not `replace`, so base's unsuffixed `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` literals pass through untouched. |
| `ATLAS_ENV` in the `atlas-env` ConfigMap | Makes the Redis key prefix inert (design §9); `computeKeyPrefix("")` is already the legacy path, so no code change. (Individual Deployments still get a container-level `ATLAS_ENV` via `patches/consumer-group-env.yaml`, for `KAFKA_CONSUMER_GROUP` uniqueness only.) |
| `wave0-create-dbs.yaml` | No per-environment databases to pre-create. |

## What it keeps, unchanged

`patches/lb-allocate.yaml`, `patches/ingress-host.yaml`,
`patches/consumer-group-env.yaml`, `patches/seed-catalog-ref.yaml`,
`ingress-route.yaml`, `atlas-env-tokens.yaml`, the replica-1 patch, the
topology-spread patch, the `imagePullPolicy: Always` patch, and the
sync-bootstrap / predelete-purge / pihole Jobs.

These are byte-identical copies of the same-named files in `../pr/`, not
references — kustomize's default load restrictor forbids a resource/patch
path from escaping the overlay's own directory tree, so a relative `../pr/`
path is rejected at build time (`security; file '...' is not in or below
'.../pr-sparse'`). Keep them in sync with `overlays/pr` by hand (or fold
the shared ones into a common base both overlays include, as a follow-up)
until Task 50 automates the check.

## What it adds

- `ATLAS_ENVIRONMENT=pr-<N>` in the `atlas-env` ConfigMap (read by
  `env.Self()`, `libs/atlas-env`'s `SelfVar`).
- The `NS_*` overrides (`ns-overrides.yaml`).
- The environment-record Job (`environment-record.yaml`).

## The override set

Currently fixed at `atlas-ingress` (always local — it is the routing
mechanism itself, not a `NS_*`-addressed service), `atlas-login`,
`atlas-channel`, and `atlas-character`. `PLACEHOLDER_DELETE_BLOCK` and
`environment-record.yaml`'s `overrides` attribute both encode this set;
Task 50's CI job is the single place that computes it, so the two never
drift independently.

## Sentinel contract

Resolved at CI time by `.github/workflows/pr-validation.yml`'s
`update-pr-overlay` job (Task 50), exactly as documented in
`overlays/pr/kustomization.yaml:1-20`:

- `PLACEHOLDER_PR_NUMBER`, `PLACEHOLDER_ATLAS_ENV`, `PLACEHOLDER_FULL_SHA` —
  the same three sentinels `overlays/pr` uses.
- `PLACEHOLDER_DELETE_BLOCK` — the `$patch: delete` entries for every base
  Deployment not in the override set.
- `PLACEHOLDER_NS_OVERRIDES` — the `NS_*` env entries for every
  non-override-set service, pointed at `PLACEHOLDER_BASELINE_NAMESPACE`.
- `PLACEHOLDER_BASELINE_ENVIRONMENT` / `PLACEHOLDER_BASELINE_NAMESPACE` —
  the baseline's `env.Id` and k8s namespace, resolved from the ACTIVE
  baseline environment record. Never a literal `"main"` / `"atlas-main"`
  (FR-1.5) — see `kustomization.yaml`'s header comment.

## Verifying locally

```sh
sed -e 's/PLACEHOLDER_PR_NUMBER/999/g' -e 's/PLACEHOLDER_ATLAS_ENV/999/g' \
    -e 's/PLACEHOLDER_FULL_SHA/deadbeef/g' \
    -e 's/PLACEHOLDER_BASELINE_ENVIRONMENT/main/g' \
    -e 's/PLACEHOLDER_BASELINE_NAMESPACE/atlas-main/g' \
    -e 's/PLACEHOLDER_DELETE_BLOCK//' -e 's/PLACEHOLDER_NS_OVERRIDES//' \
    -i.bak deploy/k8s/overlays/pr-sparse/*.yaml
kustomize build deploy/k8s/overlays/pr-sparse | grep -c "^kind: Deployment"
```

With `PLACEHOLDER_DELETE_BLOCK` empty this builds every base Deployment
(64, same as `overlays/pr`'s smoke-test shape). Restore the `.bak` files
afterwards — they must never be committed.
