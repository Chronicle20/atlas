# Task-008 Map Detail Redesign — Task Checklist

**Last Updated: 2026-04-18**
**Status: Not started — awaiting implementation approval**

## Overview

Two-phase redesign of `MapDetailPage` + introduction of a first-party map-image pipeline in `atlas-wz-extractor`. See `task-008-map-detail-redesign-plan.md` for full plan, `task-008-map-detail-redesign-context.md` for file pointers and decisions.

---

## Phase 1 — UI Redesign + Minimap Source

### Phase 1A — Extractor minimap extraction

- [ ] **1A.1** Add `ExtractMinimap(ctx, mapEntry, outputDir)` in `services/atlas-wz-extractor/atlas.com/wz-extractor/image/` (new file or extend `extract.go`).
  - Decodes `miniMap/canvas` via `wz/canvas.Decompress`.
  - Writes `{outputImgDir}/map/{mapId}/minimap.png`.
  - Skips maps with no `miniMap` (debug-level log).
  - **Acceptance:** unit test covers with-minimap and without-minimap paths.

- [ ] **1A.2** Wire minimap extraction into `extraction/processor.go`.
  - Iterates `Map.wz/Map/Map{N}/*.img` during existing `ExtractIcons` (or equivalent).
  - Logs summary (`minimaps_extracted=N`).
  - **Acceptance:** tenant upload produces `map/{mapId}/minimap.png` for all maps with minimaps; no regression in icon extraction.

- [ ] **1A.3** Extractor tests + docker build green.
  - `go test ./...` passes.
  - `docker compose build atlas-wz-extractor` passes.

### Phase 1B — Frontend asset URL helper

- [ ] **1B.1** Extend `services/atlas-ui/src/lib/utils/asset-url.ts`.
  - Add `'map'` to `category` union.
  - Add sibling `getMapImageUrl(tenant, mapId, kind: 'render' | 'minimap')` returning the full static URL.
  - Do **not** modify `getAssetIconUrl`.
  - **Acceptance:** unit tests for both `kind` values; existing tests unchanged.

### Phase 1C — Frontend sub-components

- [ ] **1C.1** `MapHeader` — `src/components/features/maps/MapHeader.tsx`.
  - Title with `TooltipContent copyable` (mirror `map-cell.tsx:43-45`).
  - Badge row: `secondary` streetName (omit if empty), `outline` spawn count (`1 spawn` / `N spawns`, skeleton if undefined).
  - Remove the existing inline `#<id>` span.
  - **Acceptance:** visually verified loading / empty-street / singular-spawn states. Tooltip click copies ID.

- [ ] **1C.2** `MapImagePanel` — `src/components/features/maps/MapImagePanel.tsx`.
  - `<img loading="lazy">` inside `Card`. Meaningful `alt`.
  - Phase 1 src: `getMapImageUrl(..., 'minimap')`.
  - `onError` swap to placeholder (`<MapIcon />` + "No render available") — single swap, no loop.
  - Reads tenant/region/version from `TenantProvider`.
  - **Acceptance:** real map renders minimap; no-minimap map shows placeholder; no broken-image icon.

- [ ] **1C.3** `MapEntitySummary` — `src/components/features/maps/MapEntitySummary.tsx`.
  - NPCs sub-section: dedup by `template`, 32px `npc` icon, link to `/npcs/{template}`, heading `NPCs ({count})`, empty state "No NPCs".
  - Monsters sub-section: dedup by `template`, 32px `mob` icon, link to `/monsters/{template}` (**verify route in `App.tsx` — see PRD §9**), `×N` suffix, heading `Monsters ({count})`, empty state "No monsters".
  - Each sub-section independent loading skeleton, capped ~400px scroll height.
  - **Acceptance:** Henesys data renders deduped lists matching UX spec. Monster link target confirmed.

- [ ] **1C.4** `ConnectedMapsRow` — `src/components/features/maps/ConnectedMapsRow.tsx`.
  - Dedup logic per `ux-flow.md` snippet: filter `999999999`, falsy, and self-links; preserve insertion order.
  - Horizontally-scrolling `w-48` card widgets; each `Link` to `/maps/{targetMapId}` using `MapCell` for name resolution.
  - Heading `Connected maps ({count})`. Whole section omitted if zero valid targets.
  - Skeleton widgets during loading.
  - **Acceptance:** mixed-portal test map renders only valid targets, no duplicates, horizontal scroll on overflow.

- [ ] **1C.5** `MapDetailTabs` — `src/components/features/maps/MapDetailTabs.tsx` (or inline).
  - Three `TabsTrigger`s: portals (default), monsters, reactors. NPCs tab removed.
  - Tab tables identical to today's rendering.
  - **Acceptance:** tables unchanged from today; deep-link `/maps/:id/portals/:portalId` still works.

### Phase 1D — Page rewrite + integration

- [ ] **1D.1** Rewrite `src/pages/MapDetailPage.tsx`.
  - Compose five sub-components in top-to-bottom order inside existing `p-10 pb-16`.
  - Hook wiring per plan §5 Phase 1D.1.
  - `useMap` error → full-page `ErrorDisplay` with retry. Other errors → inline per `ux-flow.md` error table.
  - Delete NPC-tab rendering code.
  - **Acceptance:** progressive render timeline from `ux-flow.md` observable in Network-throttled devtools. All existing deep-links still work.

- [ ] **1D.2** Responsive grid — `md:grid-cols-[2fr_1fr]` desktop, stacked mobile.
  - **Acceptance:** resize 375 → 1440; layout collapses, connected-maps row remains horizontally scrollable on touch.

- [ ] **1D.3** Accessibility pass — meaningful `alt`, keyboard-reachable tooltip-copy, no icon-only clickables.
  - **Acceptance:** tab-traverse reaches every interactive element with meaningful announcement.

### Phase 1E — Verification

- [ ] **1E.1** Manual browser walkthrough of 5 representative maps (Henesys, Perion, zero-NPC, zero-monster, loop-back portal).
  - **Acceptance:** all PRD §10 Phase-1 checkboxes pass.

- [ ] **1E.2** `rg 'maplestory\.io' services/atlas-ui/src/` returns zero matches for map images.

- [ ] **1E.3** `pnpm build` + `pnpm typecheck` clean; existing tests green.

---

## Phase 2 — Full-Map Composite Renderer

### Phase 2A — Promote spike to subpackage

- [ ] **2A.1** Create `services/atlas-wz-extractor/atlas.com/wz-extractor/mapimage/` with files:
  - `renderer.go` — public `Render(ctx, mapEntry, wzIndex, outDir, opts) error`.
  - `bounds.go` — mirror `atlas-data/map/reader.go:179-216`.
  - `index.go` — `buildBackIndex`, `buildTileIndex`, `buildObjIndex` against single `*wz.File`.
  - `background.go` — `drawBackground` + `BackgroundType` enum.
  - `blit.go` — `blit` primitive + per-sprite flip cache.
  - `sort.go` — stable sort keys.
  - Std lib only.
  - **Acceptance:** package builds; renders a fixture identical to spike output byte-for-byte.

- [ ] **2A.2** Unit + golden-PNG tests on `mapimage/`.
  - Bounds resolution, tiling step, sort stability, flip cache.
  - Byte-compare two renders of the same fixture (determinism).
  - **Acceptance:** `go test ./mapimage/...` green.

- [ ] **2A.3** Remove or `README.md`-ify `cmd/map-render-spike/`.

### Phase 2B — Wire into extraction pipeline

- [ ] **2B.1** Invoke `mapimage.Render` from `extraction/processor.go` per map.
  - `runtime.NumCPU()` worker pool.
  - Output at `.../map/{mapId}/render.png`.
  - **Acceptance:** real tenant upload produces `render.png` for every map with `back[]` + ≥1 layer.

- [ ] **2B.2** Safety caps + skip-on-error.
  - Env var `WZ_EXTRACT_MAX_MAP_PIXELS` (default `16384*16384`).
  - Skip empty maps (no back, no layers) — debug log.
  - Renderer errors logged + skipped, batch continues.
  - **Acceptance:** malformed fixture does not crash extraction; structured warning present.

- [ ] **2B.3** Structured per-map log line: `mapId`, `width`, `height`, `durationMs`, `spriteCount`, `output=render.png`.
  - **Acceptance:** logs greppable by `mapId`.

- [ ] **2B.4** **Optional:** WebP output via `WZ_EXTRACT_RENDER_FORMAT=webp`.
  - Only if it does not require a new Go dependency. If skipped, record decision in context doc.

### Phase 2C — UI fallback chain

- [ ] **2C.1** Extend `MapImagePanel` with fallback state machine: `render.png` → `minimap.png` → placeholder.
  - Keep `getMapImageUrl` single-URL; fallback logic lives in the component.
  - **Acceptance:** map with render uses render; missing render falls back to minimap; both missing → placeholder. Exactly one swap per step.

- [ ] **2C.2** UI verification: run extractor locally, reload UI, spot-check 20 representative maps from PRD §10 Phase 2.3.
  - **Acceptance:** 20/20 recognizable. Unrecognizable ones filed as Phase 2.1 follow-ups.

### Phase 2D — Verification

- [ ] **2D.1** Baseline + post-Phase-2 extraction timings; Phase 2 ≤ 2× baseline.

- [ ] **2D.2** `docker compose build atlas-wz-extractor atlas-ui` both pass; existing tests green.

- [ ] **2D.3** Walk PRD §10 Phase-2 checkboxes; all marked done or converted to follow-up.

### Phase 2E — Follow-ups (not blocking)

- [ ] **2E.1** Horizon-seam fix (Phase 2.1). Open as separate task after Phase 2 ships.
- [ ] **2E.2** `front=1` foreground validation when first real map triggers the path.

---

## Sign-off

- [ ] Phase 1 reviewed + merged.
- [ ] Phase 2 reviewed + merged.
- [ ] Follow-up tickets filed for 2E items.
- [ ] This task is tracked under `docs/tasks/` (the old dev-active convention was retired in task-016).
