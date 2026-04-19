# NPC Detail Redesign — Strategic Plan

**Last Updated: 2026-04-19**
**Status: NOT STARTED**

## 1. Executive Summary

Rewrite `NpcDetailPage` in atlas-ui from a thin icon/name/two-button card into the information-dense layout established by task-008 (map), task-010 (monster), and task-011 (item). Adds a "Spawn Locations" widget grid sourced from `npc_spawn_index`, a "Quests" widget grid sourced from a new reverse-lookup endpoint over registered quest documents, and read-only summary cards with deep-link Edit buttons for Shop and Conversation. Retires the singular `/data/npcs/:npcId/map` endpoint and its lone consumer in favour of a plural variant that exposes all maps where an NPC stands.

Authoritative spec: `docs/tasks/task-012-npc-detail-redesign/prd.md`. This plan defers to the PRD on any disagreement.

## 2. Current State Analysis

### Frontend — `services/atlas-ui/src/pages/NpcDetailPage.tsx:15-189`

- `<h2>` page heading followed by a `max-w-md` card containing a 96×96 icon, the name, a literal `ID: {npcId}` line, and two buttons (`View Shop`, `View Conversation`) that grey out when the corresponding resource does not exist.
- A `RefreshCw` icon button in the heading that manually re-fetches the NPC.
- No spawn-location surfacing — an operator cannot answer "where does this NPC stand?" without opening `/maps` and scanning.
- No quest surfacing — the page never references the quests the NPC participates in.
- Deep links to `/npcs/:id/shop` and `/npcs/:id/conversations` exist but are "view-only" buttons, not "edit" deep-links.

### Frontend — `services/atlas-ui/src/components/features/items/ItemNpcShopWidget.tsx:22`

- Uses `useNpcSpawnMap(npcId)` (singular) which calls `/data/npcs/:npcId/map`. The endpoint returns the top-1 row of `npc_spawn_index`, hiding the multi-map case (gachapon, town service NPCs).

### Backend — `services/atlas-data/atlas.com/data/npc/resource.go:28`

- `GET /data/npcs/:npcId/map` → `handleGetNpcMapRequest` (`resource.go:152-191`) returns the top row from `npc_spawn_index` ordered by `spawn_count DESC`.
- No quests-by-npc endpoint.
- `npc_spawn_index` table already populated by the map-ingest write path with the `(tenant_id, npc_id, map_id)` primary key — supports the N-row case.

### Backend — `services/atlas-data/atlas.com/data/quest/`

- Quest definitions are stored per tenant in the MongoDB-backed `document.Storage` under the `QUEST` collection (`storage.go:11-14`).
- `handleGetQuests` (`resource.go:28-42`) enumerates the full quest list. Same access pattern is reused by `atlas-quest`'s `GetAutoStartQuests` (`services/atlas-quest/atlas.com/quest/data/quest/processor.go:40-62`) — in-memory filter over the registered quest set.

## 3. Proposed Future State

Page layout, top-to-bottom, inside `flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto`:

1. **Header row** — 64×64 icon, name, tooltip-to-copy template id. Replaces the heading, the old card, and the refresh button.
2. **Spawn Locations card** — widget grid over `npc_spawn_index` rows sorted by `spawn_count DESC, map_id ASC`. Empty state reads "This NPC is not placed on any loaded map."
3. **Two-column grid: Shop + Conversation** — read-only summary cards, each with an `Edit` (or `Create` when absent) deep-link button.
4. **Quests card** — widget grid over the new reverse-lookup endpoint. Sorted Initiator → Both → Completer, ties broken by name ASC, id ASC.

Backend additions in `atlas-data`:

- `GET /data/npcs/:npcId/maps` — plural variant, returns all `npc_spawn_index` rows for `(tenant, npc)`. Reuses `NpcMapRestModel`.
- `GET /data/npcs/:npcId/quests` — in-memory scan over registered quest documents, filtering on `startRequirements.npcId`, `endRequirements.npcId`, `startActions.npcId`, `endActions.npcId`. Returns `[]quest.RestModel` (byte-identical to `GET /data/quests/:id`).

Backend retirement:

- `GET /data/npcs/:npcId/map` (singular) and `handleGetNpcMapRequest` are deleted. `ItemNpcShopWidget` migrates to the plural hook in this same PR.

No schema changes. No new Kafka topics. No new storage types.

## 4. Implementation Phases

### Phase 1 — Backend: Plural Spawn-Maps Endpoint (atlas-data) — **S**

Add `handleGetNpcMapsRequest` at `GET /data/npcs/:npcId/maps`, reusing `NpcMapRestModel`. Query `npc_spawn_index` for all `(tenant, npc)` rows ordered by `spawn_count DESC, map_id ASC`. 200-empty when no rows; 400 on bad id; 500 on DB failure. Register the route in `npc.InitResource`. Cover happy-path, empty, bad-id, and DB-failure in `resource_test.go`.

### Phase 2 — Backend: Quests-by-NPC Endpoint (atlas-data) — **M**

Add `handleGetNpcQuestsRequest` at `GET /data/npcs/:npcId/quests`. Load all tenant quest documents via `quest.NewStorage(...).GetAll(ctx)` and filter in memory across the four npcId fields. Marshal via the existing `quest.RestModel` (byte-identical to `/data/quests/:id`). Server-side sort `id ASC`. Emit a `Debugf` with `tenant_id`, `npc_id`, `result_ct`, `elapsed_ms`. Cover happy-path, empty, bad-id, and missing-tenant in tests.

### Phase 3 — Backend: Retire Singular `/map` Endpoint (atlas-data) — **S**

Remove the `/{npcId}/map` route registration and delete `handleGetNpcMapRequest`. Keep `NpcMapRestModel` — the plural handler reuses it. Remove singular-endpoint cases from `resource_test.go` and replace with plural-equivalent cases.

### Phase 4 — Frontend: Types, Services, Hooks (atlas-ui) — **M**

1. Add `NpcSpawnMapAttributes` (plural) and `NpcQuestRole` (enum: `initiator` | `completer` | `both`) to `src/types/models/npc.ts`.
2. Add `getNpcSpawnMaps(npcId)` and `getNpcQuests(npcId)` to `src/services/api/npcs.service.ts`. Remove `getSpawnMap(npcId)` — endpoint is retired.
3. Add `useNpcSpawnMaps(npcId)` hook — key `["data", "npcs", "maps", tenantId, npcId]`.
4. Add `useNpcQuests(npcId)` hook — key `["data", "npcs", "quests", tenantId, npcId]`. Derive `role` per quest on the UI side from the four npcId fields.
5. Add `useNpcConversation(npcId)` hook wrapping existing `conversationsService.getByNpcId`.
6. Delete `src/lib/hooks/api/useNpcSpawnMap.ts` and remove any barrel exports of it.

### Phase 5 — Frontend: Components (atlas-ui) — **M**

Under `src/components/features/npc/`:

1. `NpcHeader.tsx` — copy of `MonsterHeader.tsx:9-57`. 64×64 icon, name, `TooltipContent copyable` with the template id. No badges.
2. `NpcSpawnMapWidget.tsx` — mirror of `MonsterSpawnMapWidget.tsx:1-30`, linking to `/maps/:mapId`. Keyed by `${npcId}-${mapId}`.
3. `NpcQuestWidget.tsx` — MVP per PRD §4.6.1. `Scroll` icon + quest name + `parent` badge + role badge. Links to `/quests/:id`. Role badge variants: `initiator -> "default"`, `completer -> "outline"`, `both -> "secondary"`. Role label: `Initiator` / `Completer` / `Initiator & Completer`.

### Phase 6 — Frontend: Page Rewrite (atlas-ui) — **L**

Rewrite `src/pages/NpcDetailPage.tsx` per PRD §4.1–§4.8:

1. Replace outer layout with the standard scrolling container.
2. Render `NpcHeader` at the top — removes the old heading, the `max-w-md` card, the `ID: {npcId}` line, and the refresh button.
3. Render the Spawn Locations card (§4.3). Loading / error / empty / populated states as specified.
4. Render the two-column Shop + Conversation grid (§4.4, §4.5). Has-data / no-data / loading / error states per card. `Edit Shop` / `Create Shop` → `/npcs/:id/shop`; `Edit Conversation` / `Create Conversation` → `/npcs/:id/conversations`.
5. Render the Quests card (§4.6). UI-side sort: role priority (Initiator → Both → Completer), then name ASC, then id ASC.
6. All four data queries fire in parallel — no waterfalls.

### Phase 7 — Frontend: `ItemNpcShopWidget` Migration (atlas-ui) — **S**

Migrate `src/components/features/items/ItemNpcShopWidget.tsx:22` from `useNpcSpawnMap` (singular) to `useNpcSpawnMaps` (plural):

- Take the top row from the response (server sorts `spawn_count DESC, map_id ASC`).
- Render the existing `{name} · {streetName}` badge from that top row.
- If `response.length > 1`, append an outline-variant `+{N-1}` badge alongside the primary.
- `response.length === 0` → render no badge (same behaviour as the old 404 → null path).

### Phase 8 — Tests, Builds, Integration (cross-cutting) — **M**

1. Unit tests for backend happy-path + edge cases (Phases 1–3).
2. Unit tests for UI role derivation (initiator-only, completer-only, both), empty states, header tooltip copy, and per-card load-state rendering.
3. `npm run lint`, `npm run test`, `npm run build` in atlas-ui.
4. `go test ./...` in atlas-data.
5. Docker build of `atlas-data` and `atlas-ui`.
6. Bring up `docker-compose.core.yml`; verify page renders against a seeded tenant with a known NPC (e.g., Cassandra in Henesys) — ≥10 quests, ≥3 spawn maps, no console errors, no React key warnings.

## 5. Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Quest-filter in-memory scan too slow at p95 | Low | Medium | Target p95 < 50 ms at ~2000 quests. Measure in staging pre-rollout. Escalate to a persistent `quest_npc_index` table only if p95 > 200 ms — tracked as a follow-up, **not** in this task's DoD. |
| Retiring the singular `/map` endpoint breaks an un-migrated consumer | Low | High | Singular endpoint has exactly one caller (`ItemNpcShopWidget`). Grep atlas-ui and other services for `getSpawnMap`, `/npcs/.+/map` (non-`maps`), and `useNpcSpawnMap` before merging. Both sides of the migration ship in the same PR — no grace period needed. |
| Tenant has not re-ingested maps since task-010 → empty Spawn Locations card | Medium | Low | Card's empty state copy is acceptable ("This NPC is not placed on any loaded map."). Out of scope to re-seed; operational issue tracked separately. |
| `QuestDefinition` payload shape drift between `/data/quests/:id` and the new endpoint | Low | Medium | Reuse `quest.RestModel` verbatim on the server — enforced by the byte-identical acceptance criterion. UI consumes the same type used by `useQuestData`. |
| `both` role badge label ("Initiator & Completer") too long in narrow tile grids | Medium | Low | PRD §9 flags this; default to the long form. If tiles truncate in practice, swap to "Two-way" behind a one-line change. |
| Parallel queries exhaust a cold React Query cache on tenant switch | Low | Low | `TenantProvider` already calls `queryClient.clear()` on switch — queries respect the `enabled: !!activeTenant && npcId > 0` gate, so nothing fires until the new tenant is active. |
| Shop/Conversation summary cards double-fetch data the edit pages already load | Low | Low | Shop card reuses the `["npcs", "shop", tenantId, npcId]` key already owned by `NpcShopPage`. Conversation card's new hook is a shared wrapper — edit page can migrate opportunistically. |

## 6. Success Metrics

- **Operator efficiency**: Time-to-answer for "where is this NPC / what quests does it give / what does it sell" drops to one page load. Previously required ≥3 page loads (maps list + quests list + shop page).
- **Cross-reference coverage**: 100% of map, shop, conversation, and quest relationships linked from the page.
- **Backend latency**: `/data/npcs/:npcId/maps` p95 < 10 ms. `/data/npcs/:npcId/quests` p95 < 50 ms at ~2000-quest tenant load.
- **Regression surface**: Zero consumers of the retired `/data/npcs/:npcId/map` endpoint after the PR merges.
- **Parity**: `NpcHeader` tooltip-to-copy behaviour is visually and functionally identical to `MonsterHeader`, `MapHeader`, and post-task-011 `ItemDetailPage` header.

## 7. Required Resources and Dependencies

### Code dependencies (existing — no external blockers)

- `npc_spawn_index` table + ingest write path (task-010).
- `quest.RestModel` and `document.Storage` (`QUEST` collection) — already in place.
- `MonsterHeader`, `MonsterSpawnMapWidget`, `MonsterDetailPage` — reference implementations.
- `conversationsService.getByNpcId` (atlas-ui, `services/api/conversations.service.ts:140-156`).
- `TooltipContent copyable` pattern (shared primitive, used by existing *Header components).
- `ErrorDisplay`, `Skeleton`, `Card`, `Badge`, `Button`, `Link` (shadcn/ui + existing wrappers).

### Services touched

- `atlas-data` — new routes + handlers + tests only; no migrations, no Kafka, no new storage.
- `atlas-ui` — new components / hooks / services; page rewrite; one consumer migration.

### Services **not** touched

- `atlas-query-aggregator`, `atlas-quest`, `atlas-npc-shops`, `atlas-npc-conversations`.
- nginx route table (`deploy/shared/routes.conf`).

## 8. Timeline Estimates

Effort scale: **S** ≈ < half day, **M** ≈ half to full day, **L** ≈ one to two days.

| Phase | Effort | Sequencing |
|---|---|---|
| 1. Backend plural spawn-maps endpoint | S | Start here — unblocks §7 migration. |
| 2. Backend quests-by-npc endpoint | M | Independent of Phase 1; can run in parallel. |
| 3. Backend retire singular `/map` | S | Must follow Phase 1 **and** Phase 7 (consumer migration) landing together. |
| 4. Frontend types / services / hooks | M | Depends on Phase 1 + Phase 2 payloads being frozen. |
| 5. Frontend components | M | Depends on Phase 4 hooks. |
| 6. Frontend page rewrite | L | Depends on Phases 4 + 5. |
| 7. Frontend `ItemNpcShopWidget` migration | S | Depends on Phase 4 `useNpcSpawnMaps`. |
| 8. Tests, builds, integration | M | Final phase — spans both stacks and Docker. |

Single-engineer total: ~3–4 working days. Parallelizable between backend and frontend after Phases 1–2 freeze the payloads.

## 9. Out of Scope

Per PRD §2 non-goals:

- Inline editing of shop commodities or conversation states on `/npcs/:id`.
- Quest detail page enrichment (script preview, WZ enrichment, chain view, reverse NPC refs from the quest side).
- Filtering / search on `NpcsPage`.
- Changes to `atlas-npc-shops`, `atlas-npc-conversations`, `atlas-quest` runtime services.
- A persistent `quest_npc_index` table — revisit only on metric pressure.
- Cash-shop reverse-lookup.
- Preserving the refresh button.
