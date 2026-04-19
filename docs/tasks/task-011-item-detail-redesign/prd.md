# Item Detail Redesign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-19
---

## 1. Overview

`ItemDetailPage` in atlas-ui today (`services/atlas-ui/src/pages/ItemDetailPage.tsx:33-158`) is a functional but low-density page: a thin header row with an icon + name + type badge, a "General" card that repeats the template id / name / type already on that header, multiple stat/combat/properties cards each under-filled, and a flat `Table` for drop sources. For operators investigating "where does this item come from?" or "what are the stats on this weapon?" the page buries the useful data under repeated labels and gives no hint that the item might be purchasable from an NPC or cash shop.

This task refactors the page to mirror the information-dense pattern established by `MapDetailPage` (task-008) and `MonsterDetailPage` (task-010):

- A header with a tooltip-to-copy template id, dropping the redundant "General" card entirely.
- For equipment, a single merged "Stats" card combining stats / combat / properties, plus a new "Requirements" card exposing `reqLevel` / `reqJob` / `reqSTR` / `reqDEX` / `reqINT` / `reqLUK` / `reqPOP` / `reqFame` (currently not plumbed through).
- A new "Sold By" card showing which NPCs sell the item (with NPC icon, name, price, map badge, link to that NPC's shop page) and a "Cash Shop" subsection when commodity rows reference the item. The item's `price` field moves into this card so price information lives in one place.
- A compact widget-grid "Dropped By" card replacing the current flat table — monster icon + name, tooltip for chance / min / max / quest id, click navigates to the monster detail page. Sorted by chance desc.

Data dependencies require three backend changes. First, `atlas-data`'s equipment reader already parses `reqLevel`/`reqJob` in tests but the fields never reach the REST model — add them plus the four stat reqs and `reqPOP`/`reqFame`. Second, a new `npc_spawn_index` in `atlas-data` mirroring the `monster_spawn_index` from task-010 so the UI can look up "which map does this NPC live on". Third, a reverse lookup in `atlas-npc-shops` (`GET /commodities/items/{itemId}`) returning every `(npcId, price, period, …)` pair that carries the item for the active tenant, and a sibling lookup in `atlas-data` for the global cash-shop `commodity/` table.

## 2. Goals

Primary goals:
- Give operators an at-a-glance, information-dense view of an item: what it does, who sells it, what drops it, and (for equipment) what you need to wear it — all without tab-switching or reading labels twice.
- Make every reference clickable to its detail page so tracing flows are one click away (NPC → shop page, monster → monster detail, map → map detail).
- Eliminate the "raw id + name + type" General card in favor of the tooltip-to-copy pattern already used by `MapHeader` and `MonsterHeader`.
- Expose equipment requirements through REST so the UI can render them; today they're parsed by the WZ reader but dropped on the floor.
- Preserve the existing route `/items/:id`, existing breadcrumbs, and existing API contracts for the type-specific item detail endpoints.

Non-goals:
- Editing item stats, drops, requirements, shop availability, or cash-shop commodity rows from this page.
- Rendering equipment scroll/upgrade slots with progress tracking — continue to show `slots` as a bare integer.
- Filtering/sorting drops by any column other than chance desc — one canonical sort order.
- A per-map NPC position display or rendering NPCs on the map render.
- Changes to `ItemsPage` list or columns, to `NpcShopPage`, or to `MonsterDetailPage`'s drops section.
- Back-filling `npc_spawn_index` from existing map documents — re-ingest is acceptable (same call as task-010 made for `monster_spawn_index`).
- Rendering commodity items beyond flagging "available in cash shop for X NX / period Y / on-sale?". No integration with cash-shop purchase flow.
- Showing who has this item equipped, in inventory, etc. The page reflects the tenant's *data definitions*, not live state.

## 3. User Stories

- As an operator diagnosing "item X is impossible to get", I want the item page to immediately show me the NPCs selling it and the monsters dropping it, so I can spot a missing shop row or drop table.
- As a GM writing patch notes, I want to hover the item name and copy its template id into my notes, without parroting it down a second time from an "ID" field below.
- As a designer balancing a quest, I want to see at a glance which monsters drop the quest item and hover for the drop chance, so I can gauge completion time.
- As a designer building a new class, I want the equipment page to show level/job/stat requirements so I can verify a piece of gear is wearable by the intended class.
- As a shop admin, I want the item page to surface which NPC shops carry the item and at what meso/NX price, and let me jump to that shop's admin view in one click.
- As a player-support operator, I want to know whether an item is a cash-shop SKU so I can explain "you bought it with NX, not mesos" without cross-referencing another tool.

## 4. Functional Requirements

### 4.1 Page layout (`ItemDetailPage.tsx`)

From top to bottom, inside a scrolling container (`flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16` — matches the existing page):

1. **Header row** (§4.2) — item icon + name + type badge + tooltip-to-copy template id. Replaces both the current header and the "General" card.
2. **Type-specific detail card(s)** (§4.3) — for equipment, one merged "Stats" card + one "Requirements" card. For consumable / setup / etc / cash, a single "Properties" card consolidating today's scattered fields (scroll effects, spec, time windows stay as today where they apply).
3. **Sold By card** (§4.4) — NPC-shop widgets + optional Cash Shop subsection. Also the canonical home for `price` (formerly in the equipment "Properties" card).
4. **Dropped By card** (§4.5) — widget grid replacing the current flat table.

No "General" card. No duplicated template id elsewhere on the page.

### 4.2 Header

Follow `MonsterHeader.tsx:9-57` exactly:

- `flex items-center gap-3 flex-wrap`.
- `TooltipProvider` → `Tooltip` → `TooltipTrigger asChild` → `<span tabIndex={0}>` wrapping the icon and the name. Classes `inline-flex items-center gap-3 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded`.
- Icon: `<img>` at `width={64} height={64}` sourced from `getAssetIconUrl(tenant, ..., 'item', parseInt(itemId))` when `activeTenant` is available and the image hasn't errored. Fallback on error: render nothing in the icon slot (mirrors the MonsterHeader "no iconUrl = render nothing" contract; no placeholder `Package` icon that visually competes with the name).
  - Keep the existing `iconFailed` state + `useEffect` reset on tenant/item change from `ItemDetailPage.tsx:38-42` so the fallback is stable.
- Name: `<h2 className="text-2xl font-bold tracking-tight">{itemName || itemId}</h2>`.
- `TooltipContent copyable` — clicking copies the template id. Content: `<p>{itemId}</p>`.
- To the right of the span, the existing `Badge variant="secondary"` with the type-specific color via `getItemTypeBadgeVariant(itemType)` (`types/models/item.ts:17-27`).

No "General" card renders below. Template id, name, and type are surfaced exclusively through the header + badge + tooltip.

### 4.3 Type-specific detail card(s)

Routed by the existing `renderTypeSpecificSection` switch (`ItemDetailPage.tsx:170-179`), but restructured per type.

**Equipment — two cards.**

Card 1: "Stats" (merges today's three Equipment cards).
- `CardHeader` standard padding; `CardTitle` text `Stats` at default size.
- `CardContent` uses `grid gap-4 md:grid-cols-3 lg:grid-cols-4` (same dense grid as today).
- Fields (in this order, no repetition across cards): STR, DEX, INT, LUK, HP, MP, Weapon Attack, Magic Attack, Weapon Defense, Magic Defense, Accuracy, Avoidability, Speed, Jump, Upgrade Slots, Cash, Time Limited.
- `Price` is **removed** from this card and moved to the Sold By card (§4.4).

Card 2: "Requirements" — new.
- Renders only if at least one of the following is non-zero: `reqLevel`, `reqSTR`, `reqDEX`, `reqINT`, `reqLUK`, `reqPOP`, `reqFame`. If `reqJob` is non-zero it renders regardless (class restrictions imply the item is gated even without stat minimums).
- `CardContent` uses `grid gap-4 md:grid-cols-3 lg:grid-cols-4`.
- Fields (omit any whose value is 0, except `Level` which renders even at 0 for explicit "no level requirement" clarity when the card is otherwise non-empty):
  - `Level` — integer.
  - `Job` — **rendered as a list of class badges**, not a raw integer. `reqJob` is a bitmask from the WZ (0 = any, 1 = warrior, 2 = magician, 4 = bowman, 8 = thief, 16 = pirate; combined bits mean "any of these"). A tiny helper (`formatReqJob(reqJob: number): string[]`) expands the bitmask into a string list; the UI renders each as a `Badge variant="outline"`. If the bitmask is 0 the row is omitted.
  - `STR`, `DEX`, `INT`, `LUK` — integer, only if > 0.
  - `POP` (charisma/fame in some regions) — integer, only if > 0.
  - `Fame` — integer, only if > 0. (May never appear; WZ splits `reqPOP` vs `reqFame` inconsistently across regions. Expose both; suppress the empty one.)
- If the entire card would be empty (no level, no job, no stats, no pop/fame), do not render it.

**Consumable — "Properties" card (consolidated).**
- One `Card` with the existing grid. Fields: Slot Max, Required Level, Unit Price, Quest Item, Trade Block, Not For Sale, Time Limited, Rechargeable. `Price` moves to Sold By.
- Existing "Scroll Effects" card (rendered only when `success > 0`) stays as-is.
- Existing "Spec" card (rendered when `spec` is non-empty) stays as-is.

**Setup — "Properties" card.** Fields: Slot Max, Recovery HP, Required Level, Trade Block, Not For Sale, Time Limited. `Price` moves to Sold By.

**Etc — "Properties" card.** Fields: Slot Max, Unit Price, Time Limited. `Price` moves to Sold By.

**Cash — "Properties" card.** Fields: Slot Max. Existing "Spec" and "Time Windows" cards remain. No price (cash items aren't priced in mesos on the data-definition side; their NX price lives in commodity rows surfaced via Sold By).

### 4.4 Sold By card

New card. Rendered below the type-specific detail cards, above Dropped By.

- Title: `Sold By`, with combined count in parens (`Sold By (NPC: 3, Cash Shop: 1)` — see below).
- Two subsections, each rendered only when non-empty, in this order:
  1. **NPC Shops** (`h3` label `NPC SHOPS` with `text-sm font-medium text-muted-foreground uppercase tracking-wide`, count in parens).
  2. **Cash Shop** (`h3` label `CASH SHOP`, count in parens).

**Data sources:**
- `useItemSellers(itemId)` — new hook (`src/lib/hooks/api/useItemSellers.ts`). Calls the new `GET /commodities/items/{itemId}` endpoint on atlas-npc-shops (§5.1). Response shape: `Array<{ commodityId, npcId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimit }>`.
- `useItemCommodities(itemId)` — new hook. Calls the new `GET /api/data/commodity/by-item/{itemId}` endpoint on atlas-data (§5.3). Response shape: `Array<{ sn, itemId, count, price, period, priority, gender, onSale }>`.
- `useNpcMapLocation(npcId)` (batch variant preferred; see §4.6) — returns the map id / name / street name where the NPC spawns for the active tenant. Backed by the new `npc_spawn_index` (§5.2 + §6.2).
- `useNpcData(npcId)` — existing hook; returns NPC name + icon from the WZ-string registry.

**NPC Shop widget (`ItemNpcShopWidget`)** — `components/features/items/ItemNpcShopWidget.tsx`:
- Root: `Link` to `/npcs/{npcId}/shop` — `flex items-center gap-3 rounded-md border bg-card p-3 hover:bg-accent transition-colors`.
- Left column: `<img>` at `32x32 loading="lazy"` of the NPC icon via `useNpcData(npcId).iconUrl`. Fallback: `lucide-react UserCircle2` at the same box size.
- Middle column (grows): 
  - Line 1: NPC name (`text-sm font-medium truncate`) — falls back to `NPC #{npcId}` while loading.
  - Line 2: price. Render `{mesoPrice.toLocaleString()} mesos` when `mesoPrice > 0`. If `tokenPrice > 0` and a `tokenTemplateId` is set, append ` · {tokenPrice.toLocaleString()} × item {tokenTemplateId}` (the token's name is out of scope — show the id). If both are zero, show `Free`.
- Right column (collapses below 640px): map location badge.
  - `Badge variant="secondary"` text `{mapName} · {streetName}` when `streetName` is present, otherwise `{mapName}`. Omit the badge entirely if the NPC has no known spawn map for this tenant (degrades to two-column layout on that row — acceptable).
- Widget wrapped in a `Tooltip` (no `copyable`). Tooltip content surfaces the fields NOT shown on the card face:
  - `Commodity ID`, `Discount Rate` (if non-zero), `Period` (hours), `Level Limit` (if non-zero). Each as a `<p>` line.

**Cash Shop widget (`ItemCashShopWidget`)** — `components/features/items/ItemCashShopWidget.tsx`:
- Not a link (no cash-shop admin page exists at the UI level today).
- `flex items-center gap-3 rounded-md border border-amber-300/40 bg-amber-50/50 dark:bg-amber-950/20 p-3` — amber tint matches the `MonsterMesoWidget` convention.
- Left: `lucide-react Gem` icon at size 20, `text-amber-500`.
- Middle: two-line block:
  - Line 1: `NX Cash · {price.toLocaleString()} NX · {count}×` (`text-sm font-medium`). The `×count` suffix is omitted if `count === 1`.
  - Line 2 (small, muted): Period label (`{period} days` when `period > 0`, `Permanent` when `period === 0`).
- Badges on the right: `Badge variant="default"` reading `ON SALE` only when `onSale === true`. Gender-gated flag if `gender !== 2` (`Male`/`Female`).
- Tooltip on hover shows `SN: {sn}` + `Priority: {priority}`.

**Sold By — empty state:** when both subsections are empty, render `No shops or commodities sell this item.` in `text-sm text-muted-foreground` as the card body (no subsection headings). Loading state: `Loading shop data...` using the same pattern as Dropped By.

### 4.5 Dropped By card

Replaces the current flat `Table` (`ItemDetailPage.tsx:124-155`) with a widget grid.

- Title: `Dropped By` with count in parens.
- Layout: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2`.
- Sort: **by chance desc, tie-breaker by monster id asc**. One canonical order — no user-selectable sort.
- Hook: existing `useItemDrops(itemId)`.
- Pre-warm monster data: the existing `useMobData(monsterId)` is called per widget, which already dedupes via React Query cache. If a perf issue surfaces, introduce a batched `useMobBatchData` hook mirroring the item version at `useItemData.ts:170-262` (tracked as a follow-up, not in scope).

**Drop widget (`DroppedByWidget`)** — replaces `DroppedByTableRow.tsx`:
- Root: `Link` to `/monsters/{monsterId}` — `flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors`.
- Left: `<img>` at `32x32 loading="lazy"` from `useMobData(monsterId).iconUrl`. Fallback: no image (render an empty 32x32 box) — do not use a generic `Package` placeholder since that's the item fallback.
- Right: two-line block:
  - Line 1: monster name (`text-sm font-medium truncate`). Fallback `Monster #{monsterId}` while loading.
  - Line 2: `text-xs font-mono text-muted-foreground` rendering the monster id.
- Tooltip on hover (no `copyable`) surfaces:
  - `Chance: {chance.toLocaleString()}` (raw-numerator convention — matches existing table).
  - `Min Qty: {minimumQuantity}` / `Max Qty: {maximumQuantity}`.
  - `Quest ID: {questId}` — only rendered if `questId > 0`.
- Clicking the widget navigates to `/monsters/{monsterId}`. Tooltip is informational only.

Empty / loading states match the current copy: `"Loading drop sources..."` / `"No monsters drop this item."`.

### 4.6 Page data dependencies

The refactored page fetches:
- `nameQuery` (item name) — existing.
- `detailQuery` (type-specific detail) — existing.
- `useItemDrops(itemId)` — existing.
- `useItemSellers(itemId)` — new (§5.1).
- `useItemCommodities(itemId)` — new (§5.3).
- `useNpcSpawnMaps(npcIds)` — new batch hook. Takes the deduplicated list of `npcId`s from the sellers response and returns a `Record<npcId, { mapId, name, streetName }>`. Implementation: one HTTP request per NPC (simple), dedup via React Query's cache — per-NPC query keyed by `["npcs", "spawn-map", npcId, tenantId]`, 10-minute stale time. If any seller set exceeds ~20 NPCs we revisit batching; tenant-level shops are small enough that per-NPC requests are acceptable.
- `useNpcData(npcId)` — existing; called inside each NPC shop widget.
- `useMobData(monsterId)` — existing; called inside each drop widget.

Loading orchestration:
- Fetches run in parallel. Each card renders its own loading / error / empty state; a single spinner for the whole page is reserved for the base item queries (`nameQuery` + `detailQuery`).
- Sellers / drops / commodities loading fall into the per-card treatment so the page is usable even if one of them is slow.

## 5. API Surface

### 5.1 New: `GET /commodities/items/{itemId}` (atlas-npc-shops)

Returns every `commodities` row whose `TemplateId = itemId` for the active tenant, with enough context to render the widget.

- Route registration: in `shops.InitResource` or a new sibling `commodities.InitResource` — place with the other top-level `/shops` / `/commodities` routes in `atlas-npc-shops/atlas.com/npc/shops/resource.go`.
  - `router.HandleFunc("/commodities/items/{itemId}", rest.RegisterHandler(l)(db)(si)("get_commodities_by_item", handleGetCommoditiesByItem)).Methods(http.MethodGet)`
- Handler: parses `itemId` (uint32, reject non-numeric with 400), then queries `commodities.Entity` with `tenant_id = ? AND template_id = ?`, returns each as a `CommodityWithNpcRestModel`.
- Response rest model (new, at `atlas-npc-shops/.../commodities/rest_by_item.go` or similar):
  ```go
  type CommodityByItemRestModel struct {
      Id              uuid.UUID `json:"-"`
      NpcId           uint32    `json:"npcId"`
      TemplateId      uint32    `json:"templateId"`
      MesoPrice       uint32    `json:"mesoPrice"`
      DiscountRate    byte      `json:"discountRate"`
      TokenTemplateId uint32    `json:"tokenTemplateId"`
      TokenPrice      uint32    `json:"tokenPrice"`
      Period          uint32    `json:"period"`
      LevelLimit      uint32    `json:"levelLimit"`
  }
  func (r CommodityByItemRestModel) GetName() string { return "commodities" }
  func (r CommodityByItemRestModel) GetID() string   { return r.Id.String() }
  ```
- Errors:
  - `400` if `itemId` is unparseable.
  - `500` on DB failure.
  - Empty array on zero matches (200 with `data: []`) — not `404`.
- Content type: `application/vnd.api+json`.
- Tenant-scoping: inherited from `rest.HandlerDependency.Context()` / `d.Context()` — same as every other route in this service.

### 5.2 New: `GET /data/npcs/{npcId}/map` (atlas-data)

Returns the primary spawn map for an NPC in the active tenant, or 404 if the NPC is not indexed.

- Route registration: in `npc.InitResource` (`services/atlas-data/atlas.com/data/npc/resource.go`), add a new sub-route:
  - `r.HandleFunc("/{npcId}/map", registerGet("get_npc_map", handleGetNpcMapRequest(db))).Methods(http.MethodGet)`
- Handler: parses `npcId`, queries the new `npc_spawn_index` table filtered by `(tenant_id, npc_id)`, returns the *single* row with the highest `spawn_count` (tie-breaker: lowest `map_id` — deterministic). NPCs commonly appear on one map, but edge cases exist (e.g., the same NPC id wired into multiple towns); returning the primary one is simpler than a list for the UI needs here.
- Response rest model (new, colocated under `npc/`):
  ```go
  type NpcMapRestModel struct {
      NpcId      uint32 `json:"-"`
      MapId      uint32 `json:"mapId"`
      Name       string `json:"name"`
      StreetName string `json:"streetName"`
      SpawnCount uint32 `json:"spawnCount"`
  }
  func (r NpcMapRestModel) GetName() string { return "npc-maps" }
  func (r NpcMapRestModel) GetID() string   { return strconv.Itoa(int(r.NpcId)) }
  ```
- Errors:
  - `400` if `npcId` unparseable.
  - `404` if no index row exists for that NPC (caller treats as "no known location").
  - `500` on DB failure.

### 5.3 New: `GET /data/commodity/by-item/{itemId}` (atlas-data)

Returns every cash-shop `commodity` row whose `ItemId = itemId` for the active tenant. Separate from the existing `GET /data/commodity/items/{itemId}` (which looks up by the commodity SN keyed as a string — see `commodity/resource.go:45-63`) because a single itemId can have multiple commodity entries (different counts, periods, gender, on-sale status).

- Route registration: in `commodity.InitResource`, sibling to the existing routes:
  - `r.HandleFunc("/by-item/{itemId}", registerGet("get_commodities_by_item", handleGetCommoditiesByItemRequest(db))).Methods(http.MethodGet)`
- Handler: iterate the existing commodity storage `GetAll` (tenant-scoped) and filter by `ItemId == itemId` (the registry is in-memory so this is cheap); return as `[]RestModel`.
- Response rest model: reuses the existing `commodity.RestModel` — no new type required.
- Errors: `400` on bad id; empty array on zero matches (not 404).

### 5.4 Modified: `atlas-data` equipment `RestModel`

Add the following fields to `equipment.RestModel` (after `Slots`, before `Cash`):

```go
ReqLevel uint16 `json:"reqLevel"`
ReqJob   uint16 `json:"reqJob"`
ReqStr   uint16 `json:"reqStr"`
ReqDex   uint16 `json:"reqDex"`
ReqInt   uint16 `json:"reqInt"`
ReqLuk   uint16 `json:"reqLuk"`
ReqPop   uint16 `json:"reqPop"`
ReqFame  uint16 `json:"reqFame"`
```

Wire in `equipment.Read`: one `info.GetShort("reqLevel", 0)` call per field (name casing per WZ: `reqLevel`, `reqJob`, `reqSTR`, `reqDEX`, `reqINT`, `reqLUK`, `reqPOP`, `reqFame`). These are already present in XML (see `equipment/reader_test.go:22-27`); reader-side unit test already exercises them as test fixtures, just assert them on the output.

Backwards-compatibility: additive JSON fields. Existing consumers ignore unknown fields. The REST shape is unchanged for non-equipment types.

### 5.5 No change: existing endpoints

- `GET /api/data/items/{itemId}/name` — unchanged.
- `GET /api/data/<type>/{itemId}` — unchanged (equipment adds fields per §5.4; consumable / setup / etc / cash unchanged).
- `GET /api/drops?filter[itemId]=…` (or wherever `useItemDrops` ultimately resolves) — unchanged.

## 6. Data Model

### 6.1 Modified: equipment WZ reader

No schema change. Reader simply stamps more fields onto the existing `RestModel`. See §5.4.

### 6.2 New table: `npc_spawn_index` (atlas-data)

Backing entity under `services/atlas-data/atlas.com/data/npc/spawn_index.go` (new file; mirrors the `monster_spawn_index` pattern from task-010):

```go
type SpawnIndexEntity struct {
    TenantId   uuid.UUID `gorm:"type:uuid;primaryKey"`
    NpcId      uint32    `gorm:"primaryKey"`
    MapId      uint32    `gorm:"primaryKey"`
    Name       string    `gorm:"not null"`
    StreetName string    `gorm:"not null"`
    SpawnCount uint32    `gorm:"not null"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SpawnIndexEntity) TableName() string { return "npc_spawn_index" }
```

- Composite primary key `(tenant_id, npc_id, map_id)`.
- `spawn_count` = number of `npc` entries for that `(npc, map)` pair in the map document. Usually 1; >1 for duplicate spawn definitions.
- `name` / `street_name` denormalized from the owning map.
- Index: `CREATE INDEX IF NOT EXISTS idx_npc_spawn_index_lookup ON npc_spawn_index (tenant_id, npc_id, spawn_count DESC);` — supports the §5.2 "primary map" read.

Migration: added to atlas-data's migration sequence alongside the existing monster / map migrations. Forward-only; no back-fill.

### 6.3 Population

Extend `map.Storage.Add` (`services/atlas-data/atlas.com/data/map/storage.go:50-75`) — the same write hook task-010 used for `monster_spawn_index`:

1. `DELETE FROM npc_spawn_index WHERE tenant_id = ? AND map_id = ?`.
2. Aggregate `m.NPCs` by `npc.Id` into `map[uint32]uint32{npcId: count}`.
3. Bulk `INSERT INTO npc_spawn_index (...) VALUES (...)` — one row per `(npc, map)` pair.

All inside the same transaction as the map document upsert, the `monster_spawn_index` population (task-010), and the `searchindex.Upsert` — partial failure rolls back consistently.

### 6.4 Back-fill

Not required. Re-ingest populates naturally.

### 6.5 No change: cash-shop commodity table

The existing in-memory commodity registry (keyed by SN) already contains the data needed for §5.3. The endpoint iterates the registry; no storage change.

### 6.6 No change: atlas-npc-shops commodities table

`commodities.Entity` (`atlas-npc-shops/atlas.com/npc/commodities/entity.go:8-25`) already indexes by `(tenant_id, npc_id, template_id)` via column-level constraints and the primary key. The reverse lookup in §5.1 is a simple `WHERE tenant_id = ? AND template_id = ?` query — no new index required for correctness, but we SHOULD add a supporting index for performance:

```sql
CREATE INDEX IF NOT EXISTS idx_commodities_by_template
  ON commodities (tenant_id, template_id);
```

Added via a GORM migration alongside the existing `Migration` function at `entity.go:42-44`.

## 7. Service Impact

| Service | Changes |
|---|---|
| **atlas-ui** | Rewrite `pages/ItemDetailPage.tsx`. New `components/features/items/` directory: `ItemHeader.tsx`, `ItemNpcShopWidget.tsx`, `ItemCashShopWidget.tsx`, `DroppedByWidget.tsx`, `EquipmentRequirementsCard.tsx`, `formatReqJob.ts` helper. New hooks: `lib/hooks/api/useItemSellers.ts`, `lib/hooks/api/useItemCommodities.ts`, `lib/hooks/api/useNpcSpawnMap.ts`. New service modules: `services/api/npc-shop-commodities.service.ts`, extend `services/api/commodities.service.ts` (or equivalent) with `getByItem`. Retire `components/features/drops/DroppedByTableRow.tsx` (unused after this change). Extend `types/models/item.ts` with `ReqLevel`/`ReqJob`/`ReqStr`/`ReqDex`/`ReqInt`/`ReqLuk`/`ReqPop`/`ReqFame` on `EquipmentAttributes`. Add `types/models/npc.ts` shapes for the spawn-map response (or extend existing `Npc` types). |
| **atlas-data** | Add `npc/spawn_index.go` (entity + migration). Add `npc/resource.go` route + handler for `/{npcId}/map`. Extend `equipment/reader.go` to read the eight req* fields and `equipment/rest.go` to expose them. Extend `map/storage.go` `Add` to populate `npc_spawn_index`. Update the migration registry to include the new NPC migration. Add `commodity/resource.go` route + handler for `/by-item/{itemId}`. |
| **atlas-npc-shops** | Add reverse-lookup route `GET /commodities/items/{itemId}` (handler + REST model + processor method). Add the supporting `idx_commodities_by_template` index to `commodities.Migration`. |

No other services are touched. No Kafka topics, no new configuration.

## 8. Non-Functional Requirements

**Performance:**
- `npc_spawn_index` lookup is O(rows-per-npc) via the compound index — usually 1 row. Expected p99 < 10ms.
- `commodities` reverse lookup is a single indexed query — p99 < 10ms even on tenants with large shop tables (a tenant with 5000 commodity rows across 200 shops would match at most a handful per item).
- `commodity/by-item` iterates the in-memory registry per request; on a loaded cash-shop snapshot (~2k rows) this is microseconds. If profile shows hot-spotting, add a secondary registry keyed by itemId — but don't pre-optimize.
- UI: each sellers widget fires its own `useNpcData` and `useNpcSpawnMap` query. React Query dedupes; items with ≤20 sellers render in under a second end-to-end. If a single item starts returning more than ~50 sellers we revisit batching — not in scope now.

**Multi-tenancy:**
- Every new row in `npc_spawn_index` carries `tenant_id`, filtered by `tenant.FromContext` on every read.
- The reverse commodity lookup in atlas-npc-shops inherits tenant scoping from `commodities.Entity.TenantId`.
- Cash-shop commodities are registered per tenant in the in-memory registry and cleared on re-ingest (inherits existing `commodity.RegisterCommodity` semantics).

**Observability:**
- Ingest: log one line per map registration summarizing NPC index rows written, `Debug` level. Reuses the existing map ingest logger. Mirror of the monster_spawn_index log added in task-010.
- API: existing `rest.RegisterHandler` instrumentation covers the new routes.

**Security:**
- No new auth surface. Tenant scoping inherited from context on every endpoint.
- Input validation: `ParseItemId` (atlas-data, atlas-npc-shops) and `ParseNpcId` (atlas-data) both exist; reuse them.

**Backwards compatibility:**
- Adding fields to `equipment.RestModel` is additive — existing consumers unaffected.
- The `npc_spawn_index` table is new; migration forward-only.
- The existing `ItemDetailPage` route `/items/:id` stays; only the page body changes.
- `DroppedByTableRow.tsx` is deleted; grep confirms it has no other callers.

## 9. Open Questions

_None remaining._ Resolutions:

1. **Header behavior for item types without WZ icons** — confirmed to mirror the `MonsterHeader` pattern: when the icon errors or is unavailable, render nothing in the icon slot. No placeholder. The existing `iconFailed` state ensures the fallback is deterministic per item.
2. **Cash-shop subsection scope** — confirmed to include the commodity reverse lookup as a distinct subsection in Sold By. User asked what this would look like; answer embedded in §4.4 with the `ItemCashShopWidget` specification.
3. **NPC multi-map ambiguity** — NPCs are occasionally wired into multiple maps (duplicate bindings). The `npc_spawn_index` stores all (npc, map) pairs; `GET /data/npcs/{npcId}/map` returns the *single primary* (highest spawn_count, lowest map_id). For the Sold By card, this is sufficient — the badge shows "Henesys" for an NPC whose primary spawn is Henesys, even if they also appear on a side map. If operators need the full list we add a `/maps` variant later.
4. **Equipment requirements rendering when all zero** — confirmed to suppress the card entirely when no req* field has a non-default value. Does not render an empty "Requirements" card for items with no restrictions.
5. **`reqJob` bitmask rendering** — render as a list of class badges via a client-side helper that expands the bitmask. Bitmask 0 = "any class" (no badges, no row).

## 10. Acceptance Criteria

Backend (atlas-data):
- [ ] `equipment.RestModel` has `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`, `reqPop`, `reqFame`. Reader populates them from the expected WZ names (`reqLevel`, `reqJob`, `reqSTR`, `reqDEX`, `reqINT`, `reqLUK`, `reqPOP`, `reqFame`). Unit test in `equipment/reader_test.go` asserts the values from the existing test XML fixture.
- [ ] `npc_spawn_index` migration runs on startup; table exists with the compound index.
- [ ] Map re-ingest populates `npc_spawn_index` rows per spawned NPC with accurate `spawn_count`. Unit test in `map/storage_test.go` seeds a map with duplicate NPC entries and asserts the row count.
- [ ] `GET /data/npcs/{npcId}/map` returns 200 with the primary map row, 404 when the NPC has no index rows. Tenant-scoped. Unit test in `npc/resource_test.go`.
- [ ] `GET /data/commodity/by-item/{itemId}` returns all commodity rows matching the item for the tenant. 200 with empty array when none. Unit test in `commodity/resource_test.go`.
- [ ] Docker build for atlas-data succeeds.

Backend (atlas-npc-shops):
- [ ] `GET /commodities/items/{itemId}` returns all commodities for the item, scoped to the active tenant. 200 with empty array when none. Unit test in `shops/resource_test.go` (or a new `commodities/resource_test.go`).
- [ ] `idx_commodities_by_template` index exists after migration runs.
- [ ] Docker build for atlas-npc-shops succeeds.

Frontend (atlas-ui):
- [ ] `ItemDetailPage` header shows a copyable tooltip with the template id when the item name or icon is hovered. The "General" card is removed — `grep -L "CardTitle.*General" src/pages/ItemDetailPage.tsx` returns no match.
- [ ] Equipment detail renders a single "Stats" card merging stats + combat + properties (excluding price) and, when any req is non-zero, a "Requirements" card rendering `reqJob` as class badges and stat reqs as filled rows.
- [ ] Consumable / Setup / Etc / Cash type detail cards render without a `Price` field (it moves to Sold By).
- [ ] "Sold By" card renders with "NPC Shops" and "Cash Shop" subsections only when each has data. NPC widgets show NPC icon + name + price + map badge and link to `/npcs/{id}/shop`. Cash-shop widgets render distinctly (amber tint, Gem icon) and are not links.
- [ ] "Dropped By" card renders a widget grid sorted by chance desc, each widget linking to `/monsters/{id}` and surfacing chance / min / max / questId in a hover tooltip.
- [ ] Empty / loading states are handled per card.
- [ ] `DroppedByTableRow.tsx` is deleted and no references remain.
- [ ] `npm run build` and `npm run test` pass. No new ESLint errors.
- [ ] Tenant switching invalidates the new `useItemSellers`, `useItemCommodities`, and `useNpcSpawnMap` caches (automatic via `queryClient.clear()` in `TenantProvider`).

Cross-cutting:
- [ ] Docker compose up, re-ingest one tenant's data, then load `/items/{id}` for:
  - An equipment item with non-zero reqs (e.g., a job-restricted weapon) — Requirements card renders.
  - A consumable that's sold by an NPC — NPC Shops subsection renders with the correct price and map badge.
  - A cash-shop item with a commodity row — Cash Shop subsection renders.
  - An item with many monster drops — Dropped By widget grid renders, sorted by chance desc.
- [ ] No regressions on `MapDetailPage`, `MonsterDetailPage`, or `NpcShopPage` (spot-check after deploy).
