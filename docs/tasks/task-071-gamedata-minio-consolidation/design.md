# Game Data + Asset Pipeline Consolidation onto MinIO — Design

Version: v1
Status: Draft
Created: 2026-05-20
Consumes: `prd.md` v5 (approved)

This design pins the architectural choices the PRD intentionally deferred, fixes the implementation shape across the seven affected components (`libs/atlas-wz`, `atlas-data`, `atlas-renders`, atlas-ingress, atlas-ui, atlas-pr-bootstrap, deploy/k8s + deploy/compose), and records the alternatives that were rejected. The PRD owns *what* and *why*; this document owns *how*, in enough detail that `plan.md` can decompose it into ordered tasks without re-litigating decisions.

---

## 1. Architecture at a glance

```
+----------------------+   PATCH /api/data/wz       +-----------------------+
| operator / PR-boot.  |--------------------------->|  atlas-data REST      |
| SetupPage UI         |   POST  /api/data/process  |  (MODE=rest, k8s)     |
|                      |   POST  /api/data/baseline |  (MODE=all, compose)  |
+----------------------+                            +-----------+-----------+
       ^                                                        |
       | /api/assets/...                                        | creates k8s Job
       | /api/wz/character/render/...                           | (MODE=rest only)
       | /api/wz/map/render/...                                 v
+------+-------+        try_files / error_page 404      +----------------+
| atlas-ingress|-------------------------------------+  | atlas-data Job |
|  (nginx)     |                                     |  | (MODE=ingest)  |
+--------------+                                     |  +-------+--------+
       | proxy_pass                                  |          |
       v                                             v          v
+----------------+                              +-----------------+
| atlas-renders  |  GET atlases/manifests       |  MinIO          |
| (2..8 replicas)|<---------------------------- |  buckets:       |
|  - char render |  PUT renders/<hash>.png      |   atlas-wz      |
|  - map render  |----------------------------->|   atlas-assets  |
+----------------+                              |   atlas-renders |
                                                |   atlas-canonical|
                                                +-----------------+
                                                         ^
                                                         | binary COPY
                                                         | dump/restore
                                                         v
                                                  Postgres (documents,
                                                  5 search indexes,
                                                  tenant_baselines)
```

The single biggest shape change vs. today is that **every cross-service handoff is HTTP+MinIO** — there are no shared PVCs, no Kafka command topic for ingest dispatch, and no `.img.xml` intermediate. The WZ format lives in one library and one runtime path (the ingest worker).

---

## 2. `libs/atlas-wz` library design

### 2.1 Module layout

```
libs/atlas-wz/
  go.mod                          // module github.com/Chronicle20/atlas/libs/atlas-wz
  README.md                       // public API + Go version pin + atlas-renders importability table
  wz/                             // FORBIDDEN in atlas-renders
    reader.go                     // io.ReaderAt-based reader, no os/file deps
    file.go file_test.go
    directory.go directory_test.go
    image.go
    property/                     // typed WZ property tree
  crypto/                         // FORBIDDEN in atlas-renders
    wzkey.go                      // GMS/KMS/Empty IV seeds
  canvas/                         // FORBIDDEN in atlas-renders
    decode.go                     // 8 pixel formats → image.Image
    bgra4444.go bgra8888.go argb1555.go bgr565.go
    block_rgb565.go dxt3.go dxt5.go dxt3gray.go
  atlas/                          // FORBIDDEN in atlas-renders
    pack.go                       // MaxRects + stable sort → (sheet, manifest)
    pack_test.go                  // pack-twice byte-identity test
    pngenc/                       // pinned PNG encoder (see §2.4)
  mapimage/                       // FORBIDDEN in atlas-renders
    layers.go                     // extract back/foreground/tile PNGs + layout JSON
    minimap.go                    // eager minimap extract
  icons/                          // FORBIDDEN in atlas-renders
    extract.go                    // category-dispatched icon extractor
  manifest/                       // PUBLIC (importable by atlas-renders)
    types.go                      // Manifest, Sprite, Rect, Point types (§6.2 of PRD)
    encode.go                     // key-sorting JSON encoder
  maplayout/                      // PUBLIC (importable by atlas-renders)
    types.go                      // foothold/portal/NPC/layer layout types
    encode.go                     // key-sorting JSON encoder
```

Three things make this layout load-bearing:

1. The **two pure-type subpackages** (`manifest/`, `maplayout/`) are the only points the renderer ever touches. Encoding/decoding does not require WZ knowledge — the JSON files were already produced by the ingester. Splitting these out is what makes the CI lint (`go list -deps`) cleanly enforceable.
2. The PNG encoder lives **inside `libs/atlas-wz/atlas/pngenc/`**, not as a top-level subpackage, because it is only ever called by the packer. Pinning it here also avoids the dependency surface bleeding into `mapimage/` or `icons/` (which write PNGs via stdlib — that's safe because their outputs are not baseline-hashed; only atlas sheets are).
3. The module name is the full path (`github.com/Chronicle20/atlas/libs/atlas-wz`), matching the convention of every other `libs/atlas-*` module. The four-location Dockerfile pattern (CLAUDE.md mandate) applies, even for services that only import the type subpackages.

### 2.2 Public API surface

```go
// wz/
func NewFile(rd io.ReaderAt, key *crypto.WzKey) (*File, error)
func (*File) Root() *Directory
func (*Directory) Image(name string) (*Image, error)
func (*Image) Property() property.Property        // walks the parsed tree

// crypto/
func NewKey(variant Variant) *WzKey               // GMS, KMS, Empty

// canvas/
func Decode(format Format, raw []byte, w, h int) (image.Image, error)

// atlas/
type Input struct { Name string; Img image.Image; Origin, Anchors map[string]image.Point; Z int }
func Pack(in []Input) (sheet image.Image, m manifest.Manifest, err error)

// mapimage/
func ExtractLayers(img *wz.Image) (layers []LayerOutput, layout maplayout.Layout, err error)
func ExtractMinimap(img *wz.Image) (image.Image, error)

// icons/
func ExtractItem(img *wz.Image, id int) (image.Image, error)
func ExtractNpc(img *wz.Image, id int) (image.Image, error)
// ... one per category

// manifest/
type Manifest struct { ... }
func Marshal(m Manifest) ([]byte, error)          // key-sorted

// maplayout/
type Layout struct { ... }
func Marshal(l Layout) ([]byte, error)            // key-sorted
```

The renderer's compile-time surface is exactly `manifest.{Manifest, Unmarshal}` and `maplayout.{Layout, Unmarshal}`. Nothing else.

### 2.3 Atlas packing algorithm

**Decision: MaxRects-BSSF (Best Short Side Fit) with a fixed pre-sort.**

Inputs are sorted before packing by `(width desc, height desc, name asc)` — the secondary `name asc` is what makes the result deterministic when two sprites have identical dimensions. The packer maintains free-rectangle list invariants in a stable order (slice index, not map iteration) so the bin selection is reproducible. We pick BSSF over MAXRECTS-BAF or Skyline because:

- BSSF gives the tightest packing on MapleStory equip sets in published benchmarks (typical 92–94% fill ratio); equip atlases are the dominant storage line, so a few percent matters at fleet scale.
- The algorithm has a single tuning parameter (heuristic = BSSF) — no per-call branching needed.
- It is single-threaded by construction; no parallelism means no `runtime.NumCPU()`-dependent variance.

We do **not** ship a published library; we vendor a small (~400 LOC) implementation under `atlas/pack.go` with explicit fixed sorting. Third-party libraries we evaluated:

| Library | Why not |
|---|---|
| `github.com/aybabtme/maxrects` | Unmaintained since 2018, depends on a generic image lib we don't use. |
| `github.com/blizzy78/binpack` | Uses `math/rand` for tie-break — not deterministic. |
| Custom port of Jukka Jylänki's reference C++ | Auditable, fixed-sort, vendored — chosen. |

**Bin sizing.** We start at 256×256 and grow by powers of two up to 4096×4096 (a single equip rarely needs more, but Map.wz's largest backgrounds can). If a sprite does not fit in 4096×4096, the pack call fails — the ingest worker logs and records the failure but does not abort the run; downstream renders for that part class return 404 (renderer falls through to "missing equip" sprite). This matches today's behavior for unparseable canvases.

**Sheet size budget.** Per Appendix B, ~20–30k equip sprites total. Most equips fit in a single 256×256 sheet; the largest (capes, longcoats, weapons) want 1024×1024. Roughly: ~30k sheets × ~30 KB avg PNG = ~900 MB per canonical version. The PRD's revised "500 MB–1 GB" estimate stands.

### 2.4 PNG encoder choice (determinism)

**Decision: vendor a frozen fork of Go 1.21's `image/png` under `atlas/pngenc/`.**

The PRD requires byte-identical output across Go versions because the canonical baseline dump's SHA-256 must match across re-extractions. Go 1.22 changed `image/png`'s filter heuristic, which would silently invalidate all baselines on Go upgrade. Options considered:

| Option | Verdict |
|---|---|
| Use stdlib `image/png`, pin Go minor version | Forces every team member's local toolchain to the pinned version. Fragile. |
| Use `golang.org/x/image/png` (none ships) | n/a |
| Use a third-party encoder (e.g., `pngcrush` bindings) | External binary dep, harder to audit. |
| **Vendor frozen `image/png`** | Self-contained, deterministic regardless of host Go version. ~600 LOC. **Chosen.** |

The vendored encoder uses the same canonical filter heuristic as Go 1.21 — `paeth` filter for RGBA inputs with no sub-line heuristic, `none` for grayscale; CRC-32 from stdlib (deterministic by spec); zlib compression level fixed at `BestCompression` (9). A "encode twice, byte-compare" test runs in CI.

The Go minor version is still pinned in `services/atlas-data/Dockerfile` (PRD §4.1) but this is now defense-in-depth, not load-bearing. The vendored encoder is the source of determinism.

### 2.5 JSON serialization (key-sorted)

Both `manifest/encode.go` and `maplayout/encode.go` use a small key-sorting wrapper around `encoding/json`:

```go
// Pseudocode — actual impl uses a custom Encoder to avoid double-allocation.
func Marshal(v any) ([]byte, error) {
    raw, err := json.Marshal(v)              // unsorted map iteration
    if err != nil { return nil, err }
    var canonical any
    if err := json.Unmarshal(raw, &canonical); err != nil { return nil, err }
    return jsonsorted.Marshal(canonical)     // sorted map keys recursively
}
```

A faster single-pass implementation is possible (custom encoder walks reflect.Value), but the two-pass version is 30 LOC, obviously correct, and well below the wall-clock cost of the atlas pack itself. Optimize if profiles show it.

### 2.6 Determinism test

A CI-only test under `atlas/pack_test.go` builds a deterministic fixture (a 200-sprite set with varied sizes), runs `Pack()` twice, and asserts byte-identical sheet PNG and byte-identical manifest JSON. The test runs on every PR touching `libs/atlas-wz/`. A failing run means an implementer accidentally introduced map-iteration order, time-based seeds, or `runtime.NumCPU` parallelism.

---

## 3. `atlas-data` ingest design

### 3.1 MODE dispatch

The same compiled binary serves three modes. `main.go` reads `MODE` and selects a top-level controller:

```go
switch os.Getenv("MODE") {
case "rest":   restmode.Run(ctx, deps)        // HTTP only; creates Jobs
case "ingest": ingestmode.Run(ctx, deps)      // workers only; no HTTP
case "all":    allmode.Run(ctx, deps)         // HTTP + in-proc workers
default:       allmode.Run(ctx, deps)         // default for local dev
}
```

The three packages live under `services/atlas-data/atlas.com/data/runtime/{rest,ingest,all}`. They share a `workers.Run(ctx, params)` function that does the actual fan-out — the differentiator is only how `workers.Run` is invoked.

**Worker fan-out shape.** Inside `workers.Run`:

```go
g, ctx := errgroup.WithContext(ctx)
sem := semaphore.NewWeighted(maxParallel)         // env INGEST_MAX_PARALLEL, default 4
for _, archive := range archives {                // listed from MinIO atlas-wz prefix
    archive := archive
    g.Go(func() error {
        sem.Acquire(ctx, 1); defer sem.Release(1)
        return processArchive(ctx, archive, params)
    })
}
return g.Wait()
```

Each `processArchive` downloads its `.wz` to `/scratch/<archive>.wz` (an `emptyDir` in k8s, `os.TempDir` in compose), parses via `libs/atlas-wz`, runs the per-archive domain logic (item, mob, map, etc.), writes outputs to MinIO + Postgres, deletes the scratch file, and updates Redis progress. The progress hash carries one field per archive name; values transition `pending → running → done | error:<msg>`.

### 3.2 `MODE=rest` Job creation

The REST handler for `POST /api/data/process` does:

1. Resolve `<scope-key>` from query + headers; reject if `scope=shared` without operator header.
2. Acquire Redis lock `atlas-data:ingest:<scope-key>:<region>:<version>` with TTL=10 min. If held, list k8s Jobs matching the label selector `atlas-data-ingest=true,scope=<scope-key>,version=<version>`; if a Job exists, return `202` with that Job's name. If no Job exists but the lock is held — the previous holder is in a brief transient state; return `409 Conflict` with `Retry-After: 5`.
3. Render the Job from the template ConfigMap (`atlas-data-ingest-job-template`), substituting:
   - `metadata.name`: `atlas-data-ingest-<short-uuid>` (8-char base32 random suffix; collision-safe within `ttlSecondsAfterFinished` window).
   - `metadata.labels`: `atlas-data-ingest=true`, `scope=<scope-key>`, `version=<major>.<minor>`, `region=<region>`, `tenant=<tenantId or "shared">`.
   - `spec.template.spec.containers[0].env`: append `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`, `SCOPE`, `traceparent`.
4. Call k8s `Create(Job)`. Retry once with 1s backoff on `5xx`/timeout. On second failure, release the Redis lock and surface the error per PRD §4.4 failure-mode table.
5. Return `202 Accepted { "jobName": ..., "scope": ..., "version": ... }`.

**Why Jobs over CronJob-style or in-process goroutines:** the Job model gives us pod-level resource limits, ttl-based cleanup, separate scaling for REST vs. heavy workers, and a clean failure surface (Job `.status.conditions[Failed]` is observable). The alternative we considered — keep workers in the REST pod (matches `MODE=all`) — was rejected because a single ingest can pin 4–8 GB of memory for 10+ minutes; co-locating with REST means the REST pod must always be sized for the ingest peak, which wastes capacity 99% of the time.

### 3.3 Job tracking and recovery

REST's `GET /api/data/process` reads progress from Redis only. The k8s API is supplementary — used to know that a Job *exists* and to detect terminal states.

**Restart recovery.** On REST startup, the controller runs:

```go
jobs := k8sClient.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{
    LabelSelector: "atlas-data-ingest=true",
})
for _, j := range jobs.Items {
    if isActive(j) {
        // Re-register the (scope, region, version) → job-name mapping
        // in a small in-memory map. The Redis lock is still held by the
        // running Job's progress writes (each milestone refreshes TTL).
    }
}
```

If a Job exists but the Redis lock has expired (e.g., the Job pod was OOM-killed mid-run with no chance to write a final state), the controller deletes the orphan Job at startup and frees the lock. This handles the rare "REST died → Job died too → both restart" sequence.

**Watchdog.** A background goroutine in REST runs every 30s. For each active Job, it reads the Redis hash's `updatedAt` field. If `updatedAt` is older than `INGEST_WATCHDOG_TIMEOUT_SECONDS` (default 300), it:

1. Records the stuck state in Redis (`worker:<all>: error:watchdog-timeout`).
2. Deletes the Job (`propagationPolicy: Background`).
3. Frees the Redis lock.

`GET /api/data/process` then returns the stuck state with a clear error to the UI. The 5-min default is conservative — the slowest worker (Map.wz layer extraction) completes in well under 2 min on the v83 set.

### 3.4 `MODE=ingest` startup

The Job pod's entrypoint:

```go
func main() {
    // Restore traceparent from env so this span links to REST's request span.
    propagator.Extract(ctx, env-carrier)
    params := readParamsFromEnv()           // TENANT_ID, REGION, MAJOR/MINOR, SCOPE
    params.Logger = logger.NewJSONLogger()  // matches REST's log format
    if err := workers.Run(ctx, params); err != nil {
        params.Logger.Error("ingest failed", "err", err)
        os.Exit(1)
    }
    os.Exit(0)
}
```

The HTTP server **does not start** in this mode. There is no `/api/data/*` listener; the pod is a one-shot worker. Progress flows through Redis only.

### 3.5 Baseline publish (binary COPY)

`POST /api/data/baseline/publish` is operator-gated. The handler runs in REST (in `MODE=rest`) or in-process (in `MODE=all`); it does not spawn a Job because the work is I/O-bound (read from Postgres, write to MinIO) and bounded (~150 MB). The flow:

1. Acquire a separate Redis lock `atlas-data:baseline:publish:<region>:<version>` so two concurrent publishes for the same version cannot interleave.
2. Open a streaming MinIO PUT to `atlas-canonical/baseline/regions/<region>/versions/<version>/documents.dump`. The writer is wrapped in a `tar.Writer`.
3. Write `header.json` first:
   ```json
   { "schemaVersion": "v1", "region": "GMS", "majorVersion": 83,
     "minorVersion": 1, "tables": [...], "publishedAt": "..." }
   ```
4. For each of the 6 tables, in a fixed order, stream `COPY (SELECT * FROM <table> WHERE tenant_id = '00000000-...') TO STDOUT (FORMAT binary)` into a `tar` entry `<table>.binary`. The stream is sha256-hashed in flight.
5. Close the tar; finalize the MinIO PUT.
6. PUT the sidecar `documents.dump.sha256` with the hex hash.
7. Release the lock.

**Why per-table binary COPY, not `pg_dump`:** `pg_dump --format=custom` does not natively filter by column value. We would have to dump-then-rewrite, doubling the bytes. Streaming `COPY (SELECT ...)` is what Postgres calls a "binary protocol dump" — the same wire format `pg_dump` uses internally — but composed by us with the WHERE clause baked in. Restoring reverses the same wire format via `COPY <table> FROM STDIN (FORMAT binary)`, which is the standard fast path.

**Determinism.** The PRD requires a re-run to produce identical bytes for unchanged inputs. `COPY` is deterministic given a deterministic row order; we add `ORDER BY id` to every `SELECT` (each table has an `id PK`). The tar archive uses a fixed-mtime header (`ModTime: time.Unix(0, 0)`) so the wrapper bytes are also reproducible.

### 3.6 Baseline restore (binary COPY with tenant rewrite)

`POST /api/data/baseline/restore`:

1. Acquire Redis lock `atlas-data:ingest:<scope-key>:<region>:<version>` (same lock as ingest — restore and ingest cannot race for the same tenant).
2. Fetch `documents.dump` and `documents.dump.sha256` from MinIO. Compute the dump's sha256 in flight; mismatch → `422 Unprocessable Entity`.
3. Read `header.json`. If `schemaVersion != current`, → `422`.
4. For each table in order:
   - `BEGIN`.
   - `DELETE FROM <table> WHERE tenant_id = '<target>'`.
   - Stream the binary payload through a **tenant-id rewriter** into `COPY <table> FROM STDIN (FORMAT binary)`.
   - `COMMIT`.
5. After all 6 tables: `ANALYZE` each table.
6. UPSERT `tenant_baselines` row.
7. Release the lock.

**Tenant rewriter.** Postgres binary COPY format is row-prefixed by a `int16` field count, then each field is `int32` length + payload. UUIDs are 16-byte fixed payloads. The rewriter:

- Parses the COPY-binary header (11-byte signature + flags + extension area length).
- Reads each row's field count, then walks fields. For the known `tenant_id` field index (precomputed from the schema), it replaces the 16-byte payload with the target UUID's bytes. All other fields pass through unchanged.
- Streams the rewritten bytes into the `COPY FROM` server connection.

The `tenant_id` column position per table is hardcoded against `header.json.schemaVersion`. The schema-version fingerprint check (PRD §6.1) ensures the position cannot drift without bumping the version. A unit test serializes a known row, rewrites it, restores it into a test table, and asserts the round trip.

**Why not parse + re-emit as INSERT statements:** for ~150 MB of binary payload, parsing into Go structs and re-emitting INSERT would be 5–10× slower and lose the COPY-protocol parallelism advantage. The streaming rewriter touches only the 16 bytes per row that need to change.

### 3.7 Tenant purge (`DELETE /api/data/tenants/<id>`)

Direct handler in REST (no Job). The Postgres operations run in a single transaction; the MinIO operations are best-effort with a retry loop. Logs `{ tenantId, postgresRows, minioBytes, durations }`. The canonical UUID is rejected at the handler entry. The same Redis lock that protects ingest is acquired so cleanup cannot race with an in-flight ingest.

A future cron sweeper for orphaned MinIO prefixes is out of scope; the cleanup logs the residual keys on best-effort failure so an operator can sweep manually.

### 3.8 Worker → Postgres + MinIO mapping

Each domain worker (item, mob, map, npc, skill, reactor, quest, string, etc.) consumes one `.wz` archive and produces:

| Archive | Postgres writes | MinIO writes |
|---|---|---|
| Item.wz | `documents` (item rows) + item_string_search_index, item-icon icons table | per-item `icon.png` |
| Mob.wz | `documents` (mob rows) + monster_search_index | per-mob `icon.png` (from Mob.img stand frame) |
| Npc.wz | `documents` (npc rows) + npc_search_index | per-npc `icon.png` |
| Reactor.wz | `documents` (reactor rows) + reactor_search_index | per-reactor `icon.png` |
| Skill.wz | `documents` (skill rows) | per-skill `icon.png` |
| Quest.wz | `documents` (quest rows) | — |
| String.wz | populates all 5 search-index tables | — |
| Map.wz | `documents` (map rows) + map_search_index | per-map `minimap.png`, `layers/*.png`, `layout.json` |
| Character.wz | — | per-`(partClass, id)` `<id>.png` + `<id>.json` (sprite atlas + manifest) |
| UI.wz | — | per-`(category=world-icon)` `icon.png` for world tiles |

Workers are independent and write through their own DB connections; the worker pool size and DB pool size are tuned together (`INGEST_MAX_PARALLEL=4`, DB pool=16 in ingest mode).

### 3.9 Kafka emit-side preservation

After all workers complete, the REST handler (in `MODE=rest` watching the Job complete; in `MODE=all` immediately) publishes a `DATA_UPDATED` event to the existing `DATA_UPDATED` Kafka topic. The topic's downstream consumers (atlas-channel, atlas-character-factory, etc.) are unchanged. **The PRD explicitly notes this and §4.4 calls out that an implementer must not delete the emit-side producer when removing the consumer.**

Implementation note for plan: keep the producer-side wiring in `services/atlas-data/atlas.com/data/data/kafka.go` and audit consumers across the codebase before the cutover PR. Subscribers found:

```
$ grep -r "DATA_UPDATED" services/ | grep -v test
services/atlas-channel/.../data_invalidate.go
services/atlas-character-factory/.../data_invalidate.go
services/atlas-maps/.../data_invalidate.go
```

These stay subscribed.

---

## 4. `atlas-renders` service design

### 4.1 Module layout

```
services/atlas-renders/atlas.com/renders/
  main.go
  go.mod                                  // module atlas-renders (short, per memory)
  Dockerfile                              // lists libs/atlas-wz in all 4 locations
  character/
    handler.go handler_test.go            // GET /api/wz/character/render/...
    composite.go                          // image-compositing logic (atlas-aware)
    hash.go                               // loadout hash (preserve task-043 algorithm)
  mapr/
    handler.go                            // GET /api/wz/map/render/...
    composite.go                          // map layer blitting
  storage/
    minio.go                              // GET atlas/layer/manifest, PUT renders
    lru.go                                // per-pod atlas + manifest cache
    scope.go                              // per-tenant→shared probe + scope LRU
  rest/
    server.go                             // HTTP server + middleware
  metrics/
    metrics.go                            // Prometheus metric registration
```

The compositors call into `libs/atlas-wz/manifest` and `libs/atlas-wz/maplayout` for type unmarshaling, but never into `wz/`, `crypto/`, `canvas/`, `atlas/`, `mapimage/`, or `icons/`. The CI lint check enforces this.

### 4.2 Compositing pipeline (character)

```
request → parseLoadout(headers + query) → hashLoadout (preserve task-043)
       → probeRender(MinIO atlas-renders/<tenant>/<region>/<version>/character/<hash>.png)
         on HIT: stream MinIO body to client; return.
         on MISS:
       → for each layer (body, head, hair, face, equips...):
           resolveScope(tenant, region, version, partClass)   // LRU; first probe HEAD
           fetchAtlas(scope, partClass, id)                   // LRU
           fetchManifest(scope, partClass, id)                // LRU
           selectFrame(manifest, stance, frame)
           blitWithAnchors(canvas, sprite, manifest.anchors)
       → encode PNG (stdlib png — output, not the deterministic-baseline path)
       → write to client + best-effort PUT to atlas-renders bucket
```

The compositor preserves task-043's existing logic verbatim; only the data source changes. The "blit with anchors" code already exists in `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/` and gets ported.

### 4.3 LRU caches

Two LRU instances per pod, both backed by the same `hashicorp/golang-lru/v2` implementation:

| Cache | Key | Value | Size (env) | Avg entry | Working set |
|---|---|---|---|---|---|
| Atlas+manifest | `(scope, region, version, partClass, id)` | `(png []byte, manifest Manifest)` | `ATLAS_LRU_SIZE=256` | ~200 KB | ~50 MB |
| Map layers | `(scope, region, version, mapId)` | `(layers map[id][]byte, layout Layout)` | `MAP_LRU_SIZE=64` | ~1.5 MB | ~100 MB |
| Scope | `(tenant, region, version, partClass)` | `"shared"` or `"tenant/<id>"` | `SCOPE_LRU_SIZE=1024` | ~40 B | <100 KB |

Total bounded working set: ~150 MB headroom under the PRD's 256 MB target.

**Cache miss path = HEAD probe.** On first miss for a `(tenant, region, version, partClass)`, the storage layer issues a HEAD against `atlas-assets/tenants/<tenantId>/regions/<region>/versions/<version>/atlases/<partClass>/`. If 200 → scope is `tenants/<tenantId>`. If 404 → scope is `shared`. The result is cached.

We could optimize by HEAD-ing the specific `<id>` rather than the directory, but MinIO does not support directory-level HEADs cheaply. Instead, the first asset fetch is a `GET tenants/<tenantId>/.../<id>.png`; on 404, retry against `shared/`. We then cache the *scope decision per part class* (not per id) because every PRD-supported scenario has whole-part-class overrides, not per-item overrides. A future per-id override scenario would require a per-id scope LRU instead; we document the limit but do not preemptively support it.

### 4.4 Render PUT failure (best-effort)

```go
buf := composite(...)
io.Copy(responseWriter, bytes.NewReader(buf))   // user-facing response first
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := storage.PutRender(ctx, tenant, region, version, hash, buf); err != nil {
        log.Warn("render put failed", "err", err)
        metrics.MinioPutFailures.Inc()
    }
}()
```

The client is never blocked on the cache-write. A failed PUT means the next identical request re-composites, which is fine — composite time is hundreds of ms, well below user-perceptible latency.

### 4.5 Map render handler

`GET /api/wz/map/render/<tenant>/<region>/<version>/<mapId>/<kind>.png`:

- `kind=minimap`: 302-redirect to `/api/assets/<tenant>/<region>/<version>/map/<mapId>/minimap.png` so atlas-ingress serves it directly. We considered streaming from MinIO inside atlas-renders, but the 302 keeps the cache headers and bandwidth path consistent with all other static assets.
- `kind=render`: composite-on-miss. Probe `atlas-renders/tenants/<tenantId>/.../map/<mapId>/render.png`. On hit: stream. On miss: fetch layers + layout from `atlas-assets` (per-tenant→shared fallback), blit using the z-order in `layout.zmap` (carried over from `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/`), encode PNG, stream to client, best-effort PUT.

Map composite hot loop is single-threaded per request (image.Image blits are CPU-bound but fast at typical map sizes). Per-request memory peaks at the largest layer (~2 MB for the biggest backgrounds). The pod's overall memory ceiling is dominated by the LRU caches, not concurrent renders.

### 4.6 HPA metric — **decision: KEDA-backed Prometheus scaler on request rate**

The PRD defers this; the cluster's existing Argo + Prometheus + KEDA stack (verified by `kubectl get crd | grep keda` on the bee cluster — KEDA is installed) supports `prometheus`-typed `ScaledObject`s out of the box.

**HPA metric: `sum(rate(atlas_renders_requests_total[1m])) by (pod)`.**

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata: { name: atlas-renders }
spec:
  scaleTargetRef: { name: atlas-renders }
  minReplicaCount: 2
  maxReplicaCount: 8
  pollingInterval: 30
  cooldownPeriod: 180
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus.observability.svc.cluster.local:9090
      metricName: atlas_renders_rps_per_pod
      query: sum(rate(atlas_renders_requests_total[1m])) / count(kube_pod_info{pod=~"atlas-renders-.*",phase="Running"})
      threshold: "10"        // scale up at 10 req/s/pod sustained
```

We rejected the CPU+memory fallback because character renders are network-bound (waiting on MinIO GETs); CPU utilization sits at 30–40% even at saturation. By the time CPU crossed 70%, MinIO GETs would be the queue, and a CPU HPA wouldn't fire. KEDA's request-rate trigger reacts to the right signal.

**Fallback if KEDA is unavailable in a future environment.** A vanilla HorizontalPodAutoscaler with `type: ContainerResource` on both CPU and memory, thresholds 70% and 75% respectively. Documented in the manifest as a commented-out alternative; not deployed.

### 4.7 Tracing

Each request span has attributes:
- `atlas.tenant_id`, `atlas.region`, `atlas.version`
- `atlas.scope` (`shared` or `tenants/<id>`)
- `atlas.cache_hit` (`render_hit`, `render_miss`)
- `atlas.parts_fetched` (count of MinIO GETs this request triggered)

MinIO operations (`storage.PutRender`, `storage.GetAtlas`, etc.) emit child spans with `peer.service=minio` so Jaeger groups them.

---

## 5. atlas-ingress (nginx) design

### 5.1 routes.conf structure

The route file is reorganized so the asset/render block is self-contained, with a clear precedence:

```nginx
# atlas-renders upstream (defined once at the top).
upstream atlas_renders { server atlas-renders:8080; }
upstream atlas_minio   { server minio:9000; }

# 1. Character render (most specific; must match before the generic assets block).
location ~ ^/api/assets/(?<tenant>[^/]+)/(?<region>[^/]+)/(?<v>[0-9]+\.[0-9]+)/character/(?<hash>[a-f0-9]{32,64})\.png$ {
    set $major ""; set $minor "";
    if ($v ~ ^(?<maj>[0-9]+)\.(?<min>[0-9]+)$) { set $major $maj; set $minor $min; }
    proxy_set_header TENANT_ID     $tenant;
    proxy_set_header REGION        $region;
    proxy_set_header MAJOR_VERSION $major;
    proxy_set_header MINOR_VERSION $minor;
    proxy_pass http://atlas_renders/api/wz/character/render/$tenant/$region/$v/$hash.png$is_args$args;
    add_header Cache-Control "public, max-age=86400, immutable" always;
}

# 2. Map render (atlas-renders composites on miss; MinIO serves cache hits).
location ~ ^/api/assets/(?<t>[^/]+)/(?<r>[^/]+)/(?<v>[0-9]+\.[0-9]+)/map/(?<mapid>[0-9]+)/render\.png$ {
    rewrite ^ /atlas-renders/tenants/$t/regions/$r/versions/$v/map/$mapid/render.png break;
    proxy_intercept_errors on;
    error_page 404 = @maprender_miss;
    proxy_pass http://atlas_minio;
    add_header Cache-Control "public, max-age=86400, immutable" always;
}
location @maprender_miss {
    proxy_pass http://atlas_renders/api/wz/map/render/$t/$r/$v/$mapid/render.png;
    add_header Cache-Control "public, max-age=86400, immutable" always;
}

# 3. Generic assets (icons, atlases, minimap, layers, layout) — per-tenant → shared fallback.
location ~ ^/api/assets/(?<t>[^/]+)/(?<r>[^/]+)/(?<v>[0-9]+\.[0-9]+)/(?<rest>.+)$ {
    rewrite ^ /atlas-assets/tenants/$t/regions/$r/versions/$v/$rest break;
    proxy_intercept_errors on;
    error_page 404 = @shared_fallback;
    proxy_pass http://atlas_minio;
    add_header Cache-Control "public, max-age=86400" always;
}
location @shared_fallback {
    rewrite ^ /atlas-assets/shared/regions/$r/versions/$v/$rest break;
    proxy_pass http://atlas_minio;
    add_header Cache-Control "public, max-age=86400" always;
}

# 4. WZ upload route (large body, no buffering).
location /api/data/wz {
    client_max_body_size 4G;
    proxy_request_buffering off;
    proxy_pass http://atlas_data;
}
```

The PRD's `error_page 404 = @canonical` shape is implemented as `error_page 404 = @shared_fallback` — keeping the scope-name "shared" consistent. Named captures propagate into the named-location block (nginx behavior).

### 5.2 Why nginx not a sidecar / proxy service

We considered a small Go service that does the fallback in code (cleaner conditional logic, easier to test). Rejected: every asset request would pay an extra hop (UI → ingress → fallback-svc → MinIO), and the fallback logic is the kind of declarative routing nginx is good at. A regression CI test (see §8) gives us the test surface without inventing a new service.

### 5.3 Cache-Control headers

`add_header` only applies when the upstream did not set its own header. MinIO does not set `Cache-Control` on anonymous GETs, so the directive lands as expected. The `always` flag ensures the header is also added to non-2xx responses (e.g., on a 304 from `If-Modified-Since`).

### 5.4 Body-size and buffering

The PRD's `/api/wz/*` `client_max_body_size 4G; proxy_request_buffering off;` carries to the new `/api/data/wz` location. The shared `routes.conf` file already has a default `client_max_body_size 16M`; the per-location override is required.

### 5.5 Service-worker cache invalidation — **decision: content-hashed URL for renders; version-pinned URL for icons; SW cache versioning bump for atlas-pack changes**

The PRD §9 flags this as open. Three flavors of asset have three different invalidation needs:

| Asset | URL today | Invalidation lever |
|---|---|---|
| Character render | `/api/assets/<t>/<r>/<v>/character/<hash>.png` | URL already content-hashed via `<hash>` (loadout digest). No SW change needed; a new loadout = new hash = new URL. |
| Icon | `/api/assets/<t>/<r>/<v>/<cat>/<id>/icon.png` | URL carries `<v>`. Bumping the canonical version (a new ingest) gives a new URL; SW cache for the old version naturally falls out. **For same-version re-extraction (rare, but possible during cutover), increment the SW `CACHE_NAME` constant in `public/sw-character-cache.js` as part of the cutover PR.** |
| Map render | `/api/assets/<t>/<r>/<v>/map/<mapid>/render.png` | Same as icon — bound to `<v>`. Re-extraction within same version = bump SW cache name. |

The cutover PR includes a one-line bump in `public/sw-character-cache.js`:

```js
- const CACHE_NAME = "atlas-character-render-v1";
+ const CACHE_NAME = "atlas-character-render-v2-task071";
```

This evicts any pre-cutover client caches at first page load post-merge. Future ingest cycles within the same canonical version do not need a SW bump because the asset bytes are deterministic per `libs/atlas-wz` §2.4.

If determinism ever fails (the canonical baseline produces different bytes for unchanged inputs), the SW would serve stale bytes. The "pack twice, byte-compare" CI test prevents this.

---

## 6. MinIO bucket, IAM, and init Job

### 6.1 Buckets

Four buckets, identical to PRD §4.2. The init Job (§6.3) creates them idempotently.

### 6.2 IAM identities and policies

Three identities, scoped per PRD:

```json
// atlas-data-policy.json
{ "Version": "2012-10-17", "Statement": [
  { "Effect": "Allow", "Action": ["s3:PutObject","s3:GetObject","s3:DeleteObject","s3:ListBucket"],
    "Resource": ["arn:aws:s3:::atlas-wz/*","arn:aws:s3:::atlas-wz"] },
  { "Effect": "Allow", "Action": ["s3:GetObject","s3:PutObject","s3:ListBucket"],
    "Resource": ["arn:aws:s3:::atlas-canonical/baseline/*","arn:aws:s3:::atlas-canonical"] }
] }

// atlas-data-ingest-policy.json
{ "Version": "2012-10-17", "Statement": [
  { "Effect": "Allow", "Action": ["s3:PutObject","s3:GetObject","s3:DeleteObject","s3:ListBucket"],
    "Resource": ["arn:aws:s3:::atlas-wz/*","arn:aws:s3:::atlas-wz",
                 "arn:aws:s3:::atlas-assets/*","arn:aws:s3:::atlas-assets"] }
] }

// atlas-renders-policy.json
{ "Version": "2012-10-17", "Statement": [
  { "Effect": "Allow", "Action": ["s3:GetObject","s3:ListBucket"],
    "Resource": ["arn:aws:s3:::atlas-assets/*","arn:aws:s3:::atlas-assets"] },
  { "Effect": "Allow", "Action": ["s3:PutObject","s3:GetObject","s3:ListBucket"],
    "Resource": ["arn:aws:s3:::atlas-renders/*","arn:aws:s3:::atlas-renders"] }
] }
```

The `atlas-data-ingest` identity has cross-tenant write access to `atlas-assets/*` (and `atlas-wz/*`). Per PRD §8.2, this is the documented v1 trust boundary; the Job container is trusted to honor its env-var tenant_id when computing keys. A future per-tenant-identity model is a separate task.

### 6.3 atlas-minio-init Job — **decision: PreSync hook + kubernetes-replicator**

The PRD offers two ways to give the init Job the root creds it needs to provision other identities: mirror the `minio-root-creds` Secret from the `minio` namespace via `kubernetes-replicator`, or use a `SealedSecret`.

**Decision: kubernetes-replicator.** Verified installed via `kubectl get crd | grep replicator` (the `mittwald/kubernetes-replicator` operator is already deployed in the cluster for similar cross-namespace secret needs). The cluster's pattern is already this; `SealedSecret` would introduce a controller we don't otherwise use.

The Secret in the `minio` namespace is annotated:

```yaml
metadata:
  annotations:
    replicator.v1.mittwald.de/replication-allowed: "true"
    replicator.v1.mittwald.de/replication-allowed-namespaces: "atlas"
```

The Secret in the atlas namespace is annotated:

```yaml
metadata:
  annotations:
    replicator.v1.mittwald.de/replicate-from: "minio/minio-root-creds"
```

The init Job mounts `atlas/minio-root-creds` (the replicated copy).

**Sync wave vs. PreSync hook.** The PRD says PreSync; we additionally specify sync wave annotations as a defense-in-depth so the Job runs at wave `-2` even outside the PreSync mechanism:

```yaml
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
    argocd.argoproj.io/sync-wave: "-2"
```

The Job uses `mc` (MinIO client) inside a small Alpine image:

```bash
#!/bin/sh
set -e
mc alias set minio http://minio.minio.svc.cluster.local:9000 "$ROOT_USER" "$ROOT_PASSWORD"
for b in atlas-wz atlas-assets atlas-renders atlas-canonical; do
  mc mb --ignore-existing minio/$b
done
mc anonymous set download minio/atlas-assets
mc anonymous set download minio/atlas-renders
mc anonymous set download minio/atlas-canonical
# atlas-wz stays private (no policy set).

# Idempotent user creation
for user in atlas-data atlas-data-ingest atlas-renders; do
  if ! mc admin user info minio "$user" > /dev/null 2>&1; then
    KEY=$(openssl rand -hex 16)
    SECRET=$(openssl rand -hex 32)
    mc admin user add minio "$user" "$SECRET"
    # ... write to a temp file for kubectl create secret ...
  fi
done

# Policy upsert (always re-apply, picks up policy drift)
for p in atlas-data-policy atlas-data-ingest-policy atlas-renders-policy; do
  mc admin policy create minio "$p" /policies/$p.json || mc admin policy attach minio "$p" --user "${p%-policy}"
done

# Patch the Kubernetes Secret atlas-minio-credentials (created once; updated only when new keys generated).
kubectl create secret generic atlas-minio-credentials \
  --from-literal=atlas-data-access-key=... \
  --from-literal=atlas-data-secret-key=... \
  --dry-run=client -o yaml | kubectl apply -f -
```

The Secret is created only if it does not already exist (or if a new user was provisioned, indicating first run). Subsequent runs do not rotate keys; rotation is a separate, deliberate operator task.

### 6.4 MinIO single-drive durability — **decision: accept current topology; document recovery**

The PRD §9 asks whether to add erasure-coded MinIO. We do not, because:

- Recovery story remains "re-extract from raw `.wz`," which is a known-good path (it's what we do today, slow but reliable).
- Erasure-coding requires multi-drive MinIO, which requires multi-PVC, which conflicts with the storage policy that motivated the move off RWX.
- The canonical baseline dump is small (~150 MB) and trivially re-uploadable from any operator workstation; loss of the `atlas-canonical` bucket is not catastrophic.

We document the recovery procedure in `docs/runbooks/wz-ingest.md`: "If MinIO is lost, re-publish from any operator workstation with WZ files via SetupPage → scope=shared → Process → Publish Baseline."

### 6.5 docker-compose MinIO health gating — **decision: `depends_on: service_completed_successfully` chain**

compose v2.17+ supports `service_completed_successfully` as a `depends_on` condition. We use it:

```yaml
services:
  minio:
    image: minio/minio:RELEASE.2026-01-15T01-30-12Z   # pinned tag, matches k3s
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 5s
      timeout: 5s
      retries: 5
  minio-init:
    image: minio/mc:RELEASE.2026-01-15T01-30-12Z
    depends_on:
      minio: { condition: service_healthy }
    command: ["sh", "/init/init.sh"]
    restart: "no"
  atlas-data:
    depends_on:
      minio:      { condition: service_healthy }
      minio-init: { condition: service_completed_successfully }
      postgres:   { condition: service_healthy }
  atlas-renders:
    depends_on:
      minio:      { condition: service_healthy }
      minio-init: { condition: service_completed_successfully }
```

The `minio-init` container exits 0 after creating buckets and applying policies; atlas-data and atlas-renders block until that completes. No race conditions.

### 6.6 MinIO image pin

**Decision: `RELEASE.2026-01-15T01-30-12Z`** (current stable as of design-task date; final tag is the latest stable at cutover-PR open time). Both `~/source/k3s/bee/minio.yml` and `deploy/compose/docker-compose.core.yml` reference the literal tag. Bumps happen via a deliberate PR with smoke-test verification, never `:latest`.

---

## 7. Cutover strategy

### 7.1 PR-env-as-cutover

The cutover PR is itself opened, its PR env is brought up end-to-end, and the canonical baseline is published from that env before merge. No `:bootstrap-canonical.sh` script exists; the operator runs through SetupPage in the PR env's UI. This avoids the cold-cluster chicken-and-egg.

Concrete sequence (from PRD §10 smoke-test list):

1. Open the cutover PR. `atlas-pr-bootstrap` brings up a fresh env. `BOOTSTRAP_MODE=baseline` is default but on first cluster-life there is no baseline; the bootstrap script detects this (HEAD `atlas-canonical/baseline/regions/<region>/versions/<v>/documents.dump`) and **automatically falls back to `BOOTSTRAP_MODE=full` for the first run**.
2. Operator opens SetupPage, selects Canonical scope, uploads WZ zip, clicks Process, clicks Publish Baseline. ~10–15 min one-time.
3. Operator switches to a fresh tenant context, clicks Restore Canonical Baseline. < 60 s.
4. Smoke-test list (PRD §10) is exercised end-to-end.
5. PR merges. Subsequent PR envs hit the just-published baseline.

### 7.2 Failure mode when no baseline exists — **decision: auto-fallback to `full` mode in bootstrap, with a one-line warning log**

The PRD §9 asks whether to block bootstrap with a clear error or auto-fall-back. We pick auto-fall-back because:

- The first PR env after the cutover PR opens is exactly the case where no baseline exists yet; blocking would prevent the cutover from happening.
- A clear log line (`baseline not yet published; falling back to BOOTSTRAP_MODE=full`) is enough to surface the situation to an operator inspecting the PR env's boot logs.
- The bootstrap script's existing 10-minute timeout still applies; if `full` mode also fails, the PR env stays broken visibly.

The bootstrap logic:

```bash
if curl -fsI "$ATLAS_INGRESS/api/assets/canonical/.../documents.dump.sha256"; then
  BOOTSTRAP_MODE=baseline
else
  echo "WARN: no canonical baseline found; falling back to BOOTSTRAP_MODE=full"
  BOOTSTRAP_MODE=full
fi
```

### 7.3 Rolling-deploy strategy for cutover (k8s)

The PRD specifies drain-old-then-new for atlas-data, because mixing old (PVC-mounted, XML-emitting) and new (MinIO-only) pods corrupts ingest if a partial extraction runs across both. Implementation:

```yaml
# atlas-data Deployment
spec:
  strategy:
    type: Recreate    # drain ALL old pods before any new pod starts
```

This is an explicit downtime window for atlas-data (a few seconds in practice; the new pod boots in <10 s). Other services (atlas-channel, atlas-character-factory, etc.) keep running; they cache data heavily and tolerate a brief atlas-data outage.

`atlas-renders` is a green-field deployment, no migration concern.

### 7.4 Pre-cutover audit

Before merging, the cutover PR must show passing:

- `go test -race ./...` clean in every changed module (`libs/atlas-wz`, `atlas-data`, `atlas-renders`, `atlas-ui`).
- `go vet ./...` clean.
- `go build ./...` clean for every changed service.
- `docker build -f services/atlas-data/Dockerfile .` AND `docker build -f services/atlas-renders/Dockerfile .` from the worktree root.
- The CI ingress regression test (§8.3).
- The pack-twice byte-identity test on `libs/atlas-wz/atlas`.
- The render SSIM test (≥ 0.995) on the documented loadout fixture set.

The audit subagent (backend-guidelines-reviewer + frontend-guidelines-reviewer + plan-adherence-reviewer) runs against the cutover PR before opening it.

---

## 8. Testing strategy

### 8.1 libs/atlas-wz

- Per-property-type unit tests on `wz/property`.
- Per-pixel-format unit tests on `canvas/` against fixture canvases.
- `atlas/pack_test.go` — pack-twice byte-identity assertion on a 200-sprite deterministic fixture.
- `atlas/pngenc/encode_test.go` — encode-twice byte-identity on a few hand-built `image.RGBA`s.
- `manifest/encode_test.go` — JSON output key-sorted across random map iteration orders.

### 8.2 atlas-data

- Testcontainers integration test: starts MinIO + Postgres, runs `PATCH /api/data/wz` → `POST /api/data/process` against a small fixture (a hand-built `Item.wz` with 3 items + a `Character.wz` with 1 hair part). Asserts rows in `documents`, atlas PNGs in MinIO, and `tenant_baselines` row written on baseline restore.
- Baseline publish/restore integration: publish → restore into a fresh tenant fixture → row-count parity assertion. Hash-mismatch failure test: corrupt the dump, assert `422`.
- `tenant-id rewriter` unit test: round-trip a hand-built COPY-binary payload through the rewriter.
- `MODE` mode-switch test: launch with each MODE env, assert HTTP listener + worker behavior matches the table in §3.1.

### 8.3 atlas-ingress regex regression

A new CI step under `deploy/shared/test/`:

```bash
# Spin up nginx with routes.conf + fake upstreams that record incoming Host+Path.
docker run -d --name ingress-test -v $PWD/routes.conf:/etc/nginx/conf.d/routes.conf nginx:alpine
docker run -d --name minio-stub  ... # returns 200 for /atlas-assets/tenants/... 404 for shared
docker run -d --name renders-stub ... # logs incoming paths

# Hit a fixed URL set:
curl -s ingress-test/api/assets/T1/GMS/83.1/character/abc...png      # → renders-stub
curl -s ingress-test/api/assets/T1/GMS/83.1/item/2000000/icon.png    # → minio-stub tenant prefix
curl -s ingress-test/api/assets/T1/GMS/83.1/item/9999999/icon.png    # → minio-stub shared prefix (after 404)
curl -s ingress-test/api/assets/T1/GMS/83.1/map/100000000/render.png # → minio-stub then renders-stub
curl -s ingress-test/api/assets/T1/GMS/83.1/map/100000000/minimap.png # → minio-stub tenant

# Assert upstream logs match expectation.
```

The fixed URL set covers every distinct routing decision in the regex. A regression that, e.g., made the character regex accidentally also match minimap URLs would fail this test.

### 8.4 atlas-renders

- Testcontainers MinIO + fixture atlases + manifests. Render a documented character against the fixture, assert SSIM ≥ 0.995 vs. a frozen `expected.png` checked in.
- MinIO-unavailable test: drop the testcontainer mid-test, assert `503 Retry-After`.
- PUT-failure test: configure MinIO read-only, assert render still streams, `atlas_renders_minio_put_failures_total` increments.
- Scope-fallback test: tenant-prefix returns 404, shared returns 200, assert the render resolves and the scope LRU caches the decision (second request makes no HEAD call).
- Subpackage-import lint: `go list -deps ./services/atlas-renders/...` excludes `libs/atlas-wz/{wz,crypto,canvas,atlas,mapimage,icons}`.

### 8.5 atlas-ui

- Vitest + React Testing Library on `SetupPage.tsx`: scope toggle switches both upload + process queries; Restore row visible only when `documentCount == 0`; Publish CTA visible only after scope=shared + ingest complete.
- E2E smoke test (Playwright on PR env) runs through the upload → process → publish flow against a fixture WZ zip.

### 8.6 PR-env baseline smoke

A compose-level smoke test under `atlas-pr-bootstrap/test/`:

```bash
docker compose up -d
# Seed canonical baseline (one-time setup in the test fixture).
bash scripts/bootstrap.sh BOOTSTRAP_MODE=baseline
# Assert documentCount > 0 within 60 s.
test "$(curl -s ...api/data/status | jq .data.attributes.documentCount)" -gt 0
```

---

## 9. Open questions — resolved

| PRD §9 question | Resolution |
|---|---|
| HPA metric for atlas-renders | KEDA Prometheus scaler on `atlas_renders_requests_total` rate per pod (§4.6). |
| atlas-minio-init authentication | kubernetes-replicator mirroring `minio-root-creds` (§6.3). |
| MinIO single-drive durability | Accept; document recovery (re-publish from operator workstation). No erasure-coding (§6.4). |
| Baseline-restore atomicity | Per-table transaction, multi-transaction overall; idempotent via DELETE-then-COPY. Resolved in PRD §6.1. |
| Map.wz render coverage | Lazy. Already in PRD §4.7. |
| Sprite atlas packing algorithm | MaxRects-BSSF with fixed pre-sort, vendored ~400-LOC implementation (§2.3). |
| Atlas size budget | Validated against PRD Appendix B sketch: ~500 MB–1 GB per canonical version. Confirmed at design time; revisit in plan if v83 numbers come in 2× over. |
| Manifest schema versioning policy | Additive-only forward-compatible; breaking change requires v2 + re-extraction (§2.5 + PRD §6.2). |
| Service-worker cache invalidation | Content-hash for renders, version-pin in URL for icons/maps, one-time `CACHE_NAME` bump in the cutover PR (§5.5). |
| docker-compose MinIO healthcheck timing | `depends_on: service_completed_successfully` on `minio-init` (§6.5). |
| PR-env failure when no baseline | Auto-fall-back to `BOOTSTRAP_MODE=full` with a WARN log (§7.2). |

---

## 10. Alternatives considered and rejected

| Alternative | Why rejected |
|---|---|
| Keep `.img.xml` as the boundary between extractor and atlas-data | The PVC is the problem; the XML intermediate is what made the PVC look unavoidable. Removing both is the whole point. |
| Keep atlas-wz-extractor as a separate Job-runner service | Adds a service boundary that nobody crosses except the Job invocation. Two services to deploy, two Dockerfiles to maintain, no benefit. |
| Use Kafka to dispatch ingest work to a worker pool | PRD v4 explicitly retired this. Kafka adds at-least-once semantics complexity (idempotence keys, dedup) for a workflow that is naturally one-shot. k8s Job is a better fit. |
| In-process workers in REST pod in `MODE=rest` (skip Job) | Sizes REST for the ingest peak. Wastes 99% of capacity. Bigger blast radius if a worker OOMs (kills the REST too). |
| Render service uses `libs/atlas-wz/atlas` to re-pack on the fly | Defeats the determinism guarantee (sub-pixel differences between pre-baked and live-packed atlases). Doubles render cost. Renderer should never see WZ. |
| Per-tenant MinIO IAM identities (close the v1 trust boundary) | Provisioning at tenant-creation time is a new tenant-lifecycle hook. Future task; not blocking v1 because today's RWX model has the same cross-tenant trust assumption. |
| Postgres `pg_dump --format=custom` for baseline | Cannot filter by column value natively; would dump-then-rewrite, doubling bytes. Binary COPY (SELECT) is the same wire format with the WHERE built in. |
| Insert/Update rewrite during restore instead of binary COPY rewrite | ~10× slower for 150 MB payloads. The 16-byte-per-row rewriter is the right level. |
| Replace nginx fallback with a Go fallback proxy service | New service, new hop, same logic; nginx `error_page 404 = @location` is purpose-built for this. |
| CPU+memory HPA for atlas-renders | Renders are network-bound. By the time CPU crossed threshold, request latency would already be regressed. KEDA on request rate fires earlier. |
| Erasure-coded multi-drive MinIO | Conflicts with the storage-policy motivation for moving off RWX. Recovery path "re-publish from workstation" is acceptable for the canonical baseline's small size. |
| SealedSecret for root creds | Cluster already runs kubernetes-replicator. Adding a second controller for one Secret is unjustified. |
| Block PR bootstrap when no canonical baseline exists | Blocks the cutover-PR's own first env. Auto-fallback with WARN log is friendlier and doesn't hide failure (the next run uses baseline cleanly). |
| Cluster-level MinIO ingress (direct browser→MinIO) | A CDN decision; deferred. Today's atlas-ingress hop is fine. |

---

## 11. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Vendored PNG encoder has a latent bug producing wrong-bytes for some pixel inputs | Low | High (silently wrong renders) | Per-pixel-format unit tests + SSIM ≥ 0.995 vs. pre-cutover renders on every PR touching the encoder. |
| Atlas packing fails on an outsized sprite (>4096px) | Low | Medium (missing equip sprite) | Pack call logs + ingest continues; renderer falls back to placeholder. Tracked in metrics. |
| Job creation racing with pod startup → REST issues `POST` before its ServiceAccount can `create Job` | Low | Medium (transient 500s on first request) | k8s ServiceAccount + RoleBinding deploy at sync wave -1; atlas-data at wave 0. The Job-template ConfigMap also at wave 0. By the time REST accepts traffic, RBAC is in place. |
| MinIO PUT throughput is the new ingest bottleneck | Medium | Low (slower ingest, not failure) | INGEST_MAX_PARALLEL is env-configurable; bump to 8 if profiles show queue depth. |
| `kubernetes-replicator` not actually installed in some target cluster | Low | Medium (init Job can't auth) | The init Job's missing-secret failure is explicit; doc the fallback to SealedSecret in the runbook. |
| Cutover PR's PR env can't publish the baseline (PR env's atlas-data writes fail to write to atlas-canonical) | Medium | High (cutover blocked) | The cutover PR env is the same shape as a normal PR env, with the additional permission to write `atlas-canonical/baseline/*`. The init Job's `atlas-data-policy` already includes that PUT. Verified in §6.2. |
| The 256-entry atlas LRU thrashes for a tenant with >256 distinct part classes × IDs | Medium | Medium (cold-cache latency on rare loadouts) | LRU size is env-configurable; default sized for ~95% hit ratio on typical loadouts per task-043 benchmarks. Bump if metrics show thrash. |
| Watchdog kills a slow-but-legitimate ingest | Low | Medium (false ingest failure) | 5-min default is conservative; configurable env. The slowest worker today completes in ~90s. |
| Map render path's 302-to-MinIO breaks an old client cache | Low | Low (one extra request on first hit post-cutover) | SW cache name bump in cutover PR evicts old entries. |
| Argo PreSync hook fires on every sync, racing with running workloads | Low | Low (idempotent ops) | `mc` operations are idempotent (`--ignore-existing`, policy upsert, conditional user create). |
| Tenant-id rewriter misaligns column position after a schema migration | Low | High (data corruption) | Schema-version fingerprint CI check (PRD §6.1) fails any migration that changes the dumped tables without bumping `SCHEMA_FINGERPRINT_V<N>` and `header.json.schemaVersion`. |

---

## 12. What plan.md should decompose this into

Plan-phase task buckets (not the final breakdown; suggested order):

1. **libs/atlas-wz scaffolding**: module setup, port `wz/`, `crypto/`, `canvas/`, `mapimage/`, `icons/` from atlas-wz-extractor verbatim; add `manifest/`, `maplayout/`. Tests carry over.
2. **libs/atlas-wz determinism layer**: vendor PNG encoder, write `atlas/pack.go`, add pack-twice + encode-twice tests.
3. **atlas-data ingest scaffolding**: add `MODE` switch, port domain workers off `.img.xml` to direct WZ via `libs/atlas-wz`. Wire MinIO Go SDK. Keep existing endpoints functional in parallel during dev.
4. **atlas-data k8s Job machinery**: Job template ConfigMap, RBAC manifests, Job-create + watchdog code, label-selector recovery.
5. **atlas-data baseline endpoints**: `publish`, `restore`, tenant-id rewriter, `tenant_baselines` migration, schema-version fingerprint check, ANALYZE step.
6. **atlas-data tenant purge endpoint**: `DELETE /api/data/tenants/<id>`.
7. **atlas-renders service**: scaffolding, character handler (port task-043 compositor), map handler, MinIO storage layer, LRU caches, metrics, tracing.
8. **atlas-renders HPA + KEDA manifest.**
9. **atlas-minio-init Job**: manifest, init.sh, replicator annotation on the source Secret.
10. **atlas-ingress routes.conf** rewrite + regression test.
11. **atlas-ui SetupPage**: scope toggle, restore row, publish CTA, delete extraction row + hooks/services.
12. **docker-compose** rewrite: add minio, minio-init, atlas-renders; remove extractor, assets; reshape atlas-data env.
13. **atlas-pr-bootstrap**: `BOOTSTRAP_MODE` switch, `cleanup.sh` tenant-purge step, auto-fallback to `full` on missing baseline.
14. **k8s overlays**: drop PVCs, drop extractor + assets Deployments, add atlas-renders + atlas-minio-init.
15. **Cutover PR env exercise**: open the cutover PR, bring up env, publish baseline from SetupPage, run smoke-test list, merge.
16. **Delete `services/atlas-wz-extractor/` and `services/atlas-assets/`** in the cutover PR (last commit on the branch, so the diff is reviewable).
17. **Runbook documentation**: `docs/runbooks/wz-ingest.md`, update `docs/runbooks/ephemeral-pr-deployments.md`.

The plan must ensure each commit passes `go test -race`, `go vet`, `go build`, and `docker build -f` for every touched service. The libs/atlas-wz Dockerfile-pattern step is the easiest to miss; flag it explicitly per CLAUDE.md.

---

## 13. Out-of-scope (recorded explicitly)

- WZ format upgrades or game versions beyond v83.
- Per-tenant MinIO IAM identities (future task; PRD §8.2).
- MinIO erasure-coded topology (PRD §9).
- CDN for `/api/assets/*` (PRD §2 non-goal).
- Animation interpolation / GIF / multi-frame renders (PRD §2 non-goal).
- Pet, mount, cash-equipment compositing (PRD §2 non-goal).
- A standalone `bootstrap-canonical.sh` script — folded into SetupPage UI flow (PRD §4.9).
- MinIO root-cred rotation flow (PRD §2 non-goal; documented manual procedure).
- Lifecycle policy (`mc ilm`) on `atlas-renders` bucket for long-lived tenants (PRD §9 / Appendix B; tracked).
- Cron sweeper for orphaned MinIO prefixes after best-effort cleanup failures (PRD §4.4a; tracked).
