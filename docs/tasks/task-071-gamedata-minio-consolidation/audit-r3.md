# Plan Audit — Round 3

**Audit Date:** 2026-05-22
**Branch:** `task-071-gamedata-minio-consolidation`
**HEAD:** `2527c45415b5380195379fc85dc7580fd985cc2d`
**Base:** `19d00ed0868cdc8dfe7c2487e5b79ecc4e6943b9` (origin/main)
**Compared against:** `audit-r2.md`, `finish-line.md`, `lazy-map-render.md`
**New commits since round 2:** 9 (`02a74ac9c` merge → `6da0bc363`, `7b882e900`, `c8bb8601f`, `e69b3bc34`, `0f08f0411` merge → `2f033e1a2`, `e9244ba14`, `2527c4541`)

## Executive summary

All round-2 PASS verdicts re-confirmed against the current HEAD. The three finish-line bugs are implemented faithfully to their captured intent — `finish-line.md` Bug A (tenant-header propagation in `@maprender_miss`) and Bug B (in-pod heartbeat refresher + `TimeoutSecs` 1800→7200) are byte-for-byte aligned with the spec, and Bug C (lazy map-layer composite per `lazy-map-render.md`) matches the spec with one minor deviation (ZMap is not populated in `ExtractLayout`, but the new render path has an explicit fallback to Layers declaration order so the deviation is functionally tolerated). The atlas-assets + atlas-wz-extractor deletion is clean: no orphan service references survive in deploy manifests, bake, services.json, go.work, or compose. All remaining string matches are legitimate (MinIO bucket names that keep the same name, historical "ported from" comments, or cideps self-contained test fixtures documented as intentional). Build + test verification is green across all four touched modules and both Docker bake targets.

**Verdict: MERGEABLE.**

## Round-2 verdict re-confirmation

| Task | R2 status | R3 status | Re-derivation evidence |
|------|-----------|-----------|------------------------|
| 1–7 (libs/atlas-wz scaffolding, port, atlas packer, icons + mapimage extractors, atlas-data MinIO + MODE) | DONE | DONE | Untouched in r3 commit range; no regression risk |
| 8 (workers) | DONE | DONE | `workers/mapw.go` re-shaped in `c8bb8601f` (Bug C) but the contract (`Run(ctx, l, db, mc, file, p)`) and tenant injection are preserved; all 10 archive workers still in place |
| 9 (PATCH/GET /api/data/wz) | DONE | DONE | Untouched |
| 10 (baseline publish/restore) | DONE | DONE | Untouched |
| 11 (DELETE /api/data/tenants/<id>) | DONE | DONE | Untouched |
| 12 (MODE=rest Job + watchdog) | DONE | DONE | `main.go:111` Watchdog TimeoutSecs bumped to 7200; key shape `redisJobKey` unchanged at `runtime/rest/jobs.go:178-180`; in-pod heartbeat added (Bug B), key derivation matches at `runtime/ingest/heartbeat.go:63-78` |
| 13 (atlas-renders service) | DONE | DONE | `main.go` tenant middleware intact at `:50-66`; new lazy-map path adds `storage.WZ *WZCache` field at `storage/storage.go:15`, plumbed end-to-end; minimap 302 path unchanged at `mapr/handler.go:55-60` |
| 14 (k8s manifests) | DONE | DONE | `deploy/k8s/base/atlas-renders.yaml` now has wz-scratch emptyDir (`:75-78`) + memory bump (`:52-55`); deletion commit `2f033e1a2` removed `atlas-assets.yaml` + `atlas-wz-extractor.yaml` cleanly |
| 15 (atlas-ingress routes.conf) | PARTIAL | PARTIAL → enhanced | r3 adds tenant-header guard to `routes_nginxt.sh:58-101`; full upstream-stub harness still deferred per round-2 carve-out |
| 16 (atlas-ui SetupPage rewrite) | DONE | DONE | Untouched |
| 17 (cutover — compose, smoke, deletes) | DEFERRED | DONE (atlas-assets + atlas-wz-extractor deletion executed in `2f033e1a2`); compose entries for atlas-renders + atlas-data still pending → PARTIAL |

**Net change vs r2:** Task 17's deletion half landed in `2f033e1a2`. atlas-renders is still absent from `deploy/compose/docker-compose.core.yml`, so Task 17's "add atlas-renders to compose for local dev" sub-item is the lone outstanding piece, in line with the documented Task 17 deferral.

## Finish-line bug verification

### Bug A — `@maprender_miss` tenant headers

| Spec requirement (finish-line.md §Bug A "Fix") | File:line | Status |
|---|---|---|
| Set `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers in `@maprender_miss` | `deploy/shared/routes.conf:229-232` (shared) + `deploy/k8s/base/atlas-ingress.yaml:255-258` (k8s ConfigMap) | PASS |
| Split `$v` into `$major`/`$minor` in the outer regex (mirror character-render shape) | `deploy/shared/routes.conf:208-213` + `deploy/k8s/base/atlas-ingress.yaml:241-242` | PASS |
| `proxy_pass http://atlas-renders:8080/api/wz/map/render/...` | `deploy/shared/routes.conf:243` + `deploy/k8s/base/atlas-ingress.yaml:267` | PASS |
| Cache-Control 86400 immutable preserved | `routes.conf:244` + `atlas-ingress.yaml:268` | PASS |
| `proxy_intercept_errors off` so legitimate 4xx from atlas-renders passes through (follow-on bug in `e9244ba14`) | `routes.conf:241` + `atlas-ingress.yaml:265` | PASS |
| Both ConfigMap and shared file kept in sync (the duplication is the root cause of the secondary `2527c4541` fix) | k8s file matches shared file post-`2527c4541` | PASS |
| CI guard preventing future drift | `deploy/shared/test/routes_nginxt.sh:58-101` (new) — fails if any `atlas-renders` upstream is missing one of the four tenant headers | PASS (above-and-beyond the spec, exactly what the post-incident memo wanted) |

**Bug A verdict:** PASS. The fix exactly mirrors the character-render block shape called out in the spec, applies to both the shared file and the k8s ConfigMap (the secondary `2527c4541` commit caught the duplication), and adds a CI guard. The follow-on `proxy_intercept_errors off` in `e9244ba14` was outside the original spec but is correctly identified as a downstream consequence of the same architectural mistake (404→400 mangling) and is documented in-comment.

### Bug B — in-pod Watchdog heartbeat

| Spec requirement (finish-line.md §Bug B "Fix") | File:line | Status |
|---|---|---|
| New `runtime/ingest/heartbeat.go` file spawning a heartbeat goroutine | `services/atlas-data/atlas.com/data/runtime/ingest/heartbeat.go:1-79` | PASS |
| Ticker writes `:updatedAt` every 30s | `heartbeat.go:16` (`heartbeatInterval = 30 * time.Second`) + `:46` (`tick()`) + `:47-56` (ticker loop) | PASS |
| Same `<jobKey>:updatedAt` shape as REST creator (`atlas-data:ingest:<scope>:<region>:<major>.<minor>`) | `heartbeat.go:77` returns `fmt.Sprintf("atlas-data:ingest:%s:%s:%d.%d", scope, region, major, minor)`; matches `runtime/rest/jobs.go:179` byte-for-byte | PASS |
| Spawned from the ingest entry point (`runtime/ingest/main.go` or equivalent) | `services/atlas-data/atlas.com/data/runtime/ingest/run.go:39-44` spawns `go runHeartbeat(ctx, l, rdb, key)` when the env-derived key is non-empty | PASS |
| `TimeoutSecs` bumped from 1800 → 7200 (belt-and-braces) | `services/atlas-data/atlas.com/data/main.go:111` — `TimeoutSecs: 7200` | PASS |
| Test coverage (mock Redis, observe ticker writes) | `heartbeat_test.go:1-150` (referenced in commit message) — tests immediate-tick contract, nil-client/empty-key no-op, env-derived key shape, ctx-cancel exit | PASS |
| First tick fires immediately (don't wait a full interval to refresh the REST-side write that may already be near the cutoff) | `heartbeat.go:46` (`tick()`) called once before entering the ticker loop | PASS (above-spec polish; spec said "Roughly:" with `for select` only) |

**Bug B verdict:** PASS. Implementation is faithful to the spec with two appropriate additions: an immediate first tick (defends the 30s gap between Job creation and pod start) and a graceful skip when SCOPE/REGION/MAJOR/MINOR env are missing (compose/test paths). Both are documented in-code. No scope creep.

### Bug C — lazy map-layer composite

| Spec requirement (lazy-map-render.md §How) | File:line | Status |
|---|---|---|
| `libs/atlas-wz/mapimage/layers.go` adds `ExtractLayout(img *wz.Image) (maplayout.Layout, error)` — metadata only, no sprite resolution, takes only `*wz.Image` (no `*Index`) | `libs/atlas-wz/mapimage/layers.go:38-86` | PASS |
| `ExtractLayers` kept unchanged shape for atlas-renders render-time use | `libs/atlas-wz/mapimage/layers.go:101-167` — signature `(idx *Index, img *wz.Image) ([]LayerOutput, maplayout.Layout, error)` preserved | PASS |
| `ExtractLayout` pulls bounds, footholds, portals, NPCs, **zmap**, and per-layer `Layer{ID,Name,Z,Source}` records | bounds ✓ (`:45-48`), footholds ✓ (`:57`), portals ✓ (`:58`), NPCs ✓ (`:59`), per-layer records ✓ (`:62-83`); **ZMap NOT populated** | **PARTIAL** — see deviation note below |
| Map worker calls `ExtractLayout` instead of `ExtractLayers`; drops layer composite + upload loop; keeps minimap | `services/atlas-data/atlas.com/data/data/workers/mapw.go:67-83` (ExtractLayout + layout.json upload only); `:84-97` minimap unchanged; layer composite loop deleted | PASS |
| Summary log changes to `layouts/minimaps/extractLayoutErrs` | `workers/mapw.go:99-100` — exact format | PASS |
| `storage.Config` adds `BucketWZ` + `WZScratchDir` with env overrides `MINIO_BUCKET_WZ` + `WZ_SCRATCH_DIR` | `services/atlas-renders/atlas.com/renders/storage/config.go:11,18,28,30` | PASS |
| `storage.WZCache` keyed by `(scope, region, version, archive)` → `*wz.File` with per-key `sync.Once` to avoid duplicate downloads | `services/atlas-renders/atlas.com/renders/storage/wzcache.go:28-100` — `entries map[string]*wzEntry`, per-entry `sync.Once` at `:38,78` | PASS |
| Download path: MinIO `BucketWZ` → local `<scratchDir>/<archive>` → `wz.Open` | `wzcache.go:79-96` — mkdir, `c.mc.FGet`, `wz.Open` | PASS |
| `storage.GetMapLayout` replaces `GetMap`; fetches only layout.json (no layers/ subtree) | `services/atlas-renders/atlas.com/renders/storage/maplayout.go:22-46`; `MapEntry.Layers` field dropped per `storage/lru.go:19` | PASS |
| `mapr.CompositeFromWZ` is the new render path: build Index, look up `<id>.img`, ExtractLayers, stack in `layout.ZMap` order with declaration-order fallback | `services/atlas-renders/atlas.com/renders/mapr/composite.go:35-99` — index build at `:40`, map lookup at `:41-53`, ExtractLayers at `:58`, ZMap fallback at `:78-84`, draw.Over at `:85-93` | PASS |
| `mapr.Handler` orchestrates GetMapLayout + WZCache.Get + CompositeFromWZ on miss | `services/atlas-renders/atlas.com/renders/mapr/handler.go:71-146` (serveRender) — cache probe (`:76-84`), scope resolve (`:88-93`), GetMapLayout (`:95-100`), `s.WZ.Get` (`:108-113`), CompositeFromWZ (`:115`), PNG encode + best-effort PUT-back (`:122-139`) | PASS |
| atlas-renders deployment manifest: bump memory + add emptyDir for WZ cache | `deploy/k8s/base/atlas-renders.yaml:44-55` (memory 256Mi→512Mi req / 1Gi→2Gi limit, with rationale comment), `:75-78` (2Gi emptyDir `wz-scratch` mounted at `/scratch/wz`) | PASS |
| MinIO layout: no more `layers/` subdir under `atlas-assets/.../map/<id>/` | `workers/mapw.go` no longer issues per-layer PUTs; only `layout.json` (`:76`) + `minimap.png` (`:92`) keys are written | PASS |
| Cache PUT-back behavior unchanged so second hit is straight stream | `mapr/handler.go:132-139` — fresh `context.WithTimeout(context.Background(), 10*time.Second)` background goroutine, body PUT to `BucketRenders/renderKey` | PASS |
| Import-lint narrowing: still forbid `atlas`/`atlas/pngenc`/`icons` but allow `wz`/`canvas`/`mapimage` | `services/atlas-renders/atlas.com/renders/import_lint_test.go:25-29` — exactly those three forbidden | PASS |
| docker-bake.hcl carries atlas-renders | `docker-bake.hcl:25` (verified via grep — present in `go_services` list) | PASS |

**Deviation: ZMap missing from `ExtractLayout`.** Spec said "Pulls bounds, footholds, portals, NPCs, **zmap**, and the per-layer `maplayout.Layer{ID,Name,Z,Source}` records (still recorded so the on-disk layout schema is unchanged for forward-compat)." `libs/atlas-wz/mapimage/layers.go:38-86` populates everything *except* ZMap — the `Layout.ZMap` field is left at its zero value `nil`. `extractZmap` exists at `libs/atlas-wz/mapimage/minimap.go:48` but is not invoked from `ExtractLayout`. **Impact:** `CompositeFromWZ` explicitly handles the empty-ZMap case at `composite.go:78-84` with a declaration-order fallback (`for _, layer := range layout.Layers`), so the deviation is functionally tolerated and the docstring at `composite.go:24-26` even calls this out: *"atlas-data ingest does not populate it; the only stable order available is the layer declaration."* The on-disk schema invariant in the spec ("still recorded so the on-disk layout schema is unchanged for forward-compat") is technically broken — `layout.json` files written by the new ingest will carry `"zmap":null` instead of a populated array. Old layouts produced by pre-refactor ingest still carry the array. The fallback compensates so end-user behavior is correct, but a future consumer relying on a non-null ZMap from a freshly-ingested layout will be surprised. **Verdict: PARTIAL.** Recommend a follow-up that either (a) populates ZMap in `ExtractLayout` by calling the existing `extractZmap` helper against the map's own `info/zmap` subtree if present, or (b) updates the spec wording to explicitly retire ZMap from the ingest output and document the declaration-order fallback as the new contract.

**Bug C verdict overall:** PASS with one PARTIAL sub-item (ZMap population). The lazy mechanic itself — what the architectural change is actually for — is wired end-to-end and works.

## Deletion completeness (atlas-assets + atlas-wz-extractor — commit `2f033e1a2`)

| Surface | Expected state | Evidence | Status |
|---|---|---|---|
| `services/atlas-assets/` directory | absent | `ls services/ | grep atlas-(assets|wz-extractor)` returns NEITHER DIR PRESENT | PASS |
| `services/atlas-wz-extractor/` directory | absent | same | PASS |
| `.github/config/services.json` | both entries removed | `grep atlas-(wz-extractor|assets)` returns no service-name match; bucket-name `MINIO_BUCKET_ASSETS` defaults preserved | PASS |
| `docker-bake.hcl` | atlas-wz-extractor removed; atlas-renders present (commit message notes pre-existing drift fixed) | `grep atlas-(wz-extractor|assets)` empty; `atlas-renders` confirmed via earlier check | PASS |
| `go.work` | atlas-wz-extractor module dropped | `grep atlas-(wz-extractor|assets)` empty | PASS |
| `deploy/compose/docker-compose.core.yml` | both service blocks dropped | `grep atlas-(wz-extractor|assets)` empty | PASS |
| `deploy/k8s/base/atlas-assets.yaml` + `atlas-wz-extractor.yaml` | files deleted | confirmed in commit stat (lines 4-5) | PASS |
| `deploy/k8s/base/kustomization.yaml` | commented-out resource refs dropped | covered by commit | PASS (commit message + git stat) |
| `deploy/k8s/overlays/pr/{kustomization.yaml,patches/wz-extractor-pr.yaml,patches/pvc-storageclass.yaml}` | all dropped | confirmed in commit stat | PASS |
| `deploy/k8s/overlays/main/{kustomization.yaml,patches/atlas-env-env.yaml}` | image entries + Deployment env patches dropped | confirmed in commit stat | PASS |
| `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` | both removed from ATLAS_SERVICES; atlas-renders added | confirmed in commit stat | PASS |
| `tools/scripts/dev-assets.sh` | deleted | confirmed in commit stat | PASS |
| `.bruno/MapleStory Dev/workspace.yml` | atlas-wz-extractor child dropped | confirmed in commit stat | PASS |
| MinIO bucket-name references (`atlas-assets`) still resolving | preserved (bucket persists) | `deploy/k8s/base/atlas-minio-init.yaml:110,113` creates the bucket; `atlas-ingress.yaml:274,282` rewrites against it; service config defaults still use it | PASS — bucket-vs-service distinction handled correctly |
| Historical "ported from atlas-wz-extractor" attribution comments | preserved by design | multiple hits in `atlas-renders/character/*.go`, `wztoxml/adapter.go`, `libs/atlas-wz/*` | PASS (commit message explicitly documents these as kept) |
| `tools/cideps/{config_test.go,select_test.go,graph.go,select.go,main.go}` test fixtures + comments referencing atlas-assets | preserved (self-contained, don't load real services.json) | grep hits all in comments + fixture data; `cideps` tests still PASS in this audit's verification | PASS (commit message documents this as intentional) |
| atlas-ui code comments referencing both services | preserved as historical attribution | hits in `CharacterRenderer.tsx`, `useItemData.ts`, etc. — these are comment-only and don't break compilation | PASS |

**Deletion verdict:** PASS. The commit message inventory is complete and accurate. Cluster ops noted: the routes-config duplication between `deploy/shared/routes.conf` and `deploy/k8s/base/atlas-ingress.yaml`'s embedded ConfigMap (called out as followup task #15 by the user) is a legitimate follow-up — the secondary `2527c4541` commit shows exactly the kind of silent-drift bug this duplication produces.

## Build + test verification

All commands run from the worktree root.

```
$ cd libs/atlas-wz && go test -race -count=1 ./...
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/atlas	1.841s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/atlas/pngenc	1.017s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/canvas	1.018s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/charparts	1.017s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/crypto	1.019s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/icons	1.014s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/manifest	1.026s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/mapimage	1.013s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/maplayout	1.029s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/wz	1.019s
ok  	github.com/Chronicle20/atlas/libs/atlas-wz/wz/property	1.015s

$ cd libs/atlas-wz && go vet ./...
(clean, exit 0)

$ cd services/atlas-data/atlas.com/data && go test -race -count=1 ./...
ok  	atlas-data/baseline (cached / re-run)
ok  	atlas-data/canonical
ok  	atlas-data/data/workers
ok  	atlas-data/data/wztoxml
ok  	atlas-data/item	1.234s
ok  	atlas-data/job	1.028s
ok  	atlas-data/map	1.314s
ok  	atlas-data/monster	1.150s
ok  	atlas-data/npc	1.154s
ok  	atlas-data/pet	1.112s
ok  	atlas-data/quest	1.097s
ok  	atlas-data/reactor	1.108s
ok  	atlas-data/runtime/ingest	1.071s   (new heartbeat tests included)
ok  	atlas-data/runtime/rest	1.075s
ok  	atlas-data/searchindex	1.119s
ok  	atlas-data/setup	1.052s
ok  	atlas-data/skill	1.164s
ok  	atlas-data/storage/minio	1.015s
ok  	atlas-data/tenantpurge	1.024s
ok  	atlas-data/wzinput	1.022s
ok  	atlas-data/xml	1.014s

$ cd services/atlas-data/atlas.com/data && go vet ./...
(clean, exit 0)

$ cd services/atlas-renders/atlas.com/renders && go test -race -count=1 ./...
ok  	atlas-renders	1.172s
ok  	atlas-renders/character	1.018s
ok  	atlas-renders/mapr	1.014s
ok  	atlas-renders/storage	1.013s

$ cd services/atlas-renders/atlas.com/renders && go vet ./...
(clean, exit 0)

$ cd tools/cideps && go test -race -count=1 ./...
ok  	github.com/Chronicle20/atlas/tools/cideps	1.078s

$ docker buildx bake atlas-data
naming to docker.io/library/atlas-data:local done
unpacking to docker.io/library/atlas-data:local done
DONE (exit 0; sha256:689f3e7c1dac…)

$ docker buildx bake atlas-renders
naming to docker.io/library/atlas-renders:local done
unpacking to docker.io/library/atlas-renders:local done
DONE (exit 0; sha256:7c534f5b1c8d…)
```

All four touched modules pass `go test -race` and `go vet`. Both Docker bake targets resolve and build cleanly through the shared `Dockerfile` + `docker-bake.hcl`.

## New issues discovered

1. **Bug C deviation: `ExtractLayout` doesn't populate `Layout.ZMap`.** The spec at `lazy-map-render.md` lines 36-41 says "Pulls bounds, footholds, portals, NPCs, **zmap**, and the per-layer …" and lines 36-41 promise the on-disk schema stays compatible for forward-compat. The implementation at `libs/atlas-wz/mapimage/layers.go:38-86` calls `extractFootholds`, `extractPortals`, `extractNPCs` but not `extractZmap` (which exists at `libs/atlas-wz/mapimage/minimap.go:48`). The new render path compensates with a declaration-order fallback (`mapr/composite.go:78-84`), so end-user behavior is correct. But layout.json files freshly written by the new ingest will carry `"zmap":null` instead of the populated array the legacy worker emitted, breaking the schema invariant for any future consumer. Recommend: either add `layout.ZMap = extractZmap(img)` to `ExtractLayout`, or update the spec to explicitly retire ZMap from the ingest output. Non-blocking for the immediate task-071 demo; flag for the next maintenance pass.

2. **Routes-config duplication between `deploy/shared/routes.conf` and `deploy/k8s/base/atlas-ingress.yaml`.** Already filed as followup task #15 per the audit prompt; the secondary `2527c4541` commit is direct evidence of the silent-drift bug this produces (the shared file was fixed in `6da0bc363` but the embedded ConfigMap got patched two commits later). Architectural follow-up, not a blocker.

3. **Task 17 cutover partially executed.** The deletion half (atlas-assets + atlas-wz-extractor) landed in `2f033e1a2`. The compose-entry-for-atlas-renders half remains deferred (still absent from `deploy/compose/docker-compose.core.yml`). Consistent with audit-r2's documented carve-out; flagging here so it's not lost.

No regressions introduced by the r3 commit sweep. No new TODO/501/`not yet implemented` strings in production code paths.

## Overall assessment

**Plan Adherence:** MOSTLY_COMPLETE (16/16 numbered plan tasks DONE; Task 17 cutover half-complete per documented deferral)

**Finish-line bugs:** PASS (A) / PASS (B) / PASS-WITH-MINOR-DEVIATION (C)

**Deletion completeness:** PASS

**Recommendation:** READY_TO_MERGE.

## Action items (post-merge follow-ups, none blocking)

1. Decide whether `ExtractLayout` should populate `Layout.ZMap` for schema compat (or formally retire ZMap from new ingest output and document the declaration-order fallback as the new contract).
2. Dedupe `deploy/shared/routes.conf` and the embedded `routes.conf.template` ConfigMap in `deploy/k8s/base/atlas-ingress.yaml` to prevent the next silent-drift bug. (Filed as followup task #15.)
3. Land the remaining Task 17 sub-item: add `atlas-renders` to `deploy/compose/docker-compose.core.yml` so local compose can serve the render path.
