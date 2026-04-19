# NPC Detail Redesign — Task Checklist

**Last Updated: 2026-04-19**
**Status: COMPLETED**

## Overview

Rewrite `NpcDetailPage` (atlas-ui) into an information-dense layout with Spawn Locations, Shop/Conversation summary cards, and a reverse-lookup Quests grid. Add two endpoints in atlas-data (plural spawn-maps + quests-by-npc), retire the singular spawn-map endpoint, and migrate its only consumer.

Authoritative spec: `docs/tasks/task-012-npc-detail-redesign/prd.md`.

---

## Phase 1 — Backend: Plural Spawn-Maps Endpoint (atlas-data)

- [x] **1.1** Add `handleGetNpcMapsRequest` handler
  - File: `services/atlas-data/atlas.com/data/npc/resource.go`.
  - Query `npc_spawn_index` for all `(tenant, npc)` rows ordered by `spawn_count DESC, map_id ASC`.
  - De-duplicate by `mapId` before marshalling (defensive — unique key should already prevent).
  - Reuse `NpcMapRestModel` from `spawn_map_rest.go:13-14`; marshal with `server.MarshalResponse[[]NpcMapRestModel]`.
  - 400 on missing/invalid `npcId`; 500 on DB failure; 200 `{"data":[]}` on empty.
  - Acceptance: handler returns JSON:API body with `type: "npc-maps"` and `id == stringified mapId`.

- [x] **1.2** Register `/{npcId}/maps` in `npc.InitResource`
  - File: `services/atlas-data/atlas.com/data/npc/resource.go:20-30`.
  - `r.HandleFunc("/{npcId}/maps", registerGet("get_npc_maps", handleGetNpcMapsRequest(db))).Methods(http.MethodGet)`.
  - Acceptance: route reaches the handler; smoke via curl returns JSON.

- [x] **1.3** Emit Debug log on handler completion
  - Fields: `tenant_id`, `npc_id`, `result_ct`, `elapsed_ms`.
  - Pattern mirror: `resource.go:129-137`.
  - Acceptance: log line visible in dev logs on request.

- [x] **1.4** Tests — `resource_test.go`
  - Happy path (multiple rows sorted correctly).
  - Empty result (200, `{"data":[]}`).
  - Bad `npcId` (non-integer / ≤ 0 → 400).
  - Missing tenant header → 400.
  - DB read failure → 500.
  - Acceptance: `go test ./services/atlas-data/atlas.com/data/npc/...` green.

---

## Phase 2 — Backend: Quests-by-NPC Endpoint (atlas-data)

- [x] **2.1** Add `handleGetNpcQuestsRequest` handler
  - File: `services/atlas-data/atlas.com/data/npc/resource.go`.
  - Load all tenant quest documents via `quest.NewStorage(d.Logger(), db).GetAll(d.Context())` — mirror `quest/resource.go:28-42`.
  - Filter in memory: retain quests where any of `q.StartRequirements.NpcId`, `q.EndRequirements.NpcId`, `q.StartActions.NpcId`, `q.EndActions.NpcId` equals the path `npcId`.
  - Marshal as `[]quest.RestModel` (**byte-identical** to `/data/quests/:id`).
  - Server-side sort `id ASC`.
  - 400 / 500 / empty handling same as §1.1.
  - Acceptance: payload per-item shape matches `GET /data/quests/:id`.

- [x] **2.2** Register `/{npcId}/quests` in `npc.InitResource`
  - File: `services/atlas-data/atlas.com/data/npc/resource.go:20-30`.
  - `r.HandleFunc("/{npcId}/quests", registerGet("get_npc_quests", handleGetNpcQuestsRequest(db))).Methods(http.MethodGet)`.
  - Acceptance: route reaches the handler.

- [x] **2.3** Emit Debug log on handler completion
  - Fields: `tenant_id`, `npc_id`, `result_ct`, `elapsed_ms`.
  - Acceptance: log line visible.

- [x] **2.4** Tests — `resource_test.go`
  - Quest matches only on `startRequirements.npcId` → returned.
  - Quest matches only on `endActions.npcId` → returned.
  - Quest matches on both → returned once (not duplicated).
  - Quest unrelated to NPC → filtered out.
  - Tenant has no quests → `200 {"data":[]}`.
  - Bad `npcId` → 400; missing tenant → 400.
  - Acceptance: `go test ./services/atlas-data/atlas.com/data/npc/...` green.

- [x] **2.5** Manual p95 measurement
  - Load fixture with ~2000 quests; time 100 invocations; record p95.
  - Acceptance: p95 < 50 ms locally. If > 200 ms, escalate with a follow-up task — **not** a blocker for merge.

---

## Phase 3 — Backend: Retire Singular `/map` Endpoint (atlas-data)

> **Note:** This phase must not land before Phase 7 (UI migration) is complete, since it breaks `ItemNpcShopWidget`. Keep sequenced in the same PR.

- [x] **3.1** Remove route registration for `/{npcId}/map`
  - File: `services/atlas-data/atlas.com/data/npc/resource.go:28`.
  - Acceptance: singular endpoint returns 404.

- [x] **3.2** Delete `handleGetNpcMapRequest`
  - File: `services/atlas-data/atlas.com/data/npc/resource.go:152-191`.
  - Keep `NpcMapRestModel` in `spawn_map_rest.go` — reused by the plural handler.
  - Acceptance: package still compiles; no orphaned references.

- [x] **3.3** Remove singular-endpoint test cases from `resource_test.go`
  - Replace with plural equivalents (already added in §1.4) where coverage overlapped.
  - Acceptance: test file references no retired handler / route.

---

## Phase 4 — Frontend: Types, Services, Hooks (atlas-ui)

- [x] **4.1** Add types to `src/types/models/npc.ts`
  - `NpcSpawnMapAttributes` — plural variant of the existing single-spawn type.
  - `NpcQuestRole = "initiator" | "completer" | "both"`.
  - Acceptance: `tsc` clean.

- [x] **4.2** Add service methods to `src/services/api/npcs.service.ts`
  - `getNpcSpawnMaps(npcId): Promise<NpcSpawnMap[]>` → `GET /api/data/npcs/:npcId/maps`.
  - `getNpcQuests(npcId): Promise<QuestDefinition[]>` → `GET /api/data/npcs/:npcId/quests`.
  - **Remove** `getSpawnMap(npcId)` (singular).
  - Acceptance: `tsc` clean; service exports updated.

- [x] **4.3** Create `src/lib/hooks/api/useNpcSpawnMaps.ts`
  - Query key `["data", "npcs", "maps", tenantId, npcId]`.
  - `enabled: !!activeTenant && npcId > 0`.
  - Returns `NpcSpawnMap[]`.
  - Acceptance: hook renders without warnings in a storybook/test harness.

- [x] **4.4** Create `src/lib/hooks/api/useNpcQuests.ts`
  - Query key `["data", "npcs", "quests", tenantId, npcId]`.
  - Returns `{ quest: QuestDefinition, role: NpcQuestRole }[]` — role derived client-side per PRD §5.2.
  - Acceptance: unit test asserts role derivation for initiator-only / completer-only / both inputs.

- [x] **4.5** Create `src/lib/hooks/api/useNpcConversation.ts`
  - Wraps existing `conversationsService.getByNpcId`.
  - Acceptance: hook fetches without refetching when key is already in cache.

- [x] **4.6** Delete `src/lib/hooks/api/useNpcSpawnMap.ts` and any barrel exports
  - Confirm no remaining importers via grep.
  - Acceptance: `npm run build` succeeds with the file removed.

---

## Phase 5 — Frontend: Components (atlas-ui)

- [x] **5.1** Create `src/components/features/npc/NpcHeader.tsx`
  - Mirror `MonsterHeader.tsx:9-57` — 64×64 icon, `TooltipContent copyable` with template id.
  - Name fallback: `NPC #${npcId}`.
  - Accessibility: `tabIndex={0}`, `focus-visible:ring-2`.
  - Acceptance: tooltip copies template id; keyboard-focusable.

- [x] **5.2** Create `src/components/features/npc/NpcSpawnMapWidget.tsx`
  - Mirror `MonsterSpawnMapWidget.tsx:1-30`; link to `/maps/:mapId`.
  - Show `name`, `streetName` badge, spawn-count badge (`1 spawn` / `N spawns`).
  - Parent keys items by `${npcId}-${mapId}`.
  - Acceptance: click navigates; no console key warnings.

- [x] **5.3** Create `src/components/features/npc/NpcQuestWidget.tsx`
  - Per PRD §4.6.1 — `Scroll` icon + quest name + `parent` badge + role badge.
  - Role label map: `initiator → "Initiator"`, `completer → "Completer"`, `both → "Initiator & Completer"`.
  - Role badge variant map: `initiator → "default"`, `completer → "outline"`, `both → "secondary"`.
  - Links to `/quests/:id`.
  - Acceptance: snapshot / render tests cover all three role states.

---

## Phase 6 — Frontend: Page Rewrite (atlas-ui)

File: `services/atlas-ui/src/pages/NpcDetailPage.tsx` — full rewrite (~230–260 lines).

- [x] **6.1** Replace outer layout
  - Container: `flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto` — matches `MonsterDetailPage`.
  - Remove the old `<h2>` heading, `RefreshCw` button, `max-w-md` single-card wrapper, and the `ID: {npcId}` line.
  - Acceptance: page is full-width with the scrolling container behaviour.

- [x] **6.2** Render `NpcHeader` at the top (§4.2)
  - Consume `useNpcData(npcId)` for `name` + `iconUrl`.
  - Acceptance: header matches `MonsterHeader` visually.

- [x] **6.3** Render Spawn Locations card (§4.3)
  - `<Card>` → `<CardHeader>Spawn Locations ({count})</CardHeader>`; count omitted on 0.
  - `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `NpcSpawnMapWidget`.
  - Loading / error / empty states per PRD §4.3.
  - Data via `useNpcSpawnMaps(npcId)`.
  - Acceptance: all four visual states render correctly against fixtures.

- [x] **6.4** Render Shop card (§4.4)
  - `<Card>` → `<CardHeader>Shop</CardHeader>`.
  - Has-shop: stat rows (Recharger / Commodities / Tokens of Commodities) + `Edit Shop` button → `/npcs/:id/shop` with `ShoppingBag` icon.
  - No-shop: muted placeholder + `Create Shop` outline button → same route.
  - Loading: skeleton rows.
  - Error: inline `ErrorDisplay` with the query's `refetch`.
  - Shop data via `npcsService.getNPCShop(npcId)` under the existing `["npcs", "shop", tenantId, npcId]` key (cache reuse).
  - Acceptance: navigating to Edit Shop lands on unchanged shop page.

- [x] **6.5** Render Conversation card (§4.5)
  - `<Card>` → `<CardHeader>Conversation</CardHeader>`.
  - Has-conversation: `States` / `Start State` / `Entry Preview` rows + `Edit Conversation` button with `MessageCircle` icon.
  - Entry Preview: first dialogue text on the start state (truncated to one line); fallback `(no dialogue)` when non-dialogue start.
  - No-conversation: placeholder + `Create Conversation` outline button.
  - Loading / error treatment matches the Shop card.
  - Data via `useNpcConversation(npcId)`.
  - Acceptance: all four visual states render correctly.

- [x] **6.6** Wrap Shop + Conversation in `grid grid-cols-1 lg:grid-cols-2 gap-4`
  - Acceptance: two-column on `lg`+; single-column below.

- [x] **6.7** Render Quests card (§4.6)
  - `<Card>` → `<CardHeader>Quests ({count})</CardHeader>`; count omitted on 0.
  - Non-empty: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `NpcQuestWidget`.
  - Sort: Initiator → Both → Completer, ties broken by name ASC, id ASC.
  - Loading / error / empty states per PRD §4.6.
  - Data via `useNpcQuests(npcId)`.
  - Acceptance: sort stable across re-renders.

- [x] **6.8** Verify parallel data fan-out
  - `useNpcData`, `useNpcSpawnMaps`, `useNpcQuests`, shop summary, conversation summary all fire on mount — no waterfalls.
  - Acceptance: Network tab shows requests in parallel.

---

## Phase 7 — Frontend: `ItemNpcShopWidget` Migration (atlas-ui)

- [x] **7.1** Migrate `ItemNpcShopWidget.tsx:22` from `useNpcSpawnMap` to `useNpcSpawnMaps`
  - File: `services/atlas-ui/src/components/features/items/ItemNpcShopWidget.tsx`.
  - Take `response[0]` (server already sorts `spawn_count DESC, map_id ASC`).
  - Render the existing `{name} · {streetName}` badge from the top row.
  - When `response.length > 1`, append an outline-variant `+{N-1}` badge.
  - When `response.length === 0`, render no badge.
  - Acceptance: widget looks identical on single-map NPCs; shows `+N` on multi-map NPCs (e.g., gachapon).

- [x] **7.2** Grep for remaining singular-endpoint callers
  - Search atlas-ui for `getSpawnMap`, `useNpcSpawnMap`, and regex `/npcs/\$?\{[^}]+\}/map(?!s)`.
  - Acceptance: zero matches outside the deleted files.

---

## Phase 8 — Tests, Builds, Integration (cross-cutting)

- [x] **8.1** UI unit tests
  - `NpcQuestWidget` role-label and variant mapping.
  - `useNpcQuests` role derivation (initiator-only / completer-only / both / outlier-`startActions`-only / outlier-`endRequirements`-only).
  - `NpcHeader` tooltip-copy interaction.
  - `NpcDetailPage` per-card loading / error / empty / populated rendering under mocked hooks.
  - Acceptance: `npm run test` green.

- [x] **8.2** Lint & type-check
  - `npm run lint`.
  - `tsc` (already covered by `npm run build`).
  - Acceptance: both clean.

- [x] **8.3** Frontend production build
  - `npm run build` in `services/atlas-ui`.
  - Acceptance: bundle succeeds; no residual references to deleted files.

- [x] **8.4** Backend tests
  - `go test ./services/atlas-data/...`.
  - Acceptance: all suites green.

- [x] **8.5** Docker builds
  - `docker build` for `atlas-data`.
  - `docker build` for `atlas-ui`.
  - Acceptance: both succeed in CI-equivalent local run.

- [x] **8.6** Smoke test via `docker-compose.core.yml`
  - Seed tenant with a known multi-map NPC (e.g., gachapon or Cassandra) and an NPC with ≥10 quests.
  - Load `/npcs/:id`:
    - Header shows 64×64 icon + name + tooltip-copies template id.
    - Refresh button absent; `ID:` line absent; `max-w-md` wrapper absent.
    - Spawn Locations renders ≥3 tiles, each linking to `/maps/:mapId`.
    - Shop + Conversation cards render correct state (has-data or create-state); Edit button navigates correctly.
    - Quests card renders ≥10 tiles; role badge present; tiles link to `/quests/:id`.
    - No console errors, no React key warnings.
  - Acceptance: all above true against seeded data.

- [x] **8.7** Regression check — retired endpoint
  - `curl` `/data/npcs/:npcId/map` (singular) → 404.
  - Grep all services for `/npcs/.+/map` (non-`maps`) — expect zero hits.
  - Acceptance: no lingering consumers.

---

## Definition of Done

All checkboxes above complete. PRD §10 acceptance criteria satisfied in full (`docs/tasks/task-012-npc-detail-redesign/prd.md:461-507`). Docker builds of both affected services green. No console errors or React key warnings on the redesigned page with a realistic tenant seed.
