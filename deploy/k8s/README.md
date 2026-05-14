# Atlas Kubernetes manifests

Atlas's manifests are organised as a Kustomize base plus two overlays:

```
deploy/k8s/
├── base/                # Per-service Deployment+Service (no namespace)
├── overlays/
│   ├── main/            # main env: namespace=atlas-main, images=:latest, ATLAS_ENV=main
│   └── pr/              # PR env: namespace=atlas-pr-<N>, hash-suffixed, PR-tagged images
└── README.md
```

## Rendering locally

```bash
kustomize build deploy/k8s/overlays/main
kustomize build deploy/k8s/overlays/pr
```

(`kustomize` 4.5+; install from https://kubectl.docs.kubernetes.io/installation/kustomize/)

The PR overlay output contains `PLACEHOLDER_ATLAS_ENV`, `PLACEHOLDER_PR_NUMBER`, and `PLACEHOLDER_SHA` slots that Argo CD's `ApplicationSet(atlas-pr)` substitutes at sync time. Locally-rendered output is not directly applyable to a cluster.

## Adding a new service

1. Drop the service's manifest into `deploy/k8s/base/<svc>.yaml`.
2. Add the path to `deploy/k8s/base/kustomization.yaml`.
3. If the service uses Postgres, regenerate the DB-name patch (the script reads each base manifest's `DB_NAME` env var):
   ```sh
   ./deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
   ```
   And add the new service's DB name to the `ATLAS_DB_NAMES` literal in `deploy/k8s/overlays/pr/kustomization.yaml`'s `configMapGenerator`.
4. Re-run the consumer-group patch generator (reads each service's `main.go` for the consumerGroupId literal):
   ```sh
   ./deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
   ```
5. If the service emits new Kafka topics, append to `deploy/k8s/base/env-configmap.yaml` and re-render the topic literals for the PR overlay:
   ```sh
   ./deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
   # paste the output into deploy/k8s/overlays/pr/kustomization.yaml's configMapGenerator
   ```
6. Add the service's `docker_image` entry to `.github/config/services.json` so the CI matrix picks it up.
7. Commit `base/`, the updated kustomization, and the regenerated patches.

## ATLAS_ENV flow

The `ATLAS_ENV` token is the load-bearing isolation key. It propagates four ways:

| Path | Mechanism | Result |
|---|---|---|
| Postgres DB | `DB_NAME=atlas-<svc>-<env>` patched into every Deployment | One DB schema per env, same Postgres instance |
| Redis keys | `os.Getenv("ATLAS_ENV")` consumed by `libs/atlas-redis.computeKeyPrefix()` | Keys prefixed `<env>:atlas:…` for non-empty env, legacy `atlas:` for empty |
| Kafka topics | `<TOPIC>_NAME-<env>` literal materialised in `atlas-env` ConfigMap by Kustomize | Per-env topic name; consumers + producers naturally segregate |
| Kafka consumer groups | `KAFKA_CONSUMER_GROUP="<literal> [<env>]"` env, consumed by `libs/atlas-kafka/consumergroup.Resolve()` | One consumer group per env, no cross-env rebalancing |

For the `main` env, `ATLAS_ENV=main`. For PR envs, `ATLAS_ENV = sha256("pr-<N>")[:4]` (4-character hex) computed by the ApplicationSet's `goTemplate`.

## Hooks (PR overlay only)

- `presync-create-dbs.yaml` — `CREATE DATABASE IF NOT EXISTS` per service DB, idempotent
- `postsync-bootstrap.yaml` — runs the atlas-pr-bootstrap container (canonical WZ ingest + per-domain seeds + service-config write + rolling restart of SERVICE_ID-reading services)
- `postsync-pihole-add.yaml` — registers `<N>.atlas.home` on both Pi-hole servers
- `postdelete-cleanup.yaml` — drops DBs, deletes topics + consumer groups, clears Redis keys, removes ghcr image tags, unregisters Pi-hole DNS

Main env intentionally omits all four hooks. Postgres DBs survive across the one-time main cutover (see runbook §8). Subsequent fresh main installs are theoretical and have their own runbook.

## Cluster-side gitops

Argo CD itself, the `Application(atlas-main)`, the `ApplicationSet(atlas-pr)`, the cleanup CronJob, the per-PR `longhorn-pr` StorageClass, and the Pi-hole/ghcr/repo-creds Secrets all live in the maintainer's separate cluster-infrastructure repo. See `docs/runbooks/ephemeral-pr-deployments.md` for the bootstrap order and operational details.
