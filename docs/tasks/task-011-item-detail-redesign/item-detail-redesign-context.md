# Item Detail Redesign — Context

Last Updated: 2026-04-19

Anchor notes for this refactor: which files matter, what decisions are locked in, what this depends on. Read alongside `item-detail-redesign-plan.md` and the upstream PRD at `docs/tasks/task-011-item-detail-redesign/prd.md`.

---

## Key files — frontend (atlas-ui)

| File | Purpose |
|---|---|
| `services/atlas-ui/src/pages/ItemDetailPage.tsx` | Full rewrite. 373 lines today. Preserve the route `/items/:id` and breadcrumbs. |
| `services/atlas-ui/src/components/features/items/` | **Does not exist yet** — create the directory. Holds new components. |
| `services/atlas-ui/src/components/features/items/ItemHeader.tsx` | New. Mirrors `MonsterHeader.tsx`. |
| `services/atlas-ui/src/components/features/items/ItemNpcShopWidget.tsx` | New. Sold By → NPC Shops widget. |
| `services/atlas-ui/src/components/features/items/ItemCashShopWidget.tsx` | New. Sold By → Cash Shop widget. |
| `services/atlas-ui/src/components/features/items/EquipmentRequirementsCard.tsx` | New. Conditional on any `req*` > 0. |
| `services/atlas-ui/src/components/features/items/DroppedByWidget.tsx` | New. Replaces the flat drop table. |
| `services/atlas-ui/src/components/features/items/formatReqJob.ts` | New helper. Expands `reqJob` bitmask into class strings. |
| `services/atlas-ui/src/components/features/drops/DroppedByTableRow.tsx` | **Delete** after rewrite; no other callers. |
| `services/atlas-ui/src/components/features/monsters/MonsterHeader.tsx` | Pattern reference for `ItemHeader`. |
| `services/atlas-ui/src/lib/hooks/api/useItemSellers.ts` | New hook. |
| `services/atlas-ui/src/lib/hooks/api/useItemCommodities.ts` | New hook. |
| `services/atlas-ui/src/lib/hooks/api/useNpcSpawnMap.ts` | New hook. 404 → `null`. |
| `services/atlas-ui/src/lib/hooks/api/useDrops.ts` | Existing — reused for `useItemDrops`. |
| `services/atlas-ui/src/lib/hooks/api/useNpcs.ts` | Existing — reused for NPC name/icon. |
| `services/atlas-ui/src/lib/hooks/api/useMobData.ts` | Existing — reused per drop widget. |
| `services/atlas-ui/src/services/api/npc-shop-commodities.service.ts` | New service module. |
| `services/atlas-ui/src/services/api/commodities.service.ts` (or equivalent) | Extend with `getByItem`. |
| `services/atlas-ui/src/types/models/item.ts` | Extend `EquipmentAttributes` with eight `req*` fields. |
| `services/atlas-ui/src/types/models/npc.ts` | Add `NpcSpawnMap` shape (or inline). |

## Key files — backend (atlas-data)

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/equipment/rest.go` | Add eight `req*` fields to `RestModel`. |
| `services/atlas-data/atlas.com/data/equipment/reader.go` | Plumb eight `info.GetShort(...)` calls. |
| `services/atlas-data/atlas.com/data/equipment/reader_test.go` | Assert new fields on existing fixture. |
| `services/atlas-data/atlas.com/data/npc/spawn_index.go` | **New**. `SpawnIndexEntity` + migration. |
| `services/atlas-data/atlas.com/data/npc/resource.go` | Add `GET /{npcId}/map`. |
| `services/atlas-data/atlas.com/data/npc/resource_test.go` | Add 200/404/400/tenant tests. |
| `services/atlas-data/atlas.com/data/map/storage.go` | Extend `Add()` to populate `npc_spawn_index`. |
| `services/atlas-data/atlas.com/data/map/storage_test.go` | Add NPC duplicate-spawn test. |
| `services/atlas-data/atlas.com/data/commodity/resource.go` | Add `GET /by-item/{itemId}`. |
| `services/atlas-data/atlas.com/data/commodity/resource_test.go` | Add test. |

## Key files — backend (atlas-npc-shops)

| File | Change |
|---|---|
| `services/atlas-npc-shops/atlas.com/npc/shops/resource.go` (or new sibling `commodities/resource.go`) | Add `GET /commodities/items/{itemId}` route. |
| `services/atlas-npc-shops/atlas.com/npc/commodities/entity.go` | Extend `Migration` to add `idx_commodities_by_template`. |
| `services/atlas-npc-shops/atlas.com/npc/commodities/rest_by_item.go` | **New**. `CommodityByItemRestModel`. |
| `services/atlas-npc-shops/atlas.com/npc/shops/resource_test.go` (or new `commodities/resource_test.go`) | Add test. |

## Locked-in decisions

Taken from PRD §9 and embedded through the spec — do not revisit unless product explicitly revises:

1. **Header fallback:** icon error → render nothing. No placeholder icon.
2. **"General" card:** removed. Template id, name, type live only on the header + badge + copyable tooltip.
3. **Equipment cards:** single merged "Stats" card + conditional "Requirements" card. Not three separate cards.
4. **`Price` placement:** lives in "Sold By" card, not in type-specific property cards.
5. **Sold By subsections:** NPC Shops + Cash Shop, rendered independently when non-empty. Empty-both state shows a single copy line — no subsection headings.
6. **`reqJob` rendering:** expand bitmask to class-name `Badge` list via `formatReqJob`. Bitmask `0` = no row.
7. **NPC multi-map ambiguity:** `GET /data/npcs/{npcId}/map` returns a single primary row (highest `spawn_count`, lowest `map_id` tie-breaker). A list variant is *not* in scope.
8. **Dropped By:** widget grid replacing the flat table; single canonical sort `chance DESC, monsterId ASC`. No user-selectable sort.
9. **Back-fill:** none. `npc_spawn_index` populated forward-only via re-ingest.
10. **Batching:** per-NPC fan-out for `useNpcSpawnMap` is acceptable; batched variant is a follow-up trigger, not in scope.
11. **404 semantics for `/npcs/{npcId}/map`:** the UI normalizes to `null` and degrades gracefully. Not an error.

## Dependencies

**Upstream (must exist):**
- task-008 pattern: `MapDetailPage` tooltip-to-copy + card density.
- task-010 pattern: `MonsterDetailPage` + `monster_spawn_index` in `map/storage.go`.
- Existing hooks: `useItemDrops`, `useItemData`, `useNpcData`, `useMobData`.
- Existing tenant context: `tenant.FromContext` (Go) + `TenantProvider` + `queryClient.clear()` on tenant switch (TS).

**Downstream (none required):**
- No consumers of `equipment.RestModel` need updating — additive JSON is backwards-compatible.

**External:**
- Dev Postgres for atlas-data + atlas-npc-shops migrations.
- Re-ingested tenant for `npc_spawn_index` population validation.

## Not in scope (explicit exclusions from PRD §2 non-goals)

- Editing item / drop / shop / commodity data from this page.
- Progress tracking for equipment slots.
- Per-map NPC positions or rendering NPCs on maps.
- Filtering/sorting Dropped By by columns other than the canonical sort.
- `ItemsPage` list changes, `NpcShopPage` changes, `MonsterDetailPage` drops section changes.
- Back-filling `npc_spawn_index` from existing map documents.
- Cash-shop purchase flow integration.
- Live inventory/equipped-by state.

## Testing notes

- **Cross-cutting scenarios** (PRD §10) require a running compose stack + at least one re-ingested tenant. Plan a single manual pass through all four.
- **Tenant-switch invalidation** — confirm via React DevTools or cache inspection after flipping tenant that the three new query keys drop/refetch.
- **Zero-state renderings** — deliberately pick items with no drops, no sellers, and no commodities to confirm every "empty" path in PRD §4.4 / §4.5 / §4.3.
- **Equipment with non-zero `reqJob` only** (stat reqs all zero) — Requirements card renders with only the Job row. Pick a starter job-gated weapon.
