# Task-008 Map Detail Redesign — Implementation Plan

**Last Updated: 2026-04-18**
**Status: Ready for implementation (post-plan, pre-code)**
**Source PRD:** `docs/tasks/task-008-map-detail-redesign/prd.md`
**Companion docs:** `docs/tasks/task-008-map-detail-redesign/ux-flow.md`, `docs/tasks/task-008-map-detail-redesign/render-pipeline.md`
**Existing spike:** `services/atlas-wz-extractor/atlas.com/wz-extractor/cmd/map-render-spike/`

---

## 1. Executive Summary

Redesign `MapDetailPage` in `atlas-ui` from a bare four-tab table view into an overview-first layout: copyable title, metadata badges, a side-by-side map-image + NPC/monster summary panel, a connected-maps row, and a reduced three-tab detail view. Eliminate the external `maplestory.io` image dependency by producing map images from the tenant's own `Map.wz` via `atlas-wz-extractor`.

Ship in two phases to de-risk:

- **Phase 1 — UI redesign + minimap source.** Small extractor addition (`map/{mapId}/minimap.png`) + full UI rewrite. Deliverable: working redesigned page against minimap imagery.
- **Phase 2 — Full-map composite renderer.** New `mapimage/` subpackage in `atlas-wz-extractor` that composites backgrounds + tiles + objects into `map/{mapId}/render.png`. UI swaps image source to prefer `render.png` with graceful fallback. Deliverable: full-resolution recognizable map renders across the tenant catalog.

Scope is bounded to two services: `atlas-ui` and `atlas-wz-extractor`. No API changes, no schema changes, no other services touched.

---

## 2. Current State Analysis

### 2.1 Frontend — `services/atlas-ui/src/pages/MapDetailPage.tsx`
- Minimalist layout: `<h2>` title, inline `#<id>` span, optional street-name span, `Tabs` over four tables (Portals / NPCs / Monsters / Reactors).
- Existing React Query hooks: `useMap`, `useMapPortals`, `useMapNpcs`, `useMapMonsters`, `useMapReactors`.
- Existing copyable-tooltip pattern: `services/atlas-ui/src/components/map-cell.tsx` using `TooltipContent copyable`.
- Existing asset URL helper: `services/atlas-ui/src/lib/utils/asset-url.ts` → `getAssetIconUrl(tenant, region, major, minor, category, entityId)` producing `{category}/{entityId}/icon.png` for categories `npc` / `mob` / `reactor`.
- Map image is either absent or sourced from `maplestory.io` (to be removed).

### 2.2 Extractor — `services/atlas-wz-extractor/atlas.com/wz-extractor/`
- `image/extract.go` currently extracts per-entity icons (NPC / mob / reactor) into the shared assets volume.
- `extraction/processor.go` orchestrates extraction; wires `ExtractIcons` into the upload flow.
- `wz/canvas/decompress.go` already decodes WZ canvases into `*image.NRGBA` — reused by the spike and by Phase 2.
- Verified spike at `cmd/map-render-spike/` renders real maps (Henesys, Perion, Ellinia, El Nath, Leafre, Henesys hunting ground) correctly in 2–6 s each. Algorithm is known-good.

### 2.3 Data surface
- `atlas-data` `map/reader.go` already exposes map bounds resolution, foothold parsing, `VR*` + `miniMap` precedence — the extractor mirrors this logic but reads from `Map.wz` directly.
- No REST API changes required. All map/entity data already flows through existing `/api/data/maps/{id}/*` endpoints.

### 2.4 Known constraints
- External dependency on `maplestory.io` must not be reintroduced.
- All renders are tenant-scoped via the existing `/api/assets/{tenantId}/{region}/{version}/` path prefix.
- Extractor must not OOM on malformed `Map.wz` entries — canvas safety cap required.
- Full-tenant extraction SLA (~45 min currently) — Phase 2 must not blow past 2× baseline.

---

## 3. Proposed Future State

### 3.1 User-visible
- Operator lands on `/maps/:id` → sees title, street badge, spawn count, full-resolution map image, deduplicated NPC list, deduplicated monster list with counts, clickable connected-map widgets, and three detail tabs — all without clicking a tab.
- Title tooltip copies the numeric template ID to clipboard.
- Image source is tenant-owned (`atlas-assets` volume), no external calls.

### 3.2 Technical
- `MapDetailPage.tsx` rewritten as a composition of small, independently-loading sub-components:
  - `MapHeader` (title + copyable tooltip + badge row)
  - `MapImagePanel` (image + fallback placeholder; knows the Phase-1 → Phase-2 fallback chain)
  - `MapEntitySummary` (NPC + Monster dedup lists)
  - `ConnectedMapsRow` (dedup + filter + horizontal scroll)
  - `MapDetailTabs` (three tabs: portals / monsters / reactors)
- `asset-url.ts` extended with `getMapImageUrl(tenant, mapId, kind)` sibling helper (category `'map'` added to the union).
- `atlas-wz-extractor` gains:
  - Phase 1: ~50-LOC minimap extractor that writes `map/{mapId}/minimap.png` next to existing entity icons.
  - Phase 2: new `mapimage/` subpackage implementing the verified spike algorithm — backgrounds + tiles + objects composited into `map/{mapId}/render.png`, wired into `extraction/processor.go`.
- No new third-party dependencies in either service. No schema changes. No new REST endpoints.

### 3.3 Non-functional
- Progressive UI render — header + image mount before any entity query resolves.
- Per-map render cap (default 16384×16384) + per-map structured log line (`mapId`, dims, durationMs, spriteCount).
- Deterministic render output (sorted sprite lists, fixed PNG compression) to avoid asset-volume churn on re-upload.

---

## 4. Implementation Phases

### Phase 1 — UI redesign + minimap source
*Goal:* ship the redesigned page end-to-end against the minimap image so the layout is in production well before the composite renderer lands.

**Subphases:**
- **1A. Extractor minimap extraction** (backend, ~50 LOC)
- **1B. Asset URL helper** (frontend, shared utility)
- **1C. UI sub-components** (frontend, multiple small components)
- **1D. Page rewrite + integration** (frontend)
- **1E. Verification** (manual + docker build check)

### Phase 2 — Full-map composite renderer
*Goal:* replace minimap with full-resolution composite render, keeping minimap as fallback.

**Subphases:**
- **2A. Promote spike to subpackage** — move the verified `cmd/map-render-spike` implementation into a proper `mapimage/` subpackage under `atlas-wz-extractor`.
- **2B. Wire into extraction pipeline** — call the renderer from `extraction/processor.go` during `ExtractIcons`. Structured logging, safety caps, skip-on-error semantics.
- **2C. UI fallback chain** — extend `getMapImageUrl` to prefer `render.png` and fall back to `minimap.png` on 404.
- **2D. Verification** — visual spot-check of 20 representative maps; extraction time budget check.
- **2E. Phase 2.1 follow-up (optional, not blocking)** — address horizon-seam artifact for sky backgrounds.

---

## 5. Detailed Tasks

> Effort scale: **S** ≤ 2 hr, **M** ≤ 1 day, **L** ≤ 2 days, **XL** ≥ 3 days. Exact task breakdown lives in `task-008-map-detail-redesign-tasks.md`.

### Phase 1A — Extractor minimap extraction

**1A.1 Add minimap extraction function** *(Effort: S)*
- Add `ExtractMinimap(ctx, mapEntry, outputDir)` in a new file (`image/map_minimap.go`) or extend `image/extract.go`.
- Read `miniMap/canvas` from the map `.img`, decode via existing `wz/canvas.Decompress`, encode PNG to `{outputImgDir}/map/{mapId}/minimap.png`.
- Skip maps with no `miniMap` (log at debug, no error).
- **Acceptance:** unit test covering map-with-minimap and map-without-minimap paths. Running the extractor against a test `Map.wz` produces `minimap.png` files for all maps with a minimap canvas.

**1A.2 Wire minimap extraction into processor** *(Effort: S)*
- Extend `extraction/processor.go` so `ExtractIcons` (or the appropriate per-tenant extraction method) iterates over `Map.wz/Map/Map{N}/*.img` and calls `ExtractMinimap`.
- Log summary count at end (`minimaps_extracted=...`).
- **Acceptance:** Running the extractor against a tenant upload produces `map/{mapId}/minimap.png` entries in the assets volume for every map with a minimap. No regression in existing icon extraction output.

**1A.3 Extractor tests + docker build** *(Effort: S)*
- Unit test on the minimap extraction function.
- `docker compose build atlas-wz-extractor` passes.
- **Acceptance:** `go test ./...` passes inside the service. Docker image builds.

---

### Phase 1B — Frontend asset URL helper

**1B.1 Extend `asset-url.ts`** *(Effort: S)*
- Add `'map'` to the exported `category` union type.
- Add sibling function `getMapImageUrl(tenant, mapId, kind: 'render' | 'minimap')` returning `{base}/{tenantId}/{region}/{version}/map/{mapId}/{kind}.png`.
- Do **not** alter `getAssetIconUrl` — map URLs go through the new sibling helper.
- **Acceptance:** Unit test for `getMapImageUrl` covering both `kind` values. Existing `getAssetIconUrl` tests unchanged and green.

---

### Phase 1C — Frontend sub-components

**1C.1 `MapHeader` component** *(Effort: S)*
- New file `src/components/features/maps/MapHeader.tsx`.
- Props: `mapId: string`, `name: string`, `streetName?: string`, `spawnCount: number | undefined` (undefined = loading skeleton).
- Renders `<h2>` with `TooltipContent copyable` reusing the exact `map-cell.tsx:43-45` pattern.
- Renders badge row: `Badge variant="secondary"` for streetName (omitted if empty), `Badge variant="outline"` for spawns (`{n} spawns`, singular `1 spawn`, skeleton if undefined).
- Remove the existing inline `#<id>` span — tooltip replaces it.
- **Acceptance:** Storybook/visual check in app: loading, single-spawn, many-spawn, no-street states render correctly. Tooltip click copies to clipboard.

**1C.2 `MapImagePanel` component** *(Effort: M)*
- New file `src/components/features/maps/MapImagePanel.tsx`.
- Props: `mapId: string`, `mapName: string`.
- Reads tenant/region/version from `TenantProvider` (same contract as `getAssetIconUrl` callers).
- Wraps `<img loading="lazy">` inside `Card`. Alt text: `"Map render for " + mapName`.
- Phase 1 only: sets `src` to `getMapImageUrl(..., mapId, 'minimap')`.
- Implements 404 fallback: `onError` swaps to a neutral placeholder div with `<MapIcon />` + text "No render available" (no cycle — one swap only).
- **Acceptance:** Manual test against a map with minimap → image loads. Against a map without minimap → placeholder renders. No broken-image icon ever shown.

**1C.3 `MapEntitySummary` component** *(Effort: M)*
- New file `src/components/features/maps/MapEntitySummary.tsx`.
- Two sub-sections inside one `Card`: NPCs and Monsters.
- NPCs: props `npcs: Npc[] | undefined`, dedup by `template` preserving insertion order. Each row 32px icon via `getAssetIconUrl('npc', template)`, name linked to `/npcs/{template}`. Heading `NPCs ({count})`. Empty state italic "No NPCs". Skeleton while undefined.
- Monsters: props `monsters: Monster[] | undefined`, dedup + count per template. Each row 32px icon via `getAssetIconUrl('mob', template)`, name linked to `/monsters/{template}` (verify route — open question in PRD §9), `×N` muted suffix. Heading `Monsters ({count})`. Empty state italic "No monsters". Skeleton while undefined.
- Each sub-section is its own scrollable region capped at ~400px height.
- Each sub-section renders independently — an undefined `npcs` prop does not block the monsters sub-section.
- **Acceptance:** Rendering against real Henesys data produces a deduped list matching the UX spec. Confirms monster link target (monsters/mobs) — update if mismatched.

**1C.4 `ConnectedMapsRow` component** *(Effort: M)*
- New file `src/components/features/maps/ConnectedMapsRow.tsx`.
- Props: `mapId: string`, `portals: Portal[] | undefined`.
- Dedup logic per `ux-flow.md` dedup semantics snippet: filter out `targetMapId === 999999999`, falsy, and self-links; keep insertion order.
- Renders horizontally-scrolling flex row of `Card`-style widgets (width `w-48`). Each widget is a `Link` to `/maps/{targetMapId}` using `MapCell` to resolve the name (leverages existing `mapNameCache`).
- Heading `Connected maps ({count})`. Whole section omitted if zero valid targets.
- Loading: skeleton widgets while `portals === undefined`.
- **Acceptance:** Test map with mixed valid/invalid portals — widgets render only for valid targets, no duplicates, horizontal scroll triggers on overflow, clicking navigates to the target map.

**1C.5 `MapDetailTabs` component** *(Effort: S)*
- New file `src/components/features/maps/MapDetailTabs.tsx` (or inline — the three tabs are thin wrappers).
- Three `TabsTrigger`s: portals (default), monsters, reactors. NPCs tab is removed entirely.
- Tab contents: copy-paste existing table rendering from current `MapDetailPage.tsx` (portals table, monsters table, reactors table). Drop the NPC table wholesale.
- **Acceptance:** Tables render identical to today. Deep-link `/maps/:id/portals/:portalId` still works (route unchanged).

---

### Phase 1D — Page rewrite + integration

**1D.1 Rewrite `MapDetailPage.tsx`** *(Effort: M)*
- Compose the five sub-components in top-to-bottom order inside the existing `p-10 pb-16` container.
- Hook wiring:
  - `useMap(mapId)` → `MapHeader`, `MapImagePanel`.
  - `useMapNpcs(mapId)` → `MapEntitySummary` (npcs prop).
  - `useMapMonsters(mapId)` → `MapEntitySummary` (monsters prop), `MapHeader` (spawnCount).
  - `useMapPortals(mapId)` → `ConnectedMapsRow`.
  - All five hooks → `MapDetailTabs` as today.
- Error handling: `useMap` error → full-page `ErrorDisplay` with retry (today's behavior). Other hook errors → inline in their respective panel per `ux-flow.md` error table.
- Delete the NPC-tab-related rendering code.
- **Acceptance:** Progressive render timeline from `ux-flow.md` §"Progressive render timeline" holds — observable in DevTools Network throttled mode. All existing deep-links (`/maps/:id`, `/maps/:id/portals/:portalId`) continue to work.

**1D.2 Grid layout** *(Effort: S)*
- Desktop `md:grid-cols-[2fr_1fr]` for the image+summary panel row.
- Mobile stacked (default flex-column).
- **Acceptance:** Resize browser from ≥1024px down to 375px — layout collapses gracefully, no overflow, connected-maps row stays horizontally scrollable on touch.

**1D.3 Accessibility pass** *(Effort: S)*
- `<img alt>` meaningful (see `MapImagePanel` spec).
- Tooltip-as-copy keyboard accessible (inherits from `TooltipContent copyable`).
- All connected-map widgets are `<Link>` with text — no icon-only clickable elements.
- **Acceptance:** Tab-traverse the page; every interactive element is reachable and announces meaningfully.

---

### Phase 1E — Verification

**1E.1 Manual browser walkthrough** *(Effort: S)*
- Start dev server (`pnpm dev` in `services/atlas-ui`). Walk through 5 maps: Henesys (100000000), Perion (102000000), a map with zero NPCs, a map with zero monsters, a map with a portal loop-back.
- Verify: all acceptance criteria from PRD §10 Phase 1 checklist.

**1E.2 No `maplestory.io` references** *(Effort: S)*
- `rg 'maplestory\.io'` in `services/atlas-ui/src/` produces zero matches for map images.

**1E.3 Build + type-check** *(Effort: S)*
- `pnpm build` clean. `pnpm typecheck` clean. Existing tests unaffected.

---

### Phase 2A — Promote spike to subpackage

**2A.1 Create `mapimage/` subpackage** *(Effort: M)*
- New directory `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/`.
- Migrate spike code from `cmd/map-render-spike/` into named files:
  - `mapimage/renderer.go` — public `Render(ctx, mapEntry, wzIndex, outDir, opts) error`.
  - `mapimage/bounds.go` — VR / miniMap bounds resolution (mirror `atlas-data/map/reader.go:179-216`).
  - `mapimage/index.go` — `buildBackIndex`, `buildTileIndex`, `buildObjIndex` against a single `*wz.File`.
  - `mapimage/background.go` — `drawBackground` tiling logic + `BackgroundType` enum.
  - `mapimage/blit.go` — `blit` primitive + flip cache.
  - `mapimage/sort.go` — stable sort keys for tiles/objs within a layer.
- No external deps beyond std `image`, `image/draw`, `image/png`, `math`, `sort`.
- **Acceptance:** Package builds clean. Renders a known-good fixture map byte-for-byte identical to the spike output (determinism check).

**2A.2 Unit tests on `mapimage/`** *(Effort: M)*
- Fixture-based tests on bounds resolution, background tiling step computation, sort-order stability, and flip cache.
- Golden PNG test: render one fixture map twice, byte-compare.
- **Acceptance:** `go test ./mapimage/...` green.

**2A.3 Remove or mark `cmd/map-render-spike`** *(Effort: S)*
- Either delete `cmd/map-render-spike/` (its contents now live in `mapimage/`) or keep as a standalone harness with a `README.md` pointing to `mapimage/`. Prefer deletion unless the harness is useful for developer inspection.
- **Acceptance:** No orphan code paths.

---

### Phase 2B — Wire into extraction pipeline

**2B.1 Invoke renderer from processor** *(Effort: M)*
- In `extraction/processor.go`, for each map in `Map.wz/Map/Map{N}/*.img`, call `mapimage.Render(...)`.
- Parallelize with a worker pool (`runtime.NumCPU()` workers) — spike showed this is necessary to hit the SLA.
- Pass `outputImgDir=.../map/{mapId}/` and render into `render.png`.
- **Acceptance:** Running an extraction against a real tenant upload produces `render.png` for every map with `back[]` + ≥1 layer.

**2B.2 Safety caps + skip-on-error** *(Effort: S)*
- Env-var `WZ_EXTRACT_MAX_MAP_PIXELS` (default `16384*16384`). Maps exceeding the cap log + skip.
- Skip maps with no back + no layers (cash shop, system maps) — log at debug.
- Renderer errors (missing `bS`, panic recovery) log warning + continue. One bad map never kills the batch.
- **Acceptance:** Inject a synthetic malformed map fixture; extraction completes with a structured warning and no crash.

**2B.3 Structured per-map logging** *(Effort: S)*
- Each successful render emits a structured log line: `mapId`, `width`, `height`, `durationMs`, `spriteCount`, `output=render.png`.
- Failures log with `error` + `skipped=true`.
- **Acceptance:** Logs from a test extraction run can be grepped/aggregated by `mapId`.

**2B.4 Optional WebP output** *(Effort: S)*
- Env-var `WZ_EXTRACT_RENDER_FORMAT` (default `png`, opt-in `webp`).
- If `webp`, import `golang.org/x/image/webp` encoder… **actually** — std lib has no webp encoder; if WebP requested, pull in a minimal dependency or drop the flag. Decision deferred to implementer; **if adding a dep is not acceptable, skip this task and keep PNG-only**. Document the decision in context doc.
- **Acceptance:** If implemented: env-var-toggled WebP writes `render.webp` instead of `render.png`. UI fallback handles both (see 2C). If skipped: decision recorded in context doc.

---

### Phase 2C — UI fallback chain

**2C.1 Extend `getMapImageUrl` with fallback** *(Effort: S)*
- `getMapImageUrl(tenant, mapId, kind)` stays single-URL. Fallback logic moves to `MapImagePanel`.
- `MapImagePanel` initial src: `getMapImageUrl(..., mapId, 'render')`.
- `onError` swap chain: `render.png` → `minimap.png` → placeholder. Implement as a small state machine (`renderState: 'render' | 'minimap' | 'placeholder'`).
- **Acceptance:** Map with both `render.png` and `minimap.png` → renders the full composite. Map with only `minimap.png` → falls back after first 404. Map with neither → placeholder. Exactly one fallback per image (no infinite loop).

**2C.2 UI verification against Phase 2 output** *(Effort: S)*
- Run `atlas-wz-extractor` locally on a tenant upload to produce `render.png`s. Reload the UI.
- Spot-check 20 representative maps from PRD §10 Phase 2.3.
- **Acceptance:** 20/20 maps recognizable as the intended map. Any unrecognizable renders get a bug filed under Phase 2.1 follow-up.

---

### Phase 2D — Verification

**2D.1 Full-tenant extraction time** *(Effort: S)*
- Time a full tenant upload extraction before and after Phase 2 wiring.
- **Acceptance:** Phase 2 extraction time ≤ 2× the pre-Phase-2 baseline. If exceeded, tune worker count / PNG compression level.

**2D.2 Docker builds + CI** *(Effort: S)*
- `docker compose build atlas-wz-extractor atlas-ui` both pass.
- Any existing test suite in either service stays green.
- **Acceptance:** Green pipeline.

**2D.3 PRD §10 Phase 2 checklist sign-off** *(Effort: S)*
- Walk through each Phase-2 acceptance criterion in PRD §10 and mark done or open-follow-up.

---

### Phase 2E — Follow-ups (not blocking)

**2E.1 Horizon-seam fix (Phase 2.1)** *(Effort: M)*
- Fill vertical band between sky sprite and world top/bottom with nearest edge tile. ~1 day per `render-pipeline.md` §"Known limitations". Open as a separate task after Phase 2 ships.

**2E.2 `front=1` foreground validation** *(Effort: S)*
- First time a real map exercises `front=1`, verify the step-5 code path visually. If broken, fix locally.

---

## 6. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Full-tenant extraction time blows past SLA after Phase 2 | M | H | Worker pool in 2B.1. PNG BestSpeed compression if needed. Env-toggle to disable map renders in an emergency. Baseline timings captured in 2D.1. |
| OOM on malformed `Map.wz` canvas dimensions | L | H | Safety cap in 2B.2 (`MaxPixels` default 16384×16384). Skip + log on exceed. |
| Minimap 404 cascade hits every map before Phase 2 lands | L | L | `MapImagePanel` single-swap fallback to placeholder — no infinite loop. Minimap extractor runs as part of Phase 1 so presence is broad. |
| Monster detail route mismatch (`/monsters/:id` vs. `/mobs/:id`) | M | L | PRD §9 calls this out explicitly. Task 1C.3 verifies against `App.tsx` during implementation and fixes in place. |
| Render determinism regression churns asset volume on re-upload | L | M | 2A.2 golden-PNG byte-compare test catches regressions. Sort keys are stable. No random seeds. |
| `front=1` foreground code path untested against real map | M | L | Acceptable — low risk, local fix when first encountered (2E.2). |
| Large image assets bloat tenant storage | L | M | Documented in PRD §8 NFR — 15–30GB/tenant expected. WebP is opt-in for sites that need it. Operational concern, not a code defect. |
| Breaking existing deep-links during page rewrite | L | H | Route unchanged. Explicit acceptance criterion in 1D.1. Manual walkthrough in 1E.1 covers `/maps/:id/portals/:portalId`. |

---

## 7. Success Metrics

### Phase 1 (acceptance — pass/fail)
- All 9 PRD §10 Phase-1 checkboxes marked done.
- Manual walkthrough of 5 representative maps passes without regressions.
- Docker image for `atlas-wz-extractor` builds clean.
- No `maplestory.io` references for map images in `atlas-ui`.

### Phase 2 (acceptance — pass/fail)
- All 5 PRD §10 Phase-2 checkboxes marked done.
- 20/20 representative maps in spot-check are visually recognizable.
- Full-tenant extraction time ≤ 2× pre-Phase-2 baseline.
- Byte-stable render output on re-run (determinism check).

### Qualitative
- Operator feedback after 1 week of shipped Phase 1: "I can see what the map looks like at a glance" — positive confirmation from at least one GM / operator.

---

## 8. Required Resources & Dependencies

### Tooling / runtimes
- Go (existing in `atlas-wz-extractor`).
- Node + pnpm (existing in `atlas-ui`).
- Docker for service builds.

### Data fixtures
- A v83 GMS `Map.wz` upload for a test tenant (already available — used in the spike).

### No external dependencies
- No new Go modules (std lib only).
- No new npm packages.
- No new infrastructure — reuses existing atlas-assets volume + nginx.

### People
- One implementer with familiarity in both Go (extractor) and React/TypeScript (UI). Phases are independently shippable — two people can parallelize Phase 1C (UI components) and Phase 1A (extractor) if desired.

---

## 9. Timeline Estimates

Rough order-of-magnitude — adjust based on implementer speed / review latency.

| Phase | Effort (dev-days) | Notes |
|---|---|---|
| 1A Extractor minimap | 0.5–1 | ~50 LOC + test + wire-in. |
| 1B Asset URL helper | 0.25 | Trivial extension. |
| 1C UI sub-components | 2–3 | Five small components. |
| 1D Page rewrite | 0.5–1 | Composition. |
| 1E Verification | 0.5 | Manual walkthrough + build. |
| **Phase 1 total** | **4–6 days** | |
| 2A Promote spike | 1–2 | Refactor known-good code + tests. |
| 2B Wire into processor | 1–1.5 | Worker pool + logging + safety caps. |
| 2C UI fallback | 0.5 | State machine in `MapImagePanel`. |
| 2D Verification | 0.5–1 | Extraction timings + spot-check 20 maps. |
| **Phase 2 total** | **3–5 days** | |
| **Combined** | **~7–11 days** | Phases independently shippable. |

---

## 10. Out of Scope

Per PRD §2 Non-goals:
- Interactive map viewer (zoom/pan).
- Editing entities from the map page.
- Foothold / portal / spawn markers on the render.
- On-demand render at HTTP request time.
- Back-filling renders for pre-existing uploads (re-upload regenerates).
- Rendering for tenants without `Map.wz`.
- Pixel-identical in-game reproduction (parallax, lighting, camera).
