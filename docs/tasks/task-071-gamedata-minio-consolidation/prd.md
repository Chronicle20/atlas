# Game Data + Asset Pipeline Consolidation onto MinIO — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-19

---

## 1. Overview

Atlas's static game data and asset pipeline today is three services bound together by shared `ReadWriteMany` PVCs. `atlas-wz-extractor` parses proprietary WZ binary archives and writes two artifact trees: HaRepacker-compatible XML to `atlas-data-pvc` (consumed by `atlas-data`, parsed into JSON documents in Postgres, served as JSON:API) and per-entity PNG icons (plus map renders) to `atlas-assets-pvc` (served by an `atlas-assets` nginx pod). A third PVC, `atlas-wz-input-pvc`, holds raw `.wz` uploads. The character-render hot path (added in task-043) lives inside `atlas-wz-extractor`, opens `.wz` files at request time, and writes composited paperdoll PNGs back to `atlas-assets-pvc` for the nginx to serve.

This architecture has three structural problems. First, every cross-service handoff is a shared filesystem; the only reason the XML artifact exists on disk is because that's how extractor and atlas-data communicate. Second, the WZ binary parser is on the runtime hot path: character renders open `.wz` files at request time and seek to canvas offsets, which is fast over a local PVC but would be ruinous over network object storage. Third, ephemeral PR environments (task-063, task-070) re-run the entire WZ extraction + data ingest from scratch on every PR — currently the dominant cost in PR-env bootstrap (~10+ minutes), and explicitly called out in task-063's open questions as the optimization to pursue.

The cluster is moving off RWX storage onto MinIO; that move is forced by storage policy and is also a chance to fix the architecture rather than just relocate it. This task replaces the three-PVC pipeline with a MinIO-backed model in which `.wz` is the only raw input artifact and is needed *only at extraction time*. Extraction pre-materializes every visual artifact the renderer would ever need — icons, character sprite atlases with manifests, map renders — as static, content-addressable objects in MinIO. The WZ parser becomes a build-time concern, packaged as `libs/atlas-wz`. At runtime, character render becomes a stateless image compositor: fetch atlases and manifests from MinIO, composite, write the result back to MinIO. `atlas-assets` is eliminated; clients fetch assets through the existing atlas-ingress proxy, which now routes to MinIO directly. The artifact set itself is split between a **canonical baseline** (shared across all tenants on the same `(region, version)`) and **per-tenant overrides** (currently unused but supported by the layout). PR envs hydrate by referencing the canonical prefix and restoring a baked Postgres `documents` dump, collapsing bootstrap from ~10 minutes to seconds.

The wins are operational (no shared PVCs, no RWX dependency, MinIO is the only state-bearing artifact store), architectural (WZ format quirks live in one library and one Job, not in long-running services), performance (no random-access reads against object storage; the renderer never opens a `.wz` file), and developer-velocity (PR envs ready in seconds instead of minutes).

## 2. Goals

Primary goals:

- Eliminate `atlas-data-pvc`, `atlas-assets-pvc`, and `atlas-wz-input-pvc`. After cutover, no service uses an RWX volume.
- Move all extraction inputs and outputs to MinIO. `.wz` files live in MinIO and are downloaded to local scratch only during extraction.
- Pre-materialize sprite atlases + JSON manifests for every equip/hair/face/body part needed for character compositing, so the character render service no longer parses WZ at runtime.
- Extract a `libs/atlas-wz` library containing the parser, canvas decoder, and atlas packer. Used by extraction (atlas-data ingest) only; the renderer does not link it.
- Retire the `atlas-assets` service. atlas-ingress proxies asset URLs directly to MinIO.
- Split MinIO artifact key space into a **canonical** prefix (shared across tenants for a given `(region, version)`) and a **per-tenant** prefix (overlays). Reads prefer per-tenant, fall back to canonical.
- Reduce PR-env bootstrap from ~10+ minutes to seconds by referencing the canonical MinIO prefix and restoring a canonical Postgres documents dump tagged with the PR-env's tenant_id, rather than re-extracting.
- Preserve atlas-ui's URL shape (`/api/assets/...`) and all client-facing behavior. The change is invisible to the UI source other than possibly its env defaults.
- Update `docker-compose` to add MinIO, drop the extractor + assets services, and continue to support an end-to-end local dev flow.
- Cut over in a single PR ("big bang"). Revert is the rollback strategy.

Non-goals:

- WZ format upgrades, support for game versions beyond v83, or any per-version code branching.
- Redesigning the Postgres `documents` schema or per-tenant trigram search indexes (orthogonal; see task-061).
- Redesigning the character-render request/response shape from task-043. This task swaps storage and removes WZ runtime parsing; the API contract is unchanged.
- Pet, mount, or cash-equipment slot rendering. Out of scope per task-043.
- Animation interpolation, GIF output, or multi-frame responses.
- CDN selection or external-network caching. The atlas-ingress route to MinIO is the boundary; a CDN can be layered later.
- Replicated / multi-region MinIO. Single-node single-drive MinIO as currently deployed is the target.
- Multi-tenant WZ override workflows. The per-tenant prefix is reserved but not exercised in this task — no tenant currently customizes WZ data beyond region/version selection.
- Backwards-compatible URL hosts. atlas-ui's `/api/assets/...` shape is preserved, but the underlying ingress route and proxy target change.
- Production-credential rotation flow for MinIO. Initial creds match the existing `minio.yml` defaults; rotation procedure documented but not automated.

## 3. User Stories

- As an operator, I want to upload raw `.wz` files for a tenant via REST and have the resulting documents, icons, and character atlases land in Postgres + MinIO without me knowing about PVCs.
- As an operator opening a PR, I want my ephemeral env to be ready in well under a minute, with game data already loaded, because re-running WZ extraction per PR is wasted work.
- As an operator, I want to publish a new canonical baseline (new game version, new asset set) by running extraction once and having every subsequent PR env consume the result.
- As a developer, I want one binary (`atlas-data`) that owns the WZ ingest end-to-end, so I don't have to debug a serialization step between two services.
- As a developer, I want the character render service to be a thin compositor over object storage so its cold-start cost is "open HTTP client", not "mmap a 200MB WZ archive."
- As a developer running `docker compose up` locally, I want a single command to spin up the full stack including MinIO and pre-seeded canonical artifacts.
- As an admin viewing atlas-ui character tiles, I want renders and icons to load with the same latency profile as today (or better), with no externally visible URL change.
- As a cluster operator, I want to roll my storage class without coordinating with three services that each hold open file handles against an RWX PVC.

## 4. Functional Requirements

### 4.1 `libs/atlas-wz` library

- New Go module `libs/atlas-wz` exporting:
  - `wz.File`, `wz.Directory`, `wz.Image`, property types (currently in `services/atlas-wz-extractor/atlas.com/wz-extractor/wz/`).
  - `crypto.WzKey` and the three IV seed encryption variants (GMS, KMS, Empty).
  - `canvas.Decode` for canvas → `image.Image` conversion across the eight supported pixel formats (BGRA4444, BGRA8888, ARGB1555, BGR565, BlockRGB565, DXT3, DXT5, DXT3Gray).
  - `atlas.Pack(canvases) → (sheet image.Image, manifest Manifest)` taking a set of named canvases and returning a single composite PNG plus a manifest describing each canvas's rect, origin, anchors, and z-order data needed by the compositor.
- Library is I/O-agnostic: accepts `io.ReaderAt` (for WZ) and returns `image.Image` / structs. It does not call `os` or `minio` directly. Callers supply the I/O.
- The library has no dependencies on Atlas service code. Stdlib + `golang.org/x/image` (or equivalent) only.
- The existing parser code in `services/atlas-wz-extractor/atlas.com/wz-extractor/{wz,crypto,image,xml}` is moved into the library where useful, deleted where not. The XML emitter is deleted, not moved.
- Determinism guarantee: `atlas.Pack` produces byte-identical output for identical inputs, including byte-identical PNG encoding (fixed compression level, fixed filter set, sorted child ordering at every level of the layout). This is load-bearing for §4.4's canonical-baseline reuse.

### 4.2 MinIO bucket layout, access policy, and canonical/tenant split

Three buckets in the existing `minio` namespace, all served by the existing `minio` Service (cluster DNS `minio.minio.svc.cluster.local:9000`).

| Bucket | Access | Purpose |
|---|---|---|
| `atlas-wz` | private | Raw `.wz` uploads. Key prefix: `<scope>/<region>/<major>.<minor>/<filename>.wz` |
| `atlas-assets` | anonymous-read | Pre-materialized visual artifacts (icons, atlases, manifests, map renders). |
| `atlas-renders` | anonymous-read | Composited character paperdoll PNGs. |

`<scope>` is one of:

- `canonical` — shared across all tenants on the same `(region, version)`. Operator-published once per game version.
- `tenant/<tenantId>` — per-tenant overrides. Currently unused; reserved for future per-tenant WZ customization.

Asset and render reads follow a per-tenant-then-canonical lookup order. atlas-renders, when building the path to fetch an atlas, first probes `atlas-assets/tenant/<tenantId>/<region>/<version>/...` and falls back to `atlas-assets/canonical/<region>/<version>/...` if the per-tenant key is absent. Probes use HEAD requests; an in-pod LRU records the resolved scope per `(tenant, region, version, partClass)` so the fallback decision is made at most once per cold pod per part class.

For atlas-ingress-served icons (where the URL is built by the UI and the ingress proxies directly to MinIO), the fallback is resolved at the ingress layer via `try_files`-equivalent. See §4.3 for the ingress configuration.

### 4.3 atlas-ingress routing changes

atlas-ingress (`deploy/shared/routes.conf`) is the single boundary the UI talks to. The two routes that change:

- `/api/assets/(.*)` → atlas-ingress nginx serves from MinIO directly. For each request, it first probes `atlas-assets/tenant/<tenantId>/<region>/<version>/<rest>` and falls back to `atlas-assets/canonical/<region>/<version>/<rest>`. Implementation: nginx `try_files`-style cascade via two `proxy_pass` locations and an `error_page 404 = @canonical` fallback.
- `/api/wz/character/render/(.*)` → atlas-renders Service. The UI continues to build `/api/assets/...` URLs for renders (preserving today's behavior); the ingress detects the `/character/<hash>.png` segment in the assets path and routes it to atlas-renders, which produces the PNG on miss and writes it back to MinIO before responding.

The path detection lives in the ingress regex. The two relevant patterns:

```
# Character render is identified by the /character/<hex>.png suffix.
location ~ ^/api/assets/(?<tenant>[^/]+)/(?<region>[^/]+)/(?<ver>[0-9]+\.[0-9]+)/character/(?<hash>[a-f0-9]+)\.png$ {
  proxy_pass http://atlas-renders:8080/api/wz/character/render/$tenant/$region/$ver/$hash.png$is_args$args;
  proxy_set_header TENANT_ID $tenant;
  proxy_set_header REGION $region;
  proxy_set_header MAJOR_VERSION ...;   # extracted from $ver
  proxy_set_header MINOR_VERSION ...;
}

# Everything else under /api/assets/ is static, served from MinIO with
# per-tenant→canonical fallback.
location ~ ^/api/assets/(?<rest>.+)$ {
  rewrite ^ /atlas-assets/tenant/$rest break;
  proxy_intercept_errors on;
  error_page 404 = @canonical;
  proxy_pass http://minio:9000;
}
location @canonical {
  rewrite ^/api/assets/(?<rest>.+)$ /atlas-assets/canonical/$rest break;
  proxy_pass http://minio:9000;
}
```

Note: the per-tenant prefix lookup uses the `tenant/<tenantId>` MinIO key shape; the ingress maps incoming request paths to MinIO keys via rewrite. Final ingress rules will need polish (especially the version-string split into major/minor for atlas-renders headers), but the shape above is the target.

`/api/wz/*` routes other than character render disappear because they were extractor endpoints. The atlas-renders service does not expose anything under `/api/wz/` other than the rendered PNG path (preserved verbatim from task-043).

### 4.4 PR-env bootstrap optimization (canonical baseline + dump restore)

The dominant cost in `atlas-pr-bootstrap` today is WZ extraction + data processing — ~10 minutes per PR env. Both are deterministic functions of `.wz` content; the output is identical for every PR using the same canonical WZ. This task introduces a baseline-reuse mechanism that bypasses extraction entirely for the canonical path.

**Canonical baseline producer**: a new operator workflow (run once per game version on the cluster's primary tenant or a dedicated "canonical tenant"):

1. Upload `.wz` files via `PATCH /api/data/wz` with `?scope=canonical`.
2. Trigger `POST /api/data/process?scope=canonical`. atlas-data ingest writes:
   - `documents` rows in Postgres under `tenant_id = '00000000-0000-0000-0000-000000000000'` (the reserved canonical tenant UUID).
   - Search-index rows under the same tenant.
   - Icons, atlases, manifests, map renders to `atlas-assets/canonical/<region>/<version>/...`.
3. After successful ingest, atlas-data emits a `pg_dump`-formatted snapshot of the canonical tenant's documents + search indexes to `atlas-canonical/baseline/<region>/<version>/documents.dump` in the existing `atlas-canonical` bucket. Triggered by `POST /api/data/baseline/publish`.

**PR-env consumer**: `atlas-pr-bootstrap` replaces today's "upload → extract → process" sequence with:

1. `POST /api/data/baseline/restore` with body `{ "region": "GMS", "majorVersion": 83, "minorVersion": 1, "tenantId": "<pr-tenant-uuid>" }`. atlas-data:
   - Fetches `atlas-canonical/baseline/<region>/<version>/documents.dump` from MinIO.
   - Restores rows into the local Postgres with `tenant_id` substituted to the PR-env's tenant.
   - Restores search-index rows with the same substitution.
2. Asset reads automatically fall back to the canonical MinIO prefix via §4.2's scope lookup. No per-tenant assets are written by PR envs.
3. Domain seed steps (drops, gachapons, etc.) run as today.

**Idempotence**: `baseline/restore` short-circuits when the PR tenant already has a non-empty `documents` count for the target `(region, version)`.

**Determinism check**: `baseline/publish` also writes `documents.dump.sha256` next to the dump. PR-env consumers verify the hash before restoring; mismatch (e.g., from a corrupted dump) fails loudly.

**Custom WZ path (preserved)**: a PR that needs to test a WZ change rather than reuse the canonical baseline can opt out of the restore step and invoke the full `PATCH /api/data/wz` + `POST /api/data/process` flow. The bootstrap script accepts `BOOTSTRAP_MODE=full|baseline`; default is `baseline`. Tests that exercise extraction logic itself set `BOOTSTRAP_MODE=full`.

Expected PR-env bootstrap times:

| Mode | Today | After |
|---|---|---|
| `baseline` (default) | n/a | ~10–30 s (Postgres restore + seeds) |
| `full` | ~10 min | ~10–20 min (extraction + atlas packing; slightly slower than today due to atlas work) |

### 4.5 Ingest model in atlas-data

`atlas-data` gains an ingest pathway. The existing `POST /api/data/process` Kafka-dispatched worker pool is reused as the orchestration spine; workers source files from MinIO rather than `OUTPUT_XML_DIR`.

- **Upload**: `PATCH /api/data/wz` accepts a zip of `.wz` files for the requesting tenant (headers `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`; optional `?scope=canonical` for operator-published baselines). Streams entries into `atlas-wz/<scope>/<region>/<version>/<filename>.wz`. Validation matches the current `atlas-wz-extractor` rules: no path separators, no zip-slip, no symlinks, `.wz` extension only.
- **Trigger**: `POST /api/data/process` (existing endpoint). Lists the `atlas-wz` bucket prefix for that tenant+scope+version and dispatches per-archive Kafka commands. Optional `?scope=canonical` runs against the canonical scope and uses the canonical tenant UUID.
- **Worker**: each Kafka-dispatched worker downloads its assigned `.wz` archive to per-pod scratch (`emptyDir`), parses via `libs/atlas-wz`, and:
  1. **Data archives** (Item.wz, Mob.wz, Npc.wz, Map.wz, Skill.wz, Quest.wz, String.wz, etc.): walks the WZ tree, transforms each image into the existing typed domain model, and writes `documents` rows to Postgres (+ search index rows) in the existing transaction shape. **No `.img.xml` intermediate.**
  2. **Icon archives** (Item.wz, Npc.wz, Mob.wz, Reactor.wz, Skill.wz, UI.wz): extracts entity icons via the existing `image.ExtractIcons` logic, ported to `libs/atlas-wz`, and PUTs PNGs to `atlas-assets/<scope>/<region>/<version>/<category>/<id>/icon.png` and (for items/skills/etc.) directly to atlas-ingress-shaped key paths.
  3. **Character.wz**: packs equip/hair/face/body parts into sprite atlases and writes both the atlas PNG and JSON manifest to `atlas-assets/<scope>/<region>/<version>/atlases/<partClass>/<id>.{png,json}`. Manifest schema in §6.2.
  4. **Map.wz**: renders map composites and minimaps via the ported `mapimage` logic. PUTs to `atlas-assets/<scope>/<region>/<version>/map/<mapId>/<kind>.png` where `<kind>` is `render` or `minimap`. Same path shape the UI already constructs via `getMapImageUrl`.
  5. On completion, deletes the scratch `.wz`. Pod scratch is never expected to outlive a single archive.
- **Concurrency control**: the per-tenant mutex moves from atlas-wz-extractor to atlas-data. Held during `PATCH /api/data/wz` and `POST /api/data/process`. Persistence: Redis lock with TTL (atlas-data already depends on Redis).
- **Progress visibility**: `GET /api/data/process` returns the existing JSON:API shape enriched with per-worker status from Redis. Same shape as today, different data source.
- **Baseline publish / restore**: `POST /api/data/baseline/publish` and `POST /api/data/baseline/restore` per §4.4.

### 4.6 Character render service (`atlas-renders`)

Repackaged as a standalone Deployment. Code in `services/atlas-renders/atlas.com/renders/`.

- Request/response contract from task-043 preserved verbatim:
  - `GET /api/wz/character/render/<tenant>/<region>/<major>.<minor>/<hash>.png?skin=&hair=&face=&stance=&frame=&resize=&items=`
  - 200 with PNG body on success; standard error codes otherwise.
- Backing storage:
  - Input: fetches atlas PNG + manifest from `atlas-assets/<scope>/<region>/<version>/atlases/<partClass>/<id>.{png,json}`, scope-resolved per §4.2. Per-pod in-memory LRU caches `(atlas, manifest)` keyed by `(tenant, region, version, partClass, id)`. Default size: 256 entries. Scope resolution is cached per `(tenant, region, version, partClass)`.
  - Output: composited PNG PUT to `atlas-renders/tenant/<tenantId>/<region>/<version>/character/<hash>.png`. (Renders are intrinsically loadout-derived; cross-tenant reuse is impossible because the loadout points to tenant-scoped items. Renders are always under the per-tenant prefix, never canonical.)
- **No `libs/atlas-wz` import** in the final tree. A lint enforces this.
- Cache invariants:
  - Atlases and manifests are immutable per content. A re-extraction for the same `(scope, region, version)` writes byte-identical bytes when inputs are unchanged (per §4.1 determinism guarantee).
  - Rendered character PNGs are loadout-hash-keyed; the cache survives future re-extractions until the version itself is decommissioned. No wipe-on-extract behavior. (This supersedes task-043's wipe-on-extract behavior; that wipe is removed.)

### 4.7 atlas-ui changes

- **`src/lib/utils/asset-url.ts`**: no shape change. URLs continue to be `${VITE_ASSET_BASE_URL || '/api/assets'}/<tenant>/<region>/<version>/<category>/<id>/icon.png` and the corresponding map-image and character-render shapes. The default `/api/assets` prefix resolves through atlas-ingress to MinIO (per §4.3); production overrides via `VITE_ASSET_BASE_URL` continue to work.
- **`src/services/api/characterRender.service.ts`**: no change. URL stays `/api/assets/.../character/<hash>.png?...`; the ingress detects the `/character/` segment and routes to atlas-renders.
- **`public/sw-character-cache.js`** (existing service worker): no change. It caches by URL, and URLs are unchanged.
- **`useItemData`, `useMobData`, `useSkillData`, `useNpcData`** hooks: no change. They depend on `getAssetIconUrl` which is unchanged.
- **No env var renames**. `VITE_ASSET_BASE_URL` stays the only knob; in production the variable is empty and `/api/assets` is used.
- **No new components, no new pages.** This task is invisible to atlas-ui's source other than possibly a CHANGELOG note in its `CLAUDE.md`.

### 4.8 atlas-pr-bootstrap changes

- `scripts/bootstrap.sh` is rewritten to use `BOOTSTRAP_MODE` (`baseline` default, `full` opt-in).
- In `baseline` mode: skip `wz-upload` and `wz-extract` steps; call `POST /api/data/baseline/restore` instead of `POST /api/data/process`; same stable-poll pattern against `GET /api/data/status`.
- In `full` mode: existing flow, but pointed at the new endpoints (`PATCH /api/data/wz` instead of `PATCH /api/wz/input`; the `POST /api/data/process` step does both extraction and ingest in one call). The `wz-extract` step disappears.
- The `wait-ready` step drops `atlas-wz-extractor` from its readiness list and adds `atlas-renders`.
- The canonical-baseline producer workflow (`bootstrap-canonical.sh`) is a new sibling script. Run once per game version against the cluster's canonical tenant; emits the dump to `atlas-canonical/baseline/...`. Not invoked from PostSync; run by an operator on demand or wired into a separate Argo Application.

### 4.9 docker-compose changes

Compose currently runs three services with bind-mounted host directories (`tmp/data`, `tmp/assets`, `tmp/wz-input`). The new compose stack:

- **Adds `minio`**: `minio/minio:latest` with a named volume, ports `9000:9000` and `9001:9001` (console). Single-node single-drive matching the cluster pattern. Default creds match `~/source/k3s/bee/minio.yml` for parity (`minioadmin` / `minioadmin12345`).
- **Adds `minio-init`** as a short-lived sidecar: runs `mc alias set`, `mc mb` for `atlas-wz`, `atlas-assets`, `atlas-renders`, `atlas-canonical`, and `mc anonymous set download` on the public buckets. Healthcheck-gated so other services wait for buckets to exist.
- **Adds `atlas-renders`**: new Deployment-equivalent service, no volumes.
- **Removes `atlas-wz-extractor`** service definition.
- **Removes `atlas-assets`** service definition.
- **Updates `atlas-data`**: drops `ZIP_DIR` env and the `../../tmp/data` mount. Adds `MINIO_ENDPOINT=http://minio:9000`, `MINIO_BUCKET_WZ=atlas-wz`, `MINIO_BUCKET_ASSETS=atlas-assets`, `MINIO_BUCKET_CANONICAL=atlas-canonical`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`.
- **Updates `atlas-ingress`**: routes.conf reflects §4.3.
- **Removes `tmp/data`, `tmp/assets`, `tmp/wz-input`** directories from documentation and `.gitignore` cleanup (they're not in source, but referenced by mounts).
- A `compose/seed-canonical.sh` companion script runs `bootstrap-canonical.sh` against the local stack to populate the canonical baseline. Documented in `services/atlas-data/README.md` and the new top-level local-dev doc.

After cutover, `docker compose up` for the full local stack:

1. Starts MinIO + bucket init.
2. Starts atlas-data + atlas-renders + the rest.
3. Operator runs `seed-canonical.sh` once (or on WZ change) to populate canonical assets + documents dump.
4. Operator runs the existing SetupPage flow (or the equivalent `bootstrap.sh baseline` shape) to spin up a working tenant.

The WZ upload path (compose-only, for testing extraction itself) still works: the dev hits `PATCH /api/data/wz` against MinIO via atlas-data.

### 4.10 Retired artifacts and code paths

After cutover:

- `atlas-data-pvc`, `atlas-assets-pvc`, `atlas-wz-input-pvc` removed from all overlays.
- `services/atlas-assets/` deleted.
- `services/atlas-wz-extractor/` deleted (parser → `libs/atlas-wz`; character render → `services/atlas-renders/`; XML emitter and PVC paths deleted outright).
- `.img.xml` files: no producer, no consumer.
- `OUTPUT_XML_DIR`, `OUTPUT_IMG_DIR`, `INPUT_WZ_DIR` env vars removed wherever they appear.
- `ZIP_DIR` env var removed from atlas-data.
- `tmp/data`, `tmp/assets`, `tmp/wz-input` bind-mount directories no longer referenced by compose.
- `/api/wz/input`, `/api/wz/extractions` endpoints retired (their `/api/data/wz` replacements take over).
- atlas-pr-bootstrap's `wz-upload` and `wz-extract` step names retired (collapsed under `baseline-restore` or `data-process` depending on mode).

## 5. API Surface

### 5.1 New endpoints on atlas-data

#### `PATCH /api/data/wz`

Stages a `.wz` upload for the requesting tenant (or canonical scope). Streams multipart, validates, writes to `atlas-wz/<scope>/<region>/<version>/<filename>.wz`.

Request:
- `Content-Type: multipart/form-data`
- Part `zip_file` — flat `.wz` entries.
- Tenant headers: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
- Optional `?scope=canonical` (operator workflows only).

Validation: no path separators, no zip-slip, no symlinks, `.wz` extension only.

Responses: `202 Accepted` empty body / `400 Bad Request` / `409 Conflict` (per-tenant mutex held) / `500 Internal Server Error`.

#### `GET /api/data/wz`

JSON:API status of staged `.wz` objects. Returns `200` with `{ data: { type: "wzInputStatus", attributes: { fileCount, totalBytes, updatedAt } } }`. Same shape as the existing `GET /api/wz/input` for migration ergonomics.

#### `POST /api/data/baseline/publish`

Operator-only. After a successful canonical ingest, snapshots the canonical tenant's `documents` + search-index rows to `atlas-canonical/baseline/<region>/<version>/documents.dump`. Also writes `documents.dump.sha256` next to it.

Request body:
```json
{ "region": "GMS", "majorVersion": 83, "minorVersion": 1 }
```

Responses: `202 Accepted` (async) / `409 Conflict` (publish in progress) / `412 Precondition Failed` (no canonical ingest for that version).

#### `POST /api/data/baseline/restore`

Used by PR-env bootstrap. Restores the canonical baseline into the requesting tenant for the given version.

Request body:
```json
{ "region": "GMS", "majorVersion": 83, "minorVersion": 1, "tenantId": "<pr-tenant>" }
```

Behavior:
- Idempotent: if the PR tenant already has documents for that version, returns `204 No Content` with `X-Atlas-Baseline-Status: already-restored`.
- Else fetches the dump, verifies the SHA-256, restores into Postgres with `tenant_id` substituted to the request's tenantId.

Responses: `202 Accepted` (running) / `204 No Content` (already restored) / `404 Not Found` (no baseline published) / `409 Conflict` / `422 Unprocessable Entity` (hash mismatch).

### 5.2 Modified endpoints

#### `POST /api/data/process`

Existing endpoint. Workers now read from MinIO (`atlas-wz` bucket) rather than `ZIP_DIR`. Accepts optional `?scope=canonical` for operator-published baselines. Response contract unchanged.

#### `GET /api/data/process`

Existing endpoint, unchanged shape; data source moves to Redis.

#### `GET /api/data/status`

Existing endpoint. Continues to report `documentCount` and `updatedAt`. PR-env bootstrap polls this for stability after a baseline restore.

### 5.3 Retired endpoints

The following disappear with `atlas-wz-extractor`:

- `PATCH /api/wz/input`
- `GET /api/wz/input`
- `POST /api/wz/extractions`
- `GET /api/wz/extractions`

Character render preserves its endpoint shape but moves to atlas-renders.

### 5.4 atlas-renders surface

One handler: `/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png`. No new endpoints.

## 6. Data Model

### 6.1 Postgres

`documents` table unchanged. Five trigram search-index tables unchanged. Ingest writes the same rows by the same DDL.

**Canonical tenant**: the reserved UUID `00000000-0000-0000-0000-000000000000` is the operator-controlled canonical tenant. atlas-data treats it as read-mostly: operator workflows can write via `?scope=canonical`; runtime tenants never write to it. The reserved UUID is already used in compose (`atlas-drops`, `atlas-drop-information` use it as a SERVICE_ID); reusing it for the canonical tenant is intentional — it's the conventional "zero" value.

**Baseline restore semantics**: `INSERT INTO documents (...) SELECT ..., '<pr-tenant>' AS tenant_id, ... FROM canonical_documents` — driven by a server-side `documents.dump` (Postgres custom format). Restore uses `pg_restore` semantics: drops + recreates the tenant's existing rows for the affected version before insert, to keep restore deterministic.

### 6.2 Sprite atlas manifest schema

`atlases/<partClass>/<id>.json`:

```json
{
  "version": 1,
  "id": 1040002,
  "partClass": "coat",
  "sheet": { "width": 256, "height": 256 },
  "sprites": [
    {
      "stance": "stand1",
      "frame": 0,
      "part": "arm",
      "rect": { "x": 0, "y": 0, "w": 32, "h": 48 },
      "origin": { "x": 16, "y": 32 },
      "anchors": {
        "neck": { "x": 16, "y": 8 },
        "navel": { "x": 16, "y": 32 }
      },
      "z": 1
    }
  ]
}
```

- `partClass` ∈ `{coat, longcoat, pants, shoes, glove, cape, shield, cap, mask, eye-accessory, face-accessory, earrings, weapon, hair, face, body}`. Matches MapleStory equip categories already in atlas-data.
- `origin`, `anchors`, `z` carried over from the existing render logic in `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/`.
- `version` is the manifest schema version. v1 covers everything task-043's renderer uses today. Forward-compatible by additive-only fields. Renderer ignores unknown fields. Breaking changes require a v2 and a re-extraction.

### 6.3 MinIO object key conventions

```
atlas-wz/<scope>/<region>/<major>.<minor>/<filename>.wz

atlas-assets/<scope>/<region>/<major>.<minor>/<category>/<id>/icon.png
atlas-assets/<scope>/<region>/<major>.<minor>/atlases/<partClass>/<id>.png
atlas-assets/<scope>/<region>/<major>.<minor>/atlases/<partClass>/<id>.json
atlas-assets/<scope>/<region>/<major>.<minor>/map/<mapId>/render.png
atlas-assets/<scope>/<region>/<major>.<minor>/map/<mapId>/minimap.png

atlas-renders/tenant/<tenantId>/<region>/<major>.<minor>/character/<hash>.png

atlas-canonical/baseline/<region>/<major>.<minor>/documents.dump
atlas-canonical/baseline/<region>/<major>.<minor>/documents.dump.sha256
```

`<scope>` = `canonical` or `tenant/<tenantId>`.

`<category>` for icons: `npc`, `mob`, `reactor`, `item`, `skill`, `world-icon`.

### 6.4 Ingest progress (Redis)

Per-scope ingest state under key `atlas-data:ingest:<scope>:<region>:<major>.<minor>`:

- Lock (string with TTL).
- Worker status (hash, fields = worker types, values = `pending|running|done|error:<msg>`).
- `startedAt`, `updatedAt` (Unix ms).

TTL refreshed by the running worker; lock freed on completion or failure.

## 7. Service Impact

### 7.1 `atlas-data`

- Adds dependency on `libs/atlas-wz` and the MinIO Go SDK.
- New env: `MINIO_ENDPOINT`, `MINIO_BUCKET_WZ`, `MINIO_BUCKET_ASSETS`, `MINIO_BUCKET_CANONICAL`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`.
- Removes `ZIP_DIR`.
- New endpoints: `PATCH /api/data/wz`, `GET /api/data/wz`, `POST /api/data/baseline/publish`, `POST /api/data/baseline/restore`.
- Workers refactored to source files from MinIO; per-domain processors now consume WZ data via `libs/atlas-wz` instead of `.img.xml`.
- Adds atlas packing for Character.wz and map rendering for Map.wz to the ingest worker set.
- Drops `/usr/data` mount from Dockerfile.

### 7.2 New service: `atlas-renders`

- New service `services/atlas-renders/atlas.com/renders/`. Module: `atlas-renders` (short).
- Character render handler + supporting compositor.
- MinIO Go SDK for atlas + manifest reads, render writes.
- In-memory LRU for `(atlas, manifest)` pairs and scope resolution.
- No PVC. No `libs/atlas-wz` import (lint-enforced).
- Env: `MINIO_ENDPOINT`, `MINIO_BUCKET_ASSETS`, `MINIO_BUCKET_RENDERS`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `REST_PORT`, `LOG_LEVEL`, `JAEGER_HOST_PORT`, `ATLAS_LRU_SIZE`.

### 7.3 Retired: `atlas-wz-extractor`

Deleted in full. Parser → `libs/atlas-wz`. Render handler → `atlas-renders`. XML emitter and map render → ported to atlas-data ingest workers, then the original sources deleted.

### 7.4 Retired: `atlas-assets`

Deleted in full. Replaced by atlas-ingress routing to MinIO + atlas-renders.

### 7.5 New library: `libs/atlas-wz`

- New Go module. Stdlib + minimal image deps.
- Package layout: `wz/`, `crypto/`, `canvas/`, `atlas/`, `mapimage/` (the map composite logic), `icons/` (the icon-extraction dispatcher).
- README + public API documented.
- atlas-data's Dockerfile updated in all four required locations per CLAUDE.md (go.mod COPY, go.work synthesis, source COPY, `go mod edit -replace`). `docker build` from worktree root is the only check that catches drift.

### 7.6 `atlas-pr-bootstrap`

- `scripts/bootstrap.sh` rewritten per §4.8.
- New `scripts/bootstrap-canonical.sh` for operator workflows.
- Dockerfile may need updates if WZ canonical zip baking changes (out of scope unless required).

### 7.7 `atlas-ui`

- No source changes expected. The URL builder uses `/api/assets/...`; the ingress routes change shape under it but the UI doesn't know.
- One possible exception: a CHANGELOG / `CLAUDE.md` note about the route table change.

### 7.8 atlas-ingress

- `deploy/shared/routes.conf` updated per §4.3.
- Adds MinIO upstream definition.

### 7.9 docker-compose

Per §4.9.

### 7.10 Deploy / k8s

- New `atlas-renders.yaml` Deployment + Service.
- `atlas-data.yaml`: drop `/usr/data` mount + PVC; add MinIO env from secret.
- Remove `atlas-wz-extractor.yaml`, `atlas-assets.yaml`.
- Drop PVC defs for `atlas-data-pvc`, `atlas-assets-pvc`, `atlas-wz-input-pvc`.
- atlas-ingress manifest updated to reflect new routes.conf.
- Note: cluster-level MinIO ingress (allowing direct browser access to MinIO bypassing atlas-ingress) is **not** added in this task; all asset traffic flows through atlas-ingress. Direct-from-browser-to-MinIO is a future optimization deferred behind a CDN decision.

## 8. Non-Functional Requirements

### 8.1 Performance

- **Character render latency**: end-to-end p50 ≤ current p50 + 50 ms (cold compositor cache); p99 ≤ current p99 + 150 ms. Dominant new cost is one-time atlas fetch per part class per `(tenant, version)` per pod; once cached, render is GPU/CPU-bound just like today.
- **PR-env bootstrap (`baseline` mode)**: ready in ≤ 60 s from `argocd app sync`, vs. ~10 min today. Documents restored, assets fall back to canonical, no extraction.
- **PR-env bootstrap (`full` mode)**: not worse than 1.5× today's extraction time (atlas packing is added work).
- **Ingest throughput**: full extraction for one tenant/version completes in ≤ 2× the current extractor wall-clock; MinIO PUT throughput is the new bottleneck and is parallelized at the worker level.
- **Asset GET latency**: served via atlas-ingress → MinIO. Same pattern as `atlas-canonical/atlas.zip` today. ≤ 20 ms p50 within the cluster.
- **Renderer working set**: per pod ≤ ~256 MB for the atlas LRU at default size.

### 8.2 Security

- `atlas-wz` bucket private. atlas-data IAM identity: PUT + GET + DELETE on `atlas-wz/*`; PUT on `atlas-assets/*`; PUT on `atlas-canonical/*`. atlas-renders: GET on `atlas-assets/*`; PUT on `atlas-renders/*`.
- `atlas-assets` and `atlas-renders` are anonymous-read only; reads happen through atlas-ingress, which does not pass through any authentication. The MinIO console is not externally exposed.
- IAM creds in a Kubernetes Secret. Default `minio.yml` creds suitable only for local dev.
- No PII or character data in MinIO. Only WZ-derived game data and images.
- atlas-pr-bootstrap's `baseline/restore` endpoint requires the requester to assert a tenantId; the server validates the tenant exists in atlas-tenants before writing rows. Cross-tenant data leakage is impossible by construction (tenant_id is the row key, restored values are stamped server-side).

### 8.3 Observability

- atlas-data: per-worker spans + per-MinIO-op spans + new `atlas_data_baseline_*` metrics.
- atlas-renders: per-render spans with `cacheHit`, `scope` attributes; metrics `atlas_renders_requests_total{result}`, `atlas_renders_latency_seconds`, `atlas_renders_cache_hits_total{kind=atlas|scope}`, `atlas_renders_minio_{put,get}_seconds`.
- atlas-pr-bootstrap: `ATLAS_STEP=baseline-restore` in logs replaces `wz-extract` / `data-process` for the default path.
- A new Grafana panel on the existing `atlas-pr-environments` dashboard (per task-063) reports baseline-restore time-to-ready.

### 8.4 Multi-tenancy

- All MinIO keys are prefixed by `<scope>` where `<scope>` ∈ {`canonical`, `tenant/<tenantId>`}. Per-tenant prefix is reserved but not exercised today.
- Postgres rows carry `tenant_id` and are filtered via `tenant.MustFromContext(ctx)` in every read/write path.
- atlas-renders parses tenant headers via the existing middleware; refuses requests missing them.
- A tenant deletion in atlas-tenants triggers (out of scope here; tracked) MinIO prefix deletions and Postgres row purges across all three buckets / index tables.

### 8.5 Tests

- `libs/atlas-wz`: unit tests for every parser branch and every canvas pixel format.
- atlas-data ingest: testcontainers MinIO + Postgres integration test exercising upload → process → documents+atlases for a small fixture WZ set.
- atlas-data baseline: integration test for `publish` then `restore` into a fresh PR-tenant fixture; verifies row counts, search-index rebuild, and hash-mismatch failure mode.
- atlas-renders: testcontainers MinIO with fixture atlases + manifests; verifies hash-keyed cache, scope-fallback, byte-identical render output vs. a frozen task-043 baseline.
- Renderer "no WZ parser at runtime" invariant: CI greps the atlas-renders import graph for `libs/atlas-wz`.
- atlas-pr-bootstrap: a compose-level smoke test exercises `BOOTSTRAP_MODE=baseline` end-to-end against the seeded canonical baseline.

## 9. Open Questions

- **MinIO single-drive durability**: the cluster MinIO is single-node single-drive on Longhorn. With this task, MinIO becomes the canonical store for game-data artifacts. Should the design phase add an erasure-coded or replicated MinIO topology? Recovery story today is "re-extract from raw `.wz`", which still works post-cutover.
- **Documents-dump format**: `pg_dump --format=custom` (binary) vs. `--format=plain` (SQL with `\copy` blocks). Custom is smaller and faster to restore, plain is human-inspectable. Pick during design; default to custom.
- **Baseline-restore atomicity**: restore is multi-row, multi-table, potentially long-running. Wrap in a single transaction or accept partial progress with idempotence? Decision affects API contract (`POST /api/data/baseline/restore` async vs. sync).
- **Map.wz render coverage**: today `mapimage` produces both `render.png` (full composite) and `minimap.png`. Verify the v83 dataset's render success rate before committing to "every map renders successfully during ingest"; gaps need a fallback story (default placeholder? omit from MinIO and 404 from ingress?).
- **Sprite atlas packing algorithm**: a stable rectangle-packing algorithm is required for the determinism guarantee in §4.1. MaxRects with a fixed sort order is the leading candidate; design phase enumerates the alternatives and pins one.
- **Atlas size budget**: rough estimate is 20–30k equip atlases per version at a few KB each, plus ~5k icons, plus map renders (potentially MB each). Validate against an actual v83 Character.wz + Map.wz before committing the design.
- **Manifest schema versioning policy**: documented as additive-only in §6.2. Confirm in design.
- **Service-worker cache invalidation**: `public/sw-character-cache.js` caches renders by URL. If a render's bytes change (e.g., from an atlas re-pack), the URL stays the same and the SW serves stale bytes. Design must specify the cache-busting strategy (version in URL? content hash in URL? SW cache versioning?).
- **`docker-compose` MinIO healthcheck timing**: `minio-init` must run after MinIO is ready but before atlas-data starts. Design must specify the healthcheck or `depends_on: condition: service_healthy` chain.
- **PR-env failure modes**: what happens if `baseline/restore` is invoked but the canonical baseline doesn't exist yet? Block bootstrap with a clear error? Fall back to `full` mode automatically? Design pick.
- **`tools/task-numbers.sh` `next` exits 1 with `set -e`**, and the scan misses remote-tracking branches (caught the collision with task-032-dynamic-service-config in this task's creation). Out of scope here; tracked for a separate task.

## 10. Acceptance Criteria

- [ ] `libs/atlas-wz` module created with parser + crypto + canvas decoder + atlas packer + map render + icon extractor. Unit tests cover every property type and every pixel format. Determinism guarantee tested via a "pack twice, compare bytes" assertion. Public API documented.
- [ ] `atlas-data` ingests `.wz` from MinIO end-to-end: upload via `PATCH /api/data/wz`, trigger via `POST /api/data/process`, observe documents in Postgres and icons/atlases/maps in MinIO. Integration test green.
- [ ] `services/atlas-renders/` deployed; serves `GET /api/wz/character/render/...` against MinIO-only inputs. Final source tree has no import of `libs/atlas-wz`. Lint check enforces it.
- [ ] `services/atlas-wz-extractor/` deleted.
- [ ] `services/atlas-assets/` deleted.
- [ ] `atlas-data-pvc`, `atlas-assets-pvc`, `atlas-wz-input-pvc` removed from all `deploy/k8s/` overlays.
- [ ] atlas-ingress (`deploy/shared/routes.conf`) updated: `/api/assets/...` routes to MinIO with per-tenant→canonical fallback; `/api/assets/.../character/...` routes to atlas-renders. `/api/wz/input`, `/api/wz/extractions` removed.
- [ ] atlas-ui has no source changes beyond optional CHANGELOG / `CLAUDE.md` note. URL shapes preserved.
- [ ] `docker-compose` updated: adds MinIO + bucket init, removes extractor + assets, adds atlas-renders. `docker compose up` (plus a one-time `seed-canonical.sh`) produces a working local stack.
- [ ] `atlas-pr-bootstrap`'s `bootstrap.sh` updated for `BOOTSTRAP_MODE=baseline` (default) and `BOOTSTRAP_MODE=full`. New `bootstrap-canonical.sh` for operator publish. `baseline` mode end-to-end ≤ 60 s against a pre-published canonical baseline (verified in a compose smoke test).
- [ ] `POST /api/data/baseline/publish` produces a deterministic dump (re-run yields identical SHA-256 for unchanged inputs).
- [ ] `POST /api/data/baseline/restore` is idempotent (second call returns `204 No Content`).
- [ ] For a single canonical loadout (documented in tests), `atlas-renders` produces a PNG byte-identical to (or visually identical within a documented diff tolerance) the same loadout rendered by the pre-cutover task-043 service.
- [ ] `docker build -f services/atlas-data/Dockerfile .` and `docker build -f services/atlas-renders/Dockerfile .` succeed from the worktree root. atlas-data's Dockerfile correctly lists `libs/atlas-wz` in all four required locations (CLAUDE.md mandate).
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] Documentation updated: `services/atlas-data/README.md` describes the new ingest flow + canonical baseline; the obsolete `services/atlas-wz-extractor/README.md` and `services/atlas-assets/Dockerfile` are gone; a new `docs/runbooks/wz-ingest.md` covers operator workflows (raw upload, full ingest, canonical publish, baseline restore). `docs/runbooks/ephemeral-pr-deployments.md` updated for the new bootstrap modes.
- [ ] CHANGELOG / commit history reflects the cutover. The single cutover PR is reviewable as one unit.

---

## Appendix A — Affected services summary

| Service | Change |
|---|---|
| `atlas-data` | Gains WZ parser via `libs/atlas-wz`. New ingest endpoints, new baseline publish/restore endpoints. Workers read from MinIO. PVC dropped. |
| `atlas-wz-extractor` | Deleted. Parser → `libs/atlas-wz`. Render → `atlas-renders`. XML, map, PVC paths deleted outright. |
| `atlas-assets` | Deleted. atlas-ingress routes `/api/assets/...` to MinIO with per-tenant→canonical fallback. |
| `atlas-renders` (new) | Composites character renders from MinIO atlases. No WZ parser. |
| `atlas-ingress` | routes.conf updated for MinIO upstream and character-render path detection. |
| `atlas-ui` | No source changes. URL shapes preserved. |
| `atlas-pr-bootstrap` | `bootstrap.sh` rewritten for `BOOTSTRAP_MODE`. New `bootstrap-canonical.sh`. |
| `libs/atlas-wz` (new) | Shared parser + canvas + atlas packer + map render. |
| `deploy/k8s` | Three PVCs deleted; two Deployments deleted; one added (`atlas-renders`). |
| `deploy/compose` | MinIO + init added; extractor + assets removed; atlas-renders added; volume mounts removed from atlas-data. |

## Appendix B — Performance and capacity sketch (for design phase to validate)

- **MinIO object count**: ~20–30k equip atlases × 2 files (PNG + JSON) + ~5k icons + ~1k map images = ~60k objects per canonical version. Per-tenant overrides currently zero.
- **MinIO storage**: PNG-packed atlases at a few KB each ≈ 100–200 MB per canonical version. Map renders potentially 1–2 GB. Total: ~2 GB per canonical version. Single-drive 20Gi PVC fits ~10 versions comfortably.
- **Postgres documents.dump**: estimated 50–150 MB per canonical version (text + JSON). Stored in `atlas-canonical` bucket alongside today's `atlas.zip`.
- **PR-env bootstrap baseline-mode wire time**: ~50 MB dump fetch + ~10 s restore + ~5 s seed = ≤ 60 s end-to-end. Today: 10+ min.
- **Atlas LRU per atlas-renders pod**: 256 entries × ~200 KB avg = ~50 MB working set. Bounded.
