# task-043 — Implementation context

Quick reference for executing the plan. Not a substitute for `design.md` / `prd.md`.

## What we're building

Replace the runtime dependency on `https://maplestory.io` with an in-cluster character render service powered by atlas-wz-extractor. The compositor reads Character.wz extracts already on `atlas-assets-pvc`, writes deterministic 96×128 (×resize) PNGs back to the PVC, and atlas-assets nginx serves cached PNGs directly. atlas-ui swaps its URL builder and retires the platform pixel-scan.

## Key locked decisions (from design.md §1.1)

- **D1** — single URL hits atlas-assets; nginx `try_files` falls back to atlas-wz-extractor on 404.
- **D2** — extraction-time emits worn-sprite PNGs + JSON sidecars; render path does only `image.Decode` + blit.
- **D3** — two-handed weapon classification uses `libs/atlas-constants/item.IsTwoHanded(...)` (helper to be added; no such function exists today — see Task 1.1).
- **D4** — no render-time dedup. Atomic temp+rename guarantees correctness; bursts produce duplicated work but never partial bytes.
- **D5** — render endpoint at `/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png` on atlas-wz-extractor. nginx fallback rewrites the captured path.
- **D6** — new `characterimage/` package, sibling to `mapimage/`, sharing `mapimage/blit.go` and `mapimage/decoder.go` primitives.
- **D7** — frontend hash uses `js-sha256` (sync, ~5 KB).
- **D8** — new `services/api/characterRender.service.ts`; `maplestory.service.ts` is deleted.
- **FR-23 departure** — no readiness gate. Cold-window 502s ride on React Query retry + the existing fallback avatar.

## Canvas geometry

- Native canvas: **96 × 128**. Body origin lands at canvas `(48, 96)` so the foot row is row **124** (4-pixel ground padding).
- `resize ∈ {1, 2, 3, 4}`. Default 2. Output dims = `(96 × resize) × (128 × resize)`.

## Loadout hash

- Canonical string: `tenant|region|major.minor|skin|hair|face|stance|frame|resize|sortedItemsCsv`.
- Hash = first **16 hex chars** of `SHA-256(canonical)`. (16 hex chars = 64 bits.)
- Equipment item ids are sorted ascending before joining (server normalizes regardless).
- Cross-language fixture file: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/testdata/loadout-hashes.json`. Both Go and TS test suites consume it.

## Slot filtering

Drop these equipment slot keys from the request before compositing (FR-9):

- Mount: `-18`, `-19`, `-20`
- Pet: `-14`, `-21`..`-30`
- Cash: `-101`..`-114`

## Stance / frame scope

Only these stances are extracted and accepted: `stand1`, `stand2`, `walk1`, `alert`, `jump`. Any other stance value → 400 `invalid-stance`. Frame validity is checked lazily against the body skin's per-stance directory; out-of-range → 400 `frame-out-of-range`.

## Skin color mapping (server-side, internal → WZ id)

| Internal | WZ id | Internal | WZ id |
|---|---|---|---|
| 0 | 2000 | 6 | 2009 |
| 1 | 2001 | 7 | 2010 |
| 2 | 2002 | 8 | 2011 |
| 3 | 2003 | 9 | 2012 |
| 4 | 2004 | 10 | 2013 |
| 5 | 2005 | | |

Body img path: `0000{wzSkin}.img` (female) or `0001{wzSkin}.img` (male). The render request only carries `skin` (internal 0–10); the gender choice comes from the loadout's matching body img — see `data-model.md`.

## Code map (existing files to reference)

| File | Why it matters |
|---|---|
| `services/atlas-wz-extractor/atlas.com/wz-extractor/image/extract.go` | The `case name == "character":` dispatch, where `extractCharacterParts` is added next to the existing `extractEquipmentIcons`. `findSub`, `findSubCanvas`, `normalizeId`, `writeCanvasPng` are reusable helpers. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go` | `runExtraction` — wipe character cache before WZ loop; new `case name == "base":` dispatched from `image.ExtractIcons` once we extend it. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go` | Pattern for `RegisterHandler(l)(si)` route registration. New `characterrender.InitResource` mirrors this. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/tenant_path.go` | `TenantPath(t)` returns `{tenantId}/{region}/{major.minor}` — reuse for resolving assets root. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go` | Add a second `AddRouteInitializer(...)` call wiring `characterrender`. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/blit.go` | `blit(canvas, src, ex, ey, ox, oy, world, w, h)` reused for compositor blit. WorldBounds is `{X,Y}=0` for character canvases. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/decoder.go` | `decoder.decode(cp)` returns a `sprite{img, ox, oy, z, w, h}`. We do not reuse this directly (we read pre-extracted PNGs), but the algorithm informs joint resolution. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/wz/property/property.go` | `CanvasProperty`, `VectorProperty` accessors needed by `extractCharacterParts` and `extractCharacterMaps`. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/rest/handler.go` | Re-exports `RegisterHandler`, `GetHandler`, `HandlerDependency`. |
| `libs/atlas-constants/item/constants.go` | `GetWeaponType(itemId)` exists; `IsTwoHanded` is **new** (Task 1.1). |
| `services/atlas-assets/nginx.conf` | Add the character-render `try_files` block above `location /`. |
| `services/atlas-ui/src/services/api/maplestory.service.ts` | Delete after migration. Keep `characterToMapleStoryData` adapter (folded into `characterRender.service.ts`). |
| `services/atlas-ui/src/lib/hooks/useCharacterImage.ts` | Update queryKey to include loadout hash; swap `mapleStoryService.generateCharacterImage` for the new builder. |
| `services/atlas-ui/src/components/features/characters/CharacterRenderer.tsx` | No `frameMode='platform'` exists today — design assumption was wrong. The cleanup is just swapping the service import. Verified with `grep -rn "frameMode" services/atlas-ui/src/` returning zero. |
| `services/atlas-ui/src/components/features/characters/OptimizedCharacterRenderer.tsx` | Also imports `mapleStoryService`. Update or delete its consumer code paths. |
| `services/atlas-ui/src/lib/utils/character-cache-sw.ts` | Hard-codes a `maplestory.io` URL for the service-worker prewarm. Update or remove. |

## Dependencies / new

- **Go**: no new module dependencies. SHA-256 via `crypto/sha256`. PNG via `image/png` (already in use).
- **TS**: add `js-sha256` to `services/atlas-ui/package.json`.

## Test strategy gist

- Backend: pixel-fixture tests for the compositor against committed PNG fixtures. Joint-resolution unit tests use fabricated sidecars (no WZ needed). Hash parity test reads `testdata/loadout-hashes.json`.
- Frontend: hash parity test reads the same fixture file. URL-builder tests assert canonical sorting + slot dropping.
- Acceptance gate: `grep -rn "maplestory.io" services/atlas-ui/src/` MUST return zero.

## Things to watch

- The render handler **recomputes** the hash from query params and verifies it matches the path component. Mismatch → 400. This guards against stale URLs surviving an extraction wipe.
- Atomic write: `os.CreateTemp(destDir, "{hash}.png.")` → write → `Sync` → `os.Rename` to `{hash}.png`. Two concurrent renders for the same hash collide on the rename only; both writers produce byte-identical PNGs.
- The extraction-time wipe targets only `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character/`. The `character-parts/` and `character-meta/` directories are overwritten in place; wiping them would create a 404 window.
- Two-handed override: stance changes inside the compositor, but the **request** stance is what's hashed into the URL. Same loadout requested as `stand1` and `stand2` produces two cache files with identical pixels — accepted.
- nginx named regex captures (`tenant`, `region`, `ver`, `hash`) feed `proxy_pass` rewrites. Test the regex with `nginx -T` after editing.

## Out of scope (do not implement)

- Pet, mount, cash slot rendering.
- Animation interpolation, multi-frame GIF/WebP.
- LRU eviction of cache PVC.
- `showEars` / Lef-ear toggles.
- Multi-replica wz-extractor render dispatch.
- Hot-swap during extraction (clients see 400 missing-asset until run finishes — accepted for v1).
