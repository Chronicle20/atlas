# Game Data + Asset Pipeline Consolidation onto MinIO — Product Requirements Document

Version: v3
Status: Draft
Created: 2026-05-19
Updated: 2026-05-19 — v3 applies adversarial-review fixes: (a) one-version-per-tenant constraint made explicit and idempotence rewritten; (b) baseline dump format pinned to per-table `COPY (SELECT ...) TO STDOUT (FORMAT binary)` with header metadata; (c) map rendering moves from eager (ingest) to lazy (atlas-renders), halving ingest wall-clock; (d) explicit `scope=tenant|shared` toggle replaces "canonical-UUID writes implicitly to shared/"; (e) PR-env tenant cleanup pulled into scope; (f) cutover sequencing pinned to "publish baseline from ephemeral test env before merge"; (g) determinism guarantee scoped to a pinned PNG encoder; (h) tenant/<id> scope retained per user direction with the operational cost acknowledged; (i) `client_max_body_size` carried to new ingress route; (j) rolling-deploy strategy = drain old atlas-data pods (downtime acceptable). v2 changes carried forward: atlas-canonical bucket vs shared scope naming; `MODE=rest|ingest|all` switch; SetupPage UI diff.

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
- Add per-tenant cleanup hooks invoked by `atlas-pr-bootstrap`'s `cleanup.sh` (and by any future tenant-deletion flow) to purge `documents` + 5 search-index rows + the per-tenant MinIO prefixes when a tenant is decommissioned. Without this, MinIO and Postgres accumulate detritus from every closed PR env.
- Cut over via an ephemeral PR environment: the cutover PR is itself opened, its PR env is brought up end-to-end (including a one-time canonical baseline publish from that env), full smoke tests run, and only then is the PR merged. There is no in-place revert plan; if something is wrong, the PR doesn't merge.

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
- Determinism guarantee: `atlas.Pack` produces byte-identical output for identical inputs. This requires:
  - A pinned PNG encoder (the library ships its own encoder, not `image/png` from stdlib, to avoid Go-version drift).
  - A fixed rectangle-pack algorithm (MaxRects with a stable sort by `(width desc, height desc, name asc)` — to be locked in design).
  - Sorted child ordering at every level of the layout.
  - A fixed image-resampling filter for any scaling.
  - No reliance on map iteration order, time-based seeds, or `runtime.NumCPU()`-dependent parallelism in the packer.
  This is load-bearing for §4.5's canonical-baseline reuse and §4.7's "no wipe on re-extract" cache policy. A "pack twice, compare bytes" test runs in CI on every change to the library.

### 4.2 MinIO bucket layout, access policy, and shared/tenant scope split

Four buckets in the existing `minio` namespace, all served by the existing `minio` Service (cluster DNS `minio.minio.svc.cluster.local:9000`).

| Bucket | Access | Purpose |
|---|---|---|
| `atlas-canonical` | anonymous-read | **Operator-published, immutable-per-version artifacts.** Holds bundled `.wz` archives and Postgres baseline dumps. Pre-exists today (currently `atlas.zip` at the root); this task reorganizes its contents by `(region, version)` and adds the baseline dump key. |
| `atlas-wz` | private | **Non-canonical, per-tenant raw `.wz` uploads** (e.g. a tenant testing a custom Item.wz). Key: `<tenantId>/<region>/<major>.<minor>/<filename>.wz`. |
| `atlas-assets` | anonymous-read | **Extraction outputs** (icons, atlases, manifests, map renders). Key prefix scoped: `shared/...` or `tenant/<tenantId>/...`. |
| `atlas-renders` | anonymous-read | **Runtime-composited character paperdolls.** Always per-tenant (a paperdoll references tenant-scoped equip IDs, so cross-tenant reuse is impossible by construction). Key: `<tenantId>/<region>/<version>/character/<hash>.png`. |

**Terminology — keep these distinct in your head:**

- `atlas-canonical` is the **bucket name**. It holds operator-published immutable inputs and outputs (raw `.wz` bundles and baseline Postgres dumps).
- `shared` is the **scope prefix** inside `atlas-assets`. Artifacts under `shared/` are produced by ingesting the canonical-tenant's WZ archives and are valid for every tenant pinned to the same `(region, version)`.
- `tenant/<tenantId>` is the **per-tenant scope prefix** inside `atlas-assets`. Reserved for future per-tenant overrides; not exercised by this task.

v1 of this PRD overloaded "canonical" for both the bucket and the scope, which is confusing because `atlas-canonical` is a literal bucket already in your cluster. v2 separates them: bucket = `atlas-canonical`, scope = `shared` vs `tenant/<id>`.

**Asset lookup order**: atlas-renders, when building the path to fetch an atlas for a given loadout layer, first probes `atlas-assets/tenant/<tenantId>/<region>/<version>/atlases/<partClass>/<id>.png` and falls back to `atlas-assets/shared/<region>/<version>/atlases/<partClass>/<id>.png` on 404. Probes use HEAD; an in-pod LRU records the resolved scope per `(tenant, region, version, partClass)` so the fallback decision is made at most once per cold pod per part class.

For atlas-ingress-served icons (URLs built directly by the UI), the fallback is resolved at the ingress layer. See §4.3.

**`atlas-canonical` bucket layout** (this task):

```
atlas-canonical/
  wz/<region>/<major>.<minor>/atlas.zip                          # bundled .wz, versioned
  baseline/<region>/<major>.<minor>/documents.dump               # Postgres dump from canonical-tenant ingest
  baseline/<region>/<major>.<minor>/documents.dump.sha256
```

The pre-existing top-level `atlas-canonical/atlas.zip` is migrated to `atlas-canonical/wz/<region>/<version>/atlas.zip` during cutover; `deploy/k8s/overlays/pr/sync-bootstrap.yaml`'s init container URL is updated in the same PR.

### 4.3 atlas-ingress routing changes

atlas-ingress (`deploy/shared/routes.conf`) is the single boundary the UI talks to. The two routes that change:

- `/api/assets/(.*)` → atlas-ingress nginx serves from MinIO directly. For each request, it first probes `atlas-assets/tenant/<tenantId>/<region>/<version>/<rest>` and falls back to `atlas-assets/shared/<region>/<version>/<rest>`. Implementation: nginx `try_files`-style cascade via two `proxy_pass` locations and an `error_page 404 = @canonical` fallback.
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
# per-tenant→shared fallback.
location ~ ^/api/assets/(?<rest>.+)$ {
  rewrite ^ /atlas-assets/tenant/$rest break;
  proxy_intercept_errors on;
  error_page 404 = @shared;
  proxy_pass http://minio:9000;
}
location @shared {
  rewrite ^/api/assets/(?<rest>.+)$ /atlas-assets/shared/$rest break;
  proxy_pass http://minio:9000;
}
```

Notes for the ingress configuration:

- The per-tenant prefix lookup uses the `tenant/<tenantId>` MinIO key shape; the ingress maps incoming request paths to MinIO keys via rewrite. Final ingress rules will need polish: the version capture must split into `(?<major>[0-9]+)\.(?<minor>[0-9]+)` to forward as separate `MAJOR_VERSION` / `MINOR_VERSION` headers to atlas-renders, and the hash capture should be width-constrained (`[a-f0-9]{N}` where N matches task-043's hash output) to avoid future path collisions.
- **Body size config** must carry over for the new upload route: today's `client_max_body_size 4G` and `proxy_request_buffering off` block on `/api/wz/*` (for large `.wz` uploads) move to the new `/api/data/wz` location. Without these, large uploads silently fail at the ingress with `413 Request Entity Too Large`.
- `proxy_pass http://atlas-renders:8080/api/wz/character/render/...` with a URI path means nginx does NOT auto-forward the original query string; the explicit `$is_args$args` suffix (already in today's `/api/assets/*` block) must carry across.
- **Cache-Control headers**: today atlas-assets sets `Cache-Control: public, max-age=86400` on icons and `public, max-age=86400, immutable` on character renders. The new ingress block must inject equivalent headers on responses from MinIO (which by default sets no cache headers on anonymous-read GETs). Without this, browser + service-worker caches lose their TTLs and p99 page-load latency regresses silently.

`/api/wz/*` routes other than character render disappear because they were extractor endpoints. The atlas-renders service does not expose anything under `/api/wz/` other than the rendered PNG path (preserved verbatim from task-043).

### 4.4 Ingest topology: REST / Job / compose-collapsed

The atlas-data binary has three runtime modes selected by `MODE` env:

| `MODE` | Process responsibilities | Used by |
|---|---|---|
| `rest` | HTTP API only. No Kafka consumers. `POST /api/data/process` creates a Kubernetes Job from a baked template. | k8s production REST Deployment |
| `ingest` | Kafka consumers + workers + MinIO writers + Postgres writers. No HTTP. Runs to completion, pod exits. | k8s Job created on demand by REST |
| `all` | HTTP API + Kafka consumers + workers in one process. `POST /api/data/process` publishes a command that the same process consumes inline. | docker-compose; local dev |

The same compiled binary serves all three modes. Mode selection is the only difference at runtime; the Go code paths and Kafka topics are identical across modes.

**Kubernetes shape:**

- `atlas-data` Deployment runs `MODE=rest` with REST-shaped resource limits (small CPU + memory).
- A `Role`/`RoleBinding` grants the atlas-data ServiceAccount `create`, `get`, `list`, `watch`, `delete` on `batch.Job` in its own namespace only.
- A `Job` template (PodTemplate, env, resource limits — CPU `2-8`, memory `1-3Gi` matching today's atlas-wz-extractor profile) is baked into the atlas-data ConfigMap or chart. The REST handler instantiates Jobs from the template, parameterized with the current `(tenantId, scope, region, version)`.
- The Job pod runs `MODE=ingest`, parses the Kafka command(s) for its scope/version, runs every worker (data archives, icons, atlases, map renders), and exits 0 on success.
- atlas-data REST polls the Job status (or watches via the API) for the `GET /api/data/process` status response.

**Docker-compose shape:**

- atlas-data runs `MODE=all` with one process. No Job, no k8s API dependency. Ingest workers consume Kafka commands published by the REST handler in the same process.
- Resource limits in compose are not finely tuned; a developer running local ingest accepts that the same process serves REST and parser concurrently.

**Behavioral parity:**

- The REST contract is identical across all three modes. A client cannot tell whether `POST /api/data/process` was satisfied by an in-process worker or a separately-scheduled Job.
- The Kafka command topic (`COMMAND_TOPIC_DATA`) is the same. In `MODE=all` the publisher and consumer are the same process; in `MODE=rest` + `MODE=ingest` they are different pods.
- The Postgres + MinIO writes are the same; the worker code path is the same.

**Map.wz rendering happens in the ingest workers**, which means in production it runs **inside the Job pod**, not the REST pod. In compose it runs inside the single `MODE=all` process. v1 of this PRD said "atlas-data ingest workers" without distinguishing the deploy topology, which left ambiguous whether REST or Job ran the heavy work. v2 makes the answer explicit: **Job (k8s) / in-process (compose); never the REST pod.**

### 4.4a Per-tenant cleanup on PR-env teardown

Pulled into this task's scope. Without it, every closed PR env leaves behind `documents` rows, search-index rows, an `atlas-renders/<tenantId>/...` prefix, and (for any future override scenario) an `atlas-assets/tenant/<tenantId>/...` prefix. On a busy weekly PR cadence the residue would dominate MinIO storage and slow trigram search-index scans within months.

- `atlas-pr-bootstrap`'s `cleanup.sh` gains a new step **`tenant-purge`** that calls a new atlas-data endpoint `DELETE /api/data/tenants/<tenantId>` (operator-gated).
- The atlas-data handler performs, in this order:
  1. `DELETE FROM documents WHERE tenant_id = <id>`.
  2. `DELETE FROM <search-index> WHERE tenant_id = <id>` for each of the 5 search-index tables.
  3. `mc rm --recursive` (Go SDK equivalent) on:
     - `atlas-wz/tenant/<tenantId>/`
     - `atlas-assets/tenant/<tenantId>/`
     - `atlas-renders/<tenantId>/`
  4. Logs the purge with `tenantId`, row counts, and bytes freed.
- Idempotent: re-invoking on an already-purged tenant returns `204 No Content`.
- Refuses to purge the canonical UUID (`00000000-…`); returns `403 Forbidden`.
- Atomic-ish: the Postgres deletes are one transaction; the MinIO deletes are best-effort with retry. A partial MinIO failure logs the residual keys; a follow-up cron sweeps orphans (out of scope; tracked).

### 4.5 PR-env bootstrap optimization (canonical baseline + dump restore)

The dominant cost in `atlas-pr-bootstrap` today is WZ extraction + data processing — ~10 minutes per PR env. Both are deterministic functions of `.wz` content; the output is identical for every PR using the same canonical WZ. This task introduces a baseline-reuse mechanism that bypasses extraction entirely for the canonical path.

**Canonical baseline producer**: an operator workflow, surfaced in atlas-ui SetupPage via the scope toggle (§4.8):

1. Operator opens SetupPage in any tenant context and selects the **Canonical (shared)** scope toggle.
2. Upload `.wz` files via `PATCH /api/data/wz?scope=shared`. Operator-permission header (`X-Atlas-Operator: 1`) required.
3. Trigger `POST /api/data/process?scope=shared`. atlas-data ingest writes:
   - `documents` rows in Postgres under `tenant_id = '00000000-0000-0000-0000-000000000000'` (the reserved canonical tenant UUID).
   - Search-index rows under the same tenant.
   - Icons, sprite atlases, manifests, and map layer inputs to `atlas-assets/shared/<region>/<version>/...`.
4. After successful ingest, click the **Publish Baseline** CTA (or call `POST /api/data/baseline/publish` directly). atlas-data emits the per-table COPY-binary dump (per §6.1) wrapped in a `.tar` with `header.json` to `atlas-canonical/baseline/<region>/<version>/documents.dump` and writes the SHA-256 sidecar.

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

### 4.6 Ingest model in atlas-data

`atlas-data` gains an ingest pathway. The existing `POST /api/data/process` Kafka-dispatched worker pool is reused as the orchestration spine; workers source files from MinIO rather than `OUTPUT_XML_DIR`.

- **Upload**: `PATCH /api/data/wz` accepts a zip of `.wz` files. Headers `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` identify the requesting tenant. Optional query param `?scope=tenant|shared` (default `tenant`) selects the output scope. When `scope=shared`, the request requires an operator assertion (header `X-Atlas-Operator: 1` for v1; future auth gates it). Streams entries into `atlas-wz/<scope-key>/<region>/<version>/<filename>.wz` where `<scope-key>` is `shared` for `scope=shared` and `tenant/<tenantId>` for `scope=tenant`. Validation matches the current `atlas-wz-extractor` rules.
- **Trigger**: `POST /api/data/process` (existing endpoint). Same `?scope=tenant|shared` semantics. Lists the `atlas-wz` bucket prefix for the resolved scope + version and dispatches per-archive Kafka commands. Ingest workers write outputs to `atlas-assets/<scope-key>/<region>/<version>/...`. Postgres rows for `scope=shared` use the canonical tenant UUID; for `scope=tenant` they use the requesting tenant_id. The server-side check requires `scope=shared` callers to assert operator status and rejects (`403 Forbidden`) otherwise.
- **Worker**: each Kafka-dispatched worker downloads its assigned `.wz` archive to per-pod scratch (`emptyDir`), parses via `libs/atlas-wz`, and:
  1. **Data archives** (Item.wz, Mob.wz, Npc.wz, Map.wz, Skill.wz, Quest.wz, String.wz, etc.): walks the WZ tree, transforms each image into the existing typed domain model, and writes `documents` rows to Postgres (+ search index rows) in the existing transaction shape. **No `.img.xml` intermediate.**
  2. **Icon archives** (Item.wz, Npc.wz, Mob.wz, Reactor.wz, Skill.wz, UI.wz): extracts entity icons via the existing `image.ExtractIcons` logic, ported to `libs/atlas-wz`, and PUTs PNGs to `atlas-assets/<scope>/<region>/<version>/<category>/<id>/icon.png` and (for items/skills/etc.) directly to atlas-ingress-shaped key paths.
  3. **Character.wz**: packs equip/hair/face/body parts into sprite atlases and writes both the atlas PNG and JSON manifest to `atlas-assets/<scope>/<region>/<version>/atlases/<partClass>/<id>.{png,json}`. Manifest schema in §6.2.
  4. **Map.wz**: extracts the **inputs** to map rendering but does not composite them. Layer PNGs (back/foreground/tiles), foothold/portal/NPC layout JSON, and minimap PNG are written to `atlas-assets/<scope>/<region>/<version>/map/<mapId>/{layers/*.png, layout.json, minimap.png}`. Full map composites are produced on demand by atlas-renders (see §4.7) — that halves the ingest wall-clock and avoids materializing maps no one views. The minimap is materialized eagerly because it's small and almost universally viewed.
  5. On completion, deletes the scratch `.wz`. Pod scratch is never expected to outlive a single archive.
- **Concurrency control**: the per-tenant mutex moves from atlas-wz-extractor to atlas-data. Held during `PATCH /api/data/wz` and `POST /api/data/process`. Persistence: Redis lock with TTL (atlas-data already depends on Redis).
- **Progress visibility**: `GET /api/data/process` returns the existing JSON:API shape enriched with per-worker status from Redis. Same shape as today, different data source.
- **Baseline publish / restore**: `POST /api/data/baseline/publish` and `POST /api/data/baseline/restore` per §4.4.

### 4.7 Character render service (`atlas-renders`)

Repackaged as a standalone Deployment. Code in `services/atlas-renders/atlas.com/renders/`.

- Request/response contract from task-043 preserved verbatim:
  - `GET /api/wz/character/render/<tenant>/<region>/<major>.<minor>/<hash>.png?skin=&hair=&face=&stance=&frame=&resize=&items=`
  - 200 with PNG body on success; standard error codes otherwise.
- Backing storage:
  - Input: fetches atlas PNG + manifest from `atlas-assets/<scope>/<region>/<version>/atlases/<partClass>/<id>.{png,json}`, scope-resolved per §4.2. Per-pod in-memory LRU caches `(atlas, manifest)` keyed by `(tenant, region, version, partClass, id)`. Default size: 256 entries. Scope resolution is cached per `(tenant, region, version, partClass)`.
  - Output: composited PNG PUT to `atlas-renders/tenant/<tenantId>/<region>/<version>/character/<hash>.png`. (Renders are intrinsically loadout-derived; cross-tenant reuse is impossible because the loadout points to tenant-scoped items. Renders are always under the per-tenant prefix, never canonical.)
- **WZ-parser import prohibition:** `services/atlas-renders/` MUST NOT import `libs/atlas-wz/wz` or `libs/atlas-wz/crypto`. It MAY import `libs/atlas-wz/manifest` (sprite atlas manifest types) and `libs/atlas-wz/maplayout` (map layout JSON types) — these are pure type packages with no WZ format knowledge. The lint enforces the prohibition at subpackage granularity (see §7.2).
- Cache invariants:
  - Atlases and manifests are immutable per content. A re-extraction for the same `(scope, region, version)` writes byte-identical bytes when inputs are unchanged (per §4.1 determinism guarantee).
  - Rendered character PNGs are loadout-hash-keyed; the cache survives future re-extractions until the version itself is decommissioned. No wipe-on-extract behavior. (This supersedes task-043's wipe-on-extract behavior; that wipe is removed.)

**Map render endpoint (new, lazy).** atlas-renders adds a second handler so map composites are produced on demand rather than during ingest:

- `GET /api/wz/map/render/<tenant>/<region>/<major>.<minor>/<mapId>/<kind>.png` where `<kind>` ∈ `{render, minimap}`.
- `minimap`: served from `atlas-assets/<scope>/.../map/<mapId>/minimap.png` (materialized eagerly during ingest because minimaps are small and almost universally viewed). atlas-renders proxies/streams it; the atlas-ingress regex can equivalently route directly to MinIO, bypassing atlas-renders for cache hits.
- `render`: probes `atlas-renders/<tenantId>/<region>/<version>/map/<mapId>/render.png` first. On hit, streams. On miss, fetches the per-map layer PNGs + `layout.json` from `atlas-assets/<scope>/.../map/<mapId>/`, composites in the z-sort/blit logic ported from the current `mapimage` package, PUTs to MinIO, then streams the response.
- Subsequent requests are MinIO cache hits served by atlas-ingress directly. The ingress regex for `/api/assets/.../map/<mapId>/render.png` follows the same per-tenant→shared fallback as other static assets; only a MinIO 404 falls through to atlas-renders.
- Cold-cache composite time: ~hundreds of ms to ~2 s depending on map complexity. Same shape as character render.
- Ingest wall-clock impact: removing eager map render is the largest single time saving in ingest. Appendix B reflects this.

### 4.8 atlas-ui changes

**Asset URL builder, character-render URL builder, and service worker are unchanged.** URLs stay `${VITE_ASSET_BASE_URL || '/api/assets'}/<tenant>/<region>/<version>/<category>/<id>/icon.png` and `/api/assets/.../character/<hash>.png?...`. atlas-ingress (§4.3) translates them to MinIO + atlas-renders behind the scenes. `public/sw-character-cache.js` caches by URL and is unaffected. Data hooks (`useItemData`, `useMobData`, `useSkillData`, `useNpcData`) and `VITE_ASSET_BASE_URL` are untouched.

**SetupPage (`src/pages/SetupPage.tsx`) changes — this is the source change v1 incorrectly claimed was absent.** Today the page exposes three buttons against three endpoints:

| Today's row | Endpoint | Fate |
|---|---|---|
| Upload WZ | `PATCH /api/wz/input` → atlas-wz-extractor | **Renamed + scoped.** Endpoint moves to `PATCH /api/data/wz` (atlas-data) with `?scope=tenant|shared` from the new toggle (see below). Row label and badge source stay; the underlying React Query hook is repointed. |
| Run Extraction | `POST /api/wz/extractions` → atlas-wz-extractor | **Deleted.** No standalone extraction artifact exists anymore. The `useExtractionStatus`, `useRunExtraction` hooks and the `extraction.service.ts` module are removed. The "stale uploads" warning banner is deleted with it. |
| Process Data | `POST /api/data/process` → atlas-data | **Kept + scoped.** Endpoint unchanged. Same `?scope=` from the toggle. Semantics expand to "parse WZ + atlas-pack + write Postgres + write MinIO," but the client doesn't see the difference. |

**New scope toggle at the top of the card.** A two-option control (radio or segmented switch) governs whether the Upload + Process actions target the current tenant or replace the shared canonical baseline:

- **"This tenant"** (default) → `?scope=tenant`. Writes land under `atlas-wz/tenant/<tenantId>/...` and `atlas-assets/tenant/<tenantId>/...`; Postgres rows use the requesting tenant_id.
- **"Canonical (shared)"** → `?scope=shared`. Writes land under `atlas-wz/shared/...` and `atlas-assets/shared/...`; Postgres rows use the canonical tenant UUID. Requires the operator header. The card surfaces a warning when this option is selected ("This will replace the shared canonical baseline for {region} v{major}.{minor}.").

The card description rewrites from *"Upload a WZ zip, extract it into XMLs, then ingest the XMLs into atlas-data. Each step is independent."* to *"Upload a WZ zip and ingest it into atlas-data. Choose 'Canonical (shared)' to replace the baseline that PR environments and new tenants restore from."*

**One new conditional row** for baseline restore (the publish action is folded into the Upload + Process flow with `scope=shared` above, so no separate Publish row is needed):

| New row | Endpoint | Visibility |
|---|---|---|
| Restore Canonical Baseline | `POST /api/data/baseline/restore` | Visible when the active tenant has `documentCount == 0` for its configured `(region, version)`. Primary path for hydrating a new tenant against a published baseline; this is the same endpoint PR-bootstrap hits. |

Status badge: from `GET /api/data/status` augmented with `baselineRestoredAt` (new nullable timestamp).

A separate operator-only **"Publish current canonical assets as baseline"** action emits the Postgres dump from `scope=shared` to `atlas-canonical/baseline/...` via `POST /api/data/baseline/publish`. UI affordance: when `scope=shared` is selected AND ingest is complete AND a dump has not been published for the current (region, version), surface a "Publish Baseline" call-to-action under the Process row. Visibility is gated by the operator header.

**Net source changes in atlas-ui:**

- `src/pages/SetupPage.tsx` — remove Extract row, add scope toggle to the Game Data card, add Restore row, conditional Publish CTA, rewrite card description.
- `src/components/features/setup/ScopeToggle.tsx` — new; the segmented control + warning.
- `src/services/api/extraction.service.ts` — delete.
- `src/lib/hooks/api/useExtraction.ts` — delete.
- `src/services/api/wzInput.service.ts` — repoint URL to `/api/data/wz`, accept `scope` arg.
- `src/services/api/dataProcess.service.ts` — accept `scope` arg.
- `src/services/api/baseline.service.ts` — new; `restore()` and `publish()`.
- `src/lib/hooks/api/useBaseline.ts` — new mutations.
- Tests under `__tests__/` updated.

Nothing outside SetupPage and its hooks/services moves.

### 4.9 atlas-pr-bootstrap changes

- `scripts/bootstrap.sh` is rewritten to use `BOOTSTRAP_MODE` (`baseline` default, `full` opt-in).
- In `baseline` mode: skip `wz-upload` and `wz-extract` steps; call `POST /api/data/baseline/restore` instead of `POST /api/data/process`; same stable-poll pattern against `GET /api/data/status`.
- In `full` mode: existing flow, but pointed at the new endpoints (`PATCH /api/data/wz` instead of `PATCH /api/wz/input`; the `POST /api/data/process` step does both extraction and ingest in one call). The `wz-extract` step disappears.
- The `wait-ready` step drops `atlas-wz-extractor` from its readiness list and adds `atlas-renders`.
- The canonical-baseline producer workflow (`bootstrap-canonical.sh`) is a new sibling script. Run once per game version against the cluster's canonical tenant; emits the dump to `atlas-canonical/baseline/...`. Not invoked from PostSync; run by an operator on demand or wired into a separate Argo Application.

### 4.10 docker-compose changes

Compose currently runs three services with bind-mounted host directories (`tmp/data`, `tmp/assets`, `tmp/wz-input`). The new compose stack:

- **Adds `minio`**: `minio/minio:latest` with a named volume, ports `9000:9000` and `9001:9001` (console). Single-node single-drive matching the cluster pattern. Default creds match `~/source/k3s/bee/minio.yml` for parity (`minioadmin` / `minioadmin12345`).
- **Adds `minio-init`** as a short-lived sidecar: runs `mc alias set`, `mc mb` for `atlas-wz`, `atlas-assets`, `atlas-renders`, `atlas-canonical`, and `mc anonymous set download` on the public buckets. Healthcheck-gated so other services wait for buckets to exist.
- **Adds `atlas-renders`**: new service-equivalent, no volumes.
- **Removes `atlas-wz-extractor`** service definition.
- **Removes `atlas-assets`** service definition.
- **Updates `atlas-data`**: drops `ZIP_DIR` env and the `../../tmp/data` mount. Adds `MODE=all` (the compose-only "REST + workers in one process" topology), `MINIO_ENDPOINT=http://minio:9000`, `MINIO_BUCKET_WZ=atlas-wz`, `MINIO_BUCKET_ASSETS=atlas-assets`, `MINIO_BUCKET_CANONICAL=atlas-canonical`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`. No `INGEST_JOB_TEMPLATE_CM` env (not used in `MODE=all`).
- **Updates `atlas-ingress`**: routes.conf reflects §4.3.
- **Removes `tmp/data`, `tmp/assets`, `tmp/wz-input`** directories from documentation and `.gitignore` cleanup (they're not in source, but referenced by mounts).
- A `compose/seed-canonical.sh` companion script runs `bootstrap-canonical.sh` against the local stack to populate the canonical baseline. Documented in `services/atlas-data/README.md` and the new top-level local-dev doc.

After cutover, `docker compose up` for the full local stack:

1. Starts MinIO + bucket init.
2. Starts atlas-data + atlas-renders + the rest.
3. Operator runs `seed-canonical.sh` once (or on WZ change) to populate canonical assets + documents dump.
4. Operator runs the existing SetupPage flow (or the equivalent `bootstrap.sh baseline` shape) to spin up a working tenant.

The WZ upload path (compose-only, for testing extraction itself) still works: the dev hits `PATCH /api/data/wz` against MinIO via atlas-data.

### 4.11 Retired artifacts and code paths

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

Stages a `.wz` upload. Streams multipart, validates, writes to `atlas-wz/<scope-key>/<region>/<version>/<filename>.wz`.

Request:
- `Content-Type: multipart/form-data`
- Part `zip_file` — flat `.wz` entries.
- Tenant headers: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
- Optional query: `?scope=tenant|shared`. Default `tenant`.
- When `scope=shared`: header `X-Atlas-Operator: 1` required.

`<scope-key>` resolution: `tenant/<tenantId>` when `scope=tenant`; `shared` when `scope=shared`.

Validation: no path separators, no zip-slip, no symlinks, `.wz` extension only.

Responses: `202 Accepted` empty body / `400 Bad Request` / `403 Forbidden` (scope=shared without operator header) / `409 Conflict` (per-scope mutex held) / `500 Internal Server Error`.

#### `GET /api/data/wz`

JSON:API status of staged `.wz` objects. Returns `200` with `{ data: { type: "wzInputStatus", attributes: { fileCount, totalBytes, updatedAt } } }`. Same shape as the existing `GET /api/wz/input` for migration ergonomics.

#### `POST /api/data/baseline/publish`

Operator-only (header `X-Atlas-Operator: 1`). After a successful `scope=shared` ingest, snapshots the canonical tenant's `documents` + 5 search-index rows into the dump format defined in §6.1, writes to `atlas-canonical/baseline/<region>/<version>/documents.dump`, and writes `documents.dump.sha256` next to it.

Request body:
```json
{ "region": "GMS", "majorVersion": 83, "minorVersion": 1 }
```

Responses: `202 Accepted` (async) / `403 Forbidden` (missing operator header) / `409 Conflict` (publish in progress) / `412 Precondition Failed` (no `scope=shared` ingest for that version).

#### `POST /api/data/baseline/restore`

Used by PR-env bootstrap. Restores the canonical baseline into the requesting tenant for the given version.

Request body:
```json
{ "region": "GMS", "majorVersion": 83, "minorVersion": 1, "tenantId": "<pr-tenant>" }
```

Behavior:
- Acquires a Redis lock on `(tenantId, region, version)` for the duration of the restore — concurrent calls for the same target tenant serialize.
- Idempotent: if the PR tenant already has documents for that version, returns `204 No Content` with `X-Atlas-Baseline-Status: already-restored`. (Note: one version per tenant, per §6.1; presence of *any* documents under that tenant_id counts.)
- Else fetches the dump, verifies the SHA-256 of the `.tar`, validates `header.json.schemaVersion` against the compiled-in schema version, then for each table: DELETE rows for the target tenant, then COPY with tenant_id substitution.

Responses: `202 Accepted` (running) / `204 No Content` (already restored) / `404 Not Found` (no baseline published) / `409 Conflict` (lock held) / `422 Unprocessable Entity` (hash mismatch or schema-version mismatch).

#### `DELETE /api/data/tenants/<tenantId>`

Operator-gated (header `X-Atlas-Operator: 1`). Purges all Postgres rows and MinIO objects belonging to the tenant. Per §4.4a:

- Deletes from `documents` and the 5 search-index tables.
- Recursively deletes `atlas-wz/tenant/<tenantId>/`, `atlas-assets/tenant/<tenantId>/`, `atlas-renders/<tenantId>/`.

Behavior:
- Refuses the canonical UUID `00000000-0000-0000-0000-000000000000` with `403 Forbidden`.
- Idempotent: re-invoking on an empty tenant returns `204 No Content`.

Responses: `202 Accepted` / `204 No Content` (already clean) / `403 Forbidden` (canonical UUID or missing operator header) / `500 Internal Server Error`.

### 5.2 Modified endpoints

#### `POST /api/data/process`

Existing endpoint. Workers read from MinIO (`atlas-wz` bucket) rather than `ZIP_DIR`. Optional query `?scope=tenant|shared` (default `tenant`); `scope=shared` requires `X-Atlas-Operator: 1`. Output scope follows the resolved `<scope-key>` (`shared` or `tenant/<tenantId>`). Response contract unchanged.

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

**One version per tenant (invariant)**: the `documents` table has no `region` or `version` column. By construction, **each tenant holds exactly one (region, version) at a time.** A tenant's `(region, version)` is implicit in atlas-tenants's tenant config, not in `documents`. Re-ingesting or restoring for the same tenant at a new version is therefore a destructive operation: existing rows for that tenant are deleted before new rows are inserted. This is the long-standing semantics; the PRD makes it explicit because the baseline-restore flow depends on it.

**Canonical tenant**: the reserved UUID `00000000-0000-0000-0000-000000000000` is the operator-controlled canonical tenant. atlas-data ingest writes its outputs to the `shared/` scope in `atlas-assets` (and the corresponding `documents` rows in Postgres) when invoked with `scope=shared` (see §4.6). Runtime tenants never write to the canonical UUID's row range; the application layer enforces this via an operator-permission check on `PATCH /api/data/wz`, `POST /api/data/process`, and `POST /api/data/baseline/publish` when `scope=shared`. The reserved UUID is already used in compose (`atlas-drops`, `atlas-drop-information` use it as a SERVICE_ID); reusing it for the canonical tenant is intentional — it's the conventional "zero" value.

**Baseline dump format**: not `pg_dump --format=custom`, which doesn't natively filter by column value. Instead, the dump is a small, self-describing archive produced by atlas-data with this structure:

```
documents.dump
  ├── header.json                              # { "schemaVersion": "v1", "region": "GMS",
  │                                            #   "majorVersion": 83, "minorVersion": 1,
  │                                            #   "tables": ["documents", "map_search_index",
  │                                            #     "npc_search_index", "monster_search_index",
  │                                            #     "reactor_search_index", "item_string_search_index"],
  │                                            #   "publishedAt": "2026-05-19T18:00:00Z" }
  ├── documents.binary                         # COPY (SELECT * FROM documents WHERE tenant_id = '00000000-…')
  │                                            #   TO STDOUT (FORMAT binary)
  ├── map_search_index.binary                  # same shape, filtered by tenant_id
  ├── npc_search_index.binary
  ├── monster_search_index.binary
  ├── reactor_search_index.binary
  └── item_string_search_index.binary
```

Wrapped in a single `.tar` so the SHA-256 covers the whole bundle. Each `*.binary` file is the raw output of `COPY (SELECT * FROM <table> WHERE tenant_id = '00000000-…') TO STDOUT (FORMAT binary)` and is restored via `COPY <table> FROM STDIN (FORMAT binary)` with the tenant_id rewritten in flight (atlas-data streams the binary, decodes the tenant_id field, substitutes the target tenant_id, re-encodes, and pipes into the target).

**Schema version**: the `header.json` `schemaVersion` field is checked against atlas-data's compiled-in version. A mismatch fails restore with `422 Unprocessable Entity` and a clear error indicating the dump was produced by a different schema version. Increment the schema version whenever any of the 6 dumped tables gains, drops, or changes a column.

**Baseline restore semantics**: server-side per-table loop. For each table:
1. `BEGIN`.
2. `DELETE FROM <table> WHERE tenant_id = <pr-tenant>` (destroys any prior version for this tenant — see "one version per tenant" above).
3. Stream the `*.binary` payload through a tenant_id rewriter into `COPY <table> FROM STDIN (FORMAT binary)`.
4. `COMMIT`.

Across the 6 tables we accept multi-transaction restore (not one big transaction) to keep each table's WAL footprint bounded. Failure mid-restore leaves the tenant in a partial state; recovery is "call restore again" (idempotent via the Redis lock + the per-table DELETE-then-INSERT semantics).

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
atlas-wz/<tenantId>/<region>/<major>.<minor>/<filename>.wz

atlas-assets/<scope>/<region>/<major>.<minor>/<category>/<id>/icon.png
atlas-assets/<scope>/<region>/<major>.<minor>/atlases/<partClass>/<id>.png
atlas-assets/<scope>/<region>/<major>.<minor>/atlases/<partClass>/<id>.json
atlas-assets/<scope>/<region>/<major>.<minor>/map/<mapId>/render.png
atlas-assets/<scope>/<region>/<major>.<minor>/map/<mapId>/minimap.png

atlas-renders/tenant/<tenantId>/<region>/<major>.<minor>/character/<hash>.png

atlas-canonical/baseline/<region>/<major>.<minor>/documents.dump
atlas-canonical/baseline/<region>/<major>.<minor>/documents.dump.sha256
```

`<scope>` = `shared` or `tenant/<tenantId>`. (The `shared` prefix is populated by canonical-tenant ingest; v1 of this PRD called it `canonical/`, but that overloaded the `atlas-canonical` bucket name.)

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
- New env: `MODE` (`rest|ingest|all`), `MINIO_ENDPOINT`, `MINIO_BUCKET_WZ`, `MINIO_BUCKET_ASSETS`, `MINIO_BUCKET_CANONICAL`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`. In k8s: `INGEST_JOB_TEMPLATE_CM` (ConfigMap name holding the Job template).
- Removes `ZIP_DIR`.
- New endpoints: `PATCH /api/data/wz`, `GET /api/data/wz`, `POST /api/data/baseline/publish`, `POST /api/data/baseline/restore`.
- Workers refactored to source files from MinIO; per-domain processors now consume WZ data via `libs/atlas-wz` instead of `.img.xml`.
- Adds atlas packing for Character.wz and map rendering for Map.wz to the ingest worker set (running inside the Job in k8s; in-process in compose).
- In `MODE=rest`: HTTP API runs; Kafka consumers do **not** start. `POST /api/data/process` instantiates a Job from the baked template, parameterized with `(tenantId, region, version)`. The Job pod runs `MODE=ingest`. REST polls Job status for `GET /api/data/process`.
- In `MODE=ingest`: Kafka consumers start; HTTP server does **not** start. Workers run, then the process exits.
- In `MODE=all`: HTTP + Kafka consumers in one process; `POST /api/data/process` publishes a command that the same process consumes inline. Today's behavior.
- Drops `/usr/data` mount from Dockerfile.
- Adds k8s `Role`/`RoleBinding` for Job creation in atlas-data's own namespace (`MODE=rest` only).

### 7.2 New service: `atlas-renders`

- New service `services/atlas-renders/atlas.com/renders/`. Module: `atlas-renders` (short).
- Character render handler + map render handler + supporting compositors.
- MinIO Go SDK for atlas/layer/manifest reads, render writes.
- In-memory LRU for `(atlas, manifest)` pairs and map layer/layout pairs and scope resolution.
- No PVC. Prohibited from importing `libs/atlas-wz/wz` or `libs/atlas-wz/crypto`. Permitted to import `libs/atlas-wz/manifest` and `libs/atlas-wz/maplayout` (pure type packages).
- **Lint:** the prohibition is enforced by a CI step that runs `go list -deps ./services/atlas-renders/...` and greps for the disallowed subpackages. Plain text grep is insufficient because transitive imports must be caught. The lint is at subpackage granularity, not module granularity.
- Env: `MINIO_ENDPOINT`, `MINIO_BUCKET_ASSETS`, `MINIO_BUCKET_RENDERS`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `REST_PORT`, `LOG_LEVEL`, `JAEGER_HOST_PORT`, `ATLAS_LRU_SIZE`, `MAP_LRU_SIZE`.

### 7.3 Retired: `atlas-wz-extractor`

Deleted in full. Parser → `libs/atlas-wz`. Render handler → `atlas-renders`. XML emitter and map render → ported to atlas-data ingest workers, then the original sources deleted.

### 7.4 Retired: `atlas-assets`

Deleted in full. Replaced by atlas-ingress routing to MinIO + atlas-renders.

### 7.5 New library: `libs/atlas-wz`

- New Go module. Stdlib + minimal image deps.
- Package layout:
  - `wz/` — file, directory, image, property types. **Forbidden in atlas-renders.**
  - `crypto/` — `WzKey` and IV seeds. **Forbidden in atlas-renders.**
  - `canvas/` — pixel-format decoders. **Forbidden in atlas-renders.**
  - `atlas/` — sprite atlas packer. **Forbidden in atlas-renders.**
  - `mapimage/` — map layer extraction (the input-prep stage for lazy map render). **Forbidden in atlas-renders.**
  - `icons/` — icon-extraction dispatcher. **Forbidden in atlas-renders.**
  - `manifest/` — pure type definitions for the sprite atlas manifest JSON (§6.2). No WZ knowledge; importable anywhere.
  - `maplayout/` — pure type definitions for the map layout JSON used by lazy map render. No WZ knowledge; importable anywhere.
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
- `atlas-data.yaml`: drop `/usr/data` mount + PVC; add MinIO env from secret; set `MODE=rest`; add ServiceAccount + `Role`/`RoleBinding` granting `create/get/list/watch/delete` on `batch.Job` in its own namespace.
- New `atlas-data-ingest-job-template.yaml` ConfigMap: holds the Job spec atlas-data renders when ingest is requested. Resource limits CPU `2-8` / memory `1-3Gi` (today's atlas-wz-extractor profile). The Job pod runs the same atlas-data image with `MODE=ingest` and a single env override (`(tenantId, region, version)`).
- Remove `atlas-wz-extractor.yaml`, `atlas-assets.yaml`.
- Drop PVC defs for `atlas-data-pvc`, `atlas-assets-pvc`, `atlas-wz-input-pvc`.
- atlas-ingress manifest updated to reflect new routes.conf (including the `client_max_body_size 4G` and Cache-Control header injection per §4.3).
- **Cutover deploy strategy**: drain all old `atlas-data` pods (RWX-mounting, in-process worker) before any new `MODE=rest` pod accepts traffic. This is an explicit downtime window for atlas-data — accepted because the alternative (mixed old/new pods, with new commands consumed by old workers that expect `.img.xml` files) corrupts ingest. Other services keep running; only atlas-data is briefly unavailable. Operator coordinates with PR-env timing.
- Note: cluster-level MinIO ingress (direct browser access bypassing atlas-ingress) is **not** added in this task; all asset traffic flows through atlas-ingress. Direct-from-browser-to-MinIO is a future optimization deferred behind a CDN decision.

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

- All MinIO keys in `atlas-assets` are prefixed by `<scope-key>` where `<scope-key>` ∈ {`shared`, `tenant/<tenantId>`}. The per-tenant prefix is preserved for future per-tenant WZ overrides (user direction); the operational cost (HEAD-probe-on-cold-cache, scope-resolution LRU in atlas-renders, error_page fallback in the ingress) is accepted to keep the option open without a re-design.
- Postgres rows carry `tenant_id` and are filtered via `tenant.MustFromContext(ctx)` in every read/write path.
- atlas-renders parses tenant headers via the existing middleware; refuses requests missing them.
- A tenant deletion (today: PR-env teardown; future: any tenant-decommission flow) calls `DELETE /api/data/tenants/<tenantId>` per §4.4a. Postgres rows + MinIO prefixes are purged in that handler.

### 8.5 Tests

- `libs/atlas-wz`: unit tests for every parser branch and every canvas pixel format.
- atlas-data ingest: testcontainers MinIO + Postgres integration test exercising upload → process → documents+atlases for a small fixture WZ set.
- atlas-data baseline: integration test for `publish` then `restore` into a fresh PR-tenant fixture; verifies row counts, search-index rebuild, and hash-mismatch failure mode.
- atlas-renders: testcontainers MinIO with fixture atlases + manifests; verifies hash-keyed cache, scope-fallback, byte-identical render output vs. a frozen task-043 baseline.
- Renderer "no WZ parser at runtime" invariant: CI greps the atlas-renders import graph for `libs/atlas-wz`.
- atlas-pr-bootstrap: a compose-level smoke test exercises `BOOTSTRAP_MODE=baseline` end-to-end against the seeded canonical baseline.

## 9. Open Questions

- **MinIO single-drive durability**: the cluster MinIO is single-node single-drive on Longhorn. With this task, MinIO becomes the canonical store for game-data artifacts. Should the design phase add an erasure-coded or replicated MinIO topology? Recovery story today is "re-extract from raw `.wz`", which still works post-cutover.
- **Baseline-restore atomicity**: resolved per §6.1 — one transaction per table, not one transaction overall. Restore is idempotent (DELETE-then-COPY); failure mid-restore is recoverable by re-calling restore.
- **Map.wz render coverage**: now resolved by going lazy. Maps that fail to composite return 500 from atlas-renders and the UI displays its existing failure placeholder; no eager-render success-rate concern blocks ingest.
- **Sprite atlas packing algorithm**: a stable rectangle-packing algorithm is required for the determinism guarantee in §4.1. MaxRects with a fixed sort order is the leading candidate; design phase enumerates the alternatives and pins one.
- **Atlas size budget**: rough estimate is 20–30k equip atlases per version at a few KB each, plus ~5k icons, plus map renders (potentially MB each). Validate against an actual v83 Character.wz + Map.wz before committing the design.
- **Manifest schema versioning policy**: documented as additive-only in §6.2. Confirm in design.
- **Service-worker cache invalidation**: `public/sw-character-cache.js` caches renders by URL. If a render's bytes change (e.g., from an atlas re-pack), the URL stays the same and the SW serves stale bytes. Design must specify the cache-busting strategy (version in URL? content hash in URL? SW cache versioning?).
- **`docker-compose` MinIO healthcheck timing**: `minio-init` must run after MinIO is ready but before atlas-data starts. Design must specify the healthcheck or `depends_on: condition: service_healthy` chain.
- **PR-env failure modes**: what happens if `baseline/restore` is invoked but no canonical baseline exists yet? Block bootstrap with a clear error (defaulting decision favors visible failure over silent slow paths), or fall back to `full` mode automatically? Design pick. Resolved partially: the cutover itself sidesteps the cold-cluster case by publishing the baseline from the cutover PR's own ephemeral env (§2 goals) before merge.
- **`tools/task-numbers.sh` `next` exits 1 with `set -e`**, and the scan misses remote-tracking branches (caught the collision with task-032-dynamic-service-config in this task's creation). Out of scope here; tracked for a separate task.

## 10. Acceptance Criteria

- [ ] `libs/atlas-wz` module created with parser + crypto + canvas decoder + atlas packer + map layer extractor + icon extractor + manifest types + maplayout types. Unit tests cover every property type and every pixel format. Determinism guarantee tested via a "pack twice, compare bytes" assertion. Public API documented, with each subpackage's atlas-renders importability noted.
- [ ] `atlas-data` ingests `.wz` from MinIO end-to-end: upload via `PATCH /api/data/wz`, trigger via `POST /api/data/process`, observe documents in Postgres and icons/atlases/map-layers in MinIO. Integration test green for both `scope=tenant` and `scope=shared`.
- [ ] `services/atlas-renders/` deployed; serves `GET /api/wz/character/render/...` and `GET /api/wz/map/render/...` against MinIO-only inputs. Final source tree has no import of `libs/atlas-wz/wz`, `libs/atlas-wz/crypto`, `libs/atlas-wz/canvas`, `libs/atlas-wz/atlas`, `libs/atlas-wz/mapimage`, or `libs/atlas-wz/icons`. Lint check enforces it at subpackage granularity via `go list -deps`.
- [ ] `services/atlas-wz-extractor/` deleted.
- [ ] `services/atlas-assets/` deleted.
- [ ] `atlas-data-pvc`, `atlas-assets-pvc`, `atlas-wz-input-pvc` removed from all `deploy/k8s/` overlays.
- [ ] atlas-ingress (`deploy/shared/routes.conf`) updated: `/api/assets/...` routes to MinIO with per-tenant→shared fallback; `/api/assets/.../character/...` routes to atlas-renders; `/api/wz/map/render/...` routes to atlas-renders. `/api/wz/input`, `/api/wz/extractions` removed.
- [ ] atlas-ui SetupPage updated: extraction row deleted; upload row repointed to `/api/data/wz`; scope toggle (Tenant ⚪ Canonical) added; Restore row visible when `documentCount == 0`; Publish Baseline CTA visible when `scope=shared` is selected after successful ingest. URL builders for assets and character render unchanged.
- [ ] atlas-data binary respects `MODE=rest|ingest|all`. In `MODE=rest`, `POST /api/data/process` creates a Job from the baked template; in `MODE=ingest`, the process runs workers and exits; in `MODE=all`, REST + workers coexist in one process. End-to-end ingest tested in all three modes.
- [ ] `docker-compose` updated: adds MinIO + bucket init, removes extractor + assets, adds atlas-renders. `docker compose up` (plus a one-time `seed-canonical.sh`) produces a working local stack.
- [ ] `atlas-pr-bootstrap`'s `bootstrap.sh` updated for `BOOTSTRAP_MODE=baseline` (default) and `BOOTSTRAP_MODE=full`. Auto-detection: if PR diff touches `libs/atlas-wz/` or `services/atlas-data/atlas.com/data/`, force `full`. New `bootstrap-canonical.sh` for operator publish. `cleanup.sh` calls `DELETE /api/data/tenants/<tenantId>` for the PR tenant. `baseline` mode end-to-end ≤ 60 s against a pre-published baseline (verified in a compose smoke test).
- [ ] `POST /api/data/baseline/publish` produces a deterministic dump (re-run yields identical SHA-256 for unchanged inputs). Dump format matches §6.1.
- [ ] `POST /api/data/baseline/restore` is idempotent: second call returns `204 No Content`; concurrent calls for the same target tenant serialize via Redis lock (no double-insert).
- [ ] `DELETE /api/data/tenants/<tenantId>` purges Postgres rows + MinIO prefixes; refuses canonical UUID with 403; integration-tested via `atlas-pr-bootstrap cleanup.sh`.
- [ ] Lazy map render: `GET /api/wz/map/render/.../<mapId>/render.png` returns a composite on cold cache (~hundreds of ms), serves from MinIO on subsequent requests (cache hit < 50 ms via atlas-ingress).
- [ ] atlas-ingress carries `client_max_body_size 4G` + `proxy_request_buffering off` on `/api/data/wz` (verified by uploading a >1GB .wz zip).
- [ ] atlas-ingress injects `Cache-Control: public, max-age=86400` on icon responses and `immutable` on render responses (verified by curl -I).
- [ ] For a single canonical loadout (documented in tests), `atlas-renders` produces a PNG byte-identical to (or visually identical within a documented diff tolerance) the same loadout rendered by the pre-cutover task-043 service.
- [ ] `docker build -f services/atlas-data/Dockerfile .` and `docker build -f services/atlas-renders/Dockerfile .` succeed from the worktree root. atlas-data's Dockerfile correctly lists `libs/atlas-wz` in all four required locations (CLAUDE.md mandate).
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] Documentation updated: `services/atlas-data/README.md` describes the new ingest flow + canonical baseline; the obsolete `services/atlas-wz-extractor/README.md` and `services/atlas-assets/Dockerfile` are gone; a new `docs/runbooks/wz-ingest.md` covers operator workflows (raw upload, full ingest, canonical publish, baseline restore). `docs/runbooks/ephemeral-pr-deployments.md` updated for the new bootstrap modes.
- [ ] CHANGELOG / commit history reflects the cutover. The single cutover PR is reviewable as one unit.

---

## Appendix A — Affected services summary

| Service | Change |
|---|---|
| `atlas-data` | Gains WZ parser via `libs/atlas-wz`. New ingest endpoints, new baseline publish/restore endpoints. New `MODE` env: `rest` (k8s REST pod, creates Jobs), `ingest` (k8s Job, no HTTP), `all` (compose). Workers read from MinIO. PVC dropped. RBAC for Job creation. |
| `atlas-wz-extractor` | Deleted. Parser → `libs/atlas-wz`. Render → `atlas-renders`. XML, map, PVC paths deleted outright. |
| `atlas-assets` | Deleted. atlas-ingress routes `/api/assets/...` to MinIO with per-tenant→canonical fallback. |
| `atlas-renders` (new) | Composites character renders from MinIO atlases. No WZ parser. |
| `atlas-ingress` | routes.conf updated for MinIO upstream and character-render path detection. |
| `atlas-ui` | SetupPage drops the "Run Extraction" row, repoints the upload row to `/api/data/wz`, adds two conditional rows for baseline restore/publish. URL builders unchanged. |
| `atlas-pr-bootstrap` | `bootstrap.sh` rewritten for `BOOTSTRAP_MODE` (baseline-mode default). New `bootstrap-canonical.sh`. `cleanup.sh` gains `tenant-purge` step calling `DELETE /api/data/tenants/<tenantId>`. |
| `libs/atlas-wz` (new) | Shared parser + canvas + atlas packer + map render. |
| `deploy/k8s` | Three PVCs deleted; two Deployments deleted; one added (`atlas-renders`). |
| `deploy/compose` | MinIO + init added; extractor + assets removed; atlas-renders added; volume mounts removed from atlas-data. |

## Appendix B — Performance and capacity sketch (for design phase to validate)

- **MinIO object count**: ~20–30k equip atlases × 2 files (PNG + JSON) + ~5k icons + ~1k map images = ~60k objects per canonical version. Per-tenant overrides currently zero.
- **MinIO storage** (per canonical version): PNG-packed atlases ≈ 100–200 MB; map layer PNGs + layout JSON + minimaps ≈ 200–400 MB; icons ≈ 50–100 MB. **Total: ~500 MB–1 GB per canonical version**, down from v1's ~2 GB estimate now that map renders are lazy.
- **`atlas-renders` bucket growth**: unbounded per-tenant per-version, accumulating one PUT per distinct character loadout hash and one PUT per distinct map view. Mitigated by per-tenant cleanup at PR-env teardown (§4.4a). For long-lived tenants (production), a lifecycle policy (MinIO `mc ilm` with TTL on the renders bucket) is the operational answer; rules out of scope but flagged in §9.
- **Postgres documents.dump (`.tar` per §6.1)**: estimated 50–150 MB per canonical version (binary COPY format + 5 search indexes). Stored in `atlas-canonical` bucket.
- **Ingest wall-clock** (lazy map render): ~5–7 min total, down from v1's ~10–15 min estimate. Map composites materialize on first view rather than during ingest.
- **PR-env bootstrap baseline-mode wire time**: ~50 MB dump fetch + ~10 s restore + ~5 s seed = ≤ 60 s end-to-end. Today: 10+ min.
- **Map render cold-cache time**: ~hundreds of ms to ~2 s depending on map complexity (parse layout JSON, fetch layer PNGs, composite, PUT). Cached thereafter.
- **Atlas LRU per atlas-renders pod**: 256 entries × ~200 KB avg = ~50 MB working set. Map layer LRU: similar order. Total renders-pod memory bounded.
