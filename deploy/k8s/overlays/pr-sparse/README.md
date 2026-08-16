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

The replica-1 patch, the topology-spread patch, and the
`imagePullPolicy: Always` patch (all inline in `kustomization.yaml`,
copied verbatim from `overlays/pr`), plus a set of files that are
byte-identical copies — not references — of the same-named files in
`overlays/pr/`. **`tools/pr-sparse-mirror-guard.sh`'s `MIRRORS` array is
the single source of truth for exactly which files those are**; this
README does not re-list them so the two can never disagree.

Copies, not references, because kustomize's default load restrictor
forbids a resource/patch path from escaping the overlay's own directory
tree — a relative `../pr/...` path is rejected at build time (`security;
file '...' is not in or below '.../pr-sparse'`, reproduced directly). A
shared `components:` directory (e.g. a hypothetical
`deploy/k8s/overlays/_shared/`) was considered and **does not work
either** — it hits the identical restrictor (verified during task-232
Task 44's review, fix round 1). So this is forced duplication, not a
shortcut, and `tools/pr-sparse-mirror-guard.sh` (wired into
`tools/verify.sh`'s `deploy/`-gated block) is the drift guard that keeps
it safe: it byte-diffs every mirrored file against its `overlays/pr`
original and fails the build if any pair has diverged. A later task could
still eliminate the duplication entirely by promoting these files into
`deploy/k8s/base/` (a true common ancestor — `deploy/k8s/base/components/
seed-catalog/` already does this shape) but that means touching
`overlays/pr`, which is live in `main` today and out of this task's
scope; the guard does not block that follow-up, it just makes today's
duplication safe in the meantime.

## What it adds

- `ATLAS_ENVIRONMENT=pr-<N>` in the `atlas-env` ConfigMap (read by
  `env.Self()`, `libs/atlas-env`'s `SelfVar`).
- The `NS_*` overrides (`ns-overrides.yaml`).
- The environment-record Job (`environment-record.yaml`).

## The override set

Currently fixed at `atlas-ingress` (always local — it is the routing
mechanism itself, not a `NS_*`-addressed service), `atlas-login`,
`atlas-channel`, and `atlas-character`. The delete-set anchor
(`kustomization.yaml`) and `environment-record.yaml`'s `overrides`
attribute both encode this set; Task 50's CI job is the single place that
computes it, so the two never drift independently.

## Sentinel contract

Resolved at CI time by `.github/workflows/pr-validation.yml`'s
`update-pr-overlay` job (Task 50), exactly as documented in
`overlays/pr/kustomization.yaml:1-20`, extended here with two shapes that
overlay doesn't need:

- **Scalar** (safe to spell out anywhere, including in comments —
  substitution never spans a newline): `PLACEHOLDER_PR_NUMBER`,
  `PLACEHOLDER_ATLAS_ENV`, `PLACEHOLDER_FULL_SHA` (the same three
  `overlays/pr` uses), `PLACEHOLDER_BASELINE_ENVIRONMENT` /
  `PLACEHOLDER_BASELINE_NAMESPACE` (the baseline's `env.Id` and k8s
  namespace, resolved from the ACTIVE baseline environment record — never
  a literal `"main"` / `"atlas-main"`, FR-1.5), and
  `PLACEHOLDER_OVERRIDES_JSON` (a single-line JSON object, see
  `environment-record.yaml`'s header).
- **Multi-line block** (the delete-set anchor in `kustomization.yaml` and
  the NS-override-set anchor in `ns-overrides.yaml`): each is a lone
  comment line, appears exactly once across every `.yaml`/`.yml` file in
  this directory, and is never spelled out again elsewhere — a second
  bare occurrence would itself get corrupted by the same blind
  substitution. Each anchor's own comment carries its fill contract:
  the replacement must escape embedded newlines GNU-`sed`-style and
  start with one escaped newline, and — because the delete-set payload
  contains YAML's `|-` block-scalar marker — **must not use `|` as the
  `sed` delimiter** (reproduced: `sed -i "s|X|<payload with a literal |>|g"`
  fails with `unknown option to 's'`). See `kustomization.yaml`'s
  delete-set anchor comment for the verified fix (a control-character
  delimiter) and full detail.

## Verifying locally

Scalar-only smoke test (matches Task 44 Step 4 — empty block anchors,
build succeeds, Deployment count equals base's):

```sh
sed -e 's/PLACEHOLDER_PR_NUMBER/999/g' -e 's/PLACEHOLDER_ATLAS_ENV/999/g' \
    -e 's/PLACEHOLDER_FULL_SHA/deadbeef/g' \
    -e 's/PLACEHOLDER_BASELINE_ENVIRONMENT/main/g' \
    -e 's/PLACEHOLDER_BASELINE_NAMESPACE/atlas-main/g' \
    -e 's/PLACEHOLDER_OVERRIDES_JSON/{}/g' \
    -i.bak deploy/k8s/overlays/pr-sparse/*.yaml deploy/k8s/overlays/pr-sparse/patches/*.yaml
kustomize build deploy/k8s/overlays/pr-sparse | grep -c "^kind: Deployment"
```

This builds every base Deployment (64, same as `overlays/pr`'s
smoke-test shape) since both block anchors are left as no-op comments.
Restore the `.bak` files afterwards — they must never be committed. For
the full real-content substitution (both block anchors filled, exactly 4
Deployments remain) see `task-44-report.md`'s fix-round-1 entry for the
exact commands and output.
