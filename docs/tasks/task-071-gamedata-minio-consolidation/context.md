# task-071 Context

> Companion to `plan.md`. Pre-loaded knowledge an implementer needs before opening a single file. Sourced from `prd.md` v5, `design.md` v1, and direct repo inspection performed during plan authorship (2026-05-20).

## Goal in one breath

Replace the three-PVC WZ + asset + data pipeline (`atlas-wz-extractor` → PVC → `atlas-data` + `atlas-assets` PVC) with a MinIO-backed, library-extracted pipeline. WZ parser becomes `libs/atlas-wz`. Ingest lives inside `atlas-data` (REST creates a Kubernetes Job; compose runs in-process). Character + map rendering moves to a new stateless `atlas-renders` service. `atlas-wz-extractor` and `atlas-assets` are deleted. PR-env bootstrap collapses from ~10 min to ~60 s via a canonical Postgres dump in MinIO.

## Authoritative documents

| Document | Authority |
|---|---|
| `prd.md` v5 | What we are building and why. Section numbers (§N) referenced throughout the plan come from the PRD unless prefixed `design §`. |
| `design.md` v1 | How. Pins every PRD-deferred decision (HPA = KEDA on RPS; root creds via `kubernetes-replicator`; PNG encoder vendored from Go 1.21 `image/png`; pack algo = MaxRects-BSSF; bin sizing 256→4096; tar+sha256 baseline dump). |
| `CLAUDE.md` (worktree root) | Build verification + four-location Dockerfile rule. Mandatory. |

## Source-of-truth files (read before you touch)

These were inspected during plan authorship. Re-read them before executing the task that touches them — they may have moved.

### atlas-wz-extractor (to be deleted; donor source)

- `services/atlas-wz-extractor/atlas.com/wz-extractor/wz/` — `reader.go`, `file.go`, `directory.go`, `image.go`, `property/`, `canvas/`, `crypto/`. **Port verbatim** into `libs/atlas-wz/{wz,canvas,crypto}`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/image/` — `extract.go` (icon extraction), `character_parts.go`, `minimap.go`, `zmap.go`. Splits into `libs/atlas-wz/icons` (extract.go), `libs/atlas-wz/mapimage` (minimap.go, zmap.go), and the character-parts info feeds the new `libs/atlas-wz/atlas` packer.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/` — `decoder.go`, `entries.go`, `background.go`, `blit.go`, `bounds.go`, `property.go`, `renderer.go`, `sort.go`. The render/blit/sort logic moves into **`services/atlas-renders/atlas.com/renders/mapr/composite.go`** (NOT to the library — atlas-renders is the only caller). The layer-extraction half (`decoder.go`, `entries.go`, `property.go`) feeds `libs/atlas-wz/mapimage`.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/` — `handler.go`, `hash.go`, `query.go`, `path.go`, `write.go`, `resource.go`, `error.go`, `otel.go`. **Moves wholesale** to `services/atlas-renders/atlas.com/renders/character/` after rewiring storage from filesystem to MinIO.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/` — `dispatcher.go`, `processor.go`, `upload.go`, `pool.go`, `status.go`, `tenant_path.go`, `job_handler.go`, `map_render.go`. **Ports semantics** into atlas-data ingest workers + new `runtime/{rest,ingest,all}` packages, then deleted at cutover.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/xml/serializer.go` — **deleted outright.** XML intermediate retired.
- `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go`, `Dockerfile` — deleted at the end of the cutover PR.

### atlas-data (modified heavily)

- `services/atlas-data/atlas.com/data/main.go` — wraps `MODE` dispatch around the existing `service.New(...)` boot. The dispatch lives in new `runtime/` package.
- `services/atlas-data/atlas.com/data/data/processor.go` — today reads from `ZIP_DIR` filesystem; replace its source with MinIO-fetched `.wz` files in scratch. The Worker constant set (`WorkerMap`, `WorkerMonster`, …) carries over to the new ingest workers. Per-worker `documents`+`searchindex` write logic is unchanged in shape.
- `services/atlas-data/atlas.com/data/data/kafka.go` — `EnvEventTopic = "EVENT_TOPIC_DATA"`, `EventTypeDataUpdated = "DATA_UPDATED"`. **EMIT-SIDE STAYS.** PRD §4.4 mandates this.
- `services/atlas-data/atlas.com/data/kafka/consumer/data/consumer.go` — handles `COMMAND_TOPIC_DATA` (`data_command`). **RETIRED.** Delete this whole package at the cutover.
- `services/atlas-data/atlas.com/data/document/entity.go` — `documents` table schema. Stable; only the migration set must be re-fingerprinted (PRD §6.1).
- `services/atlas-data/atlas.com/data/searchindex/searchindex.go` — the shared trigram-index helper. `Migrate`, `Upsert`, `DeleteAllForTenant` are the API surface the restore path will call.
- Search-index tables — fixed list: `documents`, `monster_search_index`, `npc_search_index`, `reactor_search_index`, `map_search_index`, `item_string_search_index`. Defined in `monster/entity.go`, `npc/entity.go`, `reactor/entity.go`, `map/entity.go`, `item/string_entity.go`. The baseline dump iterates exactly these 6 tables.
- `services/atlas-data/atlas.com/data/setup/processor.go` — current XML-driven setup loader. Will be removed once ingest writes documents directly from WZ.
- `services/atlas-data/atlas.com/data/xml/` — XML reader used by current ingest. **Deleted at cutover.**
- `services/atlas-data/Dockerfile` — has the four-location pattern (go.mod COPY, go.work synthesis, source COPY, `go mod edit -replace`). The new `libs/atlas-wz` lib MUST be added in all four locations (CLAUDE.md mandate).

### atlas-ui

- `services/atlas-ui/src/services/api/seed.service.ts` — currently contains:
  - `uploadWzFiles` → `PATCH /api/wz/input`. Repoint to `/api/data/wz`, accept `scope` arg.
  - `runWzExtraction` → `POST /api/wz/extractions`. **Delete.**
  - `runDataProcessing` → `POST /api/data/process`. Accept `scope` arg.
  - `getWzInputStatus` → `GET /api/wz/input`. Repoint to `/api/data/wz`.
  - `getExtractionStatus` → `GET /api/wz/extractions`. **Delete.**
  - `getDataStatus` → `GET /api/data/status`. Augment `DataStatus` with `baselineRestoredAt`, `baselineSha256`.
- `services/atlas-ui/src/pages/SetupPage.tsx` — restructure per PRD §4.8.
- `services/atlas-ui/public/sw-character-cache.js` — bump `CACHE_NAME` constant in the cutover commit.
- New files: `src/services/api/baseline.service.ts`, `src/lib/hooks/api/useBaseline.ts`, `src/components/features/setup/ScopeToggle.tsx`.
- The `extraction.service.ts` and `useExtraction.ts` files **referenced in the PRD § 4.8 do not currently exist** — the extraction surface lives entirely in `seed.service.ts`. The plan removes the extraction-related methods from `seed.service.ts` rather than deleting files that aren't there. **Verify with `git ls-files services/atlas-ui/src | grep -i extraction` before each touching task.**

### Deploy

- `deploy/k8s/base/atlas-data.yaml` — drops PVC + mount, adds MinIO env, adds ServiceAccount/Role/RoleBinding, switches to `strategy: Recreate`.
- `deploy/k8s/base/atlas-wz-extractor.yaml` — **delete file** at cutover.
- `deploy/k8s/base/atlas-assets.yaml` — **delete file** at cutover.
- `deploy/k8s/base/kustomization.yaml` — drop the two deleted manifests; add `atlas-renders.yaml`, `atlas-data-ingest-job-template.yaml`, `atlas-minio-init.yaml`.
- `deploy/shared/routes.conf` — currently has `/api/assets` → `atlas-assets:8080`, `/api/data` → `atlas-data:8080`, `/api/wz` → `atlas-wz-extractor:8080` (lines 176, 209, 214). Rewrite the assets + wz blocks per design §5.1; the data block stays.
- `deploy/compose/docker-compose.yml`, `docker-compose.core.yml` — add `minio`, `minio-init`, `atlas-renders`. Remove `atlas-wz-extractor` and `atlas-assets`. Reshape `atlas-data`.
- `~/source/k3s/bee/minio.yml` — pin MinIO image tag. **Outside this worktree; coordinate with operator.** Plan covers what to set; the apply is operator-driven.

### PR bootstrap

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh`, `lib.sh`, `cleanup.sh` — currently call `/api/wz/input` (`wz-upload`), `/api/wz/extractions` (`wz-extract`), `/api/data/process` (`data-process`). Replace with `BOOTSTRAP_MODE` branching per design §7.2.

## Cross-cutting decisions you cannot re-litigate

| Decision | Source | Why pinned |
|---|---|---|
| WZ parser is a library, not a service | PRD §4.1 | Eliminates a service boundary that exists only because of the XML intermediate. |
| `MODE=rest` creates k8s Jobs; not in-process workers | design §3.2 | A single ingest can hold 4–8 GB for 10+ min; co-locating with REST oversizes the REST pod. |
| Vendored PNG encoder (frozen Go 1.21 `image/png`), not stdlib | design §2.4 | Go 1.22+ changed filter heuristic — stdlib would invalidate baselines on Go upgrade. |
| MaxRects-BSSF with `(width desc, height desc, name asc)` pre-sort | design §2.3 | Tightest pack on equip sets + only single-threaded heuristic with no rand tie-break. |
| Baseline dump = tar of per-table COPY-binary streams; sha256 sidecar | PRD §6.1, design §3.5 | `pg_dump` can't filter by tenant_id natively. Binary COPY is the same wire format with WHERE baked in; 16-byte rewrite per row on restore. |
| HPA = KEDA Prometheus scaler on `atlas_renders_requests_total` | design §4.6 | Renders are network-bound; CPU HPA fires too late. |
| MinIO root-cred bootstrap via `kubernetes-replicator` annotations | design §6.3 | Cluster already deploys it. |
| `atlas-minio-init` runs as Argo **PreSync** hook + sync wave -2 | PRD §7.10, design §6.3 | Apps depend on the Secret it writes; PostSync crash-loops them. |
| `tenants/`, `regions/`, `versions/` REST-plural in bucket keys | PRD §4.2 v4 fix | Avoids confusion with the legacy `tenant/` singular and reads as URLs. |
| Map render is **lazy** (atlas-renders composes on first request) | PRD §4.7 | Halves ingest wall-clock and avoids materializing unused maps. |
| Tenant-purge endpoint pulled into this task | PRD §4.4a | Without it, PR envs leak Postgres rows + MinIO prefixes. |
| `atlas-data` deploy uses `strategy: Recreate` | design §7.3 | Old (PVC, XML) and new (MinIO) pods cannot interleave without corruption. |
| Emit-side `EVENT_TOPIC_DATA` (`DATA_UPDATED` event) kept | PRD §4.4 + design §3.9 | Downstream services (atlas-channel, atlas-character-factory, atlas-maps) subscribe to it. Only the input `COMMAND_TOPIC_DATA` is retired. |
| `documents` table has no `region` or `version` column — one (region, version) per tenant | PRD §6.1 | Restore is destructive-by-construction (DELETE then COPY). |
| atlas-renders may import `libs/atlas-wz/manifest` and `libs/atlas-wz/maplayout` only | PRD §4.7 + design §4.1 | The WZ parser is never on the runtime hot path. CI lint enforces via `go list -deps`. |

## Build verification (CLAUDE.md mandate)

Per the worktree's CLAUDE.md, every commit that changes Go code must pass:

1. `go test -race ./...` in every changed module.
2. `go vet ./...` in every changed module.
3. `go build ./...` in every changed service.
4. `docker build -f services/<svc>/Dockerfile .` from the worktree root for every service whose `go.mod` or `Dockerfile` was touched. **This is mandatory** because the four-location Dockerfile pattern (go.mod COPY, go.work synthesis, source COPY, `go mod edit -replace`) is not caught by `go build` against the workspace `go.work`.

The plan's TDD steps emit per-step `go test` commands. The Docker-build step lives at the end of every task that touches a Dockerfile and at the end of every task that adds a new `libs/atlas-*` dependency in a service.

## Glossary (sticking points)

| Term | Meaning |
|---|---|
| **canonical (bucket)** | `atlas-canonical`, the MinIO bucket that holds operator-published `.wz` archives and Postgres baseline dumps. |
| **canonical (scope)** | The reserved tenant UUID `00000000-0000-0000-0000-000000000000`. Documents written under this tenant are the canonical row set. |
| **shared (scope prefix)** | The key prefix `shared/...` inside `atlas-assets`. Artifacts here are reusable across tenants on the same `(region, version)`. |
| **tenant scope prefix** | `tenants/<tenantId>/...` inside `atlas-assets`. Reserved for per-tenant overrides; not exercised by this task. |
| **MODE** | atlas-data env. `rest` (k8s REST pod, creates Jobs), `ingest` (k8s Job pod, no HTTP), `all` (compose, REST + workers in one process). |
| **scope-key** | Either `shared` or `tenants/<tenantId>`. Used in bucket keys after the bucket name. |
| **DATA_UPDATED** | Kafka event emitted by atlas-data after ingest; consumed by atlas-channel / -character-factory / -maps. Stays. |
| **COMMAND_TOPIC_DATA** | Old Kafka input topic that dispatched WZ ingest work. **Retired.** |
| **PreSync hook** | Argo CD lifecycle stage that runs the init Job before any application sync. Required so the `atlas-minio-credentials` Secret exists before atlas-data/atlas-renders start. |

## Dependencies between tasks (read the order)

The plan tasks are ordered to keep each commit shippable. The hard dependencies are:

1. `libs/atlas-wz` scaffolding and tests must land before atlas-data ingest workers can import them.
2. The four-location Dockerfile fix-up for atlas-data lands in the same commit as the first `libs/atlas-wz` import (otherwise CI is silently broken).
3. The MinIO Go SDK wiring + bucket env wiring in atlas-data must land before the new endpoints; new endpoints land before the Job-template ConfigMap (the template references env names defined by the endpoint code).
4. atlas-renders cannot land in a working state until `libs/atlas-wz/manifest` and `libs/atlas-wz/maplayout` exist as importable subpackages.
5. atlas-ingress routes.conf rewrite lands only after atlas-renders is deployable (otherwise the new regex routes to a non-existent upstream).
6. `atlas-wz-extractor` and `atlas-assets` deletion is the **final** commit on the branch: the cutover-PR-env exercise must succeed first (smoke-test list from PRD §10).

## Risk flags (carry into execution)

- **PNG encoder vendoring** — design §2.4 specifies "frozen fork of Go 1.21's `image/png`". Implementer must check in the encoder under `libs/atlas-wz/atlas/pngenc/` with a LICENSE header from the Go source (BSD-style). Don't import `image/png` from stdlib in the determinism path.
- **Job template ConfigMap drift** — the Job pod env must match what `runtime/ingest` reads from env. A typo in the ConfigMap is only caught by a working end-to-end test in `MODE=rest` (not a `go test`).
- **`go list -deps` lint** — the renderer-imports lint is a CI step, not a Go test. It belongs in the GitHub Actions workflow (typically `.github/workflows/`); plan tasks include this manifest change.
- **Recreate strategy + Replicas: 4** — current atlas-data has 4 replicas. With `Recreate`, all 4 drain simultaneously. Other services tolerate this (data is cached); confirm none are mid-startup at the same time as the cutover deploy.
- **MinIO image tag pin** — the design pins `RELEASE.2026-01-15T01-30-12Z` as the target. The literal tag at PR-open time may be newer; choose **one** tag and use it in both compose and `~/source/k3s/bee/minio.yml`.
