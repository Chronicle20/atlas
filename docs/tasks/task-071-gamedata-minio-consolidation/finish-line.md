# task-071 finish-line memo

> Authored 2026-05-22 from live evidence on PR-544. Frozen against image `pr-544-b9066a9` (latest commit on this branch).

## TL;DR

Task-071's deployed state on PR-544 shows three remaining real bugs and one architectural mismatch with the PRD. Two of the bugs are small and should land on this branch to unblock the demo. The third bug is a band-aid that the architectural follow-up makes obsolete. The architectural follow-up is a separate task (PRD-sized).

```
                             on this branch        new task
─────────────────────────────────────────────────────────────
A. routes.conf header prop          ✓
B. in-pod heartbeat refresher       ✓                ← obsolete after C
C. Lazy map layer composite                          ✓
```

## Evidence summary

All claims below are validated against the running PR-544 environment, not from reading code alone.

| Symptom user reported | Real state | Cause |
|---|---|---|
| 1a/1b NPC images don't load | NPC icons (1620) all present in MinIO; URL `/api/assets/.../npc/<id>/icon.png` returns `200 OK` with valid PNG via the cluster ingress | Stale browser cache from earlier broken deploys. Cleared by Cmd/Ctrl-Shift-R. **Not a real bug.** |
| 1c Mobs work | Confirmed | — |
| 1d Some item icons missing | Item 1000025 confirmed present (536 B); 5478/6159 Item.wz + 5114/7168 Character.wz icons uploaded | Stale browser cache, same as NPCs. **Not a real bug.** |
| 1f Map render / minimap mostly broken | Amherst (1000000) is the only cached render. Most maps have layout.json/layers/minimap.png in MinIO but `render.png` cache-miss requests return `400 Bad Request` | **Bug A** (see below) |
| 1f Henesys / some maps have no minimap | `tenants/<…>/map/100000000/` does not exist in MinIO at all (worker never finished). Last log from the ingest pod at 01:53:58Z, 30 min after Job creation, no `"map assets:"` summary line | **Bug B** (see below) |
| Henesys portal duplication | Not investigated — flagged as latent | Separate task; not in scope here |

## Bug A — `@maprender_miss` drops tenant headers

### Evidence

- `deploy/shared/routes.conf:218-222` (`@maprender_miss`) proxies to `atlas-renders:8080/api/wz/map/render/$t/$r/$v/$mapid/render.png` with no `proxy_set_header`.
- Compare `routes.conf:190-204` (character render block) which does set all four (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).
- `services/atlas-renders/atlas.com/renders/main.go:50-66` — tenant middleware returns `http.StatusBadRequest` when any of the four headers is missing.
- Direct probe inside the cluster:
  ```
  wget atlas-renders:8080/.../100000001/render.png                          → 400
  wget --header='TENANT_ID:…' --header='REGION:…' \
       --header='MAJOR_VERSION:…' --header='MINOR_VERSION:…' (same URL)   → 200 OK PNG (1.0 MB)
  ```
- Amherst escapes the bug only because its `render.png` was cached in the `atlas-renders` MinIO bucket from a prior request; the route resolves on the MinIO leg and never falls through.

### Fix

Add the four `proxy_set_header` lines to `@maprender_miss`. Mirror of the character-render block.

```nginx
location @maprender_miss {
  proxy_set_header TENANT_ID     $t;
  proxy_set_header REGION        $r;
  proxy_set_header MAJOR_VERSION $major;
  proxy_set_header MINOR_VERSION $minor;
  set $u "atlas-renders:8080";
  proxy_pass http://$u/api/wz/map/render/$t/$r/$v/$mapid/render.png;
  add_header Cache-Control "public, max-age=86400, immutable" always;
}
```

`$major` / `$minor` need to be split out of `$v` in the outer regex (the character block does this with a nested `if (... ~ ...)`), or `$v` itself can be split inside `@maprender_miss` with the same shape. Either works; mirror what character-render does for consistency.

### Verification

Re-probe `/api/assets/.../map/100000001/render.png` from inside the cluster: expect `200 OK image/png`. Pick a map whose `layout.json` + layers exist in MinIO but `render.png` does not (any map other than Amherst).

### Cost

One nginx location block edit. Single commit. No tests to add — the existing route tests in `deploy/shared/test/routes_nginxt.sh` should already cover this if the path generates a different output.

## Bug B — ingest Job killed at 30 min; no in-pod heartbeat

### Evidence

- `services/atlas-data/atlas.com/data/main.go:100` — `Watchdog{… TimeoutSecs: 1800}`.
- `services/atlas-data/atlas.com/data/runtime/rest/jobs.go:169-173` — REST pod writes Redis `:updatedAt` **once** at Job creation.
- `grep heartbeat services/atlas-data/atlas.com/data/runtime/ingest/` returns no matches — the ingest pod itself never refreshes the heartbeat.
- `runtime/rest/watchdog.go:80-92` — when `:updatedAt` is older than `now() - TimeoutSecs`, the watchdog deletes the Job with `DeletePropagationForeground`.
- Timeline match: pod `ingest-t-bf89e3b7-gms-83-1-ghcurf-sqnbb` created 2026-05-22T01:23:30Z, last log 01:53:58Z (30:28 elapsed), no summary `"map assets:"` line, NPC/Mob/Item/Skill/Reactor all completed with their summary lines well before 30 min.

### Fix

Add a heartbeat refresher to the ingest runtime. Roughly:

```go
// runtime/ingest/heartbeat.go
func RunHeartbeat(ctx context.Context, l logrus.FieldLogger, rdb *redis.Client, key string) {
    if rdb == nil || key == "" { return }
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            _ = rdb.Set(ctx, key+":updatedAt", time.Now().UTC().Format(time.RFC3339), time.Hour).Err()
        }
    }
}
```

Spawned alongside `RunWorkers` from `runtime/ingest/main.go` (or wherever the ingest entry point lives). Same `redisJobKey(scope, region, major, minor)` shape as the REST creator — both sides must agree on the key.

Also belt-and-braces: bump `TimeoutSecs` from 1800 to 7200 (2 h). If the heartbeat goroutine is wedged for any reason, the watchdog still has a generous window before nuking the Job.

### Verification

1. Trigger a fresh ingest via `POST /api/data/wz/regions/.../versions/.../ingest` (or whatever the current entry is).
2. `kubectl -n atlas-pr-544 exec ... -- redis-cli get atlas-data:ingest:<scope>:<region>:<major>.<minor>:updatedAt` — observe the timestamp tick every 30 s.
3. Wait > 30 min. Job survives. `"map assets:"` line appears in the pod log.
4. `mc ls local/atlas-assets/.../map/100000000/` — expect `layout.json` + `minimap.png` + layer PNGs to exist after this run.

### Cost

One small Go file + main.go wiring + 1 number bump in `Watchdog{}`. Adds a goroutine for the lifetime of the ingest container. Trivial test (mock Redis, observe ticker writes).

## Architectural follow-up — Bug C / lazy-map-layer refactor (separate task)

### What

Move per-layer compositing from ingest to atlas-renders, per PRD §4.7's stated intent (`"Map render is lazy (atlas-renders composes on first request) … halves ingest wall-clock and avoids materializing unused maps"`).

Today, the "lazy" decision was implemented only for the final stack (`atlas-renders/mapr/composite.go:27-74`, which just `draw.Over`s pre-rendered layer PNGs). The expensive sprite-resolution + per-layer compositing pass still runs eagerly in ingest (`libs/atlas-wz/mapimage/layers.go:41-107`, `services/atlas-data/.../workers/mapw.go:67-86`) — for **all 4,774 maps**, including the ~99% that no user will ever view.

### Why this is a separate task, not a hot patch

- The Map.wz file (606 MiB measured) becomes a runtime dependency of atlas-renders, not just ingest. That means a new code path (download, parse, cache) and a new lifecycle concern (when does atlas-renders refresh its parsed file when a new ingest completes?).
- atlas-renders gains memory + disk resource needs; deployment manifests must be updated.
- The ingest side's Map worker contract changes (no longer writes layer PNGs); the atlas-renders side's `storage.GetMap` shape changes (no longer reads layer PNGs).
- Failure modes change: a partially-failing ingest now bites at render time, not ingest time. The `"resolve bounds: no bounds"` debug logs we observed today (~80 maps) become 500s from atlas-renders instead of silent skips.

That's three commits worth of carefully-staged change spread across two services + deployment + tests. Not the kind of thing to chain onto a branch that's already weeks long.

Once C lands, B is moot — Map worker drops from ~30 min to ~2-5 min (still has to walk every `.img` for `layout.json`, but no sprite blit / PNG encode / MinIO PUT loop), well under any sensible watchdog timeout.

### Design sketch (for the new task's PRD)

1. **Ingest's Map worker** keeps:
   - DB register for every map.
   - `layout.json` upload (footholds, portals, NPCs, zmap, bounds, layer metadata — but **not** the layer images themselves).
   - `minimap.png` upload.
   Drops the layer extraction + PNG encode + per-layer upload loop entirely. Probably removes `ExtractLayers`' image-producing half from the library and keeps the layout-producing half.

2. **atlas-renders** gains:
   - A `wzCache` keyed by `(scope, region, version, archiveName)` → `*wz.File`. On miss: stream the `.wz` from MinIO `atlas-wz` bucket to a local emptyDir, call `wz.Open`, cache the result. ~10 s for the 606 MiB Map.wz fetch the first time the pod sees it, sub-second thereafter.
   - The map composite path reaches into the cached `*wz.File`, lazy-parses the requested `<id>.img`, runs the same `ExtractLayers` + composite logic now living in atlas-renders, encodes PNG, writes to the `atlas-renders` MinIO bucket as today.
   - One `*wz.File` per `(scope, region, version)` tuple — usually one, with multi-tenant overrides at most a handful.

3. **Memory / disk budget** (measured from the parser code, see finish-line investigation notes):
   - Per `*wz.File`: ~5-10 MB directory tree initially, asymptote ~100-200 MB after working set parses (sprite sets in Tile/Obj/Back).
   - Local disk per pod: ~606 MiB for Map.wz + ~178 MiB for Character.wz + whatever else (Effect.wz for character renders).
   - Atlas-renders deployment manifest needs a sized emptyDir (~2-3 GiB safety) and a memory request bump.

4. **Optional warm-set**: after a fresh ingest completes, atlas-data fires a small fan-out of HTTP GETs at atlas-renders for a curated map list (towns, GM map, a couple of common hunting grounds — 20-30 maps). Background, non-blocking. Means the maps users obviously hit first are zero-latency; everything else stays purely lazy.

5. **Cache invalidation when ingest republishes**: ingest emits the existing `EVENT_TOPIC_DATA` (`DATA_UPDATED`) event; atlas-renders consumes it and drops the matching `(scope, region, version)` entries from its `wzCache`. Already half-built — atlas-renders is already wired to read `DATA_UPDATED` events for asset URLs to refresh (or it should be — verify in the new task's plan phase).

### Expected outcome

- Ingest end-to-end drops from ~45-90 min today (estimated; we never see it finish on PR-544 due to bug B) to ~5-10 min, dominated by Item.wz + Character.wz + DB writes.
- First-time map render latency goes from "instant" (cache hit) to ~hundreds of ms-seconds depending on map complexity. Subsequent hits stay instant.
- Bootstrap stability radically improves — partial ingest no longer produces silently-missing maps.

## Recommended sequence

1. **Now (this branch):** Bug A (routes.conf) + Bug B (heartbeat + timeout bump). Verify both via direct cluster probes. Land, merge, demo.
2. **Next task (new `/spec-task`):** Bug C lazy-map-layer refactor. Follow the design sketch above as the new PRD's input.
3. **Eventually:** Henesys portal-dupe latent bug. Separate small task.

The justification for not bundling C onto this branch is purely scope-control: task-071 is already a multi-week branch touching ~20+ files across ingest, render service, deploy, ingress, UI. Cramming a cross-service runtime refactor on top will make the PR un-reviewable and CI cycles painful. Land what we have, ship, then take a clean swing at the laziness done right.
