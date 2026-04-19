# Task-009 Map Image Overlays — Task Checklist

**Last Updated: 2026-04-18**
**Status: Implemented — pending manual verification on Henesys/Perion/Ellinia (G.1–G.3)**

## Overview

Add an interactive overlay layer on top of `MapImagePanel` with bidirectional hover-to-highlight between markers, summary cards, and detail-tab rows. Single phase, broken into vertical slices. See `plan.md` for full plan and `context.md` for file pointers and the `mapArea` discovery.

---

## Phase A — atlas-data nullable mapArea

- [x] **A.1** Pointer-ize `RestModel.MapArea` in `services/atlas-data/atlas.com/data/map/rest.go:27`.
  - Change to `MapArea *RectangleRestModel \`json:"mapArea"\``.
  - Update domain model + builder for nilable field per immutable-model pattern.
  - Update transformer between domain and REST to pass `nil` through.
  - **Acceptance:** `go build ./...` clean.

- [x] **A.2** Update `getMapArea()` in `reader.go:179-216` to return `*RectangleRestModel`.
  - VR-present and miniMap-present branches return `&RectangleRestModel{...}`.
  - The `1<<18` synthetic fallback (`reader.go:198-204`) returns `nil`.
  - Update reader call site to accept `nil`.
  - **Acceptance:** Test fixture confirms nil for no-bounds map, non-nil for VR map and miniMap-only map.

- [x] **A.3** Audit other `MapArea` consumers.
  - `grep -rn "MapArea" services/atlas-data/` — nil-check every reader.
  - **Acceptance:** atlas-data `go test ./...` green.

- [x] **A.4** Docker build atlas-data.
  - **Acceptance:** `docker compose build atlas-data` passes.

---

## Phase B — Frontend type + helper plumbing

- [x] **B.1** Extend `MapAttributes` in `services/atlas-ui/src/services/api/maps.service.ts`.
  - Add optional nullable `mapArea?: { x, y, width, height } | null`.
  - **Acceptance:** `pnpm typecheck` clean. Network shape unchanged (already on wire).

- [x] **B.2** Add `worldToOverlayPercent` helper.
  - New file `src/lib/utils/map-overlay.ts` exporting `MapBounds` type and the helper.
  - Pure function returning `{ left: '<n>%', top: '<n>%' }`.
  - **Acceptance:** Unit test in `__tests__/map-overlay.test.ts` covering: positive origin, negative origin (miniMap), exact-origin entity, far-corner entity, out-of-bounds entity.

---

## Phase C — Hover context

- [x] **C.1** Create `HoverHighlightContext`.
  - New file `src/components/features/maps/HoverHighlightContext.tsx`.
  - `HoverTarget` discriminated union per PRD §4.5 (portal/monster/reactor/npc, with `spawnIndex` for monster/npc).
  - `HoverHighlightProvider` owns `useState<HoverTarget>`.
  - `useHoverHighlight()` returns `{ hovered, setHovered, isHovered(target) }`. `isHovered` encodes the matching rules from PRD §4.6.
  - **Acceptance:** Unit test covering all four kinds; per-template rule matches across spawn indices; portal/reactor rules require exact id match.

- [x] **C.2** Provide context at `MapDetailPage`.
  - Wrap the page body (everything below `MapHeader`) in `<HoverHighlightProvider>`.
  - **Acceptance:** Page renders unchanged; provider visible in React DevTools.

---

## Phase D — Overlay layer + container refactor

- [x] **D.1** `MapImageOverlay` component.
  - New file `src/components/features/maps/MapImageOverlay.tsx`.
  - Props: `bounds: MapBounds`, optional `portals/npcs/monsters/reactors` arrays.
  - Renders `<div className="absolute inset-0 pointer-events-none">` containing per-marker `<button>` children.
  - Each marker positioned via `worldToOverlayPercent` + `transform: translate(-50%, -50%)`, `pointer-events-auto`, `aria-label`, `onPointerEnter`/`onPointerLeave` wired to `useHoverHighlight`, highlight styling per PRD §4.3.
  - Wrap each marker in `Tooltip` with `TooltipContent` showing entity name; tooltip opens only on direct hover (not on row-driven highlight).
  - DOM order: monsters → reactors → npcs → portals (later children paint on top).
  - Memoize derived position list per entity array via `useMemo` keyed on `bounds` + arrays.
  - **Acceptance:** Visual check on Henesys: markers placed at sensible coordinates.

- [x] **D.2** Refactor `MapImagePanel` container sizing.
  - Add `mapArea: MapBounds | null` and entity-array props (portals/npcs/monsters/reactors).
  - When `mapArea` non-null AND `state === "render"`: wrap `<img>` and `<MapImageOverlay>` in `<div className="relative w-full max-h-[320px]" style={{ aspectRatio: \`${mapArea.width} / ${mapArea.height}\` }}>` with `<img className="w-full h-full object-cover">`.
  - Otherwise: existing `object-contain max-h-[320px]` sizing preserved; no overlay.
  - Pass entity arrays from `MapDetailPage` into the panel.
  - **Acceptance:** Render-state with bounds shows aligned markers; minimap/placeholder/no-bounds states render identical to today.

- [ ] **D.3** Cross-service invariant verification.
  - Spot-check Henesys, Perion, Ellinia: `img.naturalWidth/Height === mapArea.width/height`.
  - **Acceptance:** 3/3 match. If any fail, file follow-up and switch to `naturalWidth/Height`-based denominator.

- [x] **D.4** Dev-only out-of-bounds warning.
  - In `MapImageOverlay`, when `import.meta.env.DEV`, log `console.warn` once per render listing entities whose computed percentages fall outside `[0%, 100%]`.
  - **Acceptance:** Warning visible in dev for a deliberately-out-of-bounds fixture; absent in production build.

---

## Phase E — Bidirectional hover wiring

- [x] **E.1** Summary panel row hover.
  - In `MapEntitySummary.tsx`, add `onPointerEnter/Leave` handlers to NPC and Monster rows. Set `hovered` per-template.
  - Apply highlight styling: `bg-muted/60` + 2px left border in entity color when `isHovered(rowTarget)`.
  - **Acceptance:** Hovering a summary row highlights matching markers on the image; clears on leave.

- [x] **E.2** Detail-tab row hover.
  - In `MapDetailTabs.tsx`, add hover handlers to Portals (per-portalId), Monsters (per-spawn with `spawnIndex`), and Reactors (per-reactorId) table rows.
  - Apply identical highlight styling (color matches entity).
  - **Acceptance:** Per PRD §4.6 matching rules: portal/reactor hover highlights one marker; monster row hover highlights its specific marker AND all template siblings (template-only hover from summary highlights all per-spawn rows).

- [ ] **E.3** Marker → row highlight verification.
  - With overlay shipped, manually confirm: marker hover highlights summary row; switching to a tab and hovering markers highlights matching table rows.
  - **Acceptance:** Both directions work end-to-end.

---

## Phase F — Expanded dialog overlay

- [x] **F.1** Overlay inside dialog.
  - Inside `MapImagePanel`'s `<Dialog>` content, wrap natural-size `<img>` and `<MapImageOverlay>` in a `relative` container with `aspectRatio` style.
  - Marker CSS sizing stays at 10×10 px so they appear smaller relative to the giant image.
  - **Acceptance:** Markers align inside dialog; hover works; tooltips open.

- [x] **F.2** Clear hover on dialog close.
  - In the `Dialog onOpenChange` handler, when transitioning open → closed, call `setHovered(null)`.
  - **Acceptance:** Open → hover marker → close dialog → page underneath has no stale highlight.

---

## Phase G — Verification

- [ ] **G.1** Cross-service invariant on three sample maps.
  - Henesys (100000000), Perion (102000000), Ellinia (101000000).
  - DevTools: `img.naturalWidth/Height` equals `mapArea.width/height`. Markers visually align with map content.
  - **Acceptance:** 3/3 pass.

- [ ] **G.2** Manual walkthrough of PRD §10 acceptance criteria.
  - Backend, Frontend overlays, Hover coordination, Touch behavior, Expanded dialog, Quality/regression — every checkbox.
  - **Acceptance:** All ticked.

- [ ] **G.3** Touch-device sanity.
  - DevTools touch emulation OR real mobile browser. Confirm markers render, taps don't break the page or stick highlights.
  - **Acceptance:** No errors; no stuck highlight; no scroll hijack.

- [x] **G.4** Builds + type-check + tests.
  - `pnpm build`, `pnpm typecheck`, `pnpm test` clean in atlas-ui.
  - `go build ./...`, `go test ./...` clean in atlas-data.
  - `docker compose build atlas-data atlas-ui` both pass.
  - **Acceptance:** Green local pipeline.

- [ ] **G.5** No task-008 regressions.
  - Re-walk task-008 PRD §10 Phase 1 + Phase 2 checkboxes.
  - **Acceptance:** All previously-shipped behavior intact.

---

## Effort Summary

- Phase A (Go): ~0.5 day
- Phase B (TS types/helper): ~0.25 day
- Phase C (context): ~0.25 day
- Phase D (overlay + container): ~1–1.5 day
- Phase E (hover wiring): ~0.5–1 day
- Phase F (dialog overlay): ~0.25 day
- Phase G (verification): ~0.5 day

**Total: 3–4 dev-days.** Single PR or two (A separately, then B–G).

---

## Sign-off

Mark each phase complete by the date it ships:

- [ ] Phase A complete (date: ____)
- [ ] Phase B complete (date: ____)
- [ ] Phase C complete (date: ____)
- [ ] Phase D complete (date: ____)
- [ ] Phase E complete (date: ____)
- [ ] Phase F complete (date: ____)
- [ ] Phase G complete (date: ____)
- [ ] PR merged (date: ____)
