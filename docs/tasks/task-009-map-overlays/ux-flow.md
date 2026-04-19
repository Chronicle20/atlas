# Map Image Overlays — UX Flow

## Layout (overlay added to task-008 page)

```
┌────────────────────────────────────────────────────────────────────┐
│  Henesys                                                           │
│  [Henesys] [14 spawns]                                             │
├────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────┐  ┌───────────────────────┐    │
│ │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │  │ NPCs (3)              │    │
│ │  ░░░ ●  ░░░░░░░ ◇ ░░░░░░░░░░░  │  │ 🎭 Eddie     ←hover→  │    │
│ │  ░░░░░░░░░░░ ◼ ░░░░░░░░░░░░░░  │  │ 🎭 Cody               │    │
│ │  ░░░░ ●  ●  ●  ░░░░░░░░░░░░░  │  │ 🎭 Nells              │    │
│ │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │  ├───────────────────────┤    │
│ │  ░░░░░░░░░░ ◇ ░░░░░░░░░░░░░░░  │  │ Monsters (5)          │    │
│ └──────────────────────────────────┘  │ 👾 Snail    ×6  ←•   │    │
│                                       │ 👾 Blue Snail ×4      │    │
│                                       │ 👾 Mushroom  ×1       │    │
│                                       └───────────────────────┘    │
│                                                                    │
│  ◇ portals (emerald diamond)                                       │
│  ● npcs (sky circle)                                               │
│  • monsters (rose dot)                                             │
│  ◼ reactors (amber square)                                         │
└────────────────────────────────────────────────────────────────────┘
```

## Hover-to-highlight scenarios

### Scenario A — hover summary "Snail ×6" row

1. Pointer enters the Monsters summary row "Snail ×6".
2. `HoverHighlightContext.hovered` becomes `{ kind: "monster", template: 100100 }`.
3. All six rose dots that share `template === 100100` brighten and scale 1.5×.
4. The "Snail ×6" row's left border lights up rose.
5. In the (currently hidden) Monsters tab, all six per-spawn rows for `template === 100100` would also be highlighted — but invisible until the tab is selected.
6. No tooltip opens (row-driven highlight).
7. Pointer leaves → all highlights clear.

### Scenario B — hover a single monster marker on the image

1. Pointer enters one rose dot. The marker's `onPointerEnter` fires.
2. `hovered` becomes `{ kind: "monster", template: 100100, spawnIndex: 3 }`.
3. By template-match rules (§4.6), all six dots highlight; the summary row "Snail ×6" highlights.
4. A `Tooltip` opens above the hovered marker showing "Snail".
5. Pointer leaves → highlights clear, tooltip closes.

### Scenario C — hover a portal row in the Portals tab

1. Pointer enters portal row "in00".
2. `hovered` becomes `{ kind: "portal", portalId: "in00" }`.
3. The single emerald diamond for that portal brightens and scales 1.5×.
4. The row's left border lights up emerald.
5. No tooltip (row-driven).
6. Pointer leaves → clears.

### Scenario D — hover an NPC marker on the image

1. Pointer enters one sky circle.
2. `hovered` becomes `{ kind: "npc", template: 1002000, spawnIndex: 0 }`.
3. All sky circles with that template highlight (typically just one — NPCs rarely duplicate, but if they do all light up).
4. The NPC's row in the summary highlights.
5. Tooltip opens with the NPC's name.

### Scenario E — pointer moves directly between two markers

1. Pointer leaves marker A, enters marker B before any frame elapses.
2. `setHovered(null)` (from A's leave) is followed immediately by `setHovered(B)`.
3. React batches: net effect is `hovered === B`. A's highlight clears, B's appears, in the same frame.
4. No flicker visible.

## Interaction rules

| Action | Effect |
|---|---|
| Pointer enters marker | `setHovered` to most-specific identity; tooltip opens. |
| Pointer leaves marker | `setHovered(null)`; tooltip closes. |
| Pointer enters summary row | `setHovered` to per-template identity; no tooltip. |
| Pointer leaves summary row | `setHovered(null)`. |
| Pointer enters detail-tab row | `setHovered` to most-specific identity; no tooltip. |
| Pointer leaves detail-tab row | `setHovered(null)`. |
| Tab change | `hovered` is naturally cleared because the previously-hovered row is unmounted. |
| Dialog open | `hovered` carries over (rare — dialog opens via click which fires after hover). |
| Dialog close | `setHovered(null)` on close to avoid stale highlight on the page underneath. |
| Touch tap on marker | No-op (no hover semantics on touch — see §2 non-goal). |
| Keyboard focus on marker `<button>` | Treated as hover for highlight/tooltip purposes. |

## Edge cases

| Case | Behavior |
|---|---|
| `bounds === null` | Image renders at original `object-contain` sizing. Overlay is not mounted. Hover handlers on rows still call `setHovered`, but no marker visualizes the highlight. (Acceptable — rows still get their accent border.) |
| Panel state is `"minimap"` or `"placeholder"` | Same as `bounds === null`: overlay not mounted. |
| Entity coordinate falls outside `bounds` | Marker still renders, positioned at the clamped percentage. Dev-only `console.warn` fires once per render cycle. |
| Two entities at identical `(x, y)` | Markers stack. The frontmost (z-order: portals > npcs > reactors > monsters) takes the pointer hit; the others are visually obscured but technically present. Acceptable for v1. |
| Map has 200+ markers | All render. Performance budget: 60fps on a developer laptop. |
| Entity query in flight | Markers for that entity type don't render. As soon as the query resolves, they appear. No skeleton markers. |
| Entity query errors | Same as in-flight: just no markers. Error message remains in the affected tab/section. |

## Visual states summary

| State | Marker style |
|---|---|
| Default | semi-transparent fill (`/70`), 1–2px white border. |
| Highlighted (hovered directly OR matched by `hovered`) | full opacity, white border + yellow `ring-2`, scaled 1.5×. |
| Tooltip open | only on direct marker hover or keyboard focus. |
| Inside dialog | identical visual treatment; sized in CSS so the marker stays ~10px regardless of image zoom (use `transform: scale(1)` not pixel size scaling so the markers grow proportional to the image — TBD during implementation). |

A small implementation note for the dialog: the user wants overlays in the dialog but no extra "selected state" beyond hover. We do not introduce any sticky highlight; closing the dialog returns the page to its baseline.
