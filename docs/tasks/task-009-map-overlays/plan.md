# Task-009 Map Image Overlays — Implementation Plan

**Last Updated: 2026-04-18**
**Status: Ready for implementation (post-plan, pre-code)**
**Source PRD:** `docs/tasks/task-009-map-overlays/prd.md`
**Companion docs:** `docs/tasks/task-009-map-overlays/ux-flow.md`
**Builds on:** `docs/tasks/task-008-map-detail-redesign/` (the redesigned `MapDetailPage` and the `render.png` produced by `atlas-wz-extractor`)

---

## 1. Executive Summary

Add an interactive overlay layer on top of `MapImagePanel` that draws colored markers for every portal, monster spawn, reactor spawn, and NPC at their world-space coordinates, plus bidirectional hover-to-highlight wiring between the markers, the summary cards in `MapEntitySummary`, and the table rows in `MapDetailTabs`. Both the inline preview and the expanded `Dialog` viewer render the overlay; touch devices show markers but skip hover semantics.

Scope is dramatically smaller than originally drafted: atlas-data **already** exposes a `mapArea` rectangle on `GET /maps/:id` (`services/atlas-data/atlas.com/data/map/rest.go:27`), so no schema change, no migration, and no atlas-wz-extractor change are required. The work is overwhelmingly frontend.

The only backend touch is a small atlas-data refinement to make `mapArea` *unambiguously* nullable — the current implementation emits a `1 << 18` "huge box" fallback when neither `VR*` nor `miniMap` is present (`reader.go:198-204`). The UI needs to distinguish that from real bounds. Cleanest fix: pointer-ize `MapArea` so it's `nil`/`null` in the JSON when no real source resolved.

Ship as a single phase — there is no benefit to splitting given the small surface.

---

## 2. Current State Analysis

### 2.1 Backend — `services/atlas-data/atlas.com/data/map/`

- `rest.go:27` — `RestModel.MapArea RectangleRestModel \`json:"mapArea"\`` is already on the wire as a value type (always emitted).
- `rest.go:269` — `RectangleRestModel { X, Y, Width, Height int16 }`.
- `reader.go:179-216` — `getMapArea()` resolves bounds with the precedence: `VR*` → `miniMap` (with `-centerX/-centerY` origin) → `1 << 18` fallback. **The 1<<18 fallback is the case the UI needs to treat as "no bounds available."**
- `model.go` — domain model holds the area as a value `RectangleModel`. Builder pattern in place per Atlas conventions.

### 2.2 Frontend — `services/atlas-ui/src/`

- `pages/MapDetailPage.tsx` — composes `MapHeader`, `MapImagePanel`, `MapEntitySummary`, `ConnectedMapsRow`, `MapDetailTabs`. Hooks: `useMap`, `useMapPortals`, `useMapNpcs`, `useMapMonsters`, `useMapReactors`.
- `components/features/maps/MapImagePanel.tsx` — `Card` wrapping an `<img>` with `object-contain` + `max-h-[320px]`, plus an expanded `Dialog` view at natural size. State machine: `"render" | "minimap" | "placeholder"` with `onError` cascade.
- `components/features/maps/MapEntitySummary.tsx` — deduped NPC and Monster lists (per-template).
- `components/features/maps/MapDetailTabs.tsx` — Portals/Monsters/Reactors tabs with per-spawn rows.
- `services/api/maps.service.ts` — `MapAttributes { name, streetName }`. `getMapById` does **not** apply a sparse fieldset, so the network response already includes `mapArea` — only the TS type omits it.
- `services/api/map-entities.service.ts` — entity types include world-space `x` and `y` on every entity, ready to feed into the transform.

### 2.3 Asset path contract

- `render.png` is produced by `atlas-wz-extractor` at the world-rect dimensions resolved by the same VR/miniMap precedence. By construction, `render.png` pixel dims === `mapArea.width × mapArea.height` for any map with a real source. Verification step in §5 confirms on real data.

### 2.4 Known constraints

- No new dependencies (Go or npm) — both projects already include everything needed.
- Hover state must not trigger React Query refetches.
- Visual additions are layered on top of task-008 — must not regress its behavior.

---

## 3. Proposed Future State

### 3.1 User-visible

- Operator opens `/maps/:id` → sees the map render with colored markers overlaying every portal (emerald diamond), NPC (sky circle), monster spawn (rose dot), and reactor (amber square).
- Hovering a marker:
  - opens a tooltip with the entity name,
  - highlights all sibling markers (per-template for NPCs/monsters, exact for portals/reactors),
  - highlights the matching row in the summary card and detail tab.
- Hovering a summary or detail row highlights the matching marker(s); no tooltip is opened.
- Expanded `Dialog` view supports the same overlay + hover.
- Falls back gracefully: when `mapArea` is null, when the panel is in `"minimap"` or `"placeholder"` state, or on touch — markers are hidden / non-interactive without breaking the page.

### 3.2 Technical

- atlas-data: pointer-ize `RestModel.MapArea` so it's `*RectangleRestModel` and emits `null` when bounds resolution falls back to the `1<<18` sentinel. Reader returns nil in that branch instead of the synthetic rectangle.
- atlas-ui:
  - `MapAttributes` extended with `mapArea: { x: number; y: number; width: number; height: number } | null`.
  - New helper `lib/utils/map-overlay.ts` exposing `worldToOverlayPercent`.
  - New `components/features/maps/HoverHighlightContext.tsx` — context + provider + `useHoverHighlight` hook.
  - New `components/features/maps/MapImageOverlay.tsx` — absolute-positioned marker layer.
  - `MapImagePanel` updated: when `mapArea` is present and state is `"render"`, container uses `aspectRatio` + `object-cover` so percentage-positioned markers land on the image's content rect; overlay mounted as a sibling `<div>`. All other states behave exactly as today.
  - `MapEntitySummary` and `MapDetailTabs` row elements gain `onPointerEnter`/`onPointerLeave` handlers wired to `useHoverHighlight`.
  - `MapDetailPage` wraps everything below `MapHeader` in `<HoverHighlightProvider>`.
- No new tests required for atlas-wz-extractor. No atlas-assets change. No new endpoints anywhere.

### 3.3 Non-functional

- No additional network requests — `mapArea` piggybacks on the existing `useMap` query.
- Hover updates are O(1) context state; sibling re-render scope is contained to overlay markers and the row receiving the highlight.
- Dev-only `console.warn` when an entity falls outside `mapArea` — useful for catching renderer/bounds drift, suppressed in production builds via `import.meta.env.DEV`.

---

## 4. Implementation Phases

Single phase, broken into vertical slices ordered for early integration:

- **Phase A — atlas-data nullable mapArea** (backend, smallest piece)
- **Phase B — Frontend type + helper plumbing** (types + pure helper, no UI change)
- **Phase C — Hover context** (cross-cutting state, no visual change yet)
- **Phase D — Overlay layer + container refactor** (visible markers appear)
- **Phase E — Bidirectional hover wiring** (summary + tab rows participate)
- **Phase F — Expanded dialog overlay** (overlay in the full-size view)
- **Phase G — Verification** (cross-service invariant + manual walkthrough + builds)

A reviewer can ship after Phase D and have a useful page (markers visible, no row highlights). Phases E and F complete the feature.

---

## 5. Detailed Tasks

> Effort scale: **S** ≤ 2 hr, **M** ≤ 1 day, **L** ≤ 2 days. Acceptance criteria are testable. Detailed checklist in `tasks.md`.

### Phase A — atlas-data nullable mapArea

**A.1 Pointer-ize `RestModel.MapArea`** *(Effort: S)*
- Change `services/atlas-data/atlas.com/data/map/rest.go:27` to `MapArea *RectangleRestModel \`json:"mapArea"\`` (the field already serializes — only the type changes).
- Update the corresponding domain model field and builder to be nilable per the immutable-model pattern.
- Update the transformer between domain and REST model to pass through `nil`.
- **Acceptance:** struct compiles, no other callers break (verify with `go build ./...`).

**A.2 Update `getMapArea()` to return nil on the synthetic fallback** *(Effort: S)*
- `reader.go:179-216`: change return type to `*RectangleRestModel`.
- The VR-present and miniMap-present branches return `&RectangleRestModel{...}`.
- The `1 << 18` synthetic fallback (`reader.go:198-204`) returns `nil`.
- Update the call site in the reader that assigns to the model so it accepts `nil`.
- **Acceptance:** A unit test (or manual fixture run) covering: (a) a map with VR returns non-nil bounds, (b) a map with only miniMap returns non-nil bounds with `-centerX/-centerY` origin, (c) a map with neither returns nil.

**A.3 Verify other atlas-data consumers of MapArea** *(Effort: S)*
- `grep -rn "MapArea" services/atlas-data` to find every reader of the field.
- Any consumer that previously relied on the always-non-nil contract gets updated to nil-check (or kept on the synthetic fallback if it actually needs a non-nil rect — most likely none do).
- **Acceptance:** `go build ./...` clean for atlas-data; existing tests green.

**A.4 Docker build** *(Effort: S)*
- `docker compose build atlas-data` passes.
- **Acceptance:** image builds without error.

---

### Phase B — Frontend type + helper plumbing

**B.1 Extend `MapAttributes`** *(Effort: S)*
- `services/atlas-ui/src/services/api/maps.service.ts`:
  ```ts
  export interface MapAttributes {
    name: string;
    streetName: string;
    mapArea?: { x: number; y: number; width: number; height: number } | null;
  }
  ```
- The wire format already includes `mapArea`; this is a pure type extension.
- **Acceptance:** `pnpm typecheck` clean; no value-shape changes.

**B.2 Add `worldToOverlayPercent` helper** *(Effort: S)*
- New file `services/atlas-ui/src/lib/utils/map-overlay.ts`:
  ```ts
  export interface MapBounds { x: number; y: number; width: number; height: number; }
  export function worldToOverlayPercent(wx: number, wy: number, b: MapBounds): { left: string; top: string };
  ```
- Pure function; no React, no DOM.
- **Acceptance:** Unit test in `__tests__/map-overlay.test.ts` covering: positive origin, negative origin (miniMap case), entity at exact origin, entity at far corner, entity outside bounds (still computes — does not throw).

---

### Phase C — Hover context

**C.1 Create `HoverHighlightContext`** *(Effort: S)*
- New file `services/atlas-ui/src/components/features/maps/HoverHighlightContext.tsx`:
  - `HoverTarget` discriminated union per PRD §4.5.
  - `HoverHighlightProvider` wraps children, owns `useState<HoverTarget>`.
  - `useHoverHighlight()` hook returns `{ hovered, setHovered, isHovered(target) }` where `isHovered` encodes the matching rules from PRD §4.6.
- **Acceptance:** Unit test covering matching rules: per-template monster `hovered` matches markers regardless of `spawnIndex`; portal hover matches only the exact `portalId`; null `hovered` matches nothing.

**C.2 Provide context at `MapDetailPage`** *(Effort: S)*
- Wrap the existing children of `MapDetailPage` (everything except `MapHeader` ideally — header has nothing to highlight) in `<HoverHighlightProvider>`.
- **Acceptance:** Page renders unchanged; provider is in tree (verify via React DevTools).

---

### Phase D — Overlay layer + container refactor

**D.1 `MapImageOverlay` component** *(Effort: M)*
- New file `services/atlas-ui/src/components/features/maps/MapImageOverlay.tsx`.
- Props: `bounds: MapBounds`, `portals?, npcs?, monsters?, reactors?` (each optional — undefined entity arrays simply contribute no markers).
- Renders `<div className="absolute inset-0 pointer-events-none">` with absolutely-positioned marker `<button>` children.
- Each marker:
  - position via `worldToOverlayPercent(entity.x, entity.y, bounds)` + `transform: translate(-50%, -50%)`,
  - `pointer-events-auto`, `aria-label` describing the entity,
  - `onPointerEnter` calls `setHovered` to the most-specific identity,
  - `onPointerLeave` calls `setHovered(null)`,
  - applies highlight styling via `isHovered(target)`,
  - shadcn `Tooltip` wrapping the marker; `TooltipContent` shown only on direct marker hover (use the `Tooltip` `open` controlled prop tied to `hovered.spawnIndex` matching this marker's index — alternatively rely on `Tooltip` default behavior since hover and pointer-enter coincide).
- Z-order in DOM: monsters → reactors → npcs → portals (later children paint on top).
- Memoize derived position lists via `useMemo` keyed on `bounds` + entity arrays.
- **Acceptance:** Visual check on Henesys: markers appear at sensible locations (NPCs near building doorways, monsters spread across the ground, portals at edges).

**D.2 Refactor `MapImagePanel` container sizing** *(Effort: M)*
- When `mapArea` is present **and** `state === "render"`:
  - Wrap `<img>` and `<MapImageOverlay>` in a single `<div style={{ aspectRatio: \`${mapArea.width} / ${mapArea.height}\` }} className="relative w-full max-h-[320px]">`.
  - `<img>` becomes `className="w-full h-full object-cover"` inside that wrapper.
- When `mapArea` is null OR `state` is `"minimap"` / `"placeholder"`:
  - Existing `object-contain max-h-[320px]` sizing is preserved unchanged.
  - Overlay is not rendered.
- Add a `mapArea` prop to `MapImagePanel` (or read it via the same `useMap` query — props is cleaner since the panel currently receives `mapId` only and re-querying would be wasteful).
- Pass entity arrays into the panel as well so the overlay can render. Easiest: hoist the four `useMap*` queries' data into the panel props from `MapDetailPage`.
- **Acceptance:** With `mapArea` non-null and `state="render"`: markers and image align (verified by spot-check on Henesys, Perion, Ellinia). With `mapArea` null: panel renders identical to today; no overlay; no layout regression.

**D.3 Cross-service invariant verification** *(Effort: S)*
- Pick three maps with non-null `mapArea`. For each, verify in browser/DevTools that the rendered image's `naturalWidth × naturalHeight` equals `mapArea.width × mapArea.height`.
- If they don't match, the `aspectRatio` container will look correct but markers will be offset. Document the failure mode and either: (a) fix on the extractor side (likely a bounds-resolution drift), or (b) introduce a `naturalWidth`-based scaling fallback.
- **Acceptance:** 3/3 sampled maps match. If any mismatch, file a follow-up and use option (b) for shipping.

**D.4 Dev-only out-of-bounds warning** *(Effort: S)*
- In `MapImageOverlay`, after computing positions, if `import.meta.env.DEV` and any entity's percentage falls outside `[0%, 100%]`, `console.warn` once per render with the offending entity ids.
- **Acceptance:** Dev console shows the warning when an entity is outside; production build does not.

---

### Phase E — Bidirectional hover wiring

**E.1 Summary panel row hover** *(Effort: S)*
- `MapEntitySummary.tsx`: each NPC row gets `onPointerEnter={() => setHovered({ kind: "npc", template })}` / `onPointerLeave={() => setHovered(null)}`. Same shape for monster rows with `kind: "monster"`. Use `useHoverHighlight()`.
- Apply highlight styling: when `isHovered(rowTarget)` is true, add `bg-muted/60` + a 2px left border in the entity's color.
- **Acceptance:** Hovering a summary row highlights its matching markers on the image (verified visually). Mouse leave clears.

**E.2 Detail-tab row hover** *(Effort: M)*
- `MapDetailTabs.tsx` (Portals / Monsters / Reactors tables):
  - Portals row: `setHovered({ kind: "portal", portalId: row.id })`.
  - Monsters row (per-spawn): `setHovered({ kind: "monster", template: row.attributes.template, spawnIndex: i })`.
  - Reactors row: `setHovered({ kind: "reactor", reactorId: row.id })`.
- Highlight styling identical to summary rows (left accent border in the entity's color + `bg-muted/60`).
- **Acceptance:** Hovering each row type highlights the matching marker(s) per the rules in PRD §4.6. Pointer leave clears.

**E.3 Marker → row highlight verification** *(Effort: S)*
- With overlay markers present (Phase D shipped), confirm hovering a marker also highlights the row in the summary card. Detail tab rows light up only when their tab is open (the rows aren't mounted otherwise — no special handling needed).
- **Acceptance:** Manual walkthrough on a representative map: marker hover → summary row highlights; switch to a detail tab, marker hover → table row highlights.

---

### Phase F — Expanded dialog overlay

**F.1 Overlay inside dialog** *(Effort: S)*
- Inside `MapImagePanel`'s `<Dialog>` content, wrap the natural-size `<img>` and a `<MapImageOverlay>` in a similar `relative` container. Use `style={{ aspectRatio: ... }}` and let the image render at intrinsic size; the overlay sits above using the same percentage positioning.
- Marker visual sizing in the dialog: keep at 10×10 px (CSS `width`/`height`). They will appear smaller relative to the giant image, which is desirable — matches "I can see them but they don't dominate."
- **Acceptance:** Dialog shows markers correctly positioned; hover works; closing the dialog clears `hovered`.

**F.2 Clear hover on dialog close** *(Effort: S)*
- In `MapImagePanel`'s `Dialog onOpenChange` handler, when transitioning from open to closed, call `setHovered(null)` to avoid a stale highlight on the underlying page.
- **Acceptance:** Open dialog → hover a marker (page underneath highlights too because they share context) → close dialog → highlight clears.

---

### Phase G — Verification

**G.1 Cross-service invariant test** *(Effort: S)*
- Spot-check on Henesys (100000000), Perion (102000000), Ellinia (101000000): browser DevTools shows `img.naturalWidth/Height === mapArea.width/height` for each. Markers align with map content (NPC at the door of a building, portal at a screen edge).
- **Acceptance:** Pass for all three.

**G.2 Manual walkthrough** *(Effort: S)*
- Walk through PRD §10 acceptance criteria on a map with all four entity types (Henesys is ideal — has portals, NPCs, monsters in adjacent areas, and at least one reactor or use a hunting ground).
- **Acceptance:** Every checkbox in PRD §10 passes.

**G.3 Touch-device sanity** *(Effort: S)*
- DevTools touch-emulation OR a real mobile browser: confirm markers render but tapping does nothing weird (no error, no stuck highlight, no scroll hijack).
- **Acceptance:** Touch interactions don't break the page. Hover semantics absent as designed.

**G.4 Builds + type-check + tests** *(Effort: S)*
- `pnpm build` and `pnpm typecheck` clean in `atlas-ui`.
- `go build ./...` and `go test ./...` clean in `atlas-data`.
- `docker compose build atlas-data atlas-ui` both pass.
- **Acceptance:** Green local pipeline.

**G.5 No regressions in task-008 acceptance** *(Effort: S)*
- Re-walk task-008 PRD §10 Phase 1 + Phase 2 checkboxes; confirm none are now broken.
- **Acceptance:** All previously-shipped task-008 behavior intact.

---

## 6. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `render.png` dims drift from `mapArea` (extractor uses different precedence than reader) | L | H | G.1 cross-service invariant verification on three sample maps. If mismatched, fall back to scaling overlay by `img.naturalWidth/Height` instead of `mapArea.width/height`. |
| `mapArea` pointer-ization breaks atlas-data consumers | M | M | A.3 grep for callers; nil-check defensively. Existing tests catch downstream issues. |
| Container `aspectRatio` change visually regresses unusual map shapes (very tall vertical maps like Ellinia) | M | M | D.2 includes spot-check on Ellinia (5146px tall). If the layout looks bad, cap with both `max-h-[320px]` and `max-w-full` and live with letterboxing — overlay percentages still work because the wrapper still owns the aspect ratio. |
| Stacked markers obscure pointer hits on dense maps (3 monsters at identical coords) | M | L | Acceptable for v1 per PRD §9. Z-order rule (portals > npcs > reactors > monsters) gives consistent topmost. |
| Touch-device tap accidentally triggers `setHovered` via emulated pointer events | L | L | Markers are `<button>` with no click handler; `onPointerEnter` fires once on touch but no `onPointerLeave` until next interaction. Defensive: wrap context with a no-op when `window.matchMedia('(pointer: coarse)').matches`. Defer unless real-world testing shows breakage. |
| Hover state churn causes summary/tab re-render storm on dense maps | L | L | `useHoverHighlight` returns memoized `isHovered` callback; each consumer reads only its own predicate. React 19's automatic batching keeps it in one frame. |
| PRD said "REST extension" with new `bounds` field, plan uses existing `mapArea` | low (already noted with user) | L | Documented in §1 + this row. Nothing breaks; the data wiring is identical, just a different field name. PRD reflects the spirit; plan reflects the cheaper reality. |

---

## 7. Success Metrics

### Pass/fail
- All PRD §10 acceptance checkboxes ticked (Backend, Frontend overlays, Hover coordination, Touch, Expanded dialog, Quality/regression).
- 3/3 sampled maps pass the cross-service invariant in G.1.
- No regressions in task-008 PRD §10 acceptance.
- Local docker builds pass for both `atlas-data` and `atlas-ui`.

### Qualitative
- Operator feedback after one week: marker hover correctly correlates a row with a screen position. (Confirmed by at least one GM/operator using the page.)
- No "what's that dot?" confusion — tooltip + color legend (informally documented in `ux-flow.md`) covers it.

---

## 8. Required Resources & Dependencies

### Tooling
- Go (existing in atlas-data).
- Node + pnpm (existing in atlas-ui).
- Docker for service builds.

### Data fixtures
- A v83 GMS tenant with `Map.wz` already loaded so `mapArea` resolves and `render.png` is present (already available — task-008 produced these).

### People
- One implementer comfortable in both Go and React/TypeScript. Phases A (Go) and B/C/D/E/F (TS) are independently progressable; two implementers can parallelize once A.1 lands.

### No new external dependencies
- No new Go modules.
- No new npm packages — relies on existing `Tooltip`, `Card`, `Dialog` shadcn primitives and React state primitives.

---

## 9. Timeline Estimates

| Phase | Effort (dev-days) | Notes |
|---|---|---|
| A — atlas-data nullable mapArea | 0.5 | Tiny change; touch points are local. |
| B — TS types + helper | 0.25 | Pure additions. |
| C — Hover context | 0.25 | ~30 LOC + unit test. |
| D — Overlay + container refactor | 1–1.5 | The visual core; container refactor is the risky bit. |
| E — Hover wiring (summary + tabs) | 0.5–1 | Mechanical; one row component at a time. |
| F — Dialog overlay | 0.25 | Reuses Phase D component. |
| G — Verification | 0.5 | Manual walkthrough + builds. |
| **Total** | **3–4 days** | Single PR or two (A separately, then B–G). |

---

## 10. Out of Scope

Per PRD §2 Non-goals:
- Click-to-select / persistent-selection state.
- Overlays on the minimap fallback.
- Editing entities from the page.
- Foothold lines, collision volumes, VR boundary box.
- Touch tap-to-highlight.
- Any change to the static composite renderer in atlas-wz-extractor.

---

## 11. Notes & Discoveries

- **`mapArea` already on the wire.** PRD §4.1 calls for a new `bounds` attribute and a four-column migration. Discovery during planning: `services/atlas-data/atlas.com/data/map/rest.go:27` already emits `mapArea: { x, y, width, height }` with the right precedence in `reader.go:179-216`. Plan reuses it. Saves a migration, a backfill, and a column refactor — net change is one struct field becoming a pointer.
- **Sentinel handling.** The `1 << 18` synthetic fallback in `reader.go:198-204` is the ambiguous case ("no real bounds resolved") — Phase A converts it to `nil` so the UI can branch cleanly without sentinel-detection logic.
- **`getMapById` is not sparse-fielded.** `services/atlas-ui/src/services/api/maps.service.ts:56-58` calls `api.getOne` without `withSparseFields`, so `mapArea` already arrives in the network payload — only the TS type omits it.
