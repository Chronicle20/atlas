# Item Craft Recipes — Product Requirements Document

> **Surface rename (2026-04-25):** the original draft used `craftRecipes`/`craft-recipes`/`craft_recipes` everywhere. The implemented surface is `recipes`/`/items/{itemId}/recipes`/`/npcs/{npcId}/recipes`/`/npcs/conversations/reindex-recipes`/`recipes` table. User-facing UI titles ("Craftable At", "Crafts") are unchanged.

Version: v1
Status: Draft
Created: 2026-04-25
---

## 1. Overview

The atlas-ui item detail page currently surfaces who sells an item and which monsters drop it, but it gives no signal that an item can be obtained by crafting at an NPC. The data already exists: `atlas-npc-conversations` stores `craftAction` states inside per-NPC conversation JSON, and each `craftAction` records the output `itemId`, the materials and quantities the player must hand over, the meso cost, and (optionally) a stimulator that lets the player gamble on randomized stats. Today there is no way to query "which NPC(s) craft itemId X" — the only access pattern is "load the conversation for NPC Y."

This feature adds (a) a tenant-scoped reverse-lookup endpoint in atlas-npc-conversations that returns every craft recipe whose output is a given itemId, (b) a `Craftable At` card on the item detail page that lists each recipe with rich information (NPC, materials with names, quantities, meso cost, optional stimulator info), and (c) a symmetric `Crafts` card on the NPC detail page that lists every item that NPC can craft. The reverse index is materialized into a dedicated `recipes` table populated whenever conversations are seeded, created, or updated, so lookups are cheap regardless of how many craft NPCs exist.

## 2. Goals

Primary goals:
- Players researching an item can see at a glance whether it is obtainable through NPC crafting, and from whom.
- For each craft recipe, players can see the full cost: every material with its name and quantity, the meso cost, and stimulator info if applicable.
- NPC detail pages reciprocally surface the items that NPC crafts.
- Lookups are tenant-scoped and stay performant as the dataset grows.

Non-goals:
- Showing "this item is used as a material in the following recipes" (reverse-of-reverse). Deferred to a follow-up.
- Showing live, character-specific information ("you have 3/15 of this material").
- Editing recipes from the UI.
- Surfacing crafting paths that aren't `craftAction` (quest rewards, scrolls, monster-card synthesis, party-quest bonuses, etc.).
- Recipes for items obtained from systems outside atlas-npc-conversations (e.g., reactors, cash shop bundles).

## 3. User Stories

- As a player, when I open the detail page for an item I'm trying to acquire, I want to see which NPC (if any) crafts it so I know where to go.
- As a player, when I see a craft recipe, I want to see each material's name (not just its itemId) so I can recognize what I need to gather.
- As a player, when an NPC offers a stimulator variant of a recipe, I want to know the stimulator item, what stat outcome it implies, and the failure chance so I can decide whether to risk it.
- As a player, when I open an NPC's detail page, I want to see every item that NPC crafts so I can compare what's available without opening each item.
- As an admin, I want the craft index to stay in sync with conversation seeding/upsert so I never have to manually reindex.

## 4. Functional Requirements

### 4.1 Reverse craft index (atlas-npc-conversations)

- A `recipes` table is maintained as a derived index over the existing `npc_conversations` table.
- Each row corresponds to one `craftAction` state inside one NPC conversation, identified by `(tenant_id, conversation_id, state_id)`. Multiple recipes can share the same `(tenant_id, item_id)` (e.g., warriors-glove + stimulator variant + base variant).
- The index is rebuilt for the affected conversation whenever:
  - `Create` (POST `/npcs/conversations`) creates an NPC conversation,
  - `Update` (PATCH `/npcs/conversations/{conversationId}`) updates one,
  - `Delete` removes one (cascade-delete the rows),
  - `DeleteAllForTenant` runs (cascade-delete tenant's rows),
  - `Seed` (POST `/npcs/conversations/seed`) runs (clear-and-rebuild for the tenant).
- Index population must happen inside the same transaction as the parent conversation write so the index can never go stale relative to its parent. If the parent write succeeds and the index write fails, the whole operation must roll back.
- All queries, scans, and writes are tenant-scoped via `tenant.MustFromContext(ctx)`.

### 4.2 Reverse-lookup endpoint (atlas-npc-conversations)

- Add `GET /items/{itemId}/recipes` returning a JSON:API list of `recipe` resources for the active tenant.
- The endpoint MUST return a `200 OK` with an empty array when no recipes exist (do not 404).
- The endpoint MUST return `400 Bad Request` when `itemId` cannot be parsed as a positive integer, matching the convention in `atlas-npc-shops/atlas.com/npc/commodities/resource.go`.
- Each resource attribute set contains: `npcId`, `conversationId`, `stateId`, `itemId`, `materials[]` (each `{ itemId, quantity }`), `mesoCost`, `stimulatorId`, `stimulatorFailChance`. Non-stimulator recipes return `stimulatorId: 0` and `stimulatorFailChance: 0`.
- Recipe ordering: ascending by `npcId`, then by `stateId` (deterministic across requests).

### 4.3 Reverse-lookup endpoint, NPC side (atlas-npc-conversations)

- Add `GET /npcs/{npcId}/recipes` returning the same `recipe` resource shape, filtered to one NPC, tenant-scoped.
- Same error conventions as 4.2.
- Recipe ordering: ascending by `stateId`.

### 4.4 Item detail UI (`atlas-ui` `ItemDetailPage`)

- Add a new card titled `Craftable At (N)` after `Sold By` and before `Dropped By`. Card placement MUST match this order; no other reshuffling of the page.
- When `N === 0`, the card title reads `Craftable At` (no count) and the body reads `No NPCs craft this item.` (matches the `Sold By` / `Dropped By` empty-state phrasing).
- When `N > 0`, the body renders one `ItemCraftRecipeWidget` per recipe row.
- Each `ItemCraftRecipeWidget` shows:
  - The crafter NPC's icon, name, and a link to the NPC's detail page (mirror `ItemNpcShopWidget`).
  - A "With Stimulator" badge when `stimulatorId > 0`. The badge tooltip shows the stimulator item's name and the fail chance as a percentage (`Math.round(stimulatorFailChance * 100)%`, e.g., `10%`).
  - The meso cost rendered as `Cost: 18,000 mesos` (locale-formatted).
  - A materials block with one row per material: `{materialName} × {quantity}`. The material name MUST be fetched (see 4.5); render the raw itemId only as a fallback while the name is loading or if the lookup fails.
- Loading state: while the recipes query is in flight, the card body reads `Loading craft recipes...`.
- Error state: if the recipes query errors, render the error message inline (matches existing `Sold By` pattern).

### 4.5 Material name resolution (atlas-ui)

- Materials are item references; reuse `itemsService.getItemName` (already used for the page header) to fetch each material's name.
- Each unique material itemId triggers one React Query (`["items", "name", tenantId, materialItemId]`), enabling automatic dedup and cache reuse with the existing item-name cache the rest of the UI uses.
- If a material name fails to resolve, fall back to the literal `Item #{itemId}` and continue rendering the rest of the recipe.

### 4.6 NPC detail UI (`atlas-ui` `NpcDetailPage`)

- Add a `Crafts (N)` card at the end of the page (after the existing conversation/quests/shop cards).
- When `N === 0` the card is hidden (matches existing `NpcShopCard` behavior of conditionally rendering).
- When `N > 0`, the card body shows one row per craftable item. Each row links to the item detail page and shows the item icon (via existing `getAssetIconUrl`/asset pipeline), the item name (via `itemsService.getItemName`), the meso cost, and the same `With Stimulator` badge described in 4.4.
- The row does NOT expand the full materials list on the NPC page; clicking through to the item detail page is the canonical place to see materials. Rationale: keeps the NPC page scannable.

### 4.7 Tenant scoping & cache invalidation

- All endpoints are tenant-scoped on the backend.
- The atlas-ui `useItemCraftRecipes(itemId)` and `useNpcCraftRecipes(npcId)` hooks key on `activeTenant?.id ?? "no-tenant"` and are gated by `enabled: !!activeTenant`, matching the existing `useItemSellers` / `useItemDrops` hooks.
- No new manual cache invalidation is needed; `TenantProvider` already calls `queryClient.clear()` on tenant change.

## 5. API Surface

### 5.1 `GET /items/{itemId}/recipes` (atlas-npc-conversations)

Request headers: standard tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).

Path params:
- `itemId` — `uint32`, required. Returns `400` if not a positive integer.

Response: `200 OK`, JSON:API list. See `api-contracts.md` for the exact resource shape.

Errors:
- `400 Bad Request` — non-numeric `itemId`.
- `500 Internal Server Error` — DB failure.

### 5.2 `GET /npcs/{npcId}/recipes` (atlas-npc-conversations)

Path params:
- `npcId` — `uint32`, required. Returns `400` if not a positive integer.

Response: same resource shape, filtered to the one NPC.

Errors: same as 5.1.

### 5.3 No changes to other services

atlas-data, atlas-character, atlas-merchant, atlas-saga-orchestrator are untouched. No Kafka topics, no producers, no consumers.

## 6. Data Model

### 6.1 `recipes` (new, atlas-npc-conversations)

| Column                   | Type        | Notes                                                            |
|--------------------------|-------------|------------------------------------------------------------------|
| `id`                     | uuid        | Primary key.                                                     |
| `tenant_id`              | uuid        | NOT NULL. Scoped on every query.                                 |
| `conversation_id`        | uuid        | FK to `npc_conversations.id`. ON DELETE CASCADE.                 |
| `npc_id`                 | int (uint32)| Denormalized from parent for fast lookup.                        |
| `state_id`               | text        | The conversation state this recipe came from.                    |
| `item_id`                | int (uint32)| Output item.                                                     |
| `materials`              | jsonb       | `[{itemId: uint32, quantity: uint32}, ...]`. Aligns lengths.     |
| `meso_cost`              | int (uint32)|                                                                  |
| `stimulator_id`          | int (uint32)| `0` when not a stimulator recipe.                                |
| `stimulator_fail_chance` | float8      | `0` when not a stimulator recipe.                                |
| `created_at`             | timestamptz | Audit.                                                           |
| `updated_at`             | timestamptz | Audit.                                                           |

Indexes:
- `(tenant_id, item_id)` — supports `/items/{itemId}/recipes`.
- `(tenant_id, npc_id)` — supports `/npcs/{npcId}/recipes`.
- `(conversation_id)` — supports cascade delete and index rebuild on upsert.
- Uniqueness: `(tenant_id, conversation_id, state_id)` to make rebuilds idempotent.

Migration: pure additive. No backfill is required up front because the seeding handler (`POST /npcs/conversations/seed`) is the canonical way to populate state in any environment, and once the index-write hooks are in place, the next seed run produces a complete index. For environments that want the index without re-running a full seed, an admin-only `POST /npcs/conversations/reindex-recipes` endpoint is in scope (see 7.1) so they can rebuild without touching conversation data.

### 6.2 No changes to `npc_conversations`

The existing parent table is unchanged. Recipes are derived data and live in their own table.

## 7. Service Impact

### 7.1 atlas-npc-conversations

- Add a `craftrecipe` package alongside the existing `conversation/npc/` and `conversation/quest/` packages, containing entity, model, transform/extract, processor, administrator, resource (REST), and tests, mirroring the existing structure.
- Wire the index-write hooks into `conversation/npc/processor.go` `Create`/`Update`/`Delete`/`DeleteAllForTenant`/`Seed`. Each hook walks the conversation's `states` slice, filters to `craftAction`, and upserts/deletes the corresponding `recipes` rows in the same transaction.
- Add an admin reindex endpoint `POST /npcs/conversations/reindex-recipes` that walks every conversation for the active tenant and rebuilds the index without touching the conversations themselves. Used for rolling out the feature in environments that have already been seeded.
- Add the two GET endpoints under existing routing.
- Update the service's `docs/` to document the new resource and endpoints.

### 7.2 atlas-ui

- Add `services/api/recipes.service.ts` exposing `getByItem(itemId)` and `getByNpc(npcId)`.
- Add `lib/hooks/api/useItemRecipes.ts` and `lib/hooks/api/useNpcRecipes.ts`.
- Add `components/features/items/RecipeWidget.tsx` and `components/features/items/RecipesByItemCard.tsx`.
- Add `components/features/npc/RecipesByNpcCard.tsx` and any item-row sub-component.
- Update `pages/ItemDetailPage.tsx` to include the new card between `Sold By` and `Dropped By`.
- Update `pages/NpcDetailPage.tsx` to include the new card after existing content.
- Material/item names must come through the existing `itemsService.getItemName` cache.

### 7.3 No changes to other services

## 8. Non-Functional Requirements

- **Performance**: `GET /items/{itemId}/recipes` must be O(matching rows) — backed by the `(tenant_id, item_id)` index, not a JSON scan over conversations. Median response ≤ 50ms in the dev environment.
- **Consistency**: A successful `Create`/`Update`/`Delete`/`Seed` of a conversation MUST leave the index in a consistent state. The index-write hook lives in the same DB transaction as the parent write.
- **Tenant isolation**: All reads and writes must include a `tenant_id` predicate. Audited via `backend-guidelines-reviewer`.
- **Observability**: Each new endpoint must register through `rest.RegisterHandler` so its requests show up in the existing logging/tracing pipeline.
- **Tests**:
  - Backend: unit tests for transform/extract; integration tests for each REST endpoint covering happy path, empty result, bad input, and tenant isolation; reindex endpoint test that builds and rebuilds without duplicate rows.
  - Frontend: tests for `CraftableAtCard` and `NpcCraftsCard` covering loading, empty, error, and populated states. Stimulator badge rendering tested.
- **Docs**: `services/atlas-npc-conversations/docs/` updated with the new resource. Top-level `docs/TODO.md` updated to remove any "craft index" item if present.

## 9. Open Questions

None blocking. Items deferred for a follow-up:
- Reverse-of-reverse: "this item is used as a material in the following recipes."
- Surfacing non-`craftAction` ways an item is obtained (quest rewards, etc.).
- Showing per-character availability ("you have X of Y").

## 10. Acceptance Criteria

- [ ] `recipes` table exists with the schema in §6.1 and the listed indexes.
- [ ] `Create`/`Update`/`Delete`/`DeleteAllForTenant`/`Seed` on NPC conversations all keep `recipes` in sync atomically (proven by integration tests).
- [ ] `POST /npcs/conversations/reindex-recipes` rebuilds the index for the active tenant idempotently.
- [ ] `GET /items/{itemId}/recipes` returns recipes for the active tenant with the documented JSON:API shape, ordered by `(npcId, stateId)`, returns `[]` when none exist, and returns `400` for malformed `itemId`.
- [ ] `GET /npcs/{npcId}/recipes` does the same, filtered to one NPC and ordered by `stateId`.
- [ ] `ItemDetailPage` shows a `Craftable At` card between `Sold By` and `Dropped By`. Empty state reads `No NPCs craft this item.` Populated state shows one widget per recipe.
- [ ] Each `ItemCraftRecipeWidget` shows NPC icon + name (linked), meso cost, materials with resolved names and quantities, and a `With Stimulator` badge with tooltip (stimulator item name + fail chance %) when applicable.
- [ ] `NpcDetailPage` shows a `Crafts` card listing every item that NPC crafts, with item icon, name (linked to item detail page), meso cost, and the `With Stimulator` badge. Card is hidden when the NPC crafts nothing.
- [ ] All new backend code passes the DOM-* checklist via `backend-guidelines-reviewer`.
- [ ] All new frontend code passes the FE-* checklist via `frontend-guidelines-reviewer`.
- [ ] Verified manually against the seed dataset: NPC `2040020` produces multiple recipes for warrior-glove items including stimulator variants, and each item's detail page surfaces him as the crafter.
