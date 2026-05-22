# Lazy map render — task-071 add-on

> Authored 2026-05-22. Add-on to task-071, ships on the same branch (not a separate PR).
> Companion to `finish-line.md`: this is the Bug C portion, kept as a focused spec
> rather than a full `/spec-task` PRD because the scope is bounded and we will
> not ship it independently.

## Why

PRD §4.7 said: *"Map render is **lazy** (atlas-renders composes on first request).
Halves ingest wall-clock and avoids materializing unused maps."*

What actually shipped only deferred the **final** layer stack to render time
(`atlas-renders/mapr/composite.go:27-74` — a pure `draw.Over` over pre-rendered
layer PNGs). The expensive sprite-resolution + per-layer compositing pass still
ran eagerly in the ingest Map worker (`workers/mapw.go:67-86` calling
`libs/atlas-wz/mapimage.ExtractLayers`) for all ~4774 maps including the ~99%
that no user ever views. That is what blew the 30-minute Watchdog cutoff (see
`finish-line.md` Bug B) and produced the silently-truncated map worker.

After Bug B fixes the heartbeat, the ingest would *complete* — but it would
still take ~30-45 min for nothing. This add-on finishes the laziness so map
ingest drops to a few minutes.

## What

| Component | Before | After |
|---|---|---|
| **Ingest Map worker** | walks every `.img`; calls `ExtractLayers`; per-layer composite; PNG-encode; PUT layer PNGs to MinIO; write `layout.json`; extract + PUT `minimap.png` | walks every `.img`; calls `ExtractLayout` (new, metadata-only); write `layout.json`; extract + PUT `minimap.png`. **No per-layer composite, no layer PNG uploads.** |
| **atlas-renders** | reads `layout.json` + N pre-rendered `layers/layer-N.png` from MinIO; stacks them with `draw.Over` | reads `layout.json` from MinIO; **fetches Map.wz** from MinIO bucket `atlas-wz` to local emptyDir (once per pod startup per `(scope, region, version)`); calls `ExtractLayers` on the requested `.img`; stacks the result; caches final composite in `atlas-renders` MinIO bucket |
| **MinIO layout** | `atlas-assets/<scope>/.../map/<id>/{layout.json, minimap.png, layers/layer-N.png}` | `atlas-assets/<scope>/.../map/<id>/{layout.json, minimap.png}` — **no `layers/` subdir** |
| **`atlas-renders` deployment** | network-bound | adds an `emptyDir` for cached WZ files (~2 GiB); memory request bumped to absorb working-set sprite parses |

## How — code level

1. **`libs/atlas-wz/mapimage/layers.go`** — split:
   - `ExtractLayout(img *wz.Image) (maplayout.Layout, error)` — metadata only.
     Pulls bounds, footholds, portals, NPCs, zmap, and the per-layer `maplayout.Layer{ID,Name,Z,Source}` records (still recorded so the on-disk layout
     schema is unchanged for forward-compat). No `compositeLayer` call.
   - `ExtractLayers(idx *Index, img *wz.Image) ([]LayerOutput, maplayout.Layout, error)` — unchanged shape; keep for atlas-renders' render-time use.
   - Side effect: `ExtractLayout` doesn't need `*Index` since it never resolves sprites — accepts only `*wz.Image`.

2. **`services/atlas-data/.../data/workers/mapw.go`** — replace the
   `ExtractLayers` call + layer upload loop with `ExtractLayout`. Keep
   `ExtractMinimap` + the minimap upload. Drop the `idx := mapimage.NewIndex(file)`
   line — no longer needed. Remove `extractLayersErrs` from the summary log;
   replace with `layoutsErrs` for parity.

3. **`services/atlas-renders/.../storage/wzcache.go`** (new) — a
   `(scope, region, version, archive)` → `*wz.File` cache backed by a
   per-key `sync.Once` to avoid duplicate downloads under concurrent requests.
   Download path: stream MinIO bucket `atlas-wz` → `<localDir>/<archive>` →
   `wz.Open`. File handle stays open for the pod's lifetime; the parser is
   already lazy on canvas data (read via positional `ReadAt` from the open
   handle), so memory cost stays small (5-10 MB directory tree initially,
   asymptote ~100-200 MB).

4. **`services/atlas-renders/.../storage/maplayout.go`** — strip the layer
   download loop. `GetMap` returns only the parsed `maplayout.Layout`. (The
   `MapEntry.Layers` field becomes vestigial; drop or rename.)

5. **`services/atlas-renders/.../mapr/composite.go`** + `handler.go` — on
   cache miss inside the `atlas-renders` bucket, the new render flow is:
   1. Resolve scope (unchanged).
   2. `GetMap` to load `layout.json`.
   3. Open the cached `Map.wz` for this `(scope, region, version)` (downloads
      on first request).
   4. Index into the per-mapId image via the existing
      `mapimage.NewIndex(file).Maps()[mapID]` lookup.
   5. `mapimage.ExtractLayers` to produce the layer images.
   6. `draw.Over` the layers in `layout.zmap` order — same as today.
   7. PNG-encode, PUT to `atlas-renders` MinIO bucket, stream to client.

6. **`services/atlas-renders/.../storage/config.go`** — add
   `BucketWZ` field (default `"atlas-wz"`). `MINIO_BUCKET_WZ` env override.
   `services/atlas-renders/.../main.go` plumbs it via `ConfigFromEnv`.

7. **`deploy/k8s/base/atlas-renders.yaml`** — add the WZ-cache emptyDir and
   bump the memory request. Sizing rationale: Map.wz alone is 606 MiB
   measured; allow headroom for character archives (Character.wz 178 MiB,
   Effect.wz 60 MiB) so a single pod can serve both render paths. Pick
   2 Gi limit on the emptyDir.

## How — verification

After both A and B land **and** this work is applied, on a fresh PR env:

1. Trigger ingest. Map worker should finish in ~2-5 min, log a
   `"map assets: scanned=… layouts=… minimaps=… layoutsErrs=…"` line, no
   `layersWritten` accounting. Confirm via `kubectl logs <ingest-pod>`.
2. `mc ls --recursive .../atlas-assets/.../map/100000000/` — expect
   `layout.json` + `minimap.png`. **No** `layers/` subdir.
3. `wget http://atlas-ingress/api/assets/.../map/100000000/render.png` —
   expect 200 OK PNG. First request: takes a couple of seconds (the Map.wz
   fetch dominates the first call across all maps; subsequent maps share the
   cached `*wz.File`).
4. Same request again — expect immediate response (now cached in
   `atlas-renders` MinIO bucket).
5. `kubectl exec atlas-renders-… -- ls -lh /scratch/wz/` (or wherever the
   emptyDir mounts) — expect `Map.wz` present, ~606 MiB.
6. `kubectl top pod atlas-renders-…` — memory should asymptote in the
   200-400 MB range after a handful of requests across distinct maps.

## What this add-on does NOT do

- **No warm-set / pre-render of popular maps after ingest.** Deferred — keep
  the change focused on the laziness mechanic. Adding a warm-set is a
  one-screenful follow-up if the cold first-render latency proves
  user-annoying.
- **No DATA_UPDATED cache-invalidation in atlas-renders.** A fresh ingest
  produces a new `(scope, region, version)` only if the version bumps.
  Within the same version, ingest republishes are destructive overwrites
  to the same MinIO keys — atlas-renders' parsed `*wz.File` continues to
  reference the same byte offsets in its local cached copy. A pod restart
  picks up changes; for our PR-env demo this is sufficient. A real
  invalidation hook can be added later.
- **No per-`*wz.Image` property cache eviction.** The existing parser holds
  parsed property trees in memory until file close. Working-set size in
  practice is bounded (see `finish-line.md` analysis); if it grows beyond
  expectation we can add an LRU on `*wz.Image.properties` later.
- **Character/Effect renders are unchanged.** Character render already has
  its own working path and that's not what this task is about.

## Failure modes & mitigations

| Failure mode | Mitigation |
|---|---|
| `Map.wz` MinIO key missing on first map render request | atlas-renders returns 500 with a clear error; user re-runs ingest |
| Two concurrent requests trigger duplicate Map.wz downloads | `sync.Once` per `(scope, region, version, archive)` cache key |
| `Map.wz` parse fails mid-pod-lifetime (corrupt download) | cache the error too; subsequent requests fail-fast until pod restart |
| atlas-renders pod restart drops the cached `*wz.File` | next request re-downloads (~10s on cluster network); acceptable |
| User flips between `(region, version)` tuples rapidly | each tuple gets its own cache entry; bounded by emptyDir size; in practice 1-2 tuples per env |
| Map worker now writes layout.json for maps where `ExtractLayout` fails | `extractLayoutsErrs` counter in the worker summary log; same observability as today's layer-extract errors |
