# Shared Seeder Library and GitOps Catalog — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-20
---

## 1. Overview

Eight Atlas services today implement the same seed-orchestration pattern by copy-paste: `POST /<x>/seed` triggers an idempotent reseed of tenant-scoped tables from a JSON/script catalog bundled into the service's Docker image, `GET /<x>/seed/status` reports row counts and last-updated timestamps, and a background goroutine carries the work. The pattern is duplicated in `atlas-gachapons`, `atlas-drop-information`, `atlas-map-actions`, `atlas-reactor-actions`, `atlas-portal-actions`, `atlas-npc-conversations`, `atlas-npc-shops`, and `atlas-party-quests`. Every divergence between implementations is accidental, not deliberate.

This task extracts the shared pattern into `libs/atlas-seeder`, migrates all eight services to consume it, and decouples the catalog data from the service image. The catalog moves to a top-level `deploy/seed/<region>/<version>/` tree mounted into each pod by a `git-sync` sidecar (k8s) or a bind mount (docker-compose). Catalog versioning is observable: a `CATALOG_REVISION` file at each root and a per-tenant `seed_state` table per service record what was applied and against which revision. This makes future GitOps reconciliation a non-breaking addition rather than a refactor.

Two services with seed concerns sit outside this task. `atlas-configurations` runs a startup-only seeder over its own `seed-data/templates/` with no per-tenant POST endpoint. `atlas-tenants` exposes payload-driven `POST /tenants/{id}/configurations/{x}/seed` endpoints that accept request bodies rather than reading bundled catalogs. Both have different data flows and are deliberately deferred to a follow-up task.

## 2. Goals

Primary goals:
- Eliminate the duplicated seed-orchestration code across eight services by extracting one shared `libs/atlas-seeder` library.
- Move bundled catalog data out of every in-scope service image to a single git-tracked `deploy/seed/<region>/<version>/` tree.
- Standardize every catalog entity file as a JSON:API document so the lib needs exactly one parser.
- Split the two oversized monolithic catalog files (`drops/monsters/monster_drops.json`, 3.1 MB; `gachapons/data/gachapon_items.json`, 132 KB) into per-entity files so they become reviewable in diffs.
- Make the catalog mount work identically in k8s (via `git-sync` sidecar) and docker-compose (via bind mount).
- Record `(tenant, group) → catalog_revision` in a per-service `seed_state` table so a future reconciler can compare desired vs observed without further migrations.
- Add a CI catalog linter that fails on JSON:API envelope drift, filename↔`data.id` mismatch, or non-`_*` directories containing non-`.json` files.

Non-goals:
- Building the reconciler Job that automatically POSTs `/seed` on revision drift. The schema and revision metadata land in this task; the reconciler is a follow-up.
- Touching `atlas-configurations` or `atlas-tenants`. Their seed shapes differ; they get their own task after this pattern is proven.
- Introducing an `_base/` overlay model to dedupe entities shared across versions. Catalogs duplicate fully per `(region, version)` for now. The lib's loader takes an ordered list of roots so this becomes a non-breaking config change later if duplication becomes painful.
- Migrating `embed.FS` static-data dirs in non-seed services (`atlas-channel/data/`, `atlas-character/data/`, `atlas-maps/data/`, etc.). Those are WZ-extracted reference data with a different lifecycle.
- Bucketing per-entity files by leading id digit (e.g., `monsters/1/100100.json`). Files start flat; bucketing is revisited only if directory size becomes operationally painful.
- Auto-running the catalog linter against `tools/catalog-lint/`'s own splitter outputs. Splitters are validated against the linter as part of normal CI, not via self-test.

## 3. User Stories

- As a platform engineer, I want to edit a monster's drop table by changing one small JSON file rather than diffing a 3.1 MB blob, so review and merge are tractable.
- As a service author, I want to add seed capability to a new service by registering a `Group` of `Subdomain`s with `libs/atlas-seeder` rather than copying ~400 lines from another service.
- As an operator, I want to update a drop table without rebuilding a service image, so catalog tweaks ship at the cadence of `git push` rather than the cadence of releases.
- As an operator, I want the catalog mount to behave identically in our k8s clusters and on a developer laptop running docker-compose, so "works locally" is meaningful.
- As a tenant administrator using the UI, I want the existing seed buttons to keep working without URL or behavior changes during this migration.
- As a future GitOps reconciler implementer, I want `GET /seed/status` to report both the catalog revision the pod has mounted and the revision a tenant was last seeded against, so I can detect drift without inspecting catalog files myself.
- As a CI consumer, I want a single linter pass to fail on any malformed catalog file before it reaches a service, so seed bugs are caught at PR time.

## 4. Functional Requirements

### 4.1 `libs/atlas-seeder` library

The library exposes the following Go types and functions:

- `type Subdomain[J any, M any] interface { Name() string; Path() string; EntityIDPattern() string; DeleteAllForTenant(*gorm.DB) (int64, error); Decode([]byte) (J, error); Build(tenant.Model, uint32, J) ([]M, error); BulkCreate(*gorm.DB, []M) error; Count(*gorm.DB) (int64, *time.Time, error) }` — declares one tenant-scoped catalog dataset.
- `type Group struct { Name string; URLPrefix string; Subdomains []SubdomainAny }` — declares one `(POST /<prefix>/seed, GET /<prefix>/seed/status)` endpoint pair backed by N subdomains seeded in parallel. `SubdomainAny` is a type-erased wrapper so a `Group` can hold heterogeneously-typed subdomains.
- `type CatalogSource interface { Roots() []string; Revision(root string) (string, error); Open(root, relPath string) (io.ReadCloser, error); Walk(root, relPath string) ([]string, error) }` — abstracts where catalog files live; the v1 implementation is `FilesystemCatalogSource` rooted at the mounted catalog directory, resolved per request from `tenant.Model.Region()` + `MajorVersion()` + `MinorVersion()`.
- `Seed(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Result, error)` — orchestrates per-subdomain delete-then-bulk-insert in parallel via `errgroup`, aggregates per-subdomain `Counts{Deleted, Created, Failed, Errors}`, persists a row to the per-service `seed_state` table on success, and returns a structured `Result` keyed by subdomain name.
- `Status(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Status, error)` — fans `Count()` across subdomains in parallel via `errgroup`, reads the catalog revision via `CatalogSource.Revision()`, reads the tenant's last-seeded revision from `seed_state`, returns a struct containing all three plus per-subdomain counts and the most-recent `UpdatedAt`.
- `RegisterRoutes(router *mux.Router, db *gorm.DB, si jsonapi.ServerInformation, src CatalogSource, g Group)` — wires `POST <prefix>/seed` (returns 202, runs `Seed` in a goroutine with a fresh `tenant.WithContext(context.Background(), t)` so the work survives the request) and `GET <prefix>/seed/status` (synchronous, returns JSON:API-formatted `Status`).

The library handles JSON:API parsing centrally: every catalog file is decoded as `{data: {type, id, attributes, relationships?}}` and the `data.id` is validated against the entity id extracted from the filename via the subdomain's `EntityIDPattern`. Files whose names start with `_` or `.` are skipped during walks. Files in subdirectories starting with `_` are also skipped (this supports the `_global/` convention for non-entity files like gachapons' global item pool).

`CatalogSource.Roots()` returns an ordered list. The v1 implementation returns a single root resolved from `tenant.MajorVersion`/`MinorVersion`/`Region`, but the loader walks each root in order and later roots override earlier roots by filename match (with `*.tombstone` files suppressing earlier roots). This is dead code in v1 (only one root is ever configured); it exists so introducing the `_base/<region>/` overlay model later requires zero loader changes.

`SEED_CATALOG_ROOT` is the env var pointing to the mounted catalog directory (`/var/run/seed-catalog` by default in containers, repo-relative `deploy/seed` on dev machines). The lib resolves `<root>/<region>/<major>_<minor>/<subdomain-path>/` per tenant on each request. The mounted catalog must always have a `CATALOG_REVISION` file at the per-`(region, version)` directory level.

### 4.2 Per-service `seed_state` table

Each of the eight services gains a `seed_state` table via GORM `AutoMigrate`:

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | UUID | composite primary key (with `group_name`) |
| `group_name` | text | composite primary key (with `tenant_id`); value matches `Group.Name` |
| `catalog_revision` | text | last successful seed's catalog revision |
| `seeded_at` | timestamptz | last successful seed's wall-clock time |
| `result_summary` | jsonb | the `Result` struct (counts per subdomain, errors) serialized for ops introspection |

The lib's `Seed` writes one row per `(tenant, group)` on completion. `Status` reads this row to populate `tenantSeededRevision` and `tenantSeededAt` on responses. The CI catalog linter ignores this table; it is purely runtime state.

### 4.3 Catalog directory layout

The repo gains `deploy/seed/<region>/<major>_<minor>/`. On day one this is six directories, mirroring `atlas-configurations`'s existing template axes:

```
deploy/seed/
├── gms/12_1/
├── gms/83_1/
├── gms/87_1/
├── gms/92_1/
├── gms/95_1/
├── jms/185_1/
└── _schema/                       # JSON Schema files used by the linter
```

Each `<region>/<version>/` directory contains:

```
CATALOG_REVISION                   # one-line opaque string (typically commit SHA)
drops/
  monsters/monster-<id>.json
  continents/continent-<id>.json
  reactors/reactor-<id>.json
gachapons/
  <id>.json                         # gachapon definition + inline item pool
  _global/items.json                # global gachapon item pool (not entity-shaped)
npc-conversations/
  npc/npc-<id>.json
  quests/quest-<id>.json
map-actions/
  map/map-<id>.json
portal-actions/
  portals/portal-<map-id>-<portal-name>.json
reactor-actions/
  reactors/reactor-<id>.json
npc-shops/
  shops/shop-<npc-id>.json
  npc/npc-<id>.json
party-quests/
  definitions/<definition-id>.json
```

Every `*.json` file is a JSON:API document. Filenames follow `<entity-type>-<id>.json` (or `<id>.json` where the parent directory already disambiguates) so the linter can extract the id from the filename without parsing the body.

For v83 (today's primary), the splitter scripts produce the initial content from the current in-image data. For v12, v87, v92, v95, and jms/185, the directories are initially populated with a copy of the v83 catalog and a `CATALOG_REVISION` indicating "bootstrapped from v83" — services running on those versions today consume identical catalogs to v83, so the copy preserves current behavior. Real per-version divergence happens lazily as edits land.

### 4.4 Splitter scripts

Each oversized or non-JSON:API catalog file gets a one-shot Go splitter under `tools/seed-splitters/`:

- `tools/seed-splitters/split-monster-drops/` — reads `services/atlas-drop-information/drops/monsters/monster_drops.json` (3.1 MB), groups drops by `MonsterId`, writes one JSON:API doc per monster to `deploy/seed/<region>/<version>/drops/monsters/monster-<id>.json`.
- `tools/seed-splitters/split-continent-drops/` — reads `services/atlas-drop-information/drops/continents/continent_drops.json`, groups by `ContinentId`, writes one JSON:API doc per continent.
- `tools/seed-splitters/split-gachapons/` — reads `services/atlas-gachapons/data/{gachapons.json,gachapon_items.json}`, merges by `GachaponId`, writes one combined JSON:API doc per gachapon plus a flat `_global/items.json` for the global pool.
- `tools/seed-splitters/wrap-jsonapi/` — generic transformer that wraps a plain-JSON file into a JSON:API doc given an `id` field name. Used for npc-conversations, map-actions, portal-actions, reactor-actions, npc-shops, party-quests where files already exist per-entity but lack the JSON:API envelope. The reactor-drops files (already JSON:API) are exempt.

Splitter scripts are deterministic (no timestamps, no random ids, files written in id-sorted order so reruns produce byte-identical output). They are committed to the repo but are not run by CI; running them at refactor time produces the initial `deploy/seed/` content that then becomes the source of truth.

### 4.5 Per-service migration

For each of the eight services, the migration touches:

1. Delete the duplicated `seed/` package (or in cases like npc-conversations where the seed lives next to the domain, delete the `seed.go`, `status.go`, `resource.go` files in each affected package).
2. Add a tiny new `seed/groups.go` that registers each `Group` and its `Subdomain[J, M]` implementations with `libs/atlas-seeder.RegisterRoutes`.
3. Remove the `COPY` lines in the service's `Dockerfile` that previously pulled in `data/` or `scripts/` or `drops/` or `conversations/` or `party-quests/`.
4. Delete the now-orphaned `data/` / `scripts/` / etc. directory from the service tree.
5. Update the service's k8s `Deployment` YAML in `deploy/k8s/base/atlas-<svc>.yaml` to mount `/var/run/seed-catalog` from the shared catalog volume and add the `git-sync` sidecar (see 4.6).
6. Update `deploy/compose/docker-compose.yml` to bind-mount `./deploy/seed:/var/run/seed-catalog:ro` on the service.
7. Add a GORM `AutoMigrate(&SeedState{})` call in the service's bootstrap so the new table is created on startup.

Endpoint URLs do not change. `POST /drops/seed`, `POST /gachapons/seed`, `POST /shops/seed`, `POST /party-quests/definitions/seed`, etc. all continue to function. The atlas-ui seed buttons need no changes.

Services with two seed groups today keep two endpoints:
- `atlas-npc-conversations`: `POST /npc-conversations/npc/seed` + `POST /npc-conversations/quests/seed` (URLs may need normalization from current paths; see Open Questions).
- `atlas-npc-shops`: keeps its existing two endpoints with current URLs.

### 4.6 k8s `git-sync` sidecar

A `git-sync` sidecar runs alongside every in-scope service pod, repolling the catalog repo at a fixed interval (default 60s) into a shared `emptyDir` mounted at `/var/run/seed-catalog`. Key configuration:

- Image: `registry.k8s.io/git-sync/git-sync` pinned to a specific tag.
- Repo: the same repo as the service code (Atlas monorepo); the sidecar checks out the path `deploy/seed/`.
- Ref: parameterized per ArgoCD overlay (`main` for prod, PR SHA for ephemeral envs).
- Mount: `emptyDir{}` named `seed-catalog`, mounted into both the sidecar (at `/git`) and the service container (at `/var/run/seed-catalog`).
- Sidecar runs alongside the service for the lifetime of the pod (not an initContainer); the service lib reads the catalog on each `/seed` POST so updates flow through without pod restart.
- Resource limits: small (the sidecar holds one git checkout at a time); explicit `requests`/`limits` set in the base manifest.

The shared catalog mount path and env var convention are encoded in a Kustomize component under `deploy/k8s/base/components/seed-catalog/` so the eight service manifests reference it identically.

### 4.7 docker-compose bind mount

`deploy/compose/docker-compose.yml` gets a top-level YAML anchor:

```yaml
x-seed-catalog: &seed-catalog
  volumes:
    - ../seed:/var/run/seed-catalog:ro
  environment:
    SEED_CATALOG_ROOT: /var/run/seed-catalog
```

Each of the eight service blocks references the anchor with `<<: *seed-catalog`. No git-sync container in compose — the bind mount means the laptop's working copy is the catalog.

### 4.8 CI catalog linter

`tools/catalog-lint/` is a Go binary invoked by CI on every PR. It performs:

- Walks `deploy/seed/**/*.json`, skipping `_*` files and directories.
- Validates each file parses as JSON:API (`{data: {type: string, id: string, attributes: object, relationships?: object}}`).
- Validates `data.id` matches the id captured from the filename by the directory's owning subdomain's `EntityIDPattern`.
- Validates `data.type` matches the subdomain's expected type (e.g., `monster-drop`, `gachapon`).
- Validates every `<region>/<major>_<minor>/` directory contains a non-empty `CATALOG_REVISION` file.
- Validates filenames are kebab-case and end in `.json`.
- Exits non-zero with a per-file error list on any violation.

A new GitHub Actions job runs `go run ./tools/catalog-lint deploy/seed/` on every PR that touches `deploy/seed/` or `tools/catalog-lint/`.

## 5. API Surface

No new HTTP endpoints. Every endpoint listed below already exists; this task changes their backing implementation without changing their URL or request/response shape.

### 5.1 Preserved endpoints

| Service | Endpoint | Notes |
|---|---|---|
| atlas-drop-information | `POST /drops/seed` | 202 Accepted, async |
| atlas-drop-information | `GET /drops/seed/status` | adds `catalogRevision` and `tenantSeededRevision` fields |
| atlas-gachapons | `POST /gachapons/seed` | 202 Accepted, async |
| atlas-gachapons | `GET /gachapons/seed/status` | adds revision fields |
| atlas-map-actions | `POST /map-actions/seed` | (verify exact URL during migration) |
| atlas-map-actions | `GET /map-actions/seed/status` | adds revision fields |
| atlas-reactor-actions | `POST /reactor-actions/seed` | (verify exact URL) |
| atlas-reactor-actions | `GET /reactor-actions/seed/status` | adds revision fields |
| atlas-portal-actions | `POST /portal-actions/seed` | (verify exact URL) |
| atlas-portal-actions | `GET /portal-actions/seed/status` | adds revision fields |
| atlas-npc-conversations | `POST /npc-conversations/npc/seed` + `POST /npc-conversations/quests/seed` | preserve existing URLs even if today's paths differ (Open Question) |
| atlas-npc-conversations | matching status endpoints | adds revision fields |
| atlas-npc-shops | existing two `POST /shops/...` endpoints | preserve URLs |
| atlas-npc-shops | matching status endpoints | adds revision fields |
| atlas-party-quests | `POST /party-quests/definitions/seed` | preserve URL |
| atlas-party-quests | `GET /party-quests/definitions/seed/status` | adds revision fields |

### 5.2 Status response shape changes

Every `GET .../seed/status` response gains two top-level fields without renaming or removing existing fields:

```json
{
  "data": {
    "type": "<existing>",
    "id": "<tenant uuid>",
    "attributes": {
      "<existing count fields>": "...",
      "updatedAt": "<existing timestamp>",
      "catalogRevision": "abc123...",
      "tenantSeededRevision": "abc012...",
      "tenantSeededAt": "2026-05-19T14:23:00Z"
    }
  }
}
```

`catalogRevision` is the revision the pod has mounted (read from `CATALOG_REVISION` per request). `tenantSeededRevision` is what the tenant was last seeded against (read from `seed_state`). When divergent, a future reconciler triggers `POST /seed`. When the tenant has never been seeded, `tenantSeededRevision` and `tenantSeededAt` are `null`.

### 5.3 Error cases

- `POST /seed` returns 202 if accepted (existing behavior; the actual work happens in a goroutine and errors are logged).
- `GET /seed/status` returns 500 with a logged error if reading the catalog `CATALOG_REVISION` fails or if `seed_state` query fails.
- The lib does not return 4xx for "catalog missing" — services start successfully without a mounted catalog; the first `POST /seed` is when the absence surfaces (logged error, 202 still returned, no rows inserted).

## 6. Data Model

### 6.1 New table per service: `seed_state`

```sql
CREATE TABLE seed_state (
  tenant_id        UUID         NOT NULL,
  group_name       TEXT         NOT NULL,
  catalog_revision TEXT         NOT NULL,
  seeded_at        TIMESTAMPTZ  NOT NULL,
  result_summary   JSONB        NOT NULL,
  PRIMARY KEY (tenant_id, group_name)
);
```

GORM `AutoMigrate(&SeedState{})` is added to each of the eight services' bootstrap. The lib owns the entity definition; services just register it for migration.

### 6.2 Catalog file shape (every file)

```json
{
  "data": {
    "type": "monster-drop",
    "id": "100100",
    "attributes": {
      "drops": [
        {
          "itemId": 2000000,
          "minimumQuantity": 1,
          "maximumQuantity": 1,
          "questId": 0,
          "chance": 1000000
        }
      ]
    }
  }
}
```

- `type` matches the owning subdomain's type identifier.
- `id` is the entity id as a string (even when the domain type is numeric); the lib parses to the correct numeric type via the subdomain's `Build` method.
- `attributes` shape is subdomain-specific; the lib treats it as opaque JSON forwarded to `Subdomain.Decode([]byte)`.
- `relationships` is allowed but currently unused by any subdomain.

### 6.3 `CATALOG_REVISION` file

A text file at each `<region>/<major>_<minor>/` directory containing a single line: typically the git SHA of the commit producing the catalog content. The lib treats it as opaque. The `git-sync` sidecar overwrites it whenever a new commit is synced (CI step writes it; see 6.4).

### 6.4 Catalog revision provenance

The repo's CI pipeline writes `deploy/seed/<region>/<version>/CATALOG_REVISION = $GITHUB_SHA` on every commit to `main` and on every PR build. This means every commit that lands produces a unique revision string. Manual edits to `CATALOG_REVISION` are discouraged but not blocked.

## 7. Service Impact

### 7.1 `libs/atlas-seeder` (new)

New module under `libs/`. Depends on `libs/atlas-rest`, `libs/atlas-tenant`, `gorm.io/gorm`, `github.com/jtumidanski/api2go/jsonapi`, `github.com/sirupsen/logrus`, `golang.org/x/sync/errgroup`, `github.com/gorilla/mux`. Has its own `go.mod`, `go.sum`. Unit-tested with fixture catalogs under `libs/atlas-seeder/testdata/`.

### 7.2 `atlas-gachapons`

- Delete `atlas.com/gachapons/seed/{seed.go,processor.go,resource.go,status.go}` and the `_test.go` files for `seed.go` and `status.go`.
- Add `atlas.com/gachapons/seed/groups.go` registering one `Group` with subdomains for `gachapon` and `global` (the `_global/items.json` file is loaded via a one-shot non-entity subdomain).
- Drop `COPY data/ ...` from `Dockerfile`. Delete `data/` from the service tree.
- Splitter `split-gachapons` produces `deploy/seed/gms/83_1/gachapons/<id>.json` + `_global/items.json` from current `data/` contents.
- Add `SeedState` to AutoMigrate.

### 7.3 `atlas-drop-information`

- Delete `atlas.com/dis/seed/{seed.go,processor.go,resource.go,status.go}` and `_test.go` siblings.
- Delete the `Load*` helpers in `atlas.com/dis/{monster,continent,reactor}/drop/seed.go` (loader logic moves into per-subdomain `Decode`/`Build` methods).
- Add `atlas.com/dis/seed/groups.go` registering one `Group` with three subdomains (monster, continent, reactor drops).
- Drop `COPY drops/ ...` from `Dockerfile`. Delete `drops/` from the service tree.
- Splitters `split-monster-drops`, `split-continent-drops`, and the generic `wrap-jsonapi` for reactor drops produce the per-entity files.
- Add `SeedState` to AutoMigrate.

### 7.4 `atlas-map-actions`

- Delete `atlas.com/map-actions/script/{seed.go,seed_status.go}` and their `_test.go` siblings.
- Add `atlas.com/map-actions/seed/groups.go` registering one `Group` with one subdomain (map scripts).
- Drop `COPY scripts/ ...` from `Dockerfile`. Delete `scripts/` from the service tree.
- Splitter `wrap-jsonapi` produces `deploy/seed/gms/83_1/map-actions/map/map-<id>.json`.
- Add `SeedState` to AutoMigrate.

### 7.5 `atlas-reactor-actions`

- Delete `atlas.com/reactor/script/{seed.go,seed_status.go}` and `_test.go` siblings.
- Add `atlas.com/reactor/seed/groups.go` registering one `Group` with one subdomain.
- Drop `COPY scripts/ ...` from `Dockerfile`. Delete `scripts/`.
- Splitter `wrap-jsonapi`.
- Add `SeedState` to AutoMigrate.

### 7.6 `atlas-portal-actions`

- Same shape as 7.5. Catalog files at `deploy/seed/gms/83_1/portal-actions/portals/portal-<map-id>-<portal-name>.json` (portal id is composite — confirm naming convention during implementation).
- Add `SeedState` to AutoMigrate.

### 7.7 `atlas-npc-conversations`

- Delete `atlas.com/npc/conversation/{npc,quest}/{seed.go,seed_status.go}` and `_test.go` siblings.
- Add `atlas.com/npc/seed/groups.go` registering **two** `Group`s: one for `npc-conversations/npc/` and one for `npc-conversations/quests/`.
- Drop `COPY conversations/ ...` from `Dockerfile`. Delete `conversations/`.
- Splitter `wrap-jsonapi` against both `conversations/npc/` and `conversations/quests/` trees.
- Add `SeedState` to AutoMigrate (one table, used for both groups via the `group_name` discriminator).

### 7.8 `atlas-npc-shops`

- Delete the two existing seed packages (`atlas.com/npc/seed/` and the seed code in `atlas.com/npc/shops/seed.go`).
- Add `atlas.com/npc/seed/groups.go` registering two `Group`s preserving the existing two endpoints.
- Drop `COPY data/ ...` from `Dockerfile`. Delete the `data/` tree under the service.
- Splitter `wrap-jsonapi`.
- Add `SeedState` to AutoMigrate.

### 7.9 `atlas-party-quests`

- Delete `atlas.com/party-quests/definition/{Seed* code in processor.go, SeedDefinitionsHandler in resource.go}`.
- Add `atlas.com/party-quests/seed/groups.go` registering one `Group` with one subdomain.
- Drop `COPY party-quests/ ...` from `Dockerfile`. Delete `party-quests/` from the service tree.
- Splitter `wrap-jsonapi`.
- Add `SeedState` to AutoMigrate.

### 7.10 `deploy/k8s/base/`

- New Kustomize component `deploy/k8s/base/components/seed-catalog/` containing the `git-sync` sidecar spec, the `emptyDir` volume, the volume mount, and the `SEED_CATALOG_ROOT` env.
- Each of the eight `atlas-<svc>.yaml` manifests references the component.
- The component is parameterized so the git ref (`main` vs PR SHA) flows through per-overlay.

### 7.11 `deploy/compose/`

- `docker-compose.yml` gets the `x-seed-catalog` anchor and `<<: *seed-catalog` references on each of the eight services.
- `up.sh` and `build.sh` need no changes.

### 7.12 `tools/`

- `tools/seed-splitters/{split-monster-drops,split-continent-drops,split-gachapons,wrap-jsonapi}/` — each a Go program with `main.go`.
- `tools/catalog-lint/main.go` — the CI linter.

### 7.13 `.github/workflows/` (or repo's CI config)

- New workflow step running `go run ./tools/catalog-lint deploy/seed/` triggered on PRs touching `deploy/seed/**` or `tools/catalog-lint/**`.

### 7.14 Out-of-scope services (no changes)

- `atlas-configurations`: untouched. Its `seeder/` package and `seed-data/templates/` remain. Migration deferred.
- `atlas-tenants`: untouched. Its payload-driven `POST /tenants/{id}/configurations/{routes,vessels,instance-routes}/seed` endpoints continue to function as today.

## 8. Non-Functional Requirements

### 8.1 Performance

- Catalog walk + bulk insert for the largest single service (npc-conversations at 5.2 MB across hundreds of files) must complete in under 60s on a default Postgres dev cluster. The current implementation already achieves this; the lib's parallel-per-subdomain orchestration is equal or better.
- `GET /seed/status` must respond within the existing latency budget (~50ms p95 today across the eight services); the added `seed_state` lookup is one indexed PK read and negligible.
- The `git-sync` sidecar's per-poll bandwidth and disk I/O must be bounded — only the `deploy/seed/` subtree is synced, not the full monorepo.

### 8.2 Security

- The catalog is read-only from the service container's perspective (`:ro` bind mount in compose, the emptyDir is writable only by git-sync). No service can mutate its own catalog.
- The `seed_state` table is service-local; tenants can read their own row via the existing status endpoint, never another tenant's.
- The CI catalog linter rejects files containing executable references, shell commands, or anything beyond JSON:API attributes; this is the existing posture (catalog files have always been pure data).
- No new secrets. The git-sync sidecar accesses the same Atlas monorepo it already has visibility to via ArgoCD.

### 8.3 Observability

- The lib logs structured entries on `Seed` start, per-subdomain completion, and `Seed` completion, all keyed by `tenant_id`, `group_name`, and `catalog_revision`.
- The lib logs `WARN` when `catalogRevision` on status read differs from `tenantSeededRevision` (this is the signal a future reconciler would consume).
- The lib emits a Prometheus counter `atlas_seeder_runs_total{service, group, outcome}` and a histogram `atlas_seeder_duration_seconds{service, group}` so seeding behavior is visible alongside other service metrics.
- The git-sync sidecar's logs land in the same pod log stream; operators reading service logs see both.

### 8.4 Multi-tenancy

- Every `Seed` and `Status` invocation extracts `tenant.MustFromContext` at the boundary; per the existing pattern, a fresh `tenant.WithContext(context.Background(), t)` is used for the background goroutine so cancellation of the original request does not abort the seed.
- `seed_state` rows are tenant-scoped; PK is `(tenant_id, group_name)`.
- The catalog path resolution uses `tenant.Region()` + `MajorVersion()` + `MinorVersion()`. Tenants with the same `(region, major, minor)` share catalog files via the read-only mount — no per-tenant file duplication on disk.

### 8.5 Backward compatibility

- All existing endpoint URLs remain functional.
- The atlas-ui seed buttons continue to work without modification.
- Existing `SEED_*` env vars (e.g., `GACHAPONS_DATA_PATH` in gachapons today) are removed; the migration is atomic per service, no transitional period with both old and new paths.
- The pre-migration containers' bundled catalogs are deleted on the same commit that adds the mount; rollback means rolling back the commit (the image without the catalog and without the mount cannot seed).

## 9. Open Questions

1. **Today's actual endpoint URLs for the script services.** The PRD assumes `POST /map-actions/seed`, `POST /reactor-actions/seed`, `POST /portal-actions/seed`. The grep showed they exist but did not show the exact paths. The implementation phase must confirm and the PRD requirement is "preserve whatever exists today" — if the actual URLs differ, those are the URLs the migrated implementation uses.
2. **npc-conversations endpoint URLs.** The two seed packages live at `conversation/npc/seed.go` and `conversation/quest/seed.go`; the actual mounted URL prefixes (likely `/conversations/npc/seed` and `/conversations/quests/seed`, or possibly `/npc/seed` and `/quest/seed`) need verification before the `Group.URLPrefix` is set. Whatever they are today, they stay.
3. **Portal id encoding in filenames.** Portals are identified by `(map_id, portal_name)` in MapleStory data, not a single integer id. The PRD proposes `portal-<map-id>-<portal-name>.json` but portal names contain characters that may need escaping (spaces, special characters). The implementation phase decides the escaping rule.
4. **`SEED_CATALOG_ROOT` default in dev.** For local `go run` against the service from source (no container), should the lib default `SEED_CATALOG_ROOT` to `./deploy/seed` (repo-relative) or require explicit setting? Default makes from-source dev frictionless; explicit setting reduces ambient state.
5. **git-sync ref strategy for PR overlays.** ArgoCD's `overlays/pr` is parameterized per PR; the git-sync sidecar's ref must follow. Implementation phase confirms whether this is a Kustomize patch, an ArgoCD ApplicationSet parameter, or an env-var-driven sidecar arg.
6. **CI catalog linter — strict on all branches, or only on PRs?** Recommendation: strict on PRs, advisory (log-only) on `main` so a catalog mistake on `main` doesn't block deploys — but this is a polish decision deferred to implementation.

## 10. Acceptance Criteria

This task is complete when all of the following hold:

### 10.1 Library

- [ ] `libs/atlas-seeder/` exists with `Subdomain`, `Group`, `CatalogSource`, `Seed`, `Status`, `RegisterRoutes` exported.
- [ ] Unit tests in `libs/atlas-seeder/` cover: parallel subdomain fan-out, error aggregation, JSON:API parsing, filename-id mismatch detection, missing `CATALOG_REVISION` handling, `seed_state` row write/read, tombstone file behavior (forward-compat).
- [ ] `go test -race ./...` clean inside `libs/atlas-seeder/`.

### 10.2 Per-service migration (all 8)

For each of `atlas-gachapons`, `atlas-drop-information`, `atlas-map-actions`, `atlas-reactor-actions`, `atlas-portal-actions`, `atlas-npc-conversations`, `atlas-npc-shops`, `atlas-party-quests`:

- [ ] The old `seed/` package or seed code is deleted.
- [ ] The service registers its `Group`(s) with `libs/atlas-seeder.RegisterRoutes`.
- [ ] The service's `Dockerfile` no longer `COPY`s catalog data.
- [ ] The service's bundled catalog directory is deleted from `services/<svc>/`.
- [ ] The service's `k8s/base/atlas-<svc>.yaml` includes the seed-catalog Kustomize component.
- [ ] The service's compose entry uses `<<: *seed-catalog`.
- [ ] The service's bootstrap calls `db.AutoMigrate(&seeder.SeedState{})`.
- [ ] `POST /<existing-prefix>/seed` returns 202 against a tenant.
- [ ] `GET /<existing-prefix>/seed/status` returns existing fields plus `catalogRevision`, `tenantSeededRevision`, `tenantSeededAt`.
- [ ] `go test -race ./...` clean in the service module.
- [ ] `go vet ./...` clean in the service module.
- [ ] `go build ./...` clean in the service module.
- [ ] `docker build -f services/<svc>/Dockerfile .` from the worktree root succeeds.

### 10.3 Catalog

- [ ] `deploy/seed/gms/{12_1,83_1,87_1,92_1,95_1}/` and `deploy/seed/jms/185_1/` all exist with `CATALOG_REVISION` and the subdomain directories populated.
- [ ] For v83_1: every per-entity file is the splitter-script output of the corresponding old bundled data.
- [ ] For v12_1, v87_1, v92_1, v95_1, jms/185_1: contents bootstrap from v83_1 with `CATALOG_REVISION` indicating the bootstrap.
- [ ] No service's `services/<svc>/` tree contains catalog data anymore.

### 10.4 Infra

- [ ] `deploy/k8s/base/components/seed-catalog/` exists and is referenced by all eight `atlas-<svc>.yaml` manifests.
- [ ] `deploy/compose/docker-compose.yml` has the `x-seed-catalog` anchor and `<<: *seed-catalog` on all eight service blocks.
- [ ] `kubectl apply --dry-run=server -k deploy/k8s/overlays/main` passes.
- [ ] `docker compose -f deploy/compose/docker-compose.yml config` succeeds.

### 10.5 Tooling

- [ ] `tools/seed-splitters/{split-monster-drops,split-continent-drops,split-gachapons,wrap-jsonapi}/` exist, each is a runnable Go program with a `--help` flag.
- [ ] Each splitter is deterministic: rerunning it produces byte-identical output.
- [ ] `tools/catalog-lint/main.go` exists; `go run ./tools/catalog-lint deploy/seed/` exits 0 on clean catalog and non-zero on intentionally-malformed fixtures under `tools/catalog-lint/testdata/`.
- [ ] CI workflow step invokes `tools/catalog-lint` on PRs touching `deploy/seed/**` or `tools/catalog-lint/**`.

### 10.6 End-to-end

- [ ] In a local docker-compose stack, `POST /drops/seed` (with a tenant header) against atlas-drop-information returns 202, then `GET /drops/seed/status` reports non-zero counts and a `catalogRevision` matching `cat deploy/seed/gms/83_1/CATALOG_REVISION`.
- [ ] Editing one file under `deploy/seed/gms/83_1/drops/monsters/` and re-POSTing `/drops/seed` updates `tenantSeededRevision` on next status read (after CI updates `CATALOG_REVISION`; for local testing, manually overwriting `CATALOG_REVISION` simulates this).
- [ ] All eight services boot in compose without crashing when `deploy/seed/gms/83_1/` is mounted.
