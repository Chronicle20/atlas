# Adding a New Service — Registration Checklist

Every new service must be enumerated in a fixed set of files. **None of these are
derived from each other** — most are hand-maintained lists, and several fail
*silently* when an entry is missing (see [Silent-failure traps](#silent-failure-traps)).
Work through every section; a service is not "added" until all applicable rows
are done and the [verification commands](#verification) pass.

> **Why this doc exists:** atlas-mts (task-121) was wired into CI, the k8s base,
> and the PR overlay — but missed all four of its main-overlay enumerations.
> Result: crash-looping pods on main (`DB_NAME` pointed at a nonexistent
> database), an unpinned `:latest` image the bump workflow could never pin, and
> Kafka topics silently resolving to unsuffixed names. Each miss was invisible
> until runtime.

For code-level scaffolding (model/entity/processor layout, Bruno collections,
tenant opcode templates), see the backend-dev-guidelines skill resource:
`.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md`.

## 1. Build & CI

| # | File | What to add |
|---|---|---|
| 1.1 | `.github/config/services.json` | Entry in `services[]` with `name`, `type: go-service`, `path`, `module_path`, `docker_image`, `docker_context: "."`. Both `main-publish.yml` and `pr-validation.yml` read this dynamically. |
| 1.2 | `docker-bake.hcl` | Add `"atlas-<svc>"` to the hardcoded `go_services` list. **Hand-synced** with services.json — adding to one does NOT add to the other. |
| 1.3 | `go.work` | Add `./services/atlas-<svc>/atlas.com/<svc>` to `use()`. |
| 1.4 | Repo-root `Dockerfile` | Nothing per-service (it is parameterized by `ARG SERVICE`). Only a new **shared lib** needs edits: two `COPY libs/<name>` lines (mod-only block + source block) plus a `go.work` line. |

Verify the image builds: `docker buildx bake atlas-<svc>` from the repo root.

## 2. Kubernetes base (`deploy/k8s/base/`)

| # | File | What to add |
|---|---|---|
| 2.1 | `deploy/k8s/base/atlas-<svc>.yaml` | Deployment + Service. Copy an existing DB-backed service as the template. No `namespace:` (overlays set it). `DB_NAME` gets the **unsuffixed** base value (`atlas-<db>`); overlays patch the env suffix. Container `name:` is the short service name (e.g. `mts`) — the overlay patches match on it. |
| 2.2 | `deploy/k8s/base/kustomization.yaml` | Add `atlas-<svc>.yaml` to `resources:`. |
| 2.3 | `deploy/k8s/base/env-configmap.yaml` | Every **new Kafka topic env var** the service introduces, as `KEY: "KEY"` (identity value). Producers and consumers in *other* services read these too. |
| 2.4 | Seed catalog (optional) | If the service consumes seed data, add the `atlas.seed-catalog: "true"` label to the Deployment — the `components/seed-catalog` kustomize component injects the git-sync sidecar and `SEED_CATALOG_ROOT` automatically. |

## 3. Main overlay (`deploy/k8s/overlays/main/`) — the ones missed for MTS

| # | File | What to add |
|---|---|---|
| 3.1 | `patches/db-name-suffix.yaml` | New patch document: `DB_NAME: "atlas-<db>-main"` targeting the container name from 2.1. DB-backed services only. |
| 3.2 | `patches/atlas-env-env.yaml` | New patch document: `ATLAS_ENV: "main"`. Every service gets this. |
| 3.3 | `kustomization.yaml` → `images:` | `- name: ghcr.io/chronicle20/atlas-<svc>/atlas-<svc>` with `newTag:` set to the current fleet tag (`main-<sha>`; confirm the tag exists on ghcr, e.g. `docker manifest inspect`). **The bump workflow only rewrites entries already present** — a missing entry means the service runs `:latest` forever. |
| 3.4 | `kustomization.yaml` → `configMapGenerator` literals | Every topic var from 2.3 as `KEY=KEY-main`. The generator uses `behavior: replace`, so any base key not re-listed here is **absent** on main. |

Note: `KAFKA_CONSUMER_GROUP` is intentionally NOT injected on main (see the
comment at the top of the main kustomization) — do not add it there.

## 4. PR overlay (`deploy/k8s/overlays/pr/`)

| # | File | What to add |
|---|---|---|
| 4.1 | `kustomization.yaml` → `ATLAS_DB_NAMES` literal | Add the DB base name (e.g. `atlas-mts`). This single list drives **both** the wave-0 create-DBs job and ephemeral-env teardown (the drop list is derived from it). |
| 4.2 | `kustomization.yaml` → `images:` | Same entry shape as 3.3. |
| 4.3 | `kustomization.yaml` → `configMapGenerator` literals | Every topic var as `KEY=KEY-PLACEHOLDER_ATLAS_ENV`. |
| 4.4 | `patches/db-name-suffix.yaml` | `DB_NAME: "atlas-<db>-PLACEHOLDER_ATLAS_ENV"`. |
| 4.5 | `patches/consumer-group-env.yaml` | `KAFKA_CONSUMER_GROUP` patch document (PR envs DO inject it, unlike main). |

## 5. Ingress (REST services only)

| # | File | What to add |
|---|---|---|
| 5.1 | `deploy/shared/routes.conf` | nginx location block(s), alphabetically placed, bare container name (`http://atlas-<svc>:8080`). |
| 5.2 | regenerate | Run `./deploy/scripts/sync-k8s-ingress-routes.sh` to rebuild `deploy/k8s/ingress.yaml`. Commit both. |

## 6. Databases

| # | Where | What |
|---|---|---|
| 6.1 | postgres.home (main) | Create `atlas-<db>-main` **manually** — main has no wave-0 create job. Owner = the app role; `uuid-ossp` extension is inherited from `template1`. |
| 6.2 | `tools/db-bootstrap.sh` | Add the **unsuffixed** DB name to the hand-edited `DBS` list (local/dev bootstrap). |
| 6.3 | PR envs | Nothing beyond 4.1 — create and drop are derived from `ATLAS_DB_NAMES`. |

## 7. Socket services only

A new socket-exposing service (or a new client version) needs LB port rows in
`versions.json` + `gen-lb-ports.sh`; CI's `check-version-coverage.sh` gates
socket templates. See `docs/packets/PROCESS.md` for the packet/version side.

## Silent-failure traps

These are the failure modes that make missing entries invisible until runtime:

1. **`images:` bump is a no-op for missing entries.** The main-publish workflow
   runs `yq '(.images[] | select(.name == …) | .newTag) = …'` — if the service
   has no entry, nothing is written and no error is raised. The service runs
   `:latest`.
2. **`configMapGenerator` with `behavior: replace` drops unlisted keys.** The
   overlay does not *merge* with `env-configmap.yaml`; it replaces it. A topic
   var present in base but not in the overlay literals simply doesn't exist in
   that environment.
3. **Missing topic env vars don't crash.** `libs/atlas-kafka/topic/provider.go`
   falls back to the token itself (with only a warn log), so a service missing
   `COMMAND_TOPIC_X=COMMAND_TOPIC_X-main` silently produces/consumes on the
   unsuffixed topic `COMMAND_TOPIC_X`. It "works" only while every participant
   is equally misconfigured, then splits the moment one side gets the var.
4. **`DB_NAME` without the env suffix crash-loops at startup** (`SQLSTATE
   3D000`) — the only trap of the four that is loud.

## Verification

Run all of these before opening the PR:

```bash
# Overlays render and contain the service with correct values
kubectl kustomize deploy/k8s/overlays/main | grep -B2 -A6 "name: atlas-<svc>$"
#   expect: DB_NAME=atlas-<db>-main, ATLAS_ENV=main, image pinned to main-<sha>

# No base configmap key is dropped by either overlay (empty output = pass)
for ov in main pr; do
  comm -23 \
    <(grep -oE '^  [A-Z_0-9]+' deploy/k8s/base/env-configmap.yaml | tr -d ' ' | sort -u) \
    <(grep -oE '^      - [A-Z_0-9]+=' deploy/k8s/overlays/$ov/kustomization.yaml | sed 's/^      - //; s/=$//' | sort -u)
done

# Service present in every hand-maintained list
grep -l "atlas-<svc>" .github/config/services.json docker-bake.hcl go.work \
  deploy/k8s/base/kustomization.yaml \
  deploy/k8s/overlays/main/patches/db-name-suffix.yaml \
  deploy/k8s/overlays/main/patches/atlas-env-env.yaml \
  deploy/k8s/overlays/pr/patches/db-name-suffix.yaml \
  deploy/k8s/overlays/pr/patches/consumer-group-env.yaml \
  tools/db-bootstrap.sh

# Image target builds
docker buildx bake atlas-<svc>
```
