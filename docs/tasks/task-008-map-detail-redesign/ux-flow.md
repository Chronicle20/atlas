# Map Detail UX Flow

## Layout (desktop, ≥ md breakpoint)

```
┌────────────────────────────────────────────────────────────────────┐
│  Henesys                          ← hover copies template ID       │
│  [Henesys] [14 spawns]            ← badge row                      │
├────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────┐  ┌───────────────────────┐    │
│ │                                  │  │ NPCs (3)              │    │
│ │                                  │  │ 🎭 Eddie              │    │
│ │         map render.png           │  │ 🎭 Cody               │    │
│ │      (or minimap.png P1)         │  │ 🎭 Nells              │    │
│ │                                  │  ├───────────────────────┤    │
│ │                                  │  │ Monsters (5)          │    │
│ │                                  │  │ 👾 Snail   ×6         │    │
│ └──────────────────────────────────┘  │ 👾 Blue Snail  ×4     │    │
│                                       │ 👾 Red Snail   ×2     │    │
│                                       │ 👾 Mushroom   ×1      │    │
│                                       │ 👾 Shroom     ×1      │    │
│                                       └───────────────────────┘    │
├────────────────────────────────────────────────────────────────────┤
│  Connected maps (4)                                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  →             │
│  │Henesys   │ │Bazaar    │ │Hunting   │ │Pig Park  │               │
│  │Market    │ │          │ │Ground 1  │ │          │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
├────────────────────────────────────────────────────────────────────┤
│  [ Portals ] [ Monsters ] [ Reactors ]     ← three tabs            │
│                                                                    │
│  (existing detail table for the selected tab)                      │
└────────────────────────────────────────────────────────────────────┘
```

## Mobile (< md)

Sections stack vertically in the same top-to-bottom order. The connected-maps row remains horizontally scrollable via touch.

## Interaction notes

- **Title hover** — uses `TooltipContent copyable`. Touch users get a long-press fallback per the existing shadcn tooltip behavior (same as `MapCell`).
- **Badges** — non-interactive. The spawn-count badge shows a skeleton while `useMapMonsters` is loading.
- **Image panel** — static `<img loading="lazy">`. On 404, render a placeholder `div` with a neutral background and the text "No render available".
- **Summary panel** — each sub-section is its own scrollable region capped at ~400px height; the panel itself does not grow unbounded with 100+ spawns.
- **Connected-maps row** — horizontal scroll on overflow (no wrap). Each widget is a full-size clickable link with a hover state. `MapCell`'s `mapNameCache` prevents N+1 fetches when multiple widgets reference recently-viewed maps.
- **Tabs** — identical to today's tables except the `npcs` tab is removed. The Portals tab is the default-open tab (matches today).

## Progressive render timeline

```
t=0        useMap succeeds
           ├── header renders (title + copyable tooltip)
           ├── street-name badge renders
           ├── spawn-count badge shows skeleton
           ├── image panel renders (img tag mounted, browser fetches)
           └── tabs shell renders with per-tab loading states

t=varies   useMapNpcs succeeds      → NPCs summary section fills in
t=varies   useMapMonsters succeeds  → Monsters summary section fills in
                                    → spawn-count badge resolves
                                    → Monsters tab fills in
t=varies   useMapPortals succeeds   → connected-maps row renders
                                    → Portals tab fills in
t=varies   useMapReactors succeeds  → Reactors tab fills in
t=img-load img resolves or 404s     → placeholder swaps in if 404
```

No query blocks another. No "everything loads together" gate.

## Error handling

| Failure | UI response |
|---|---|
| `useMap` fails | Full-page `ErrorDisplay` with retry (today's behavior, unchanged). |
| `useMapNpcs` fails | NPCs summary section shows "Failed to load NPCs" inline. Monsters section and tabs unaffected. |
| `useMapMonsters` fails | Monsters summary shows "Failed to load monsters" inline. Spawn-count badge renders as `— spawns`. Monsters tab shows its own error. |
| `useMapPortals` fails | Connected-maps section hidden (same as zero-targets case). Portals tab shows its own error. |
| `useMapReactors` fails | Reactors tab shows its own error. Other sections unaffected. |
| Image 404 | Placeholder swap. No user-facing error — absence of a render is normal for system maps. |

## Dedup semantics (reference for implementation)

**Connected maps:**
```ts
const distinctTargets = new Map<number, string>();
for (const p of portals) {
  const tm = p.attributes.targetMapId;
  if (!tm || tm === 999999999 || String(tm) === mapId) continue;
  if (!distinctTargets.has(tm)) distinctTargets.set(tm, p.attributes.name);
}
// render in insertion order
```

**NPCs summary:**
```ts
const distinctNpcs = Array.from(
  new Map(npcs.map(n => [n.attributes.template, n])).values(),
);
```

**Monsters summary (with count):**
```ts
const counts = new Map<number, { name: string; count: number }>();
for (const m of monsters) {
  const existing = counts.get(m.attributes.template);
  if (existing) existing.count++;
  else counts.set(m.attributes.template, { name: /* resolved */, count: 1 });
}
```

Monster names come from the monster REST model — today's Monsters tab already resolves these via `MonsterTableRow`. The summary panel should reuse whatever name-resolution path that component uses (likely a `useMonster` hook per template or a bulk resolver). If bulk resolution doesn't exist, a simple per-template `useMonster` is acceptable for first cut given React Query will dedupe identical keys.
