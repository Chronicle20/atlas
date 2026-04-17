# Deployment Artifact Reorganization — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-16

---

## 1. Overview

Atlas currently scatters deployment artifacts across the repo: per-service Kubernetes manifests live inside each service directory (`services/atlas-<name>/atlas-<name>.yml`), infrastructure manifests sit at the repo root (`base.yaml`, `atlas-ingress.yml`), and the shared environment ConfigMap lives under `services/atlas-env.yaml`. There is no docker-compose artifact at all — local development requires either a full Kubernetes cluster or one-off `docker-build.sh`/`go run` workflows per service.

This task consolidates all deployment artifacts under a single top-level `deploy/` tree, mirroring the structure used in the `home-hub` project: `deploy/k8s/` for Kubernetes manifests (flat layout, one file per service plus shared namespace/secret/env/ingress files) and `deploy/compose/` for a new local Docker Compose stack that brings up every Atlas service (but **not** backing infrastructure — Postgres, Redis, Kafka, Tempo remain externally provided).

The resulting layout makes deployment concerns discoverable in one place, eliminates the scatter-gather problem when auditing or editing manifests, and gives developers a one-command path (`./deploy/compose/up.sh`) to run the full Atlas service graph locally against their existing host-level infra.

## 2. Goals

### Primary goals

- Centralize all Kubernetes manifests under `deploy/k8s/` in a flat layout matching `home-hub/deploy/k8s/`.
- Centralize local-development Docker Compose artifacts under `deploy/compose/` matching `home-hub/deploy/compose/`.
- Produce a set of Docker Compose files that build and run the 56 Atlas services locally, split so that **core HTTP/Kafka services** and the **TCP socket servers** (`atlas-login`, `atlas-channel`) can be brought up independently or together, with a companion nginx container replicating the K8s ingress routes.
- Mirror `home-hub`'s dual-file secret pattern: `secrets.example.yaml` (committed, sanitized) and `secrets.yaml` (gitignored, real); similarly `.env.example` (committed) and `.env` (gitignored).
- Generate real, functional `secrets.yaml` and `.env` from the current environment so developers can use them immediately after the reorg.
- Update all in-repo references (README.md, CLAUDE.md, DOCS.md, tools scripts) to the new paths so documented workflows still work.

### Non-goals

- Adding Postgres, Redis, Kafka, Zookeeper, Tempo, or any backing infrastructure to the compose file. Infra is assumed to be provided on the host (e.g., `postgres.home`, `kafka.home`, `redis.home`, `tempo.home`).
- Converting K8s manifests to Helm, Kustomize, Kpt, Timoni, or any templating layer.
- Modifying existing `Dockerfile`/`Dockerfile.dev`/`docker-build.sh` content (they continue to build the same images they build today).
- Modifying CI workflows (`pr-validation.yml`, `main-publish.yml`) unless a reference to a moved file is discovered.
- Changing any service code, Go module layout, or the `go.work`-based build.
- Implementing automatic cross-cluster ingress rerouting to the local compose nginx. The nginx is designed so this is *possible* later, but wiring it up is follow-up work.
- Migrating the live K8s deployment. This task changes file locations in the repo only; applying the new manifests to a cluster is a separate operational step performed by the developer.

## 3. User Stories

- As an Atlas developer, I want all deployment files in one folder so I can audit or edit them without grepping across 54+ service directories.
- As an Atlas developer, I want to run `./deploy/compose/up.sh` and get every Atlas service running locally against my already-running Postgres/Redis/Kafka, so I can iterate without standing up k3s/minikube.
- As an Atlas developer, I want the compose stack to expose an nginx entry point (identical routes to the cluster ingress), so frontend/client traffic works the same locally as in-cluster.
- As an operator, I want `secrets.yaml` and `.env` to be gitignored and real, with `.example` variants committed, so I never risk leaking credentials and new contributors have a template to clone.
- As a contributor, I want the README's deployment section to still match reality after the move, so I don't get mislead by stale paths.

## 4. Functional Requirements

### 4.1 K8s directory layout

Produce `deploy/k8s/` containing, flat:

- `namespace.yaml` — extracted from current `base.yaml` (the `Namespace` resource only).
- `secrets.example.yaml` — extracted from current `base.yaml` (the `Secret db-credentials` resource, with all `data:` values replaced by clearly-placeholder base64 strings such as `CHANGE_ME`).
- `secrets.yaml` — same schema as `secrets.example.yaml`, populated with the real base64-encoded values taken from the current `base.yaml`. Gitignored.
- `env-configmap.yaml` — moved from `services/atlas-env.yaml`, unchanged content.
- `ingress.yaml` — adapted from root `atlas-ingress.yml`. The **route-definition content** is sourced from `deploy/shared/routes.conf` (see §4.9). The K8s ConfigMap inlines that content verbatim; a sync script regenerates it whenever `routes.conf` changes.
- One file per service named `atlas-<name>.yaml` for each of the **54 services that have a manifest today** — e.g., `atlas-account.yaml`, `atlas-character.yaml`. The `atlas-` prefix is preserved so filenames match `metadata.name`, container names, module names, and image names used everywhere else in the repo. `atlas-families` and `atlas-marriages` have no manifest today (not deployed) and are explicitly excluded from this task; they'll be added when their manifests are authored (see §7).
- Subfolder `tls/` reserved for future TLS material (optional — create only if the current deployment has TLS artifacts; otherwise skip).

All moved files must preserve their current `metadata.name`, `metadata.namespace`, labels, image references, `containerPort` values, env wiring, and replica counts. This task is a **file move + rename + nginx-route single-sourcing only**; service manifests are not rewritten beyond relocation.

### 4.2 Compose directory layout

The compose stack is split into three files so developers can mix-and-match which tiers run locally (e.g., run core services in compose while keeping `atlas-login`/`atlas-channel` running elsewhere, or vice versa). All files share a single user-defined bridge network so containers across files can still reach each other by name when stacks are brought up together.

Produce `deploy/compose/` containing:

- `docker-compose.yml` — **base stack**. Declares the shared network (`name: atlas`) and the `nginx` reverse-proxy container. Does not define any Atlas service. Always required; other files are overlays that attach to the same network.
- `docker-compose.core.yml` — **core services overlay**. Defines every Atlas service *except* `atlas-login` and `atlas-channel`. Covers 51 HTTP/Kafka domain services plus `atlas-ui` = **52 services total**. Excludes `atlas-families` and `atlas-marriages` (no K8s manifest today; parity with K8s — see §7).
- `docker-compose.socket.yml` — **socket servers overlay**. Defines only `atlas-login` and `atlas-channel`, which expose raw TCP ports for game clients.
- `nginx.conf` — compose-specific server/http header (resolver, server_name, etc.). Includes `routes.conf` for all `location` blocks.
- `routes.conf` — **symlink** to `../shared/routes.conf` (see §4.9), bind-mounted into the nginx container alongside `nginx.conf`. This is the single source of nginx routes, shared with the K8s ingress.
- `up.sh`, `down.sh`, `logs.sh` — wrappers matching `home-hub` style that accept a stack selector: `core` (default), `socket`, or `all`. Each resolves the script directory, sources `$SCRIPT_DIR/.env`, and invokes `docker compose -f docker-compose.yml -f docker-compose.<stack>.yml ...` with the correct overlay set. See §4.5 for the exact flag wiring.
- `.env.example` — committed template with every variable referenced by any of the three compose files, using sanitized placeholders (e.g., `DB_HOST=postgres.local`, `BOOTSTRAP_SERVERS=kafka.local:9093`, `DB_USER=CHANGE_ME`, `DB_PASSWORD=CHANGE_ME`).
- `.env` — real values taken from the current `atlas-env.yaml` ConfigMap and `base.yaml` Secret. Lives at `deploy/compose/.env` (not repo root); gitignored.

**Stack composition matrix.**

| Stack selector | Files composed                                                           | Contents                                                              |
| -------------- | ------------------------------------------------------------------------ | --------------------------------------------------------------------- |
| `core`         | `docker-compose.yml` + `docker-compose.core.yml`                         | 52 services (51 HTTP/Kafka + `atlas-ui`) + nginx                      |
| `socket`       | `docker-compose.yml` + `docker-compose.socket.yml`                       | `atlas-login` + `atlas-channel` + nginx (nginx starts but is mostly idle when core isn't running; acceptable) |
| `all`          | `docker-compose.yml` + `docker-compose.core.yml` + `docker-compose.socket.yml` | All 54 deployed services + nginx                                       |

Because all three files declare `networks: atlas: external: false` on the shared definition and attach every service to it, running `core` and `socket` as separate invocations (different terminals, different times) still results in cross-stack name resolution when both are up.

### 4.3 Compose — service entries

Every Atlas service *that has a K8s manifest today* must have a matching compose entry, placed in the correct overlay file per §4.2. `atlas-families` and `atlas-marriages` have no manifest; they are **excluded** from compose for parity.

- **`docker-compose.core.yml` (52 services)**: `atlas-account, atlas-asset-expiration, atlas-assets, atlas-ban, atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs, atlas-chalkboards, atlas-character, atlas-character-factory, atlas-configurations, atlas-consumables, atlas-data, atlas-drop-information, atlas-drops, atlas-effective-stats, atlas-expressions, atlas-fame, atlas-gachapons, atlas-guilds, atlas-inventory, atlas-invites, atlas-keys, atlas-map-actions, atlas-maps, atlas-merchant, atlas-messages, atlas-messengers, atlas-monster-death, atlas-monsters, atlas-notes, atlas-npc-conversations, atlas-npc-shops, atlas-parties, atlas-party-quests, atlas-pets, atlas-portal-actions, atlas-portals, atlas-query-aggregator, atlas-quest, atlas-rates, atlas-reactor-actions, atlas-reactors, atlas-saga-orchestrator, atlas-skills, atlas-storage, atlas-tenants, atlas-transports, atlas-ui, atlas-world, atlas-wz-extractor`.
- **`docker-compose.socket.yml` (2 services)**: `atlas-login, atlas-channel`.

Each compose entry must include:

- `container_name: <service>` — matches the K8s service name so nginx routes work unchanged and cross-overlay name resolution is stable.
- `networks: [atlas]` — attaches to the shared bridge declared in the base file so core ↔ socket cross-talk works when both overlays are up simultaneously.
- `build.context: ../..` (repo root, required by the `Dockerfile` which `COPY`s `libs/` and `services/`).
- `build.dockerfile: services/<service>/Dockerfile` — the production Dockerfile (NOT `Dockerfile.dev`), so the compose build exercises the same code path as the CI image build and benefits from Docker layer caching on subsequent builds.
- `image: <service>:local` — stable local tag so `docker compose up` without `--build` reuses cached images.
- `env_file: .env` — single source of truth for shared environment values (loaded from `deploy/compose/.env`).
- `environment:` overrides only for values hard-coded in each service's K8s manifest (`env:` entries with literal `value:`, not `valueFrom:` refs). The full transcription table is in §4.10.
- `extra_hosts:` entries mapping infra hostnames to `host-gateway` so containers can reach host-provided infra DNS:
  ```yaml
  extra_hosts:
    - "postgres.home:host-gateway"
    - "kafka.home:host-gateway"
    - "redis.home:host-gateway"
    - "tempo.home:host-gateway"
  ```
  Infra hostnames themselves come from `.env` variables (`DB_HOST`, `BOOTSTRAP_SERVERS`, `REDIS_URL`, `TRACE_ENDPOINT`) so users can override them for non-standard setups, but the default `extra_hosts` map the shipped defaults. (See §8.2 Network.)
- `restart: unless-stopped`.
- `depends_on:` entries only where strictly required (e.g., `nginx` depends on every service; socket services have no intra-atlas hard dependencies since Kafka is external).

Services with TCP socket ports (`atlas-login`, `atlas-channel`) — defined in `docker-compose.socket.yml` — must expose those ports to the host:

- `atlas-login`: publish `1200, 8300, 8700, 9200, 9500, 18500`.
- `atlas-channel`: publish `1201, 8301, 8701, 18501`.

Services with HTTP ports used for direct probing (`atlas-ui` on 3000, defined in the core overlay) should publish that port as well. All other services are reachable only through the nginx container — no host-port publishing required.

**Volume mounts.** Three services need bind-mounts under `tmp/` (already gitignored at repo root) to replicate their K8s PVC wiring:

| Service               | Mount                                                                                        |
| --------------------- | -------------------------------------------------------------------------------------------- |
| `atlas-assets`        | `../../tmp/assets:/usr/assets`                                                               |
| `atlas-data`          | `../../tmp/data:/usr/data`                                                                   |
| `atlas-wz-extractor`  | `../../tmp/wz-input:/usr/wz-input`, `../../tmp/data:/usr/data`, `../../tmp/assets:/usr/assets` |

Implementation must also ensure `tmp/assets/` and `tmp/data/` exist (create empty) and that `tmp/` remains gitignored (confirmed: `.gitignore:27`). Other services that reference container paths in env vars (`/scripts/map`, `/drops/continents`, etc.) do not mount anything in K8s either — those paths are baked into the image at build time via `COPY`, so no compose volume is needed.

**Cross-overlay dependency note.** `atlas-channel` depends on several core services (tenants, character, etc.) at runtime. The socket overlay must *not* declare hard `depends_on` links across to the core overlay (those services may not be in the same compose project), and must tolerate them being absent at start time — the services already retry Kafka and REST calls. If a developer brings up `socket` alone, they are expected to point its Kafka/REST targets at whichever core services are reachable via the shared `atlas` network or via `.env` overrides.

### 4.4 Compose — nginx entry

A dedicated `nginx` service lives in the **base `docker-compose.yml`** using the official `nginx:alpine` image so it's always present regardless of which overlay(s) a developer starts:

- `container_name: atlas-ingress` (matches K8s deployment name).
- `networks: [atlas]`.
- Mounts two files: `./nginx.conf:/etc/nginx/conf.d/default.conf:ro` (compose-specific header) and `./routes.conf:/etc/nginx/conf.d/routes.conf:ro` (shared routes — see §4.9).
- Publishes port `${INGRESS_HOST_PORT:-8080}:80`.
- **No `depends_on` entries.** Because nginx is in the base file and upstream services live in overlays, declaring `depends_on` would fail when that overlay is not composed. Instead, rely on nginx's built-in upstream resolution (`resolver 127.0.0.11 valid=30s;` — Docker's embedded DNS) so it tolerates services appearing/disappearing.
- Does not publish 443 or carry TLS — plain HTTP is sufficient for local dev; TLS is provided by the K8s ingress in real deployments.

### 4.5 Compose — scripts

`up.sh`, `down.sh`, `logs.sh` follow home-hub style but take a first positional argument selecting the stack: `core` (default), `socket`, or `all`. Implementation contract:

- Use `set -euo pipefail`.
- Resolve `SCRIPT_DIR` from `${BASH_SOURCE[0]}`. `.env` lives at `$SCRIPT_DIR/.env` (scoped to compose, not repo root).
- Read and shift the first positional arg as `STACK` (default `core`). Validate against `{core, socket, all}`; if invalid, print usage and exit 2.
- Build a `COMPOSE_FILES` flag list based on `STACK`:
  - `core`   → `-f docker-compose.yml -f docker-compose.core.yml`
  - `socket` → `-f docker-compose.yml -f docker-compose.socket.yml`
  - `all`    → `-f docker-compose.yml -f docker-compose.core.yml -f docker-compose.socket.yml`
- Pass `--project-name atlas` explicitly so all invocations (regardless of stack) share one compose project, keeping the `atlas` network and container names consistent across separate `up.sh` calls.
- Require `$SCRIPT_DIR/.env` to exist; print a friendly error referencing `.env.example` if missing.
- `cd "$SCRIPT_DIR"` then `exec docker compose --env-file "$SCRIPT_DIR/.env" --project-name atlas $COMPOSE_FILES <verb> "$@"`.
- `up.sh` passes `--build "$@"` so first run builds images; subsequent runs can pass `--no-build` to skip.

Examples:

```bash
./deploy/compose/up.sh                  # core stack (default)
./deploy/compose/up.sh core -d          # core stack, detached
./deploy/compose/up.sh socket           # just login/channel
./deploy/compose/up.sh all -d           # everything
./deploy/compose/logs.sh core -f atlas-account
./deploy/compose/down.sh all
```

### 4.6 Secrets & env — real values

- Generate `deploy/k8s/secrets.yaml` by copying the current `base.yaml` secret block verbatim (values already base64-encoded).
- Generate `deploy/compose/.env` by flattening the current `services/atlas-env.yaml` ConfigMap data block into `KEY=value` lines, plus decoded `DB_USER` and `DB_PASSWORD` taken from the current `base.yaml` secret. Also include a few compose-specific keys: `INGRESS_HOST_PORT=8080`, `ATLAS_IMAGE_TAG=local`.
- Generate `.example` variants with the same keys but sanitized values (`CHANGE_ME`, or representative defaults like `postgres.home`, `kafka.home:9093` for hostnames that aren't sensitive).
- Add `.env` and `deploy/k8s/secrets.yaml` to `.gitignore`. Verify that `.env.example` and `secrets.example.yaml` are *not* gitignored.

### 4.7 Reference updates

Update every in-repo reference to moved files. Known call sites (from current grep):

- `README.md`: `base.yaml`, `atlas-ingress.yml`, `services/atlas-env.yaml`, `services/atlas-<name>/atlas-<name>.yml`, and the "Create Kubernetes manifest" bullet in the contribution flow.
- `CLAUDE.md`: anywhere deployment paths are documented. Currently references are minimal, but the existing multi-service build/verification note should be reviewed.
- `DOCS.md`: no known references, but re-grep during implementation.
- **Service-local docs** — `services/atlas-<name>/README.md` and `services/atlas-<name>/docs/*.md`. Sweep across all 56 service directories for references to `atlas-<name>.yml`, `base.yaml`, `atlas-ingress.yml`, `services/atlas-env.yaml`, or anything in the contribution/deployment sections.
- `tools/debug-start.sh`, `tools/debug-stop.sh`: they do not reference manifest paths directly (they operate on running cluster state via `kubectl`), but any baked-in `NAMESPACE`, `INGRESS_DEPLOYMENT`, or `CONFIGMAP_NAME` must still match the renamed files.
- `.github/workflows/*.yml`: current grep shows no references to deployment paths, but re-verify during implementation.

All references must point to the new canonical paths (`deploy/k8s/…`, `deploy/compose/…`, `deploy/shared/…`). Do not leave symlinks or backwards-compatibility shims behind.

### 4.8 Old-path cleanup

After the move, the following must not exist:

- `base.yaml` (root)
- `atlas-ingress.yml` (root)
- `services/atlas-env.yaml`
- `services/atlas-<name>/atlas-<name>.yml` (every service that had one)
- `services/atlas-<name>/docker-build.sh` (every service) — superseded by compose `build:` directive.
- `services/atlas-<name>/docker-build.bat` (every service) — same reason.

Service directories retain everything else (Go source, `Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug`, `docs/`, `README.md`, any service-specific auxiliary files like `atlas-assets/nginx.conf`).

### 4.9 Shared nginx routes

Problem: today's `atlas-ingress.yml` embeds the nginx config (~230 lines) inside a K8s ConfigMap. If compose ships its own separate `nginx.conf`, every new route or edit has to be duplicated in two places with near-certain drift.

Observation: **route bodies are identical between environments.** All HTTP services run on container port 8080 in both K8s and compose, and bare service names (`atlas-account`) resolve correctly in both:
- In K8s, pods' `resolv.conf` search list includes `atlas.svc.cluster.local`, so `atlas-account` → `atlas-account.atlas.svc.cluster.local`.
- In compose, Docker's embedded DNS resolves container names directly.

So `proxy_pass http://atlas-account:8080;` works verbatim in both environments — no hostname suffix needed.

The only environment-specific bits live in the server-level header (resolver IP, `server_name`).

**Design.**

```
deploy/
├── shared/
│   └── routes.conf          # SINGLE SOURCE — all `location ~ ^/api/...` blocks with bare container-name proxy_pass targets
├── k8s/
│   └── ingress.yaml         # ConfigMap with K8s-specific header + inlined copy of routes.conf (maintained by sync script)
├── compose/
│   ├── nginx.conf           # Compose-specific header with `include /etc/nginx/conf.d/routes.conf;`
│   └── routes.conf          # Symlink → ../shared/routes.conf (bind-mounted into nginx container)
└── scripts/
    └── sync-k8s-ingress-routes.sh  # Regenerates ingress.yaml's ConfigMap from deploy/shared/routes.conf
```

**Contents.**

- `deploy/shared/routes.conf` — contains *only* the `location` blocks (including tenant/region header propagation). No `http {}`, no `server {}`, no `resolver` — just routes. Proxy targets are bare container/service names.
- `deploy/k8s/ingress.yaml` — the ConfigMap's `nginx.conf` key holds the K8s header (resolver `10.43.0.10`, `server_name dev.atlas.home`) and uses an `include /etc/nginx/routes.conf;` directive pointing to a second ConfigMap key. The ConfigMap has two keys: `nginx.conf` (K8s header) and `routes.conf` (inlined copy of shared). The sync script rewrites the `routes.conf` key's value block from `deploy/shared/routes.conf` whenever routes change. A `grep` CI check (future) can detect drift.
- `deploy/compose/nginx.conf` — compose header (resolver `127.0.0.11`, `server_name _`) with `include /etc/nginx/conf.d/routes.conf;`. The compose container bind-mounts `deploy/compose/routes.conf` (which is a symlink to `deploy/shared/routes.conf`) at that path. No regeneration needed — compose reads the live shared file on container start.
- `deploy/scripts/sync-k8s-ingress-routes.sh` — bash script that reads `deploy/shared/routes.conf`, indents each line to match the YAML scalar block in `deploy/k8s/ingress.yaml`, and rewrites the `routes.conf: |` block. Idempotent; exit 0 if file is already in sync.

**Workflow when adding or editing a route.**

1. Edit `deploy/shared/routes.conf`.
2. Run `./deploy/scripts/sync-k8s-ingress-routes.sh` (or let a pre-commit hook run it — out of scope here).
3. Commit both `routes.conf` and `ingress.yaml`.

**Drift detection (future).** A CI step that runs the sync script with a `--check` flag and fails if output differs from the committed `ingress.yaml`. Not in scope for this task but explicitly enabled by the design.

### 4.10 Per-service environment-block transcription

From the survey of all 54 service manifests, the literal `env:` entries that compose must transcribe into each service's `environment:` block:

| Service                  | Literal env entries to transcribe                                                                                                                        |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `atlas-account`          | `LOG_LEVEL=debug`, `DB_NAME=atlas-accounts`                                                                                                              |
| `atlas-asset-expiration` | `LOG_LEVEL=debug`, `EXPIRATION_CHECK_INTERVAL_SECONDS=60`                                                                                                |
| `atlas-assets`           | (none)                                                                                                                                                   |
| `atlas-ban`              | `DB_NAME=atlas-ban` (no LOG_LEVEL in manifest)                                                                                                            |
| `atlas-buddies`          | `LOG_LEVEL=debug`, `DB_NAME=atlas-buddies`                                                                                                               |
| `atlas-buffs`            | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-cashshop`         | `LOG_LEVEL=debug`, `DB_NAME=atlas-cashshop`                                                                                                              |
| `atlas-chairs`           | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-chalkboards`      | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-channel`          | `LOG_LEVEL=debug`, `SERVICE_ID=e7fb1d7e-47b8-46bd-97dc-867d93530000`, `SERVICE_TYPE=channel-service`                                                      |
| `atlas-character`        | `LOG_LEVEL=debug`, `DB_NAME=atlas-characters`, `SERVICE_MODE=MIXED`                                                                                       |
| `atlas-character-factory`| `LOG_LEVEL=debug`, `SERVICE_ID=00000000-0000-0000-0000-000000000000`, `SERVICE_TYPE=character-factory`                                                    |
| `atlas-configurations`   | `LOG_LEVEL=debug`, `DB_NAME=atlas-configurations`                                                                                                        |
| `atlas-consumables`      | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-data`             | `LOG_LEVEL=debug`, `DB_NAME=atlas-data`, `ZIP_DIR=/usr/data`                                                                                             |
| `atlas-drop-information` | `LOG_LEVEL=debug`, `DB_NAME=atlas-drops`, `SERVICE_ID=0...0`, `SERVICE_TYPE=drops-information-service`, `MONSTER_DROPS_PATH=/drops/monsters`, `CONTINENT_DROPS_PATH=/drops/continents`, `REACTOR_DROPS_PATH=/drops/reactors` |
| `atlas-drops`            | `LOG_LEVEL=debug`, `SERVICE_ID=0...0`, `SERVICE_TYPE=drops-service`                                                                                      |
| `atlas-effective-stats`  | (none beyond secret refs)                                                                                                                                |
| `atlas-expressions`      | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-fame`             | `LOG_LEVEL=debug`, `DB_NAME=atlas-fame`                                                                                                                  |
| `atlas-gachapons`        | `DB_NAME=atlas-gachapons`                                                                                                                                |
| `atlas-guilds`           | `LOG_LEVEL=debug`, `DB_NAME=atlas-guilds`                                                                                                                |
| `atlas-inventory`        | `LOG_LEVEL=debug`, `DB_NAME=atlas-inventory`                                                                                                             |
| `atlas-invites`          | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-keys`             | `LOG_LEVEL=debug`, `DB_NAME=atlas-keys`                                                                                                                  |
| `atlas-login`            | `LOG_LEVEL=debug`, `SERVICE_ID=e7fb1d7e-47b8-46bd-97dc-867d93530856`, `SERVICE_TYPE=login-service`                                                        |
| `atlas-map-actions`      | `DB_NAME=atlas-map-actions`, `MAP_SCRIPTS_DIR=/scripts/map`                                                                                              |
| `atlas-maps`             | `LOG_LEVEL=debug`, `DB_NAME=atlas-maps`                                                                                                                  |
| `atlas-merchant`         | `DB_NAME=atlas-merchant`                                                                                                                                 |
| `atlas-messages`         | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-messengers`       | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-monster-death`    | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-monsters`         | `LOG_LEVEL=info`                                                                                                                                         |
| `atlas-notes`            | `LOG_LEVEL=debug`, `DB_NAME=atlas-notes`                                                                                                                 |
| `atlas-npc-conversations`| `LOG_LEVEL=debug`, `DB_NAME=atlas-npc-conversations`, `NPC_CONVERSATIONS_PATH=/conversations/npc`, `QUEST_CONVERSATIONS_PATH=/conversations/quests`      |
| `atlas-npc-shops`        | `LOG_LEVEL=debug`, `DB_NAME=atlas-npc-shops`                                                                                                             |
| `atlas-parties`          | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-party-quests`     | `DB_NAME=atlas-party-quests`                                                                                                                             |
| `atlas-pets`             | `LOG_LEVEL=debug`, `DB_NAME=atlas-pets`                                                                                                                  |
| `atlas-portal-actions`   | `DB_NAME=atlas-portal-actions`, `PORTAL_SCRIPTS_DIR=/scripts/portals`, `QUERY_AGGREGATOR_URL=http://atlas-query-aggregator:8080`                         |
| `atlas-portals`          | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-query-aggregator` | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-quest`            | `DB_NAME=atlas-quest`                                                                                                                                    |
| `atlas-rates`            | (none beyond secret refs)                                                                                                                                |
| `atlas-reactor-actions`  | `DB_NAME=atlas-reactor-actions`, `REACTOR_ACTIONS_DIR=/scripts/reactors`                                                                                 |
| `atlas-reactors`         | `LOG_LEVEL=debug`                                                                                                                                        |
| `atlas-saga-orchestrator`| `LOG_LEVEL=debug`, `DB_NAME=atlas-saga-orchestrator`, `SAGA_DEFAULT_TIMEOUT=5m`, `SAGA_REAPER_INTERVAL=30s`, `SAGA_RECOVERY_ENABLED=true`                  |
| `atlas-skills`           | `LOG_LEVEL=debug`, `DB_NAME=atlas-skills`                                                                                                                |
| `atlas-storage`          | `DB_NAME=atlas-storage`, `SERVICE_MODE=MIXED`, `COMMAND_TOPIC_STORAGE=COMMAND_TOPIC_STORAGE`, `EVENT_TOPIC_STORAGE_STATUS=EVENT_TOPIC_STORAGE_STATUS`     |
| `atlas-tenants`          | `LOG_LEVEL=info`, `DB_NAME=atlas-tenants`                                                                                                                |
| `atlas-transports`       | `LOG_LEVEL=info`                                                                                                                                         |
| `atlas-ui`               | `NEXT_PUBLIC_ROOT_API_URL=http://atlas-ingress:8080` (compose override of K8s cluster DNS value; pre-existing build-time bake limitation flagged in §8.3) |
| `atlas-world`            | `LOG_LEVEL=debug`, `SERVICE_ID=0...0`, `SERVICE_TYPE=world-service`                                                                                      |
| `atlas-wz-extractor`     | `LOG_LEVEL=debug`, `INPUT_WZ_DIR=/usr/wz-input`, `OUTPUT_IMG_DIR=/usr/assets`, `OUTPUT_XML_DIR=/usr/data`                                                 |

Secret-ref env vars (`DB_USER`, `DB_PASSWORD`, consumed via `envFrom: configMapRef: atlas-env` and `secretKeyRef: db-credentials`) are not transcribed per-service — they come from `.env` via `env_file`.

## 5. API Surface

No external or internal API changes. This is a packaging/layout change only.

## 6. Data Model

No schema changes, no new entities, no migrations.

## 7. Service Impact

No service code changes. Impact is purely on repo layout:

| Area | Change |
| --- | --- |
| `base.yaml`, `atlas-ingress.yml` | Deleted from root; content split into `deploy/k8s/{namespace,secrets{,.example},ingress}.yaml` + `deploy/shared/routes.conf`. |
| `services/atlas-env.yaml` | Moved to `deploy/k8s/env-configmap.yaml`. |
| `services/atlas-<name>/atlas-<name>.yml` (×54) | Moved to `deploy/k8s/atlas-<name>.yaml` and renamed with `.yaml` extension. |
| `services/atlas-families/`, `services/atlas-marriages/` | No manifest to move; not deployed today; excluded from compose too. Re-add when manifests exist. |
| `services/atlas-<name>/docker-build.sh`, `docker-build.bat` | Deleted (superseded by compose `build:`). |
| `services/atlas-<name>/` (Go source, Dockerfiles, docs) | Unchanged. |
| `deploy/compose/`, `deploy/shared/`, `deploy/scripts/` | Newly created. |
| `README.md`, `CLAUDE.md`, `DOCS.md`, `tools/debug-*.sh`, `services/*/README.md`, `services/*/docs/*.md` | Path references swept and updated. |
| `.gitignore` | Add `deploy/compose/.env`, `deploy/k8s/secrets.yaml`. (`tmp/` already covered.) |

## 8. Non-Functional Requirements

### 8.1 Build performance

- Compose builds must share Docker layer cache across services. Because every service `Dockerfile` copies the same `libs/atlas-*` module sources early in the build, the BuildKit cache should hit on those layers after the first build. Expect cold-start of the full stack to take several minutes on a typical dev machine; warm rebuilds of a single service should be seconds.
- Builds must not require buildx-specific features that aren't available in vanilla `docker compose build`. Standard BuildKit enabled by default in modern Docker Desktop / Docker Engine ≥ 23 is acceptable.

### 8.2 Network

- Compose stack uses a user-defined bridge network named `atlas`, declared once in the base `docker-compose.yml` and referenced by every overlay service via `networks: [atlas]`. Containers reach each other by service name (e.g., `http://atlas-account:8080`) regardless of which overlay they were started in.
- Because all scripts pass `--project-name atlas`, the network is shared across independent `up.sh` invocations (e.g., `up.sh core` in one terminal and `up.sh socket` in another both attach to `atlas_atlas` and can resolve each other's containers).
- Containers reach host-provided infra via `extra_hosts: ["<hostname>:host-gateway"]`, which maps the hostname to the Docker host gateway IP. On Docker Desktop (macOS/Windows) this resolves to `host.docker.internal`; on Linux Docker Engine ≥ 20.10 it maps to the host's gateway address.
- Developers who route infra via alternate hostnames override `DB_HOST`, `BOOTSTRAP_SERVERS`, `REDIS_URL`, `TRACE_ENDPOINT` in their local `.env` and add matching `extra_hosts` entries if needed. Document this in a README section in `deploy/compose/`.

### 8.3 Observability

- Containers emit logs to stdout/stderr (default Docker log driver). No log rotation or forwarding configured — developers use `./logs.sh <service>` to tail.
- Tracing continues to export to `$TRACE_ENDPOINT` (`tempo.home:4317`) via the same OTel wiring as K8s. If Tempo is not running, services must continue to function (ensure existing graceful-degradation behavior is preserved — no code change, just worth confirming in acceptance testing).
- **`atlas-ui` `NEXT_PUBLIC_*` caveat.** The atlas-ui Dockerfile invokes `npm run build` without passing `NEXT_PUBLIC_ROOT_API_URL` as a build ARG or ENV. Next.js bakes `NEXT_PUBLIC_*` variables into the client bundle at build time; any runtime env setting only affects server-side code (server components, route handlers). This is **pre-existing** behavior in the K8s deployment and is **replicated verbatim** in compose — the reorg does not fix or change it. Flag as a separate follow-up task to either (a) pass the URL as a build ARG and rebuild per environment, or (b) switch the UI to read the API base URL at runtime via a `/api/config` endpoint or relative URLs.

### 8.4 Security

- Real `.env` and `secrets.yaml` must never be committed. Both must appear in `.gitignore` before the first real file is written in the working tree. Implementation order: add `.gitignore` entries first, then generate real files second.
- `.env.example` and `secrets.example.yaml` must contain only placeholder/sanitized values — never real credentials, never real hostnames that would expose internal infrastructure.

### 8.5 Multi-tenancy

- No change. The existing tenant-header propagation through nginx is preserved in `deploy/compose/nginx.conf` by copying the relevant `proxy_set_header` directives from `atlas-ingress.yml`.

## 9. Open Questions

All previously-tracked open questions were resolved during spec review on 2026-04-16. Decisions:

| # | Question                                              | Decision                                                                                         |
| - | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| 1 | K8s filename prefix                                   | **Keep `atlas-` prefix** — match `metadata.name`, container names, and repo-wide naming.         |
| 2 | nginx default host port                               | **8080**, overridable via `INGRESS_HOST_PORT`.                                                   |
| 3 | `atlas-wz-extractor` lifecycle                        | **Long-running service** — include in `docker-compose.core.yml` with `restart: unless-stopped`. No `profiles:` gate. |
| 4 | Per-service `DB_NAME` wiring                          | **Literal in each service's `environment:` block**, transcribed from today's K8s manifest. No `.env` variable proliferation. |
| 5 | Compose image source                                  | **Always build** via `build:` directive. Tag `atlas-<name>:${ATLAS_IMAGE_TAG:-local}`. No `ghcr.io` pull profile in this task. |
| 6 | Cluster→local-compose ingress rerouting               | **Already supported** by `tools/debug-start.sh --service <X> --target <laptop-ip>:<INGRESS_HOST_PORT>`. The cluster nginx proxies service X's routes to the developer's laptop; compose nginx (listening on `INGRESS_HOST_PORT`) then dispatches to the compose container or to a locally-debugged process on the same port. No new tooling needed. |
| 7 | `atlas-ui` overlay placement                          | **Core overlay** (`docker-compose.core.yml`). Not split into its own file.                       |
| 8 | Stack-selector CLI shape                              | **Positional arg** on `up.sh`/`down.sh`/`logs.sh`: `core` (default), `socket`, `all`.            |
| 9 | `.env` location                                       | **`deploy/compose/.env`** (scoped to compose), not repo root. Scripts use `$SCRIPT_DIR/.env`.    |
| 10 | nginx config drift                                   | **Single-source routes** via `deploy/shared/routes.conf`. K8s ConfigMap is regenerated by `deploy/scripts/sync-k8s-ingress-routes.sh`; compose bind-mounts the shared file directly. See §4.9.    |
| 11 | Login/channel smoke-test port                        | **TCP `1200` for login, `1201` for channel** (the canonical v83 game-client ports; first in each manifest's port list).                                                                       |
| 12 | `atlas-families`, `atlas-marriages`                  | **Exclude from this reorg.** No K8s manifest today → no compose entry either. Add when manifests are authored.                                                                                 |
| 13 | Shared-volume host path                              | **`tmp/`** at repo root — already gitignored (`.gitignore:27`). Bind-mount subpaths per §4.3.                                                                                                  |

## 10. Acceptance Criteria

- [ ] `deploy/k8s/` exists with: `namespace.yaml`, `secrets.example.yaml`, `secrets.yaml` (gitignored), `env-configmap.yaml`, `ingress.yaml`, and **54** `atlas-<name>.yaml` service manifests.
- [ ] `deploy/shared/routes.conf` exists and contains all `location` blocks with bare-container-name `proxy_pass` targets (no `.atlas.svc.cluster.local` suffix).
- [ ] `deploy/scripts/sync-k8s-ingress-routes.sh` exists, is executable, and run with no args exits 0 when the K8s ConfigMap is in sync with `routes.conf`.
- [ ] `deploy/compose/` exists with: `docker-compose.yml` (base: network + nginx), `docker-compose.core.yml` (52 services), `docker-compose.socket.yml` (2 services), `nginx.conf`, `routes.conf` (symlink → `../shared/routes.conf`), `up.sh`, `down.sh`, `logs.sh`, `.env.example`, `.env` (gitignored).
- [ ] `./deploy/compose/up.sh core` brings up the 52-service HTTP tier + nginx; `./deploy/compose/up.sh socket` (in a separate terminal) starts `atlas-login`/`atlas-channel` on the same `atlas` network; both invocations report the same docker compose project name (`atlas`).
- [ ] `base.yaml`, `atlas-ingress.yml`, and `services/atlas-env.yaml` no longer exist at their old paths.
- [ ] No `services/atlas-<name>/atlas-<name>.yml` files remain.
- [ ] No `services/atlas-<name>/docker-build.sh` or `docker-build.bat` files remain.
- [ ] `.gitignore` includes `deploy/compose/.env` and `deploy/k8s/secrets.yaml`; `git status` shows neither as untracked after generation.
- [ ] `grep -rE "base\.yaml|atlas-ingress\.yml|services/atlas-env\.yaml|services/atlas-[a-z-]+/atlas-[a-z-]+\.yml"` across the repo returns only results inside `docs/tasks/` and the new `deploy/` tree.
- [ ] `./deploy/compose/up.sh core --no-cache` succeeds on a clean machine with host-provided Postgres/Redis/Kafka reachable at the `.env`-specified hostnames.
- [ ] After `up.sh core`, `curl -H "TENANT_ID: <uuid>" -H "REGION: GMS0" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" http://localhost:8080/api/accounts` returns a JSON:API response from `atlas-account` via the compose nginx.
- [ ] After `up.sh socket`, `nc -zv localhost 1200` succeeds (atlas-login) and `nc -zv localhost 1201` succeeds (atlas-channel).
- [ ] `kubectl apply -f deploy/k8s/namespace.yaml && kubectl apply -f deploy/k8s/` brings up a functioning Atlas cluster equivalent to the current one (one full end-to-end smoke test in a cluster before merge).
- [ ] README's deployment section describes the new layout and shows the new commands (`kubectl apply -f deploy/k8s/...`, `./deploy/compose/up.sh`).
- [ ] `tools/debug-start.sh`/`tools/debug-stop.sh` continue to function against the renamed ingress ConfigMap and Deployment (or are updated if their baked-in names change).
- [ ] Service-local docs (`services/*/README.md`, `services/*/docs/*.md`) no longer reference old paths.
