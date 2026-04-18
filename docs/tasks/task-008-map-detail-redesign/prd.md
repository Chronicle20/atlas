# Map Detail Redesign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-18
---

## 1. Overview

`MapDetailPage` in atlas-ui today is a minimalist four-tab layout: a plain title, an optional street name, and a `Tabs` strip over four tables (Portals / NPCs / Monsters / Reactors) — see `services/atlas-ui/src/pages/MapDetailPage.tsx:28-294`. The page does its job but gives no at-a-glance sense of the map itself: you can't see what the map looks like, can't tell how many monsters spawn on it without clicking into a tab, and can't jump to the maps it connects to without opening the Portals tab and scanning a table.

This task redesigns the page around a visual, overview-first layout: a clickable-to-copy title, metadata badges (street name, spawn-point count), a side-by-side **map image + NPC/monster summary** panel, a **connected-maps** widget row for portals with a valid target, and a reduced three-tab detail view (Portals / Monsters / Reactors — NPCs are promoted into the summary and removed from the tabs).

The biggest open question was how to get the map image without an external dependency on `maplestory.io` and without settling for the low-resolution minimap in `Map.wz`. The research is captured in `ux-flow.md` and `render-pipeline.md`; the solution is a full-map composite renderer in `atlas-wz-extractor` that writes `map/{mapId}/render.png` into the existing atlas-assets volume, consumed by the UI via the same static-URL pattern already used for NPC/mob/reactor icons. To de-risk shipping, the work is phased: **Phase 1** ships the UI with the minimap as the image source, **Phase 2** adds the composite renderer and swaps the image URL. Both phases live in this task.

## 2. Goals

Primary goals:
- Give operators an at-a-glance view of a map: what it looks like, what spawns on it, and where it leads.
- Keep detailed inspection (positions, mob times, script names, delays) one click away via tabs.
- Eliminate the external `maplestory.io` image dependency — render the map from the tenant's own WZ data so modded/private content renders correctly.
- Preserve existing deep-link URLs (`/maps/:id`, `/maps/:id/portals/:portalId`) and existing API contracts unless explicitly extended.

Non-goals:
- Interactive map viewer (zoom/pan/click-to-select spawn points). The image is a static render.
- Editing maps, NPCs, monsters, reactors, or portals from this page.
- Changing `MapDetailPage`'s route or breadcrumb behavior.
- Rendering foothold overlays, collision volumes, portal markers on the image, or any debug overlays. The image is the composited "screenshot", nothing more.
- Re-rendering maps on demand at HTTP request time. Renders are produced at extraction time and cached as static PNG/WebP.
- Back-filling renders for WZ uploads that pre-date the renderer — a re-upload triggers re-extraction; we don't need a separate migration.
- Rendering maps for tenants whose `Map.wz` we don't have (e.g., global/uuid.Nil fallback). Those fall back to the minimap (Phase 1 source) if present, or a placeholder.

## 3. User Stories

- As an operator investigating a bug report ("monsters won't spawn on Henesys Bazaar"), I want to land on the map page and immediately see the map image, the spawn count, and the list of monsters without opening any tab.
- As a GM documenting a quest, I want to click the title's tooltip to copy the template ID into my notes.
- As an operator tracing a portal chain (Perion → Perion Mountain Path → Wild Boar Land), I want a row of clickable map widgets under the image so I can step through connected maps without clicking into the Portals tab.
- As a designer reviewing spawn balance, I want the monster summary to show a deduplicated list with counts ("Mushroom ×4, Blue Mushroom ×2") rather than one row per spawn point.
- As a frontend developer, I want the page to render progressively so the header and image don't wait on the slowest entity query.
- As a platform engineer, I want map renders to come from our own asset pipeline so the UI doesn't break when maplestory.io rate-limits us or drifts from our server's WZ version.

## 4. Functional Requirements

### 4.1 Page layout (`MapDetailPage.tsx`)

From top to bottom, inside the existing `p-10 pb-16` container:

1. **Header row** — map name as `<h2>` (existing `text-2xl font-bold`). Wrapped in a `Tooltip` using the existing `TooltipContent copyable` pattern (see `services/atlas-ui/src/components/map-cell.tsx:43-45`). Hover shows the template ID; clicking the tooltip content copies the ID to the clipboard. The raw `#<id>` span next to the title is removed — the tooltip replaces it.
2. **Metadata badge row** — a flex row of `Badge` components:
   - `Badge` with `variant="secondary"` for `streetName` (omitted if empty — do not render an empty badge).
   - `Badge` with `variant="outline"` for `# spawns` where the count is `monsters.length` — i.e., the number of monster spawn-point entries, not distinct templates. The badge renders as `{n} spawns` (singular: `1 spawn`). While the monsters query is loading, show a skeleton badge.
   - Future badges (town flag, boss flag, PQ flag) are out of scope but the row's flex container must accommodate additional badges without layout changes.
3. **Image + summary panel** (desktop: two-column grid, `md:grid-cols-[2fr_1fr]`; mobile: stacked):
   - **Left panel** (`Card`): the map image. Source URL is built via `getAssetIconUrl(tenantId, region, major, minor, 'map', mapId)` — Phase 1 resolves to `map/{mapId}/minimap.png`, Phase 2 adds a `renderKind: 'render' | 'minimap'` argument and resolves `render.png` when available with graceful fallback to `minimap.png`. On missing image (404), render a neutral placeholder with `<MapIcon />` and the text "No render available". Use `<img loading="lazy">` per project conventions (`services/atlas-ui/CLAUDE.md` → Images section).
   - **Right panel** (`Card`): a vertical summary with two sub-sections, each inside its own scrollable region:
     - **NPCs** — deduplicated by `template`. Each row: 32px icon (`NpcImage` with `getAssetIconUrl(..., 'npc', template)` — reuse the exact pattern from the current NPCs tab), NPC name as a `Link` to `/npcs/{template}`. Heading "NPCs ({distinctCount})" above the list. Empty state: italic "No NPCs".
     - **Monsters** — deduplicated by `template`. Each row: 32px icon (`getAssetIconUrl(..., 'mob', template)`), monster name as a `Link` to `/monsters/{template}`, and a muted "×N" suffix where N is the occurrence count in the `monsters[]` array. Heading "Monsters ({distinctCount})" above the list. Empty state: italic "No monsters".
4. **Connected-maps row** — a horizontally-scrolling flex row of `Card`-style widgets, one per **distinct non-NONE target map**:
   - Source: `portals.map(p => p.attributes.targetMapId)` filtered to exclude `999999999` (NONE), deduplicated.
   - Each widget is a `Link` to `/maps/{targetMapId}` rendering a compact card containing only the target map's **name** (resolved via `useMap(targetMapId)` or the existing `MapCell` component). Width is fixed per widget (e.g., `w-48`) so the row scrolls horizontally on overflow rather than wrapping.
   - Heading "Connected maps ({count})" above the row. If there are zero valid targets, omit the entire section (heading included).
   - Loading: show skeleton widgets while portals are still loading. Individual widget names load independently via `MapCell`'s existing cache.
5. **Detail tabs** — `Tabs` with three `TabsTrigger`s, in order: `portals`, `monsters`, `reactors`. The `npcs` tab is removed. Tab contents and tables are identical to today's implementation.

### 4.2 Progressive rendering

The page must render in this order regardless of which queries are in flight:

- `useMap` success → header + metadata badges render immediately. Spawn-count badge shows a skeleton until `useMapMonsters` resolves.
- Image panel renders immediately once `useMap` succeeds. The `<img>` fetches on its own; no React Query wait.
- Summary panel sub-sections render independently: NPCs appears as soon as `useMapNpcs` resolves; Monsters appears as soon as `useMapMonsters` resolves. Each shows its own skeleton independently.
- Connected-maps row renders as soon as `useMapPortals` resolves.
- Detail tabs render their loading states per-tab as today (unchanged).

If `useMap` fails or returns 404, render `ErrorDisplay` with a retry, as today. Failures in entity queries do **not** block the page — they render localized error messages inside their panel/tab.

### 4.3 Copy-to-clipboard tooltip

Reuse `TooltipContent copyable` exactly as it's used elsewhere (`services/atlas-ui/src/components/map-cell.tsx`, `services/atlas-ui/src/pages/MapDetailPage.tsx:173-184`). The tooltip payload is the numeric map ID as a string. No new component is introduced.

### 4.4 Connected-maps dedup and filtering

- Deduplicate by `targetMapId`. The first portal encountered in the array order determines ordering; no sorting by name.
- Filter out portals where `targetMapId === 999999999` (NONE) or `targetMapId` is falsy/missing.
- Filter out self-links (`targetMapId === mapId`) — a portal that loops back to the current map should not render a widget.

### 4.5 Phase 2: full-map composite renderer (atlas-wz-extractor)

`atlas-wz-extractor` gains a new extraction path that produces `map/{mapId}/render.png` for each map in `Map.wz`. The algorithm:

1. Parse the per-map `.img` for `back[]`, numbered layers `0..7` each with `info`, `tile[]`, `obj[]`, and `miniMap` / `VRLeft,VRTop,VRRight,VRBottom` to determine the canvas bounds.
2. Resolve canvas bounds: prefer `VRLeft/VRRight/VRTop/VRBottom`. If absent, fall back to `miniMap.width` / `miniMap.height` with `-centerX, -centerY` origin — same precedence already used in `services/atlas-data/atlas.com/data/map/reader.go:179-216`.
3. Allocate an `image.RGBA` the size of the resolved bounds (clamped to a per-tenant safety max — see NFRs).
4. In z-order, composite:
   - Backgrounds with `front=0`, honoring `type` (NORMAL, HORIZONTAL_TILE, VERTICAL_TILE, TILE_BOTH, H_SCROLL, V_SCROLL). Resolve image via `bS` (background set) + `no` against `Back.wz`.
   - Per layer 0..7, in order: tiles (resolved via the layer's `info/tS` against `Map.wz/Tile/{tS}.img` with `u` + `no`), then objects (resolved via `Map.wz/Obj/{oS}.img/l0/l1/l2/0` — `oS`, `l0`, `l1`, `l2` come from the object entry).
   - Backgrounds with `front=1` (foregrounds) last.
5. Apply per-sprite `a` (alpha) and `f` (flipX) if present.
6. Encode as PNG (default) or WebP (if `WZ_EXTRACT_RENDER_FORMAT=webp` env var set). Write to `{outputImgDir}/map/{mapId}/render.png`.

Scope and safety:
- Skip maps whose bounds exceed a configurable safety max (default `16384×16384`) — log and continue rather than OOM.
- Skip maps with no `back` / no layers (typically cash-shop or system maps) — emit no render, UI falls back to minimap.
- Skip on unresolved references (missing `bS`, missing tile set) with a warning — partial renders are acceptable.
- The renderer is a new subpackage under `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/` (siblings: `image/`, `wz/`). It reuses `wz/canvas.Decompress` for sprite decoding and `image/draw` for compositing — no new third-party dependencies.
- Foothold overlays and portal markers are explicitly **not** rendered in this task.

### 4.6 Phase 1: minimap extraction (interim image source)

Because Phase 2 is large, Phase 1 ships the UI against a smaller extractor change: extract each map's `miniMap/canvas` (already decoded by the existing `wz/canvas` package) and write `map/{mapId}/minimap.png`. This is a <50-LOC extractor that mirrors `extractEntityIcons` shape. The UI's image panel consumes whichever file exists — Phase 2 simply adds `render.png` and prefers it.

### 4.7 Asset URL contract

`src/lib/utils/asset-url.ts` is extended:

- Add `'map'` to the `category` union.
- Add an optional fourth component: current pattern is `{category}/{entityId}/icon.png`; for maps it becomes `{category}/{mapId}/render.png` **or** `{category}/{mapId}/minimap.png`.
- Rather than forking the function, add a sibling helper `getMapImageUrl(tenant, mapId, kind: 'render' | 'minimap')` that returns the full URL. Keep `getAssetIconUrl` unchanged for entity icons.

### 4.8 Entity pages stay linked correctly

- NPC summary rows link to `/npcs/{template}`. Verify that route accepts the numeric template as `:id` (existing `NpcDetailPage`).
- Monster summary rows link to `/monsters/{template}`. Verify the same for `MonsterDetailPage` / whatever route handles monsters — if the existing monsters route uses a different param, the summary must match.
- Portal rows (unchanged) already link via `/maps/{id}/portals/{portalId}`.

## 5. API Surface

No new REST endpoints in atlas-data. All map / entity data already flows through:
- `GET /api/data/maps/{id}` → `mapsService.getMapById` (unchanged)
- `GET /api/data/maps/{id}/portals` → `useMapPortals` (unchanged)
- `GET /api/data/maps/{id}/npcs` → `useMapNpcs` (unchanged)
- `GET /api/data/maps/{id}/monsters` → `useMapMonsters` (unchanged)
- `GET /api/data/maps/{id}/reactors` → `useMapReactors` (unchanged)

Static asset surface (served by atlas-assets nginx, no API layer):
- `GET /api/assets/{tenantId}/{region}/{version}/map/{mapId}/minimap.png` (Phase 1)
- `GET /api/assets/{tenantId}/{region}/{version}/map/{mapId}/render.png` (Phase 2)

No atlas-ui-side service-module changes beyond extending `asset-url.ts`.

## 6. Data Model

No database schema changes. No new entities. Map renders are filesystem artifacts under the existing atlas-assets shared volume, scoped by tenant path (`{tenantId}/{region}/{version}/`) exactly like existing entity icons.

The `monsters[]` array on a map already contains the data needed for the spawn-count badge and the deduplicated monster summary. No new fields on any REST model.

## 7. Service Impact

| Service | Change | Reason |
|---|---|---|
| **atlas-ui** | Rewrite `MapDetailPage.tsx`; new components under `components/features/maps/` (e.g., `MapImagePanel`, `MapEntitySummary`, `ConnectedMapsRow`); extend `lib/utils/asset-url.ts` with `getMapImageUrl`. | The entire redesign lives here. |
| **atlas-wz-extractor** | Phase 1: add minimap extraction to `image/extract.go` or a sibling file. Phase 2: new `mapimage/` package that composites backgrounds/tiles/objects and writes PNG output. Wire both into `extraction/processor.go` so they run as part of `ExtractIcons`. | The render is produced at extraction time and lands in the shared volume. |
| **atlas-assets** | No source changes — the nginx config already serves the shared volume. New path `/api/assets/.../map/{mapId}/*.png` is automatically reachable. | Static-serving container; no code. |
| **atlas-data** | No changes. | All needed data already exposed. |

No other services touched.

## 8. Non-Functional Requirements

**Performance (UI):**
- First meaningful paint (header + badges + image `<img>` tag mounted) must not wait on any entity query other than `useMap`.
- Summary panel sub-sections must render independently — an NPCs query that takes 800ms must not delay the Monsters section.
- Connected-maps widgets resolve names via `MapCell`'s existing `mapNameCache` — no N+1 round-trip explosion when a map has many distinct targets.

**Performance (extractor, Phase 2):**
- A single-map render must complete in <500ms on a developer laptop for a typical town map (~1500×800 canvas, ~200 tiles, ~50 objects).
- Full-tenant re-render (30k maps) must not exceed the existing `atlas-wz-extractor` upload SLA. If projected above budget, emit renders in parallel with XML serialization rather than sequentially.
- Per-map safety cap on canvas dimensions (default 16384×16384) to prevent OOM on malformed WZ data.

**Storage:**
- Expected footprint: PNG ~500KB–1MB per map × 30k maps ≈ 15–30GB per tenant-region-version. Document expected storage growth; volume sizing is an operational concern, not a code concern.
- WebP output is opt-in via `WZ_EXTRACT_RENDER_FORMAT=webp` env var (cuts storage ~60% at q85).

**Multi-tenancy:**
- Map renders are tenant-scoped via the existing asset path prefix. No cross-tenant leakage possible — the URL includes the tenant UUID.
- `getMapImageUrl` must read the active tenant from `TenantProvider` (same contract as `getAssetIconUrl`).

**Observability:**
- Phase 2 extractor emits a structured log line per rendered map with `mapId`, `dimensions`, `durationMs`, `spriteCount`. Failed renders log the reason and continue.
- No new metrics required for atlas-ui — the page uses existing React Query hooks whose errors surface to the existing error boundary.

**Accessibility:**
- The `<img>` tag must have a meaningful `alt` (`alt={"Map render for " + map.attributes.name}`) so screen readers don't announce it as "image".
- Connected-map widgets are `<Link>` elements with descriptive text; no icon-only interactive elements.
- Tooltip-as-copy must remain keyboard-accessible (existing `TooltipContent copyable` pattern already handles this — verify in testing).

**Security:**
- No new user-controlled input flows into file paths; the `mapId` is validated as numeric by the existing route.
- Asset URLs are already public within the tenant's network boundary — no change to the security posture.

## 9. Open Questions

- **Monsters detail route:** the current NPCs tab links to `/npcs/{template}`. The new monster summary needs to link to the equivalent monster page — confirm the route shape (`/monsters/:templateId` vs. `/mobs/:templateId`) during implementation and match whatever exists in `App.tsx`.
- **Connected-map widget visual density:** this PRD specifies "just the name." If widgets feel visually empty, a small map-icon placeholder or a tiny inline minimap thumbnail could be added in a follow-up without changing the data model.
- **Phase 2 render determinism:** do we commit to pixel-identical renders across extractor re-runs? Probably yes (deterministic sprite ordering, no random seeds), but worth explicit verification — it affects whether a re-upload churns the asset volume for unchanged maps.
- **Global (uuid.Nil) fallback for images:** entity icons fall back to global-tenant assets when tenant-scoped ones are missing. Should map renders do the same? Leaning yes for consistency, but the tenant/global file layout for `map/` needs a line in `render-pipeline.md` once implementation starts.

## 10. Acceptance Criteria

**Phase 1 (UI + minimap):**
- [ ] `MapDetailPage` renders the new layout: title (with copyable tooltip), badge row (street + spawn count), image-summary panel, connected-maps row, three-tab detail view.
- [ ] NPCs tab is removed from the tab strip; NPCs appear deduplicated in the summary panel linking to `/npcs/:template`.
- [ ] Monsters summary is deduplicated with `×N` counts linking to the monster detail route.
- [ ] Connected-maps row filters out NONE (`999999999`), self-links, and duplicate targets.
- [ ] Header + badges + image panel appear before entity queries resolve; each summary/tab section renders independently.
- [ ] `atlas-wz-extractor` writes `map/{mapId}/minimap.png` alongside existing NPC/mob/reactor icon output.
- [ ] The image panel resolves to the minimap URL and falls back to a placeholder on 404.
- [ ] Page route, breadcrumbs, and all existing deep links still work unchanged.
- [ ] No references to `maplestory.io` added to the UI for map images.

**Phase 2 (composite renderer):**
- [ ] `atlas-wz-extractor` writes `map/{mapId}/render.png` for every map with `back[]` + at least one layer.
- [ ] `getMapImageUrl` prefers `render.png` and falls back to `minimap.png` on missing.
- [ ] Visual spot-check of 20 representative maps (Henesys, Perion, Ellinia, Orbis, El Nath, Ludibrium towers, Aqua Road, Deep Ludi, a boss map, a PQ map) produces renders that are recognizably the map with correct background, tile, and object placement.
- [ ] Renderer logs per-map structured output; failures are logged and skipped without aborting the extraction run.
- [ ] Full-tenant extraction time increases by no more than 2× the pre-Phase-2 baseline.
