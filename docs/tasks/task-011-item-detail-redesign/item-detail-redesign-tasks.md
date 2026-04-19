# Item Detail Redesign — Tasks

Last Updated: 2026-04-19

Checklist drives day-to-day progress. Keep in sync with `item-detail-redesign-plan.md`. Mark items `[x]` as completed.

---

## Phase 1 — Backend data surfaces

### 1.1 Equipment `req*` fields (atlas-data) — S
- [x] 1.1.1 Add `ReqLevel`, `ReqJob`, `ReqStr`, `ReqDex`, `ReqInt`, `ReqLuk`, `ReqPop`, `ReqFame` (uint16) to `equipment.RestModel` in `services/atlas-data/atlas.com/data/equipment/rest.go` (JSON tags: `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`, `reqPop`, `reqFame`).
- [x] 1.1.2 In `equipment/reader.go`, read `reqLevel`, `reqJob`, `reqSTR`, `reqDEX`, `reqINT`, `reqLUK`, `reqPOP`, `reqFame` via `info.GetShort(...)` and set them on the model.
- [x] 1.1.3 Extend `equipment/reader_test.go` to assert the eight new fields read from the existing test XML.
- [x] 1.1.4 `go test ./services/atlas-data/atlas.com/data/equipment/...` passes.
- [x] 1.1.5 `docker build services/atlas-data` succeeds.

### 1.2 NPC spawn index (atlas-data) — M
- [x] 1.2.1 Create `services/atlas-data/atlas.com/data/npc/spawn_index.go` with `SpawnIndexEntity` (composite PK `tenant_id, npc_id, map_id`; fields per PRD §6.2) and `TableName() = "npc_spawn_index"`.
- [x] 1.2.2 Register entity in the atlas-data migration sequence; add `idx_npc_spawn_index_lookup (tenant_id, npc_id, spawn_count DESC)`.
- [x] 1.2.3 Extend `map/storage.go` `Add()` (alongside monster_spawn_index block): delete rows by `(tenant, map)`, aggregate `m.NPCs` by `npc.Id`, bulk insert. Inside the existing upsert transaction.
- [x] 1.2.4 Log `"npc_spawn_index: tenant=%s map=%d rows=%d"` at Debug in `storage.go`.
- [x] 1.2.5 Add test in `map/storage_test.go` with a seed map containing duplicate NPC bindings; assert row count + `spawn_count`.
- [x] 1.2.6 Add `GET /{npcId}/map` under `/data/npcs` subrouter in `npc/resource.go`; handler returns top row by `spawn_count DESC, map_id ASC`. 200 / 404 / 400 / 500 per PRD §5.2.
- [x] 1.2.7 Define `NpcMapRestModel` (per PRD §5.2) colocated under `npc/`.
- [x] 1.2.8 Add `npc/resource_test.go` cases: primary-row 200, no-index 404, bad-id 400, tenant-scoping.
- [x] 1.2.9 `docker build services/atlas-data` succeeds.

### 1.3 Commodity reverse lookup (atlas-data) — S
- [x] 1.3.1 Add `GET /by-item/{itemId}` to `commodity/resource.go` sibling to existing routes.
- [x] 1.3.2 Handler iterates tenant-scoped commodity registry, filters by `ItemId == itemId`, returns `[]RestModel`.
- [x] 1.3.3 Test in `commodity/resource_test.go`: 200 with rows, 200 empty array, 400 bad id.
- [x] 1.3.4 Docker build confirms (covered by 1.2.9 if done together).

### 1.4 Commodity reverse lookup (atlas-npc-shops) — M
- [x] 1.4.1 Define `CommodityByItemRestModel` per PRD §5.1 (colocate under `commodities/`).
- [x] 1.4.2 Add `GET /commodities/items/{itemId}` route in `shops/resource.go` (or new `commodities/resource.go`).
- [x] 1.4.3 Handler parses `itemId`, queries `commodities.Entity` by `tenant_id + template_id`, returns array. 200 / 400 / 500 per PRD §5.1 (no 404).
- [x] 1.4.4 Extend `commodities.Migration` to `CREATE INDEX IF NOT EXISTS idx_commodities_by_template ON commodities (tenant_id, template_id)`.
- [x] 1.4.5 Test (`shops/resource_test.go` or new `commodities/resource_test.go`): 200 non-empty, 200 empty, 400 bad id, tenant-scoping.
- [x] 1.4.6 `docker build services/atlas-npc-shops` succeeds.

---

## Phase 2 — Frontend page refactor (atlas-ui)

### 2.1 Types + service modules — S
- [x] 2.1 Extend `EquipmentAttributes` in `src/types/models/item.ts` with `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`, `reqPop`, `reqFame` (all `number`).
- [x] 2.2 Add `NpcSpawnMap` shape to `src/types/models/npc.ts` (or inline in hook).
- [x] 2.3 New `src/services/api/npc-shop-commodities.service.ts` exporting `getByItem(itemId)` against atlas-npc-shops `/commodities/items/{itemId}`.
- [x] 2.4 Extend `src/services/api/commodities.service.ts` (or equivalent) with `getByItem(itemId)` against atlas-data `/api/data/commodity/by-item/{itemId}`.
- [x] 2.5 Extend NPC service with `getSpawnMap(npcId)` — distinguish 404 from error, resolve 404 to `null`.
- [x] 2.6 `tsc` + `npm run build` clean.

### 2.2 Hooks — S
- [x] 2.1 `src/lib/hooks/api/useItemSellers.ts` with key `["items", itemId, "sellers", tenantId]`, 10-minute stale time.
- [x] 2.2 `src/lib/hooks/api/useItemCommodities.ts` with key `["items", itemId, "commodities", tenantId]`.
- [x] 2.3 `src/lib/hooks/api/useNpcSpawnMap.ts` with key `["npcs", "spawn-map", npcId, tenantId]`, normalizes 404 → `{ data: null }`.
- [x] 2.4 All three hooks gated on id + tenant truthiness; verify tenant-switch invalidation via existing `queryClient.clear()`.

### 2.3 New components — M
- [x] 2.1 `components/features/items/ItemHeader.tsx` — tooltip-to-copy id, 64×64 icon + `iconFailed` state, badge via `getItemTypeBadgeVariant`.
- [x] 2.2 `components/features/items/formatReqJob.ts` — expand bitmask `1|2|4|8|16` → `[Warrior, Magician, Bowman, Thief, Pirate]`. Return `string[]`; `0` → `[]`.
- [x] 2.3 `components/features/items/EquipmentRequirementsCard.tsx` — renders only when any `req*` non-zero; Job row renders bitmask as `Badge variant="outline"` list; Level row renders when card renders; stat / pop / fame rows suppressed at 0.
- [x] 2.4 `components/features/items/ItemNpcShopWidget.tsx` — Link to `/npcs/{npcId}/shop`; icon + name/price + map badge; uses `useNpcData` + `useNpcSpawnMap`; tooltip surfaces commodity id, discount rate, period, level limit.
- [x] 2.5 `components/features/items/ItemCashShopWidget.tsx` — amber tint, `Gem` icon, NX price + period + ON SALE badge when `onSale`; tooltip for `sn` + `priority`; non-link.
- [x] 2.6 `components/features/items/DroppedByWidget.tsx` — Link to `/monsters/{id}`; icon + name + id; tooltip shows Chance / Min / Max / QuestID (QuestID row suppressed when `questId === 0`).

### 2.4 `ItemDetailPage` rewrite — L
- [x] 2.1 Replace header + "General" card with `<ItemHeader />`.
- [x] 2.2 Equipment branch: render merged "Stats" card (PRD §4.3 field order, no `Price`) + conditional `<EquipmentRequirementsCard />`.
- [x] 2.3 Consumable branch: single "Properties" card (no `Price`); keep Scroll Effects and Spec cards as-is.
- [x] 2.4 Setup branch: single "Properties" card (no `Price`).
- [x] 2.5 Etc branch: single "Properties" card (no `Price`).
- [x] 2.6 Cash branch: single "Properties" card (Slot Max only); keep Spec + Time Windows cards as-is.
- [x] 2.7 Insert "Sold By" card below type-specific cards; subsection headings per PRD §4.4; empty-both state renders single copy line.
- [x] 2.8 Replace Dropped By table with `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `<DroppedByWidget />`, sorted `chance DESC, monsterId ASC`.
- [x] 2.9 Loading orchestration: page spinner gated on `nameQuery + detailQuery` only; per-card treatment for sellers / commodities / drops.
- [x] 2.10 Delete `src/components/features/drops/DroppedByTableRow.tsx`; grep the repo to confirm zero references remain.
- [x] 2.11 Preserve route `/items/:id` and existing breadcrumbs.

### 2.5 Build + lint + test — S
- [x] 2.1 `npm run build` clean.
- [x] 2.2 `npm run test` clean.
- [x] 2.3 `npm run lint` — no new errors.

---

## Phase 3 — Integration + acceptance

- [ ] 3.1 Docker compose up all affected services.
- [ ] 3.2 Re-ingest at least one tenant's data (required for `npc_spawn_index`).
- [ ] 3.3 Load `/items/{id}` for an **equipment item with non-zero reqs** — Requirements card renders with correct job badges and stat rows.
- [ ] 3.4 Load `/items/{id}` for a **consumable sold by an NPC** — NPC Shops subsection renders with price + map badge.
- [ ] 3.5 Load `/items/{id}` for a **cash-shop SKU** — Cash Shop subsection renders with NX price, period, ON SALE badge when applicable.
- [ ] 3.6 Load `/items/{id}` for an **item with many monster drops** — widget grid renders, sort is `chance DESC`.
- [ ] 3.7 Load `/items/{id}` for an **item with zero sellers and zero drops** — empty states render per copy.
- [ ] 3.8 Flip active tenant — confirm `useItemSellers`, `useItemCommodities`, `useNpcSpawnMap` caches invalidate via `TenantProvider.queryClient.clear()`.
- [ ] 3.9 Spot-check `MapDetailPage` — no regression.
- [ ] 3.10 Spot-check `MonsterDetailPage` — no regression.
- [ ] 3.11 Spot-check `NpcShopPage` — no regression.

---

## Final acceptance (mirrors PRD §10)

Backend:
- [x] `equipment.RestModel` exposes all eight `req*` fields and reader populates from WZ.
- [x] `npc_spawn_index` migration + index run on startup.
- [x] Map re-ingest populates `npc_spawn_index` with accurate `spawn_count`.
- [x] `GET /data/npcs/{npcId}/map` — 200 primary row / 404 missing / 400 bad id, tenant-scoped.
- [x] `GET /data/commodity/by-item/{itemId}` — 200 matches / 200 empty / 400 bad id.
- [x] `GET /commodities/items/{itemId}` (atlas-npc-shops) — 200 / 400 / 500 per spec, tenant-scoped.
- [x] `idx_commodities_by_template` present after migration.
- [ ] atlas-data and atlas-npc-shops Docker builds succeed. (pending CI run)

Frontend:
- [x] No "General" card remains (`grep` returns no match).
- [x] Equipment renders single "Stats" card + conditional "Requirements" card.
- [x] Consumable / Setup / Etc / Cash property cards omit `Price`.
- [x] "Sold By" card renders with correct subsections and empty state.
- [x] "Dropped By" widget grid renders, sorted chance desc, each widget links + has tooltip.
- [x] `DroppedByTableRow.tsx` deleted, zero remaining references.
- [x] `npm run build` clean; lint has no new errors from task-011 files (pre-existing repo-wide errors unchanged). `npm run test` — 1 pre-existing unrelated failure in `tenant-context.test.tsx`.
- [ ] Tenant switch invalidates the three new caches. (hook keys include `activeTenant?.id` so `queryClient.clear()` at switch will invalidate; verify via live integration.)
