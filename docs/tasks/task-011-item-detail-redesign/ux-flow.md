# Item Detail Redesign — UX Flow

Supplementary layout reference for §4 of `prd.md`.

---

## Top-to-bottom layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  [icon]  Zakum Helmet 3      [EQUIPMENT]                            │  ← Header (tooltip-to-copy id)
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌── Stats ──────────────────────────────────────────────────────┐  │  ← Equipment only.
│  │  STR 15   DEX 15   INT 15   LUK 15                            │  │    Merged stats / combat
│  │  HP  0    MP  0    Speed 0  Jump 0                            │  │    / properties card.
│  │  W.Atk 0  M.Atk 0  W.Def 150 M.Def 150                        │  │    Price is NOT here.
│  │  Acc   20 Avoid 20 Slots 10 Cash No                           │  │
│  │  TimeLimited No                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌── Requirements ───────────────────────────────────────────────┐  │  ← Rendered only when any
│  │  Level 50                                                     │  │    req* field is non-zero.
│  │  Job   [any]                                                  │  │    Job renders as class
│  │  STR   -     DEX   -     INT   -     LUK   -                  │  │    badges (if not 0).
│  │  POP   -     Fame  -                                          │  │    Zero rows omitted.
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌── Sold By (NPC: 2, Cash Shop: 1) ─────────────────────────────┐  │
│  │                                                               │  │
│  │  NPC SHOPS (2)                                                │  │
│  │  ┌─────────────────────────┐ ┌─────────────────────────┐      │  │
│  │  │ [icon] Henesys Weapon   │ │ [icon] Perion Weapon    │      │  │
│  │  │        50,000 mesos     │ │ 5 × 4031000 (token)     │      │  │  ← Each widget links to
│  │  │        [Henesys · VR]   │ │ [Perion · Victoria Rd]  │      │  │    `/npcs/{id}/shop`.
│  │  └─────────────────────────┘ └─────────────────────────┘      │  │
│  │                                                               │  │
│  │  CASH SHOP (1)                                                │  │
│  │  ┌───────────────────────────────────────────┐                │  │
│  │  │ 💎 NX Cash · 3,900 NX              [SALE] │                │  │  ← Amber-tinted, non-link.
│  │  │    30 days                                │                │  │
│  │  └───────────────────────────────────────────┘                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌── Dropped By (17) ────────────────────────────────────────────┐  │
│  │  ┌────────────────────┐ ┌────────────────────┐                │  │
│  │  │ [m] Zakum Arm      │ │ [m] Zakum Body     │                │  │  ← Sort: chance DESC.
│  │  │     8800000        │ │     8800001        │                │  │    Hover → chance + qty
│  │  └────────────────────┘ └────────────────────┘                │  │    + questId tooltip.
│  │  ┌────────────────────┐ ┌────────────────────┐                │  │    Click → monster page.
│  │  │ [m] Zakum (Summon) │ │ [m] Zakum Arm 2    │                │  │
│  │  │     8800002        │ │     8800003        │                │  │
│  │  └────────────────────┘ └────────────────────┘                │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Header interaction

- Hovering the item icon **or** the item name opens the same tooltip (single shared trigger, `<span>` wraps both).
- Tooltip content is the raw template id (`1002357`). `copyable` prop means clicking the tooltip copies to clipboard.
- Type badge ("Equipment" / "Consumable" / "Setup" / "Etc" / "Cash") uses the existing `getItemTypeBadgeVariant` color.
- When the icon errors, the icon slot renders nothing — the `<h2>` + badge still render.

## Requirements card — special behavior

| Field | Always-show rule |
|---|---|
| `Level` | Show when card renders (may be 0) |
| `Job` | Show as badges only when `reqJob !== 0`. Bitmask expands to Warrior / Magician / Bowman / Thief / Pirate |
| `STR`, `DEX`, `INT`, `LUK` | Show only when `> 0` |
| `POP`, `Fame` | Show only when `> 0` |

The card itself renders only when *any* of `reqLevel`, `reqJob`, the four stats, `reqPop`, or `reqFame` is non-zero. For the vast majority of gear none of these are set and the card is omitted.

## Sold By — subsection rules

- **Card renders** when sellers OR commodities are non-empty.
- **NPC SHOPS subsection** renders when at least one seller row exists.
- **CASH SHOP subsection** renders when at least one commodity row exists.
- Empty card: render once with copy `No shops or commodities sell this item.` instead of a subsection layout.

## NPC Shop widget — column collapse

```
Wide (≥640px):   [icon] [name + price]         [map badge]
Narrow (<640px): [icon] [name + price + map badge on line 3]
```

Map badge falls below the price line instead of to the right when the viewport is narrow. When the NPC has no `npc_spawn_index` entry for the tenant, the map badge is omitted entirely (the widget stays in its two-column layout).

## Dropped By — tooltip

```
┌──────────────────────────┐
│ Chance: 1,250,000        │
│ Min Qty: 1               │
│ Max Qty: 1               │
│ Quest ID: 3213           │ ← only if questId > 0
└──────────────────────────┘
```

Quest ID row is suppressed when `questId === 0`. The rest of the tooltip is always present for every drop row.

## Empty / loading / error states

| Card | Loading | Empty | Error |
|---|---|---|---|
| Stats / Requirements | Hidden until detail query resolves; skeleton OK | Requirements hidden if all zero | `ErrorDisplay` inside card body |
| Sold By | `Loading shop data…` | `No shops or commodities sell this item.` | `ErrorDisplay` per sub-query |
| Dropped By | `Loading drop sources…` | `No monsters drop this item.` | `ErrorDisplay` inside card body |

Top-level page loader renders only while `nameQuery` or `detailQuery` is pending. Sub-queries degrade gracefully per card.
