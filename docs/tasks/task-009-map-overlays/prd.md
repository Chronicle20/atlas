# Map Image Overlays — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-18
---

## 1. Overview

Task-008 redesigned `MapDetailPage` around a static composite map render plus a side panel of NPCs/monsters and a tabbed detail view of portals/monsters/reactors (`services/atlas-ui/src/pages/MapDetailPage.tsx:42-73`). The image is currently a flat picture — operators can see *what* the map looks like, but not *where* on the map a given monster spawns, where a portal sits, or which NPC stands at the dock.

This task adds an **interactive overlay layer** on top of `MapImagePanel` that draws colored markers for every portal, monster spawn point, reactor spawn point, and NPC at their world-space coordinates, plus a **bidirectional hover-to-highlight** interaction: hovering a marker lights up the matching row(s) in the summary panel / detail tabs, and hovering a row/card lights up the matching marker(s) on the image. Both the inline preview and the expanded `Dialog` view support the overlay.

The key enabler is exposing the map's resolved world-space bounds (origin + width/height) on the map REST attributes, so the UI can transform world `(x, y)` coordinates into overlay percentages without inferring them from the rendered image. atlas-data's map reader already computes these bounds (`services/atlas-data/atlas.com/data/map/reader.go:179-216`); this task surfaces them through the model and JSON:API attributes.

## 2. Goals

Primary goals:
- Let operators see *where* every entity is positioned on a map at a glance, without opening any tab.
- Let operators correlate a row in the summary or a row in a detail tab with its physical position on the map (and vice versa) via hover.
- Keep the overlay an additive layer on top of the existing `MapImagePanel` — do not change the image-fallback chain or the static-render contract.
- Expose the map's world bounds through the REST contract so any future feature (foothold overlays, custom annotations) can re-use the same transform.

Non-goals:
- Click-to-select / persistent-selection state. Hover is transient; we do not track a "currently selected" marker across views or persist selection in the URL.
- Overlays on the **minimap fallback**. When `render.png` is missing and the panel falls back to `minimap.png`, the overlay layer is hidden. Minimap pixel scale (`mag`) and the lack of any rendered overlay testing on it are out of scope.
- Editing entities (drag a portal, move a spawn point) — read-only overlays only.
- Foothold lines, collision volumes, VR boundary box, or any other debug overlay beyond the four entity types.
- Touch-device overlay interaction. Markers still render on touch screens, but tap-to-highlight is not implemented; row-hover-to-highlight implicitly does not exist on touch either.
- Any change to the static composite renderer in atlas-wz-extractor. The renderer's pixel output is already the source of truth for image dimensions; this task assumes `render.png` dims equal the world rect width × height (verified during implementation).

## 3. User Stories

- As an operator chasing a "monsters not spawning" bug, I want to hover a monster row in the summary and immediately see all of that template's spawn points light up on the image so I can tell whether the spawn area looks reasonable.
- As an operator inspecting a portal chain, I want to hover a portal row in the Portals tab and see the portal's marker pulse on the image so I can confirm I'm looking at the right portal before clicking through to its target.
- As a GM placing an NPC, I want to hover the NPC's marker on the image and see the NPC's name highlighted in the summary panel so I don't have to memorize templates.
- As a designer reviewing reactor placement, I want every reactor's position visible on the image so I can confirm the layout matches the quest design without opening the Reactors tab.
- As a frontend developer, I want the world→image transform delivered by the REST API so I don't reverse-engineer it from the PNG dimensions or duplicate the bounds-resolution logic that already lives in atlas-data.

## 4. Functional Requirements

### 4.1 atlas-data: expose map bounds

`MapAttributes` (REST + DB model) gains a nullable `bounds` object:

```ts
bounds: {
  x: number;       // world-space X of the rendered image's top-left
  y: number;       // world-space Y of the rendered image's top-left
  width: number;   // pixel width of the rendered image (and world-rect width — they match)
  height: number;  // pixel height of the rendered image (and world-rect height — they match)
} | null
```

Resolution rules (mirror the existing precedence in `services/atlas-data/atlas.com/data/map/reader.go:179-216` and `docs/tasks/task-008-map-detail-redesign/render-pipeline.md` §"Concrete static-composite pipeline" step 1):

1. If `VRLeft / VRRight / VRTop / VRBottom` are all present, `bounds = { x: VRLeft, y: VRTop, width: VRRight - VRLeft, height: VRBottom - VRTop }`.
2. Else if `miniMap` is present, `bounds = { x: -centerX, y: -centerY, width: miniMap.width, height: miniMap.height }`.
3. Else `bounds = null`.

`bounds` MUST be `null` whenever atlas-wz-extractor would have skipped writing `render.png` for the same reason (no VR + no miniMap, or dims clamped). Cross-service invariant: **for every map with non-null `bounds`, the corresponding `render.png` is `bounds.width × bounds.height` pixels.** Verified during implementation by spot-checking a handful of maps; an integration test compares the PNG header dims to the REST `bounds` for at least three maps in the Phase 2 fixture set.

The field is part of the default JSON:API `attributes` payload (no sparse-fieldset opt-in needed — the overlay layer fails closed when it's missing, but the page still works without it).

### 4.2 atlas-ui: world→image transform

A pure helper in `src/lib/utils/map-overlay.ts`:

```ts
export interface MapBounds { x: number; y: number; width: number; height: number; }

export function worldToOverlayPercent(
  worldX: number,
  worldY: number,
  bounds: MapBounds,
): { left: string; top: string } {
  return {
    left: `${((worldX - bounds.x) / bounds.width) * 100}%`,
    top:  `${((worldY - bounds.y) / bounds.height) * 100}%`,
  };
}
```

Markers position via CSS percentages so the overlay scales freely with the displayed image regardless of `object-contain` letterboxing — see §4.4 for the container contract that makes percentages work.

### 4.3 Marker visual language

| Entity   | Color (Tailwind token)     | Shape           | Default size | Border          |
|----------|----------------------------|-----------------|--------------|-----------------|
| Portal   | `bg-emerald-500/70`        | rotated square (diamond) | 10×10 px | `border-2 border-white` |
| NPC      | `bg-sky-500/70`            | circle          | 10×10 px     | `border-2 border-white` |
| Monster  | `bg-rose-500/70`           | circle (smaller) | 8×8 px      | `border border-white` |
| Reactor  | `bg-amber-500/70`          | square          | 10×10 px     | `border-2 border-white` |

All markers are absolutely positioned via `transform: translate(-50%, -50%)` so the percentage coords land on the marker's center. Markers are non-interactive on touch (no tap handler) but render so the visual "where things are" cue is preserved.

Highlight state (any of: marker is hovered, OR a sibling row in the summary/detail is hovered):

- Saturation goes from `/70` to full opacity.
- Border becomes `ring-2 ring-yellow-400` in addition to the white border.
- Marker scales `1.5×` via `transition-transform`.
- A small `Tooltip` (shadcn `TooltipContent`) appears on direct marker hover showing the entity's display name (e.g., "Mushroom" for monsters, portal `name` for portals). Row-driven highlights do **not** open a tooltip — only direct marker hover does.

Z-order on the overlay (lowest to highest paint): monsters → reactors → npcs → portals. Portals on top because they're the rarest and most actionable for navigation.

### 4.4 Overlay container contract (`MapImagePanel`)

The image's display container must enforce the render's intrinsic aspect ratio so percentage-positioned overlays land on the image's content rect rather than getting offset by `object-contain` letterboxing:

- Wrap the `<img>` and the overlay in a single positioned container with `style={{ aspectRatio: bounds ? `${bounds.width} / ${bounds.height}` : undefined }}`.
- Inside that container, the `<img>` uses `className="w-full h-full object-cover"` (instead of today's `object-contain` + `max-h-[320px]`). The wrapper's `max-h` provides the height ceiling.
- The overlay is a sibling `<div className="absolute inset-0 pointer-events-none">` containing per-entity marker buttons. Markers re-enable pointer events with `pointer-events-auto`.
- When `bounds` is null **or** the panel state is `"minimap"` or `"placeholder"`, the overlay is not rendered at all — the original `object-contain` sizing returns so non-render fallbacks are visually unchanged. (Spec restated: minimap-fallback overlays are explicitly out of scope.)

Inline preview keeps its `max-h-[320px]` ceiling. The expanded `Dialog` view applies the same wrapper but with no max-height — the image renders at intrinsic size and scrolls within the dialog. Overlays render in both contexts using the same component.

### 4.5 Hover-state coordination

A new lightweight context in `src/components/features/maps/HoverHighlightContext.tsx` exposes:

```ts
type HoverTarget =
  | { kind: "portal"; portalId: string }
  | { kind: "monster"; template: number; spawnIndex?: number }
  | { kind: "reactor"; reactorId: string }
  | { kind: "npc"; template: number; spawnIndex?: number }
  | null;

interface HoverHighlightContextValue {
  hovered: HoverTarget;
  setHovered: (t: HoverTarget) => void;
}
```

Provided once at `MapDetailPage`. Consumers:

- `MapImagePanel`'s overlay layer reads `hovered` and computes "is this marker highlighted?" per the matching rules in §4.6.
- `MapEntitySummary` rows set `hovered` on `onPointerEnter` and clear on `onPointerLeave`. NPC rows use `{ kind: "npc", template }`; monster rows use `{ kind: "monster", template }` (no `spawnIndex` — summary is per-template).
- `MapDetailTabs` table rows set `hovered` similarly. Portal rows use `{ kind: "portal", portalId }`. Monster table rows (per-spawn) use `{ kind: "monster", template, spawnIndex: i }`. Reactor rows use `{ kind: "reactor", reactorId }`.
- The overlay markers themselves set `hovered` on `onPointerEnter` to the most-specific identity available (`spawnIndex` included for monsters/NPCs).

Hover state is transient. It's not persisted, not URL-encoded, not surfaced to React Query.

### 4.6 Highlight matching rules

"Marker M is highlighted" iff either (a) the user is directly hovering M, or (b) the current `hovered` matches M per these rules:

| `hovered` kind | Marker matches when... |
|---|---|
| `portal` | marker's `portalId === hovered.portalId` (exact) |
| `monster` | marker's `template === hovered.template` (regardless of `spawnIndex`) |
| `reactor` | marker's `reactorId === hovered.reactorId` (exact) |
| `npc` | marker's `template === hovered.template` (regardless of `spawnIndex`) |

"Row R is highlighted" iff `hovered`'s identity satisfies the row's display granularity:

| Row context | Highlighted when `hovered` is... |
|---|---|
| Summary panel — NPC row (per-template) | any `{ kind: "npc", template: rowTemplate }` (any spawnIndex) |
| Summary panel — Monster row (per-template) | any `{ kind: "monster", template: rowTemplate }` (any spawnIndex) |
| Portals tab row | exact `{ kind: "portal", portalId: rowPortalId }` |
| Monsters tab row (per-spawn) | exact `{ kind: "monster", template, spawnIndex }` OR per-template `{ kind: "monster", template }` (template-only highlights all sibling rows) |
| Reactors tab row | exact `{ kind: "reactor", reactorId: rowReactorId }` |

Highlight visual on rows: `bg-muted/60` + a 2px left accent border in the matching entity color (emerald/sky/rose/amber).

### 4.7 Marker source data

- **Portals**: every entry in `useMapPortals` is rendered, including `targetMapId === 999999999` (NONE) and any type. Marker's `portalId` is the JSON:API `id`. Tooltip shows `attributes.name`.
- **Monsters**: every entry in `useMapMonsters` is rendered (one marker per spawn point). Tooltip shows the resolved monster name (reuse the same name-resolution path as the summary panel — likely `useMonster(template)`).
- **Reactors**: every entry in `useMapReactors`. Tooltip shows `attributes.name`.
- **NPCs**: every entry in `useMapNpcs`. Tooltip shows `attributes.name`.

If a query is loading or errored, that entity type's markers simply don't render. No skeleton markers.

Coordinate source on each entity attribute: `x` and `y` are world-space — fed straight into `worldToOverlayPercent`.

### 4.8 Performance considerations

- A typical hunting ground has on the order of 10–30 monster spawns + a handful of NPCs/portals/reactors. PQ maps and bossing rooms can hit 80–150 markers. The overlay must remain smooth at 200 markers; absolute-positioned `<button>` elements with `pointer-events: auto` are fine at this scale.
- All marker positions are pure functions of `bounds` + entity coords — memoize with `useMemo` keyed on `bounds`, the entity arrays, and `hovered`.
- Hover updates are local context state; they MUST NOT cause `MapDetailPage` or sibling sections to refetch. Verified by ensuring hover does not invalidate any React Query key.

### 4.9 Expanded `Dialog` view

The `Dialog` (full-size image viewer) renders the same `MapImageOverlay` component over the natural-size image. Hover-to-highlight works inside the dialog using the same `HoverHighlightContext`, so a marker hover in the dialog also lights up its row in the underlying summary/tab — but since the dialog covers most of the page, this is mostly cosmetic. Tooltip on direct marker hover still fires.

Per the user requirement, the dialog does **not** introduce any "selected marker" state — closing the dialog clears any hover, no selection persists.

## 5. API Surface

No new endpoints. One field added to an existing response:

- `GET /api/data/maps/{id}` — `attributes.bounds: { x, y, width, height } | null` added to the JSON:API attributes object. Backwards-compatible: clients ignoring the field continue to work.

`GET /api/data/maps` (list) is unchanged — `bounds` is detail-only and not needed by the index page. The sparse fieldset (`fields[maps]=name,streetName`) currently used by `mapsService.fetchAll` does **not** request `bounds`, keeping list payloads small.

No atlas-assets, atlas-wz-extractor, atlas-ingress, or other-service surface changes.

## 6. Data Model

atlas-data persistence:

- Map entity gains four nullable columns: `bounds_x`, `bounds_y`, `bounds_width`, `bounds_height` (all `int`). A migration adds the columns; a follow-up backfill populates them by re-reading `Map.wz` for the existing tenant data. Maps with no VR + no miniMap leave the columns null.
- The map domain model exposes a `Bounds()` getter returning `(*Bounds, bool)` per the immutable-model pattern. Builder gains `SetBounds(b *Bounds)`.
- The REST transformer in atlas-data emits the `bounds` attribute as `null` when all four columns are null, otherwise as the four-field object.

No new entities. No multi-tenancy changes — bounds are per-map and fall under the existing tenant scoping for the maps table.

## 7. Service Impact

| Service | Change | Reason |
|---|---|---|
| **atlas-data** | New nullable bounds columns + migration; reader populates from existing `reader.go:179-216` precedence; domain model + builder + REST transformer expose the field. | Source of truth for map metadata; UI overlay needs the world rect. |
| **atlas-ui** | New `HoverHighlightContext`; new `MapImageOverlay` component layered inside `MapImagePanel`; row-hover handlers added to `MapEntitySummary`, the Portals/Monsters/Reactors tab tables in `MapDetailTabs`; new `worldToOverlayPercent` helper; small refactor of `MapImagePanel` container sizing to enforce aspect ratio when `bounds` is present. | All UI work lives here. |
| **atlas-wz-extractor** | No code change. Verified that existing `render.png` dimensions match the bounds atlas-data exposes. | Render contract is already aligned via shared bounds precedence in render-pipeline.md. |
| **atlas-assets, atlas-data list endpoint, atlas-ingress, others** | No change. | Out of scope. |

## 8. Non-Functional Requirements

**Performance:**
- Marker render budget: 200 markers must paint without dropping below 60fps on a typical developer laptop. No virtualization is required at this density.
- Hover-to-highlight latency: ≤16ms perceived (one frame). Achieved by keeping `HoverHighlightContext` updates synchronous and not memoizing across the whole page tree.
- No additional network requests. Bounds piggyback on the existing `useMap` query.

**Correctness:**
- Cross-service invariant: for every map where atlas-data returns non-null `bounds`, the corresponding `render.png` (when present) MUST be `bounds.width × bounds.height` pixels. Verified by an integration check on at least three sample maps (Henesys, Perion, Ellinia) during PR review.
- Marker positions MUST be center-anchored — entity `(x, y)` maps to the marker's geometric center, not its top-left.

**Accessibility:**
- Markers are `<button>` elements with `aria-label` describing the entity (e.g., "Portal: in00"). Keyboard users can Tab through them; focus state is visually equivalent to hover.
- The hover-to-highlight relationship is **decorative**, not the only way to discover the data — every entity is also visible in the summary or tabs. Screen-reader users are not penalized.
- Tooltip text is visible/announced via the existing shadcn `Tooltip` primitive.

**Multi-tenancy:**
- Bounds are per-map per-tenant via the existing maps table tenant scoping. No new cross-tenant surface.
- The overlay reads no global state beyond the active tenant (already required for asset URLs).

**Observability:**
- No new metrics or logs. The overlay is a pure UI feature.
- A console warning fires (dev only, via a small `import.meta.env.DEV` guard) if an entity's `(x, y)` falls outside the map's bounds — useful for catching renderer/bounds drift but not a user-visible error.

**Security:**
- No new user input flows. Hover state is client-side only.
- Bounds values come from trusted backend WZ parsing; no validation needed beyond the int type.

## 9. Open Questions

- **Backfill migration strategy.** Adding the bounds columns is trivial; populating them for existing map rows requires re-reading `Map.wz` per tenant. Two options: (a) lazy backfill on next read, (b) one-shot maintenance command. Punt to the implementer — `null` is a valid steady state, so any approach is correct.
- **NPC tooltip name resolution.** The summary panel resolves NPC names via `getAssetIconUrl` + the entity REST attribute. The marker tooltip should reuse the same name; confirm the field is on `MapNpcData` (it is — `attributes.name`).
- **Marker collision.** Maps with overlapping spawns (e.g., 3 mushrooms at the same coord) will draw stacked markers. v1 ships as-is; if visual clutter becomes a complaint, a future task can add per-cluster spread or a count badge. Out of scope.
- **Portal type-specific shapes.** Portals have a numeric `type` (spawn point, regular, hidden, script, etc.). v1 uses one diamond shape regardless. Could be split by shape in a follow-up if useful.

## 10. Acceptance Criteria

**Backend:**
- [ ] `GET /api/data/maps/{id}` returns `attributes.bounds` as `{x, y, width, height}` for maps with VR or miniMap, and `null` otherwise.
- [ ] For at least three sampled maps (Henesys, Perion, Ellinia), the `bounds.width × bounds.height` matches the `render.png` pixel dimensions exactly.
- [ ] Migration adds `bounds_x/y/width/height` nullable columns; existing list endpoint and other consumers are unaffected.

**Frontend overlays:**
- [ ] When `bounds` is present and the panel is in `"render"` state, markers appear for every portal, monster, reactor, and NPC at world-correct positions.
- [ ] When the panel falls back to `"minimap"` or `"placeholder"`, no markers render and the panel reverts to its pre-overlay sizing.
- [ ] Markers use the colors and shapes from §4.3.
- [ ] Direct marker hover opens a tooltip with the entity name; row-driven highlights do not open the tooltip.

**Hover coordination:**
- [ ] Hovering a marker highlights:
  - the deduped row in the summary panel (for NPCs/monsters),
  - all sibling markers (same template) for monsters and NPCs,
  - the matching row(s) in the relevant detail tab.
- [ ] Hovering a summary row (per-template) highlights every matching marker on the image.
- [ ] Hovering a portal/reactor row highlights exactly that marker; hovering an exact monster table row highlights exactly that marker.
- [ ] Pointer leaving the row or marker clears the highlight.
- [ ] No persistent selection: re-opening the dialog or scrolling does not preserve any hover state.

**Behavior on touch / mobile:**
- [ ] Markers still render. Tap on a marker does not trigger any hover/highlight behavior. Page does not error.

**Expanded dialog:**
- [ ] Overlay markers render in the expanded `Dialog` view at the image's natural size, positioned correctly.
- [ ] Hover-to-highlight works inside the dialog (same context).
- [ ] Closing the dialog clears any active hover state.

**Quality / regression:**
- [ ] No regressions in task-008 acceptance criteria — all existing map-detail behavior continues to work.
- [ ] No new external dependencies.
- [ ] Unit test for `worldToOverlayPercent` covering positive/negative origins.
