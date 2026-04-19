# UX Flow — Monster Detail Redesign

This document captures the page's visual structure and interaction details that are too specific for the PRD. It is the authoritative reference during implementation for component ordering, spacing, and hover/click behavior.

## 1. Page anatomy (top to bottom)

```
┌──────────────────────────────────────────────────────────────────────────┐
│  🐙  Mushroom                [Boss] [Undead]  ←── tooltip on hover/focus │
│      (icon + name share a single tooltip → click content to copy id)     │
├──────────────────────────────────────────────────────────────────────────┤
│  ┌─ Combat Stats ──┐  ┌─ Attack / Defense ─┐  ┌─ Properties ──┐          │
│  │ Level     5     │  │ W.Atk     10       │  │ First Atk  No │          │
│  │ HP      250     │  │ W.Def      5       │  │ FFA Loot   No │          │
│  │ MP        0     │  │ M.Atk      0       │  │ Explosive  No │          │
│  │ EXP      14     │  │ M.Def      0       │  │ CP          0 │          │
│  └─────────────────┘  └────────────────────┘  └───────────────┘          │
├──────────────────────────────────────────────────────────────────────────┤
│  Skills                                                                  │
│  [Power Up · L2]  [Physical Barrier · L1]  [Summon Minions · L3]         │
├──────────────────────────────────────────────────────────────────────────┤
│  Drops (12)                                                              │
│                                                                          │
│  MESOS                                                                   │
│  ┌── 🪙 Mesos ─────────────┐                                             │
│  │ 🪙 Mesos                 │                                             │
│  │    100 – 250             │   ←── tooltip: Chance 5,000                │
│  └─────────────────────────┘                                             │
│                                                                          │
│  EQUIPMENT (3)                                                           │
│  ┌── 🗡  Red Whip ──┐  ┌── 🛡 Leather Shield ┐  ┌── 🎩 Old Hat ┐         │
│  │   1002140        │  │   1092000            │  │   1002001     │         │
│  └──────────────────┘  └─────────────────────┘  └───────────────┘         │
│   ← each widget clickable, links to /items/{id}; hover for drop stats    │
│                                                                          │
│  CONSUMABLE (4)                                                          │
│  ┌── 🧪 Red Potion ┐  ┌── 🍞 Bread ┐  ┌── 📜 Scroll ┐  ┌── 🧪 Elixir ┐    │
│  │   2000000       │  │   2022003  │  │   2040000    │  │   2000003 │    │
│  └─────────────────┘  └────────────┘  └──────────────┘  └───────────┘    │
│                                                                          │
│  ETC (3)                                                                 │
│  ...                                                                     │
│                                                                          │
│  CASH (1)                                                                │
│  ...                                                                     │
├──────────────────────────────────────────────────────────────────────────┤
│  Spawn Locations (4)                                                     │
│  ┌── Henesys Hunting  [Victoria Road] ─┐  ┌── Lith Harbor  [Victoria] ┐  │
│  │                         [6 spawns]  │  │                 [4 spawns] │  │
│  └─────────────────────────────────────┘  └────────────────────────────┘  │
│  ┌── Ellinia  [Victoria Road] ─────────┐  ┌── Perion  [Victoria Road] ┐  │
│  │                         [3 spawns]  │  │                 [1 spawn]  │  │
│  └─────────────────────────────────────┘  └────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────┘
```

Outer container: `flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto` — the `overflow-y-auto` on the outer container preserves full-page scroll for drop-heavy monsters.

## 2. Header interaction

- Trigger: single `TooltipTrigger` wrapping a `<span className="inline-flex items-center gap-3">` containing `<img>` + `<h2>`.
- `TooltipContent copyable`: `<p>{monster.id}</p>` — same treatment as `MapHeader.tsx:23-34`.
- Keyboard: the trigger span is `tabIndex={0}` so the tooltip opens on focus. `Enter`/`Space` on the focused trigger triggers the tooltip's copy action (the `copyable` prop handles this).
- The name's `cursor-help` class signals the interactive affordance; the icon inherits a `cursor-help` from the trigger span.
- Badges (`Boss`, `Undead`, `Friendly`) render **outside** the tooltip trigger — they sit on the same flex row but are not part of the tooltip target.

## 3. Stat cards spacing

Existing → new:

| Property | Current | New |
|---|---|---|
| Card grid gap | `gap-4` | `gap-3` |
| CardHeader padding | default (py-6) | `py-2 px-4` override |
| CardContent padding | default | `py-2 px-4` override |
| Row spacing inside card | `space-y-2` | `space-y-1` |

Fields and `text-sm` body size stay the same. The goal is ~30% vertical height reduction per card.

## 4. Skills chips

- Rendered only when `attrs.skills.length > 0`.
- Container: `CardContent` with `flex flex-wrap gap-2`.
- Each chip: shadcn `Badge variant="outline"` with custom padding `px-2.5 py-1 text-xs font-medium`.
- Content: when name loaded → `{name} · L{level}`; while loading or missing → `#{id} · L{level}` (numeric prefix makes the fallback obvious).
- No click / navigation behavior, no tooltip.

## 5. Drops grouping — decision tree

```
for each drop d:
  if d.attributes.itemId === 0        → group "mesos"
  else getItemType(itemId):
    "Equipment"                        → group "equipment"
    "Consumable"                       → group "consumable"
    "Setup"                            → group "setup"
    "Etc"                              → group "etc"
    "Cash"                             → group "cash"
    "Pet" | "Unknown"                  → group "other"
```

Render order: `mesos`, `equipment`, `consumable`, `setup`, `etc`, `cash`, `other`. Empty groups are omitted.

Within each group, drops are rendered in the order returned by the API (atlas-drop-information already orders them; no client re-sort).

## 6. Drop widget (non-meso) — hover contract

Widget layout:
```
┌─────────────────────────────────────────┐
│ [icon 32px]  Item Name (truncate)       │
│              1002140                     │  ← itemId, font-mono text-xs
└─────────────────────────────────────────┘
     whole rectangle is a <Link to="/items/:id">
```

Tooltip content (on hover, simple shadcn `Tooltip`, not `copyable`):
```
Chance     1 / 12,500
Min Qty    1
Max Qty    1
Quest      2000
```

- `Chance` is rendered as `{chance.toLocaleString()}` only (current display).
- `Quest` row is omitted when `questId === 0`.
- Multiple drop rows for the same itemId (rare, but possible for quest-gated drops) render as separate widgets — do not merge.

Icon fallback: if `useItemData(itemId).iconUrl` is falsy, render a `lucide-react Package` icon at size 32 with `text-muted-foreground`.

## 7. Meso widget — visual distinction

Intentionally different from item widgets so currency reads at a glance:

```
┌─────────────────────────────────────────┐
│ 🪙 Mesos                                │
│     100 – 250                            │  ← {min} – {max}, toLocaleString
└─────────────────────────────────────────┘
     (amber tint, not clickable, no link)
```

Styling specifics:
- Border: `border-amber-300/40` light, `dark:border-amber-700/40`.
- Background: `bg-amber-50/50 dark:bg-amber-950/20`.
- Icon: `lucide-react Coins` at 20px, `text-amber-500`.
- Tooltip: Chance only (quantity already visible on the widget).

Multiple meso rows (very rare): each renders as its own widget stacked in the grid.

## 8. Spawn Locations widget

```
┌─────────────────────────────────────────────┐
│ Henesys Hunting Ground 1  [Victoria Road]   │
│ [6 spawns]                                   │
└─────────────────────────────────────────────┘
     whole rectangle is a <Link to="/maps/:id">
```

- Layout: `flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors`.
- Line 1: name (`text-sm font-medium truncate`) + optional `Badge variant="secondary"` for street name.
- Line 2: `Badge variant="outline"` with the count.
- Pluralization: `1 spawn` vs `N spawns`.
- Grid responsive: 1 col on mobile, 2 on `sm`, 3 on `lg`. Starts with 3-col on desktop per the user's "2 or 3 seems fine" guidance — pick 3 (matches stat cards and drop widgets). Easy to iterate to 2 later.

## 9. Loading and error states

| Section | Loading | Error | Empty |
|---|---|---|---|
| Page-level | Existing `<PageLoader />` while `useMonster` loads | Existing `<ErrorDisplay>` | (N/A — if monster not found → error) |
| Drops card | "Loading drops..." | `<ErrorDisplay>` inside card content | "No drops configured for this monster." |
| Spawn Locations card | "Loading spawn locations..." | `<ErrorDisplay>` inside card content | "This monster does not spawn on any loaded map." |
| Skills chips | Per-chip numeric-id fallback while `useMobSkillData` pending | Numeric-id fallback | N/A (card hidden when no skills) |

## 10. Accessibility

- Header tooltip: trigger is keyboard-focusable (`tabIndex=0`), focus ring via `focus-visible:ring-2`.
- Drop widgets: `<Link>` elements inherit keyboard focus; tooltip opens on focus per shadcn defaults. Each link has an implicit accessible name from the item name text content.
- Meso widget: not interactive — no focus ring, no keyboard handler. Decorative role.
- Spawn widgets: same as drop widgets — focusable links.
- Skills chips: not focusable (no interaction).

## 11. Out-of-scope interactions (do not implement)

- Clicking the skills chip — mob skills have no detail page yet.
- Bulk-copying drop ids from the page.
- Filtering drops by chance threshold.
- A compact/expanded toggle for the Drops card — we ship with the dense layout as the default.
- Hover-highlighting a map-detail spawn overlay from this page. (That would require cross-page context that isn't on the table.)
