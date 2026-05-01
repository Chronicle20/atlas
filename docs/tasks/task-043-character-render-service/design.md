# Self-hosted character render service — Design

Version: v1
Status: Draft
Created: 2026-05-01
PRD: `prd.md`
API contracts: `api-contracts.md`
Data model: `data-model.md`
Risks: `risks.md`

This document describes the implementation architecture for replacing `maplestory.io` with an in-cluster character render service. It assumes the PRD's purpose, scope, and acceptance criteria are settled.

---

## 1. Architecture overview

Three services cooperate. atlas-ui builds a single URL pointed at atlas-assets. atlas-assets nginx tries the cached PNG; on miss it falls back to atlas-wz-extractor, which composites, atomically writes the PNG to the shared PVC, and returns the bytes. Subsequent requests hit the file directly via nginx.

```
browser
  │  GET /api/assets/{tenant}/{region}/{v}/character/{hash}.png?<render-query>
  ▼
atlas-ingress
  │  /api/assets/* → atlas-assets:8080   (existing rule)
  ▼
atlas-assets nginx
  │  try_files $uri @render;
  │      hit:  serve PNG (Cache-Control: public, max-age=86400, immutable)
  │      miss: proxy_pass http://atlas-wz-extractor:8080
  │             /api/wz/character/render/{tenant}/{region}/{v}/{hash}.png?<render-query>
  ▼
atlas-wz-extractor characterrender handler
  │  validate → composite (characterimage) → atomic write
  │  PNG written to PVC at the same path nginx tried.
  ▼
browser receives PNG
```

The PVC mounted at atlas-assets's `/usr/assets` is the same volume atlas-wz-extractor writes to via `OUTPUT_IMG_DIR`. No new volume.

### 1.1 Architectural decisions (locked during design)

| # | Decision | Notes |
|---|---|---|
| D1 | Single URL hits atlas-assets; nginx `try_files` falls back to atlas-wz-extractor on 404. | FR-12: cache hits never touch wz-extractor. |
| D2 | Asset extraction emits a PNG + JSON sidecar per `(templateId, stance, frame, partName)` at extraction time. | Render path does only `image.Decode` + blit. |
| D3 | Two-handed weapon classification uses `libs/atlas-constants/item.IsTwoHanded(...)` server-side. | Single source of truth for all backend consumers. |
| D4 | No render-time dedup. Each cache miss renders independently; atomic temp+rename guarantees clients never see partial bytes. | Single-replica wz-extractor today; bursts produce duplicated work but correct outputs. |
| D5 | atlas-wz-extractor exposes the render endpoint at `/api/wz/character/render/{tenant}/{region}/{v}/{hash}.png`. nginx fallback rewrites the captured path; the wz-extractor handler is the only place that knows how to render. | Service-prefix ownership: `/api/assets/*` = atlas-assets, `/api/wz/*` = atlas-wz-extractor. |
| D6 | New `characterimage/` package, sibling to `mapimage/`, sharing `mapimage/blit.go` and `mapimage/decoder.go` primitives. | Joints + zmap differ enough from world-bounds map composition that fusing the packages couples two domains. |
| D7 | Frontend hash uses `js-sha256` (sync, ~5KB). | Matches `data-model.md`'s SHA-256-truncated-16-hex spec without async/SubtleCrypto. |
| D8 | New `services/api/characterRender.service.ts`; the `maplestory.io` constants and URL builder in `services/api/maplestory.service.ts` are deleted. | FR-18, FR-20. |

### 1.2 Open items resolved during design

The PRD's open questions:

- **Smap location.** Confirmed: `Base.wz/smap.img` and `Base.wz/zmap.img` both exist in extracted XML at `tmp/<tenant>/GMS/83.1/Base.wz/`. The render service parses both: zmap for z-order (entries are declared in render order — first entry on top, or last; verify against `mapimage/sort.go` semantics during implementation), smap for slot-precedence/cover logic (e.g. `capOverHair → CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe` indicates which categories the cap supersedes). No hardcoded fallback needed.
- **Two-handed determination.** Resolved per D3 above: `atlas-constants/item.IsTwoHanded(...)`.
- **Skin color mapping.** Server-side, per `data-model.md` §"Skin color mapping". Endpoint accepts internal 0–10; maps to WZ id 2000–2013 inside the handler.
- **Walk/jump frame counts.** Validated lazily against the body skin's stance directory: if the requested frame doesn't exist for that stance, the handler returns 400.
- **Lef ear / showEars toggles.** Skipped for v1. Documented in `risks.md` as low risk; added back if a use case appears.

---

## 2. Backend components

All paths under `services/atlas-wz-extractor/atlas.com/wz-extractor/`.

### 2.1 `image/` (extended)

`extract.go` already dispatches by WZ name; the `case name == "character":` branch is extended.

- `extractEquipmentIcons` (existing) keeps emitting equipment icons (info/icon canvas) — this is the inventory icon flow, untouched.
- `extractCharacterParts` (new) walks every `.img` under Character.wz root and gendered subdirs:
  - For each top-level `*.img` (body skins `0000{skin}.img` / `0001{skin}.img`) and each subdir img (`Cap/*.img`, `Coat/*.img`, …), iterate `info/`, every stance subdir (`stand1`, `stand2`, `walk1`, `alert`, `jump`), every frame index, every part canvas.
  - For each canvas, decode via `wz/canvas.Decompress` (same primitive `mapimage/decoder.go` uses) and write to disk:
    - PNG: `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character-parts/{templateId}/{stance}/{frame}/{partName}.png`
    - Sidecar JSON: same path with `.json`, schema per `data-model.md` §"Sprite metadata sidecar" (origin, map, z, group, delay, face).
  - For each `.img`, emit `{templateId}/info.json` with `{islot, vslot, cash}`.
  - Equipment subdirs covered (per FR-3): `Cap/`, `Coat/`, `Longcoat/`, `Pants/`, `Shoes/`, `Glove/`, `Cape/`, `Shield/`, `Weapon/`, `Hair/`, `Face/`, `Accessory/` (templateId prefixes 101xxxx face accessories, 102xxxx eye accessories, 103xxxx earrings — all live under `Accessory/`). Body skin imgs live at the Character.wz root (`0000{skin}.img`, `0001{skin}.img`).
  - Out of scope: `Ring`, `Pendant`, `PetEquip`, `TamingMob`, `Dragon`, `Afterimage`. Stance scope: `stand1`, `stand2`, `walk1`, `alert`, `jump` only — other stances (fly, prone, swing) are skipped.

`zmap.go` (new file in `image/`):
- `extractCharacterMaps(l, baseFile, outputDir)` reads `Base.wz/zmap.img` and `smap.img`.
- Emits `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character-meta/zmap.json` (ordered list of layer-string names) and `smap.json` (object: layer-string → slot-category-string).
- Called from `extract.go` when `name == "base"`.

### 2.2 `characterimage/` (new package)

```
characterimage/
├── doc.go
├── errors.go              ErrUnknownTemplateId, ErrInvalidStance, ErrFrameOutOfRange, ErrAssetsMissing
├── meta.go                LoadZmap / LoadSmap / LoadPartMeta / LoadInfo (read JSONs from disk)
├── meta_cache.go          in-process LRU keyed by templateId (sync.Map of *templateMeta)
├── joints.go              joint-tree resolution: for each part, computes (anchor_x, anchor_y) on the canvas
├── stance.go              stance/frame validation against extracted body sprite
├── compositor.go          Composite(req CompositeRequest) (*image.RGBA, error)
├── compositor_test.go
└── scale.go               nearest-neighbor upscale of the final 96×128 canvas to (resize×96)×(resize×128)
```

Key types:

```go
type CompositeRequest struct {
    Tenant      tenant.Model
    Skin        int          // internal 0..10
    Hair        int          // templateId
    Face        int          // templateId
    Equipment   map[string]int // slotName -> templateId, mount/pet/cash already filtered out
    Stance      string       // stand1 | stand2 | walk1 | alert | jump
    Frame       int
    Resize      int          // 1..4
    AssetsRoot  string       // {OUTPUT_IMG_DIR}/{tenant}/{region}/{v}
}

type compositor struct {
    zmap        []string                    // ordered layer-string list from zmap.json
    smap        map[string]string           // layer-string -> slot-category from smap.json
    metaCache   *sync.Map                   // templateId -> *templateMeta
    twoHanded   func(item.TemplateId) bool  // wraps libs/atlas-constants/item.IsTwoHanded
}
```

Algorithm (`Composite`):

1. **Filter equipment.** Drop slots -14, -18, -19, -20, -21..-30, -101..-114 (mount/pet/cash) per FR-9.
2. **Stance override.** If any equipped weapon templateId is two-handed (`item.IsTwoHanded`), force `stance = stand2` (FR-16).
3. **Map skin.** Internal skin → WZ skin id via the table in `data-model.md`.
4. **Load body metadata.** Read `{templateId}/info.json` and per-stance/frame part metadata for the body img. The body sprite carries the canonical joint anchors (`neck`, `navel`, `hand`, `body`).
5. **Load equipment metadata.** For each equipped slot, load `{templateId}/info.json`. Validate the templateId exists; otherwise return `ErrUnknownTemplateId`.
6. **Slot-precedence resolution (smap).** For each layer-string in the zmap, check `smap[layerString]` against `info.vslot` to determine whether a sprite from that template is allowed in that layer. (e.g. if `vslot=Hb` covers `Hd*`, hat sprites can replace head sprites for that slot.)
7. **Walk joint tree.** Starting from body's `origin` placed at canvas `(48, 96)` (chosen so foot row lands at 124 on a 96×128 canvas — exact constants verified against extracted body sprite during implementation), recursively position each part by mapping its `origin` to its parent's joint coordinate.
8. **Z-sort.** Order all parts by `zmap.IndexOf(part.z)` (ties broken by load order, per FR-14).
9. **Blit.** Use `mapimage/blit.go`'s `blit` primitive onto a `image.NewRGBA(96, 128)` canvas.
10. **Scale.** If `resize > 1`, nearest-neighbor upscale to `(96*resize) × (128*resize)`. Pixel art preservation.
11. **Encode.** PNG-encode the upscaled canvas.

Observability: each `Composite` records `len(equipment)` (post-filter), wall time, and a flag for two-handed override.

### 2.3 `characterrender/` (new package)

HTTP layer, separate from compositor. Uses the project's standard `RegisterHandler(l)(si)` pattern (see `extraction/resource.go`).

```
characterrender/
├── doc.go
├── handler.go         HandleRender — parses path + query, validates, calls compositor, atomic-writes, writes response
├── handler_test.go
├── hash.go            CanonicalLoadoutString / LoadoutHash (SHA-256 truncated 16 hex, matches client)
├── hash_test.go
├── path.go            ParseRenderPath (extracts tenant/region/version/hash from URL)
├── query.go           ParseRenderQuery (skin, hair, face, items, stance, frame, resize)
├── query_test.go
├── error.go           JSON:API-style error responses with code/title/detail/meta (per api-contracts.md §error body shape)
├── write.go           AtomicWrite (write to .tmp, fsync, os.Rename)
└── otel.go            character.render span helpers + counters
```

Wiring into `extraction/resource.go`:

```go
ren := router.PathPrefix("/wz/character").Subrouter()
ren.HandleFunc("/render/{tenant}/{region}/{version}/{hash}.png",
    register("render_character", rh.handleRender())).Methods(http.MethodGet)
```

The handler:

1. Parses `tenant`, `region`, `majorVersion`, `minorVersion`, `hash` from the path.
2. Parses `skin`, `hair`, `face`, `stance`, `frame`, `resize`, `items` from the query.
3. **Verifies the hash matches the canonicalized loadout.** If not, returns 400. This guards against stale URLs surviving an extraction wipe and against malformed clients pointing at the wrong cache file.
4. Resolves the assets root: `{OUTPUT_IMG_DIR}/{tenant}/{region}/{majorVersion}.{minorVersion}`. If it doesn't exist, returns 404 with `code: "tenant-not-extracted"`.
5. Builds the tenant context (`tenant.WithContext`) and constructs `CompositeRequest`.
6. Calls compositor. Errors map to:
   - `ErrUnknownTemplateId` → 400 `code: unknown-template-id`
   - `ErrInvalidStance` → 400 `code: invalid-stance`
   - `ErrFrameOutOfRange` → 400 `code: frame-out-of-range`
   - `ErrAssetsMissing` → 404 `code: missing-asset`
   - other → 500 `code: compositor-error`
7. Atomic-writes the PNG bytes to `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character/{hash}.png`. Because D4 forbids dedup, two concurrent renders for the same hash must not collide on the temp filename. Write to `{hash}.png.{pid}.{rand}.tmp` (e.g. `os.CreateTemp` in the destination directory with prefix `{hash}.png.`), fsync, `os.Rename` to `{hash}.png` (atomic on the same filesystem; safe under nginx open). Last-writer-wins; both writers produce byte-identical PNGs.
8. Writes the response: `Content-Type: image/png`, `Cache-Control: public, max-age=86400, immutable`, `ETag: "{hash}"`, `X-Render-Cache: miss`, `X-Render-Ms: {duration}`.

### 2.4 `extraction/processor.go` (modified)

Two changes:

1. Extend the existing per-WZ-file dispatch in `image/extract.go`'s `ExtractIcons` (which already switches on `name == "character" | "base" | …`):
   - `case name == "character":` runs both `extractEquipmentIcons` (existing — inventory icons) **and** the new `extractCharacterParts` (worn-sprite assets + sidecars per FR-1/FR-2).
   - `case name == "base":` (new) runs `extractCharacterMaps` (zmap.json + smap.json per FR-4).
   `runExtraction` itself isn't aware of WZ names; the dispatch stays in `extract.go`.
2. Before processing any WZ file (i.e. at the top of `runExtraction`), wipe `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character/` (the rendered character cache directory). The `character-parts/` and `character-meta/` directories are not wiped — they're regenerated by the extraction itself, and wiping them would create a window where renders 404. Atomic replacement is sufficient.

Per FR-5: scope is `(tenant, region, version)` because the existing tenant-path rooting (`extraction/tenant_path.go#TenantPath`) already namespaces by that triple.

### 2.5 `main.go` wiring

`main.go` constructs `Processor` once, passing `INPUT_WZ_DIR`/`OUTPUT_XML_DIR`/`OUTPUT_IMG_DIR`. Add a second route initializer for `characterrender`:

```go
rh := characterrender.NewHandler(characterrender.Config{
    AssetsRoot:    outputImgDir,
    Compositor:    characterimage.NewCompositor(),
})
server.New(l).
    ...
    AddRouteInitializer(extraction.InitResource(p, tdm.WaitGroup(), extraction.Dirs{...})(GetServer())).
    AddRouteInitializer(characterrender.InitResource(rh)(GetServer())).
    Run()
```

`characterrender.InitResource` follows the pattern in `extraction/resource.go`.

---

## 3. Frontend components

### 3.1 New module: `services/api/characterRender.service.ts`

```ts
import sha256 from 'js-sha256';

export interface CharacterLoadout {
  skin: number;
  hair: number;
  face: number;
  equipment: Record<string, number>;
}

export interface RenderOptions {
  stance?: 'stand1' | 'stand2' | 'walk1' | 'alert' | 'jump';
  frame?: number;
  resize?: number;
}

const ITEM_SLOTS_TO_DROP = new Set([-14, -18, -19, -20, /* -21..-30 */ -21, -22, -23, -24, -25, -26, -27, -28, -29, -30]);
// Cash slots (-101..-114) are also dropped client-side.

function canonicalLoadoutString(
  tenant: string, region: string, major: number, minor: number,
  loadout: CharacterLoadout, opts: Required<RenderOptions>,
): string {
  const items = Object.entries(loadout.equipment)
    .filter(([slot]) => !ITEM_SLOTS_TO_DROP.has(parseInt(slot)))
    .map(([, id]) => id)
    .sort((a, b) => a - b)
    .join(',');
  return `${tenant}|${region}|${major}.${minor}|${loadout.skin}|${loadout.hair}|${loadout.face}|${opts.stance}|${opts.frame}|${opts.resize}|${items}`;
}

export function loadoutHash(canonical: string): string {
  return sha256(canonical).slice(0, 16);
}

export function generateCharacterUrl(
  tenant: string, region: string, major: number, minor: number,
  loadout: CharacterLoadout, options: RenderOptions = {},
): string {
  const opts = {
    stance: options.stance ?? 'stand1',
    frame: options.frame ?? 0,
    resize: options.resize ?? 2,
  };
  const canonical = canonicalLoadoutString(tenant, region, major, minor, loadout, opts);
  const hash = loadoutHash(canonical);
  const params = new URLSearchParams({
    skin: String(loadout.skin),
    hair: String(loadout.hair),
    face: String(loadout.face),
    stance: opts.stance,
    frame: String(opts.frame),
    resize: String(opts.resize),
    items: canonical.split('|').pop()!, // the sorted item list
  });
  return `/api/assets/${tenant}/${region}/${major}.${minor}/character/${hash}.png?${params.toString()}`;
}
```

The hash and the canonical string are both consumed by the server: the server recomputes the hash from the query and verifies it matches the path component, rejecting tampered/stale URLs with 400.

### 3.2 `services/api/maplestory.service.ts` (cleanup)

Delete:

- `apiBaseUrl`, `apiVersion`, and all maplestory.io-related constants (FR-20).
- `generateCharacterUrl`, `getCharacterImageUrl`, and any helpers that build `maplestory.io` URLs.
- `SKIN_COLOR_MAPPING` (mapping moves to the server side).
- `TWO_HANDED_WEAPON_RANGES` (server decides the stance).

Keep: `characterToMapleStoryData` (the adapter that flattens `Character` + `Asset[]` into the loadout shape). The new `characterRender.service.ts` consumes it. If retaining the file with only the adapter feels cluttered, fold the adapter into `characterRender.service.ts` and delete `maplestory.service.ts` entirely.

### 3.3 `lib/hooks/useCharacterImage.ts`

- Replace `mapleStoryService.generateCharacterUrl` with `characterRenderService.generateCharacterUrl`.
- The `region` and `majorVersion`/`minorVersion` inputs come from `useTenant()` (already imported).
- React Query key extends to include the loadout hash; cache invalidation flows naturally on tenant switch via `queryClient.clear()` in `TenantProvider`.

### 3.4 `components/features/characters/CharacterRenderer.tsx`

- Delete the entire `frameMode='platform'` pixel-scan path. Foot alignment is now guaranteed by the canvas contract; `<img>` wraps directly with `object-contain` in a fixed-aspect container.
- The `scale` prop maps directly to `resize` in the URL builder.

### 3.5 Tests

- `services/api/__tests__/characterRender.service.test.ts` — canonical string formation, hash stability across input permutations, mount/pet/cash slot dropping, default options.
- `services/api/__tests__/maplestory.service.test.ts` — update or delete; ensure no `maplestory.io` references remain anywhere under `services/atlas-ui/src/`.
- `components/features/characters/__tests__/CharacterRenderer.test.tsx` — drop pixel-scan tests, add tests for the new URL contract.

A repo-wide `grep -rn "maplestory.io" services/atlas-ui/src/` MUST return zero after the migration (acceptance criterion §10).

---

## 4. Deploy and ingress

### 4.1 atlas-assets `nginx.conf`

Current:

```nginx
location / {
  try_files $uri =404;
  add_header Access-Control-Allow-Origin *;
  add_header Cache-Control "public, max-age=86400";
}
```

Add (above the catch-all `location /`):

```nginx
# Character renders: try cache file, fall back to atlas-wz-extractor on miss.
location ~ ^/(?<tenant>[^/]+)/(?<region>[^/]+)/(?<ver>[^/]+)/character/(?<hash>[a-f0-9]{16})\.png$ {
  try_files $uri @character_render;
  add_header Access-Control-Allow-Origin *;
  add_header Cache-Control "public, max-age=86400, immutable";
}

location @character_render {
  proxy_pass http://atlas-wz-extractor:8080/api/wz/character/render/$tenant/$region/$ver/$hash.png$is_args$args;
  proxy_set_header Host $host;
  proxy_read_timeout 30s;
}
```

The named regex captures (`tenant`, `region`, `ver`, `hash`) become available as `$tenant`, `$region`, `$ver`, `$hash` for the proxy_pass rewrite. The proxied URL preserves the original query string via `$is_args$args`.

The `try_files` directive rewrites the URI path internally before the file probe — atlas-assets's `root /usr/assets` resolves to `/usr/assets/{tenant}/{region}/{v}/character/{hash}.png`, which is exactly where atlas-wz-extractor writes.

### 4.2 No new ingress location

`/api/assets/*` already proxies to `atlas-assets:8080` in `deploy/shared/routes.conf`, `deploy/compose/routes.conf`, and `deploy/k8s/ingress.yaml`. The atlas-assets-internal nginx changes are the only deploy delta.

### 4.3 Readiness gate (FR-23 — PRD departure)

**PRD FR-23 mandates a K8s readiness gate ensuring atlas-wz-extractor is ready before atlas-ui pods accept traffic on first deploy.** This design does not add ordering between Deployments; instead it relies on graceful degradation: if a render request arrives while atlas-wz-extractor is not yet ready, atlas-assets nginx returns a 502 from the `@character_render` upstream, atlas-ui's React Query retries with backoff, and the next attempt succeeds. The CharacterRenderer fallback avatar covers the brief window.

**Why depart from FR-23.** Adding an `initContainer` or `readinessGate` between two unrelated Deployments couples deploy choreography, complicates rolling restarts, and serves a transient cold-start window that's already visually handled. Cached PNGs on the PVC also mean the readiness gap is rare — a fresh PVC + simultaneous restart is the only triggering scenario.

If the user wants the literal FR-23 behavior, the design adds:

- atlas-wz-extractor exposes `GET /api/wz/health` returning 200 once route initialization completes.
- atlas-ui's Deployment gains a `readinessProbe` with an init delay or an `initContainer` that polls atlas-wz-extractor's health endpoint.

This is a deliberate departure flagged for the spec-review gate. Override or accept.

---

## 5. Data flow scenarios

### 5.1 Cold render (cache miss)

1. atlas-ui builds `/api/assets/T/R/83.1/character/abc123.png?skin=0&hair=30030&...`.
2. ingress → atlas-assets nginx.
3. nginx `try_files`: `/usr/assets/T/R/83.1/character/abc123.png` doesn't exist → falls into `@character_render`.
4. nginx proxy_pass: `http://atlas-wz-extractor:8080/api/wz/character/render/T/R/83.1/abc123.png?skin=0&hair=30030&...`.
5. Handler parses path + query; recomputes canonical string + hash; verifies hash matches `abc123`.
6. Resolves assets root `{OUTPUT_IMG_DIR}/T/R/83.1`; loads zmap/smap (cached after first call); loads per-templateId metadata (in-process LRU).
7. Composites onto 96×128, scales to 192×256.
8. Atomic-writes `{OUTPUT_IMG_DIR}/T/R/83.1/character/abc123.png.tmp` → renames to `abc123.png`.
9. Writes response: 200 image/png, headers as listed in §2.3.
10. nginx forwards bytes to client. Browser caches per `Cache-Control`.

### 5.2 Warm render (cache hit)

1. atlas-ui builds the same URL.
2. ingress → atlas-assets nginx.
3. nginx `try_files`: the file exists at `/usr/assets/T/R/83.1/character/abc123.png` → serves directly with `Cache-Control: public, max-age=86400, immutable`.
4. atlas-wz-extractor receives no traffic. Verifiable in pod metrics. (FR-12, acceptance criterion.)

### 5.3 Concurrent identical cold requests

Per D4: each request renders independently. Each writes to a unique `{hash}.png.{pid}.{rand}.tmp` (per §2.3 step 7), then renames onto the final path. Last writer wins; output is byte-identical because the inputs are identical. No client sees a partial file because rename is atomic. Wasted CPU is accepted.

### 5.4 Extraction wipe

1. Operator calls `POST /api/wz/extractions` (existing endpoint).
2. Handler acquires the per-tenant extraction mutex.
3. Before processing any WZ file: `os.RemoveAll({OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character/)`. This wipes all rendered loadouts for the triple; `character-parts/` and `character-meta/` are kept and overwritten in place by the extraction.
4. Extraction proceeds (Character.wz emits parts + sidecars; Base.wz emits zmap/smap).
5. Cache directory begins repopulating from the next render request.

Subtle point: while the extraction is running, a concurrent render request might compose against partially-extracted assets (e.g. body sprites updated, hair not yet). Mitigation: extract via `*.tmp.{run}` directory, swap with `os.Rename` at the end. **Out of scope for v1** — single-replica wz-extractor + the per-tenant extraction mutex makes it harmless in practice (clients see 400 on missing-asset, not a corrupted render). Documented in §8 (Future work).

### 5.5 Two-handed override

1. Client requests `stance=stand1` with a polearm equipped.
2. Handler builds `CompositeRequest` with `Stance: "stand1"`.
3. Compositor (step 2 of the algorithm) iterates equipped weapons, calls `item.IsTwoHanded(weaponTemplateId)`. True → overrides `Stance` to `"stand2"` for the rest of the algorithm.
4. The output PNG reflects the stand2 sprite. The hash baked into the URL is still based on the *requested* stance — meaning two URLs (`stance=stand1` and `stance=stand2`) for the same two-handed loadout produce two different cache files with identical pixels. Acceptable; the wasted slot is small and avoids cache-key surprises.

---

## 6. Hash function alignment

Server (`characterrender/hash.go`):

```go
func CanonicalLoadoutString(tenantId, region string, major, minor uint16,
    skin, hair, face int, stance string, frame, resize int, sortedItems []int) string {
    ids := make([]string, len(sortedItems))
    for i, id := range sortedItems { ids[i] = strconv.Itoa(id) }
    return fmt.Sprintf("%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s",
        tenantId, region, major, minor, skin, hair, face, stance, frame, resize, strings.Join(ids, ","))
}

func LoadoutHash(canonical string) string {
    sum := sha256.Sum256([]byte(canonical))
    return hex.EncodeToString(sum[:8])
}
```

Client (`characterRender.service.ts`): same algorithm via `js-sha256`. Both implementations are pinned to a shared fixture file checked in alongside the implementation: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/testdata/loadout-hashes.json`. The Go and TS test suites both load this file and assert each canonical-string-to-hash row matches. Adding a row in either language requires regenerating from the canonical string (deterministic), so drift is caught at test time.

Cross-language fixture verification is part of acceptance criteria.

---

## 7. Error handling

Errors map to JSON:API-style bodies (per `api-contracts.md`).

| Condition | HTTP | code | meta |
|---|---|---|---|
| `tenant/region/version` directory missing | 404 | `tenant-not-extracted` | `{tenant, region, version}` |
| Hash in path doesn't match recomputed hash from query | 400 | `hash-mismatch` | `{expected, got}` |
| Unknown stance | 400 | `invalid-stance` | `{stance, supported: [...]}`  |
| Frame index out of range for stance | 400 | `frame-out-of-range` | `{stance, frame, max}` |
| `resize` out of `1..4` | 400 | `invalid-resize` | `{resize}` |
| Equipment templateId not in extract | 400 | `unknown-template-id` | `{templateId}` |
| Assets root present but specific sprite missing (e.g. body for stance) | 404 | `missing-asset` | `{templateId, stance, frame, partName}` |
| Compositor panic / io.Writer failure | 500 | `compositor-error` | `{}` (logged with hash) |

The atlas-ui hook surfaces these as user-friendly messages per `api-contracts.md` §"Error mapping".

---

## 8. Observability

### 8.1 OTel span: `character.render`

Recorded in the wz-extractor handler. Attributes:

- `tenant.id` (string)
- `region` (string)
- `version` (string, e.g. `83.1`)
- `stance` (string — the *resolved* stance after two-handed override)
- `frame` (int)
- `resize` (int)
- `equipped_slot_count` (int — post-filter)
- `loadout_hash` (string)
- `cache_hit` (bool — always false, since we only run on miss)
- `render_ms` (int)
- `two_handed_override` (bool)
- `error.code` (string, on failure)

Cache hits don't get a span — they never touch wz-extractor. nginx access logs cover them.

### 8.2 Prometheus counters/histogram

- `character_render_total{stance, two_handed_override}` — counter.
- `character_render_errors_total{reason}` — counter, where `reason ∈ {invalid-input, hash-mismatch, unknown-template-id, missing-asset, compositor-error, tenant-not-extracted}`.
- `character_render_duration_ms` — histogram, render path only.

### 8.3 Gap acknowledged

The PRD §8 mentions the "no progress indicator" gap from the wz-extractor status discussion. Not blocking; not addressed here.

---

## 9. Testing strategy

### 9.1 Backend unit tests

- `characterimage/joints_test.go` — joint resolution: given a body sprite map and a child sprite origin, the resulting canvas position is correct. Uses fabricated sprites (no WZ needed).
- `characterimage/compositor_test.go` — bare body, equipped warrior, mage-with-tall-hat, archer-with-long-hair, polearm-wielder. Pixel-compare against committed PNG fixtures generated from a known reference.
- `characterimage/stance_test.go` — stance/frame validation, two-handed override.
- `characterrender/hash_test.go` — canonical string formatting, hash stability across input permutations (item order, optional defaults), parity with TS via shared fixture file.
- `characterrender/handler_test.go` — path/query parsing, hash mismatch rejection, error mapping table.
- `characterrender/write_test.go` — atomic write semantics: a reader during write either sees the old file or the new file, never partial bytes (use a separate goroutine + `tail -f`-style read in the test).
- `image/zmap_test.go` — zmap/smap parsing from a small XML fixture.

### 9.2 Backend integration test

- A test that runs `extractCharacterParts` against a tiny synthetic Character.wz fixture, then calls `Composite` end-to-end, then verifies the PNG bytes equal a checked-in fixture.
- A test that runs the full HTTP handler against an in-memory router (`net/http/httptest`) with an extracted assets root, verifying the wire response matches `api-contracts.md`.

### 9.3 Frontend tests

- `characterRender.service.test.ts` — generateCharacterUrl produces stable URLs for permuted item orders; mount/pet/cash slots are dropped; defaults applied; hash matches a known fixture (cross-checked with the Go fixture).
- `useCharacterImage.test.ts` — uses the new builder; React Query key includes the hash; tenant switch invalidates.
- `CharacterRenderer.test.tsx` — no pixel-scan path; renders an `<img>` with the expected URL.

### 9.4 Manual verification (acceptance gate)

- Browser network tab on a full atlas-ui session viewing accounts/characters/presets shows zero requests to `maplestory.io`.
- Cluster with no internet egress renders every test loadout end-to-end.
- pod-level metrics confirm atlas-wz-extractor receives no traffic on warm-cache page reloads.

---

## 10. Future work (intentionally out of scope)

- LRU eviction on the rendered cache PVC (today: bounded by extraction wipe).
- Multi-replica wz-extractor: requires either flock dedup or a render dispatcher service.
- Live preview hot-swap during extraction (today: clients see 400 on missing-asset until the run finishes).
- Walk/jump animation as multi-frame GIF / WebP-animated output.
- Pet, mount, and cash slot rendering.
- Lef ear / `showEars` toggles.
- Zone-time progress indicator for in-flight extractions.

---

## 11. Implementation order

This section is prescriptive guidance for the planner — each step is a vertical slice that can be tested before the next begins.

1. **Hash + canonical string** in both languages, with cross-language fixture parity. (Trivial; unblocks cache routing.)
2. **zmap/smap extraction.** Parse `Base.wz/zmap.img` and `smap.img` to JSON. (Pure refactor of an existing dispatch.)
3. **Character parts extraction.** Walk Character.wz, emit PNGs + sidecars + info.json for every covered slot. (Bulk of the extraction work.)
4. **Compositor primitives.** Joint tree, z-sort, blit onto 96×128, scale.
5. **Render handler + atomic write.** Path/query parsing, validation, error mapping, response shape.
6. **nginx config + route registration.** Smallest deploy delta.
7. **atlas-ui new module + hook update.** Adopt the URL builder.
8. **CharacterRenderer cleanup.** Delete pixel-scan; tighten container.
9. **maplestory.service.ts purge.** Remove all `maplestory.io` references; verify zero greps.
10. **Observability.** Spans + counters; verify in dev with a few cold/warm requests.

Each step is independently shippable; cutover to atlas-ui (steps 7–9) lands in a single PR but the backend (steps 1–6, 10) can be staged behind the unused render route.
