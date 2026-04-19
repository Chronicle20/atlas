# Item Detail Redesign — Implementation Plan

Last Updated: 2026-04-19
Status: Draft
Source PRD: `docs/tasks/task-011-item-detail-redesign/prd.md`
API Contracts: `docs/tasks/task-011-item-detail-redesign/api-contracts.md`
UX Flow: `docs/tasks/task-011-item-detail-redesign/ux-flow.md`

---

## Executive Summary

Refactor atlas-ui's `ItemDetailPage` from a low-density, label-repeating layout into an information-dense page mirroring the pattern established by `MapDetailPage` (task-008) and `MonsterDetailPage` (task-010). The page gains two new data surfaces — "Sold By" (NPC shops + cash shop commodities) and equipment "Requirements" — and replaces the flat drop table with a widget grid. Three backend changes unblock the UI: equipment REST gains eight `req*` fields, atlas-data gains an `npc_spawn_index` + matching lookup endpoint, and atlas-npc-shops gains a reverse commodity lookup.

The change is scoped to one page, three services (atlas-ui, atlas-data, atlas-npc-shops), and additive APIs — no Kafka, no cross-service protocol churn, no backwards-incompatible schema changes.

## Current State Analysis

**atlas-ui** (`services/atlas-ui/src/pages/ItemDetailPage.tsx:1-373`)
- 373-line page with header + "General" card (redundant with header) + type-specific cards + flat Dropped By table.
- Equipment type renders three separate cards (Stats / Combat / Properties) — each sparse.
- `Price` field lives inside type-specific cards, not tied to any shop context.
- No visibility into NPC shops or cash-shop commodities that reference the item.
- Directory `src/components/features/items/` does not exist — no `ItemHeader` yet.
- `DroppedByTableRow.tsx` (`src/components/features/drops/DroppedByTableRow.tsx`) is used only by this page.
- `EquipmentAttributes` (`src/types/models/item.ts:36-55`) has 15 fields; no `req*` fields.

**atlas-data**
- `equipment/rest.go:16-40` — `RestModel` exposes stats/combat/properties + `BonusExp` + `EquipSlots`. No `req*` fields despite `reader_test.go` fixtures containing them.
- `npc/` — has `resource.go` with `GET /data/npcs` + `GET /data/npcs/{npcId}` + search. No `spawn_index.go`.
- `map/storage.go:51-108` — `Add()` already populates `monster_spawn_index` (task-010 pattern) inside the map upsert transaction. The NPC equivalent plugs into the same function.
- `commodity/resource.go:15-25` — exposes `/data/commodity/items` (list) and `/data/commodity/items/{itemId}` (by-SN lookup). No `by-item` reverse path.

**atlas-npc-shops**
- `shops/resource.go:1-39` — CRUD on `/shops` and `/npcs/{npcId}/shop` (+ commodity relationships).
- `commodities/entity.go:9-21` — GORM `Entity` with `tenant_id`, `npc_id`, `template_id`, `meso_price`, `discount_rate`, `token_template_id`, `token_price`, `period`, `level_limit`. `Migration` at `entity.go:42-44` runs `AutoMigrate`.
- No reverse lookup endpoint exists — there is no way today for the UI to ask "which shops sell item X?".

## Proposed Future State

1. Single `ItemHeader` component: icon + name + badge + tooltip-to-copy template id. "General" card removed.
2. Equipment type renders **one** merged "Stats" card (stats + combat + properties sans price) plus a conditional "Requirements" card driven by the eight new `req*` fields. `reqJob` renders as expanded class badges.
3. "Sold By" card below the type-specific cards: NPC shop widgets (linking to `/npcs/{id}/shop`) + Cash Shop widgets (non-link, amber-tinted). `Price` for non-equipment types moves here.
4. "Dropped By" becomes a widget grid (not a table), sorted chance desc, each widget linking to `/monsters/{id}`.
5. Three new REST endpoints: `GET /commodities/items/{itemId}` (atlas-npc-shops), `GET /data/npcs/{npcId}/map` (atlas-data), `GET /data/commodity/by-item/{itemId}` (atlas-data). Equipment REST additively gains eight `req*` fields. New `npc_spawn_index` table populated during map ingest.

## Implementation Phases

The work is ordered so backend precedes frontend within each feature stream, but the three feature streams (equipment reqs, NPC spawn index, commodity reverse lookups) are otherwise independent and can be parallelized once tasks are carved up.

### Phase 1 — Backend data surfaces (atlas-data + atlas-npc-shops)

Unblocks every UI piece. No UI work until Phase 1 is green.

#### 1.1 Equipment `req*` fields (atlas-data) — **S**
1. **Extend `equipment.RestModel` with eight uint16 fields.**
   - Add `ReqLevel`, `ReqJob`, `ReqStr`, `ReqDex`, `ReqInt`, `ReqLuk`, `ReqPop`, `ReqFame` to `services/atlas-data/atlas.com/data/equipment/rest.go` after `Slots` and before `Cash`.
   - Acceptance: struct compiles; JSON tags match PRD §5.4 casing (`reqLevel`, `reqJob`, `reqStr`, …).
2. **Plumb reader.**
   - In `equipment/reader.go`, call `info.GetShort(...)` for each WZ key (`reqLevel`, `reqJob`, `reqSTR`, `reqDEX`, `reqINT`, `reqLUK`, `reqPOP`, `reqFame`) — note the uppercase stat suffixes on the WZ side vs. the camelCase JSON on the REST side.
   - Acceptance: values propagate from XML → Model → REST.
3. **Unit test.**
   - Extend `equipment/reader_test.go` (fixture already carries the fields per PRD §5.4). Assert values on the output model.
   - Acceptance: `go test ./equipment/...` passes.
4. **Docker build.**
   - Acceptance: `docker build services/atlas-data` succeeds.

Dependencies: none. Blocks Phase 2.4 (UI Requirements card).

#### 1.2 NPC spawn index (atlas-data) — **M**
1. **Add `SpawnIndexEntity` + migration.**
   - Create `services/atlas-data/atlas.com/data/npc/spawn_index.go` with the struct from PRD §6.2 (composite PK on `tenant_id, npc_id, map_id`; columns `name`, `street_name`, `spawn_count`, `updated_at`). Register in the migration sequence.
   - Index: `idx_npc_spawn_index_lookup (tenant_id, npc_id, spawn_count DESC)`.
   - Acceptance: migration runs; table + index exist in dev Postgres.
2. **Populate during map ingest.**
   - Extend `map/storage.go` `Add()` (alongside the existing monster_spawn_index population circa `storage.go:73-99`): delete old rows for `(tenant, map)`, aggregate `m.NPCs` by `npc.Id`, bulk insert. Inside the same transaction as the map upsert + monster_spawn_index + search index.
   - Log `"npc_spawn_index: tenant=%s map=%d rows=%d"` at Debug.
   - Acceptance: unit test seeds a map with duplicate NPC entries and asserts the row count + spawn_count aggregation.
3. **Add `GET /data/npcs/{npcId}/map` route + handler.**
   - Register in `npc/resource.go` under the existing `/data/npcs` subrouter.
   - Handler: parse `npcId`, query by `(tenant_id, npc_id)`, return top row by `spawn_count DESC, map_id ASC`. 404 when no rows; 400 on bad id; 500 on DB error.
   - Response: `NpcMapRestModel` per PRD §5.2.
   - Acceptance: `npc/resource_test.go` covers 200 primary-row, 404 no-index, 400 bad-id, tenant-scoping.
4. **Docker build for atlas-data.**

Dependencies: none (independent from 1.1). Blocks Phase 2.3 (Sold By NPC widgets).

#### 1.3 Commodity reverse lookup (atlas-data) — **S**
1. **Add `GET /data/commodity/by-item/{itemId}` route.**
   - Register in `commodity/resource.go` sibling to the existing `/items` routes.
   - Handler: iterate the tenant-scoped in-memory commodity registry, filter by `ItemId == itemId`, return `[]RestModel`. Reuses the existing `RestModel`.
   - Acceptance: 200 with array (may be empty), 400 on bad id. Unit test in `commodity/resource_test.go`.
2. **Docker build for atlas-data** (covered by 1.2.4 if run together).

Dependencies: none. Blocks Phase 2.3 (Sold By Cash Shop widgets).

#### 1.4 Commodity reverse lookup (atlas-npc-shops) — **M**
1. **Add `GET /commodities/items/{itemId}` route + handler.**
   - Register under `atlas.com/npc/shops/resource.go` or a new `commodities/resource.go` sibling — place alongside `/shops` top-level routes.
   - Handler: parse `itemId` (uint32, 400 on failure), query `commodities.Entity` by `tenant_id + template_id`, return `[]CommodityByItemRestModel` per PRD §5.1.
   - Acceptance: 200 with array, empty array on no matches, 400 on bad id, 500 on DB error, tenant-scoped. Unit test (`shops/resource_test.go` or new `commodities/resource_test.go`).
2. **Add supporting index in migration.**
   - Extend `commodities.Migration` (`commodities/entity.go:42-44`) to create `idx_commodities_by_template (tenant_id, template_id)` if not exists.
   - Acceptance: index present after migration run.
3. **Docker build for atlas-npc-shops.**

Dependencies: none. Blocks Phase 2.3 (Sold By NPC widgets).

### Phase 2 — Frontend page refactor (atlas-ui)

All Phase 2 tasks depend on Phase 1 — backend endpoints + additive fields must be live.

#### 2.1 Types + service modules — **S**
1. **Extend `EquipmentAttributes`.**
   - Add `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`, `reqPop`, `reqFame` (all `number`) to `src/types/models/item.ts:36-55`.
   - Acceptance: TS compiles; page consumers see the fields.
2. **Add NPC spawn-map type.**
   - New shape `NpcSpawnMap = { mapId: number; name: string; streetName: string; spawnCount: number }` (or extend `src/types/models/npc.ts`).
3. **Service modules.**
   - New `src/services/api/npc-shop-commodities.service.ts` with `getByItem(itemId)`.
   - Extend `src/services/api/commodities.service.ts` (or equivalent) with `getByItem(itemId)` against `/api/data/commodity/by-item/{itemId}`.
   - Extend NPC service with `getSpawnMap(npcId)` against `/api/data/npcs/{npcId}/map`, distinguishing 404 from other errors so the hook can return `null`.
   - Acceptance: each function tenant-scopes via existing header injection; 404 from `getSpawnMap` resolves to `null` rather than throwing.

#### 2.2 New React Query hooks — **S**
1. **`useItemSellers(itemId)`** — `src/lib/hooks/api/useItemSellers.ts`. Key `["items", itemId, "sellers", tenantId]`, 10-minute stale time.
2. **`useItemCommodities(itemId)`** — `src/lib/hooks/api/useItemCommodities.ts`. Key `["items", itemId, "commodities", tenantId]`.
3. **`useNpcSpawnMap(npcId)`** — `src/lib/hooks/api/useNpcSpawnMap.ts`. Key `["npcs", "spawn-map", npcId, tenantId]`. Treat 404 as "data: null", not error.
4. **Acceptance:** each hook gated on `itemId`/`npcId` truthiness + tenant; tenant switch invalidates via the existing `queryClient.clear()` in `TenantProvider`.

Dependencies: 2.1.

#### 2.3 New components — **M**
1. **`ItemHeader.tsx`** — `src/components/features/items/ItemHeader.tsx`. Mirror `MonsterHeader` (PRD §4.2). Tooltip-to-copy id; 64×64 icon with `iconFailed` state; fallback renders no icon.
2. **`ItemNpcShopWidget.tsx`** — `src/components/features/items/ItemNpcShopWidget.tsx`. Link to `/npcs/{npcId}/shop`. Uses `useNpcData` + `useNpcSpawnMap`. Per PRD §4.4 layout (icon / name+price / map badge) with column collapse <640px. Tooltip exposes commodity id / discount rate / period / level limit.
3. **`ItemCashShopWidget.tsx`** — `src/components/features/items/ItemCashShopWidget.tsx`. Non-link, amber-tinted, `Gem` icon. ON SALE badge when `onSale`; gender flag when `gender !== 2`. Tooltip for `sn` + `priority`.
4. **`EquipmentRequirementsCard.tsx`** — `src/components/features/items/EquipmentRequirementsCard.tsx`. Renders only when any `req*` is non-zero; `Job` expanded via `formatReqJob` to `Badge variant="outline"` list.
5. **`formatReqJob.ts`** — `src/components/features/items/formatReqJob.ts` (or `src/lib/utils/`). Expand bitmask `1|2|4|8|16 → Warrior | Magician | Bowman | Thief | Pirate`. Return `string[]`.
6. **`DroppedByWidget.tsx`** — `src/components/features/items/DroppedByWidget.tsx`. Replaces `DroppedByTableRow`. Per PRD §4.5.

Dependencies: 2.1, 2.2.

#### 2.4 `ItemDetailPage` rewrite — **L**
1. Replace header + General card with `<ItemHeader />`.
2. Equipment branch: render merged "Stats" card (list per PRD §4.3 — omit `Price`) plus conditional `<EquipmentRequirementsCard />`.
3. Non-equipment branches: fold current multi-card clusters into one "Properties" card per type; strip `Price` from each (it moves to Sold By). Preserve existing Scroll Effects / Spec / Time Windows cards as-is for the types that have them.
4. Insert "Sold By" card below type-specific cards: subsection headings + empty state per PRD §4.4. Count in title reflects both subsections.
5. Replace Dropped By table with widget grid (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2`), sorted `chance DESC, monsterId ASC`, using `<DroppedByWidget />`.
6. Loading orchestration: single page spinner gated on `nameQuery + detailQuery` only; sellers/drops/commodities loading states per-card.
7. Delete `src/components/features/drops/DroppedByTableRow.tsx`; verify zero remaining references via grep.
8. Acceptance: manual walk-through of the four flows in PRD §10 cross-cutting (equipment w/ reqs, consumable w/ NPC seller, cash item, high-drop-count item).

Dependencies: 2.1, 2.2, 2.3.

#### 2.5 Build + lint + test — **S**
1. `npm run build` clean.
2. `npm run test` clean.
3. `npm run lint` — no new errors.
4. Manual smoke against a running backend stack.

### Phase 3 — Integration + acceptance

1. **Docker compose up; re-ingest a tenant.** Required because `npc_spawn_index` is forward-only (PRD §6.4).
2. **Verify the four acceptance scenarios** in PRD §10 cross-cutting.
3. **Regression spot-check** on `MapDetailPage`, `MonsterDetailPage`, `NpcShopPage`.
4. **Confirm tenant-switch invalidation** on the three new hooks (flip tenant in UI, ensure the widgets re-fetch).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `npc_spawn_index` population races with existing monster index population in the shared map-storage transaction | Low | High (ingest corruption) | Add inside the same DB transaction the monster index uses; unit test with duplicate NPC entries. |
| 404 semantics on `/data/npcs/{npcId}/map` confuses the UI into a broken state | Med | Low | `useNpcSpawnMap` normalizes 404 → `null`; widget degrades to two-column layout without map badge (PRD §4.4 covers this). |
| Per-NPC `useNpcSpawnMap` fan-out saturates network for items with many sellers | Low | Med | React Query cache de-dupes; PRD §4.6 sets ≥50-sellers as follow-up batching trigger. Add a TODO, don't pre-optimize. |
| `reqJob` bitmask semantics vary by region (some WZ ship `reqPOP` vs `reqFame`) | Med | Low | Expose both fields; UI suppresses zeros. Documented in PRD §9 Q5. |
| Existing `DroppedByTableRow.tsx` has external callers we missed | Low | Med | Grep during 2.4.7; if any callers exist, redirect or preserve the file. Survey already showed none. |
| Missing tenant scoping on new endpoints leaks cross-tenant data | Low | High | Reuse `rest.HandlerDependency.Context()` / `tenant.FromContext`; assert tenant filter in every unit test. |
| Cash-shop registry in-memory filter becomes a hotspot | Low | Low | Registry is small (~2k rows). PRD §8 allows adding a secondary item-keyed map later. |
| Equipment REST field additions break an existing consumer | Low | Low | Additive JSON fields — Go unmarshal ignores unknown, TS types are extended not replaced. |

## Success Metrics

- **Zero "General" cards on the page:** `grep -L "CardTitle.*General" src/pages/ItemDetailPage.tsx` returns no match (PRD §10).
- **`DroppedByTableRow.tsx` deleted with no remaining references.**
- **Cross-cutting acceptance scenarios (PRD §10) all pass** against a fresh re-ingested tenant.
- **No regressions on `MapDetailPage` / `MonsterDetailPage` / `NpcShopPage`** per spot-check.
- **Docker builds green** for atlas-data, atlas-npc-shops, atlas-ui.
- **All new backend routes return correct status codes** across 200/empty, 400 bad-id, 404 (npc map only), 500 error paths.
- **p99 latency <10ms** for `/commodities/items/{id}` and `/data/npcs/{id}/map` under dev load.

## Required Resources & Dependencies

- **Services in scope:** atlas-data, atlas-npc-shops, atlas-ui.
- **External data:** requires a re-ingested tenant to populate `npc_spawn_index`; no back-fill.
- **Prior work relied on:**
  - task-008 (`MapDetailPage` tooltip-to-copy pattern).
  - task-010 (`MonsterDetailPage` pattern + `monster_spawn_index` storage hook in `map/storage.go`).
- **No new infra.** No Kafka topics, no new config, no new auth surface.

## Timeline Estimates

Sizing: S ≈ 2–4 hr, M ≈ ½–1 day, L ≈ 1–2 days.

| Phase | Effort |
|---|---|
| 1.1 Equipment reqs | S |
| 1.2 NPC spawn index | M |
| 1.3 Commodity reverse (atlas-data) | S |
| 1.4 Commodity reverse (atlas-npc-shops) | M |
| 2.1 Types + service modules | S |
| 2.2 Hooks | S |
| 2.3 Components | M |
| 2.4 Page rewrite | L |
| 2.5 Build + lint + test | S |
| 3 Integration | S |

Rough aggregate: **~5–7 days of focused work** if single-threaded; ~3–4 days if Phase 1 streams are parallelized across contributors.
