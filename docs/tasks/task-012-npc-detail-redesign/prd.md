# NPC Detail Redesign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-19
---

## 1. Overview

`NpcDetailPage` in atlas-ui today (`services/atlas-ui/src/pages/NpcDetailPage.tsx:15-189`) is the thinnest of the detail pages: a page heading, a card with a 96×96 icon, the name, a literal "ID: {npcId}" line, and two buttons — "View Shop" and "View Conversation" — each greyed out when the corresponding resource doesn't exist. To answer any useful question about an NPC ("what quests use them?", "where do they stand?", "what do they sell?") the operator has to click through to a separate page, and quest relationships are not surfaced at all.

This task refactors the page to mirror the information-dense pattern established by `MapDetailPage` (task-008), `MonsterDetailPage` (task-010), and `ItemDetailPage` (task-011):

- A header with a tooltip-to-copy template id, dropping the "ID: {npcId}" line and the refresh button.
- A new "Spawn Locations" card — a widget grid sourced from `npc_spawn_index`, mirroring `MonsterSpawnMapWidget` — answering "where does this NPC stand?" at a glance, including the multi-map case that the existing singular endpoint hides.
- A new "Quests" card listing every quest where this NPC is an initiator (`startRequirements.npcId`) or a completer (`endActions.npcId`), each tile showing the quest name, parent category, and a role badge, and linking to `/quests/:id`.
- A summary-with-deep-link treatment for Shop and Conversation: read-only preview cards ("Recharger: yes / 12 commodities" or "32 states / entry preview dialogue") with `Edit Shop` and `Edit Conversation` buttons that navigate to the existing `/npcs/:id/shop` and `/npcs/:id/conversations` routes.

Data dependencies require two backend additions in `atlas-data`, plus the retirement of one existing endpoint. First, a new plural spawn-maps endpoint `GET /data/npcs/:npcId/maps` returning all rows from `npc_spawn_index` for the active tenant. The existing singular `GET /data/npcs/:npcId/map` endpoint (returning top-1 by `spawn_count DESC`) is retired as part of this task — it hides the fact that many NPCs (gachapon, town service NPCs) stand in multiple maps, and the single consumer (`ItemNpcShopWidget.tsx:22`) is migrated to the plural hook in this same PR. Second, a new reverse-lookup `GET /data/npcs/:npcId/quests` returning every quest definition where `startRequirements.npcId == npcId` or `endActions.npcId == npcId` — implemented as an in-memory scan over the tenant's registered quest documents, no new GORM table (quest count is ~O(thousands) per tenant and this page is admin-only; same tradeoff rationale as `GetAutoStartQuests` in `services/atlas-quest/atlas.com/quest/data/quest/processor.go:40-62`).

Out of scope is any enrichment of the quest detail page itself — script-conversation preview, WZ enrichment, and cross-references from the quest side will be handled in a follow-up task. This PRD stops at "click the quest tile and you land on `/quests/:id`."

## 2. Goals

Primary goals:

- Give operators a single NPC page that answers "where is it, what does it sell, what conversation does it run, and what quests does it participate in" without tab-switching.
- Make every cross-reference (map, quest, shop, conversation) clickable to its detail/editor page.
- Eliminate the "ID:" literal line in favour of the tooltip-to-copy pattern already used by `MapHeader`, `MonsterHeader`, and `ItemDetailPage`'s header (post-task-011).
- Surface spawn locations — the NPC page today never shows where the NPC actually stands.
- Surface quest relationships — the NPC page today never shows what quests reference the NPC.
- Keep `/npcs/:id/shop` and `/npcs/:id/conversations` routes alive so direct links from other pages (`ItemNpcShopWidget`, breadcrumbs, bookmarks) do not break.

Non-goals:

- Editing shop commodities or conversation state machines inline on `/npcs/:id`. Edit flows stay on the existing pages; the NPC page is read-only summary + deep link (option C from the scope-review Q1).
- Enriching the quest detail page. Quest cards show name + parent + role and link out. Per-quest script-conversation preview, WZ enrichment, quest chain view, NPC cross-refs from the quest side — all deferred to a separate task.
- New filtering/search on the NPC list page (`NpcsPage`).
- Changes to `atlas-npc-shops`, `atlas-npc-conversations`, or `atlas-quest` runtime services. All new backend work is in `atlas-data`.
- A new persistent reverse-lookup table (`quest_npc_index`) analogous to `npc_spawn_index`. In-memory scan is acceptable for the volume and access pattern — revisit only if the endpoint shows up hot in metrics.
- Cash-shop commodity reverse-lookup on the NPC page (confirmed non-issue — no cash-shop dummies to surface).
- Handling the "refresh" UX — the button is dropped entirely, in line with the sibling detail pages post-redesign.

## 3. User Stories

- As a GM triaging a bug report ("this NPC gives the wrong quest reward"), I want to land on the NPC page and see every quest the NPC initiates or completes so I can jump straight to the broken one.
- As a content designer writing patch notes, I want to hover the NPC name and copy the template id into my notes without re-typing it.
- As a GM answering a player ticket ("where is Cassandra?"), I want the NPC page to show the map name + street name in one place rather than memorising her template id and searching the map list.
- As a shop admin, I want to see "Recharger: on, 14 commodities" on the NPC overview and one-click into the shop editor, rather than loading the shop page just to check the headline.
- As a dialogue writer, I want to see at a glance whether an NPC has a conversation defined and how many states it has, before deciding whether to edit it.
- As a platform engineer, I want the new spawn-maps endpoint to work off the existing `npc_spawn_index` with no schema or ingestion change.
- As a platform engineer, I want the quests-by-npc endpoint to have no new storage requirement, and to degrade gracefully (empty list, not 500) when the tenant has no registered quests.

## 4. Functional Requirements

### 4.1 Page layout (`NpcDetailPage.tsx`)

From top to bottom, inside a scrolling container (`flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto` — matches `MonsterDetailPage`):

1. **Header row** (§4.2) — NPC icon + name + tooltip-to-copy template id. Replaces the current `<h2>` heading, the `Card` with icon-and-id, and the `RefreshCw` button.
2. **Spawn Locations card** (§4.3) — widget grid of maps where this NPC is placed, sourced from `npc_spawn_index`. Shows a friendly empty state when the NPC is not placed on any ingested map.
3. **Two-column grid: Shop card + Conversation card** (§4.4, §4.5) — `grid grid-cols-1 lg:grid-cols-2 gap-4`. Each card shows a read-only summary and an `Edit Shop` / `Edit Conversation` button that navigates to the existing edit page. When the NPC has no shop / conversation, the card renders a muted placeholder with a `Create Shop` / `Create Conversation` button linking to the same edit page (the edit pages already handle the "create on save" case).
4. **Quests card** (§4.6) — widget grid of quests where this NPC appears in `startRequirements.npcId` or `endActions.npcId`.

No separate "Actions" card. No "ID: {npcId}" line. No refresh button.

### 4.2 Header (`NpcHeader.tsx` — new component under `components/features/npc/`)

Follow `MonsterHeader.tsx:9-57` exactly.

- Outer `div` with `flex items-center gap-3 flex-wrap`.
- `TooltipProvider` → `Tooltip` → `TooltipTrigger asChild` → `<span tabIndex={0}>` wrapping the icon and the name.
  - Span classes: `inline-flex items-center gap-3 cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded`.
- Icon: `<img>` at `width={64} height={64} className="object-contain"` using `iconUrl` from `useNpcData(npcId)`. If `iconUrl` is null/undefined, render nothing in the icon slot (same contract as `MonsterHeader` — no placeholder).
- Name: `<h2 className="text-2xl font-bold tracking-tight">{name || `NPC #${npcId}`}</h2>`.
- `TooltipContent copyable` with `<p>{npcId}</p>`. Clicking the tooltip copies the template id.

No badges on the NPC header — NPCs don't have the equivalent of boss/undead/friendly flags.

Props:

```ts
interface NpcHeaderProps {
  npcId: number;
  name?: string | undefined;
  iconUrl?: string | undefined;
}
```

### 4.3 Spawn Locations card

Layout matches the monster page Spawn Locations section (`MonsterDetailPage.tsx:245-268`):

- `<Card>` with `<CardHeader>` title `Spawn Locations ({count})` (count omitted when 0).
- `<CardContent>` contains a `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `NpcSpawnMapWidget` items (new component, §4.3.1), sorted by `spawnCount desc, name asc`.
- Loading state: `<p className="text-sm text-muted-foreground">Loading spawn locations...</p>`.
- Error state: `<ErrorDisplay>` with the query's `refetch`.
- Empty state: `<p className="text-sm text-muted-foreground">This NPC is not placed on any loaded map.</p>`.

#### 4.3.1 `NpcSpawnMapWidget.tsx` (new)

Mirror `MonsterSpawnMapWidget.tsx:1-30` verbatim, except the route is `/maps/:mapId`:

```tsx
<Link
  to={`/maps/${entry.attributes.mapId}`}
  className="flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
>
  <div className="flex items-center gap-2 flex-wrap">
    <span className="text-sm font-medium truncate">{entry.attributes.name}</span>
    {entry.attributes.streetName && (
      <Badge variant="secondary">{entry.attributes.streetName}</Badge>
    )}
  </div>
  <Badge variant="outline">
    {entry.attributes.spawnCount === 1
      ? "1 spawn"
      : `${entry.attributes.spawnCount} spawns`}
  </Badge>
</Link>
```

The widget is keyed by `${npcId}-${mapId}` in the parent list.

### 4.4 Shop card

`<Card>` with `<CardHeader>` title `Shop`.

- **Has-shop state** (`npc.hasShop === true`):
  - `<CardContent>` shows three stat rows:
    - `Recharger — {recharger ? "Yes" : "No"}`
    - `Commodities — {commodities.length}`
    - `Tokens — {commoditiesWithTokenPrice.length} of {commodities.length}` (count rows with `tokenPrice > 0 && tokenTemplateId > 0`).
  - `<CardFooter>` (or a `flex justify-end` row inside content) with an `<Button asChild><Link to={/npcs/:id/shop}>Edit Shop</Link></Button>`.
  - Button icon: `ShoppingBag` from lucide (keeps the current signifier).
- **No-shop state** (`npc.hasShop === false`):
  - `<CardContent>` shows `<p className="text-sm text-muted-foreground">No shop configured.</p>`.
  - `<Button variant="outline" asChild><Link to={/npcs/:id/shop}>Create Shop</Link></Button>` — the existing shop page handles the "first commodity triggers shop creation" flow.
- **Loading state**: three `<Skeleton>` rows + a disabled-looking button skeleton.
- **Error state**: inline `<ErrorDisplay>` inside `<CardContent>`.

Commodities are fetched by the same query key the shop page uses: `["npcs", "shop", activeTenant?.id ?? "no-tenant", npcId]` via `npcsService.getNPCShop(npcId)` — reusing the cache so navigating to the edit page does not re-fetch.

### 4.5 Conversation card

`<Card>` with `<CardHeader>` title `Conversation`.

- **Has-conversation state** (`npc.hasConversation === true`):
  - Fetch the conversation via `conversationsService.getByNpcId(npcId)` — already implemented in `services/atlas-ui/src/services/api/conversations.service.ts:140-156`.
  - Stat rows:
    - `States — {conversation.attributes.states.length}`
    - `Start State — {conversation.attributes.startState}`
    - `Entry Preview —` one-line truncated preview of the first dialogue text on the start state, or "(no dialogue)" when the start state is non-dialogue.
  - `<Button asChild><Link to={/npcs/:id/conversations}>Edit Conversation</Link></Button>` with a `MessageCircle` icon.
- **No-conversation state**:
  - `<p className="text-sm text-muted-foreground">No conversation configured.</p>`.
  - `<Button variant="outline" asChild><Link to={/npcs/:id/conversations}>Create Conversation</Link></Button>`.
- **Loading / error** states match the Shop card treatment.

### 4.6 Quests card

`<Card>` with `<CardHeader>` title `Quests ({count})` (count omitted when 0).

- `<CardContent>`:
  - Loading: `<p className="text-sm text-muted-foreground">Loading quests...</p>`.
  - Error: `<ErrorDisplay>` with the query's `refetch`.
  - Non-empty: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2` of `NpcQuestWidget`.
  - Empty: `<p className="text-sm text-muted-foreground">This NPC does not participate in any quest.</p>`.

Sort order (stable): `Initiator` roles first, then `Both`, then `Completer`, ties broken by `quest.name` ASC then `quest.id` ASC.

#### 4.6.1 `NpcQuestWidget.tsx` (new)

Minimum viable card — no WZ enrichment, no level range, no conversation preview. Deferred to the quest-detail redesign.

```tsx
<Link
  to={`/quests/${quest.id}`}
  className="flex flex-col gap-1 rounded-md border bg-card p-3 hover:bg-accent transition-colors"
>
  <div className="flex items-center gap-2 flex-wrap">
    <Scroll className="h-4 w-4 text-muted-foreground shrink-0" />
    <span className="text-sm font-medium truncate">
      {quest.attributes.name || `Quest #${quest.id}`}
    </span>
  </div>
  <div className="flex items-center gap-2 flex-wrap">
    {quest.attributes.parent && (
      <Badge variant="secondary">{quest.attributes.parent}</Badge>
    )}
    <Badge variant={roleBadgeVariant(role)}>{roleLabel(role)}</Badge>
  </div>
</Link>
```

Props:

```ts
interface NpcQuestWidgetProps {
  quest: QuestDefinition;
  role: "initiator" | "completer" | "both";
}
```

`roleLabel` maps to `Initiator` / `Completer` / `Initiator & Completer`.
`roleBadgeVariant` maps `initiator -> "default"`, `completer -> "outline"`, `both -> "secondary"`.

Role derivation on the UI side: for each `QuestDefinition` returned by the new endpoint, compute the role from the attribute shape rather than from any server-side flag — see §5.2.

### 4.7 Data flow / query keys

New React Query hooks under `services/atlas-ui/src/lib/hooks/api/`:

- `useNpcSpawnMaps(npcId)` → query key `["data", "npcs", "maps", tenantId, npcId]`, fetches `/api/data/npcs/:npcId/maps`. Returns `NpcSpawnMap[]`.
- `useNpcQuests(npcId)` → query key `["data", "npcs", "quests", tenantId, npcId]`, fetches `/api/data/npcs/:npcId/quests`. Returns `QuestDefinition[]` with each element tagged with the derived role.

Both hooks respect the standard `enabled: !!activeTenant && npcId > 0` gate.

Existing hooks reused unchanged:

- `useNpcData(npcId)` — name + icon.
- `npcsService.getNPCById(npcId)` via the existing `["npcs", "detail", ...]` query — `hasShop` / `hasConversation` flags.
- `npcsService.getNPCShop(npcId)` via the shop page's query key — shop summary.
- `conversationsService.getByNpcId(npcId)` — conversation summary (new query hook `useNpcConversation(npcId)`).

### 4.8 Removed behaviours

- The `RefreshCw` button and its `fetchNpcData` handler are removed.
- The "ID: {npcId}" `<p>` line is removed.
- The `max-w-md` card wrapper around the icon/name/actions is removed — the page is now full-width like the sibling detail pages.
- The "This NPC has no shop or conversation configured." fallback line is removed — each card owns its own empty state.

## 5. API Surface

### 5.1 `GET /data/npcs/:npcId/maps` — new

Returns all `npc_spawn_index` rows for the active tenant and given npc.

Request:

```
GET /api/data/npcs/9201000/maps
Headers: TENANT_ID, REGION, MAJOR_VERSION, MINOR_VERSION
```

Response (200, JSON:API):

```json
{
  "data": [
    {
      "type": "npc-maps",
      "id": "100000000",
      "attributes": {
        "mapId": 100000000,
        "name": "Henesys",
        "streetName": "Free Market Entrance",
        "spawnCount": 1
      }
    }
  ]
}
```

`data[].id` is the stringified `mapId` (matches the existing `NpcMapRestModel.GetID()` for the singular endpoint — see `services/atlas-data/atlas.com/data/npc/spawn_map_rest.go:13-14`). If multiple rows share an npc/map pair (should not happen — unique key is `(tenant, npc, map)`), the server de-duplicates by `mapId` before responding.

Response (200, empty):

```json
{ "data": [] }
```

Response (400): missing/invalid `npcId` path param.

Sort: `spawn_count DESC, map_id ASC` — matches the existing singular endpoint's `ORDER BY`.

Not 404 when the NPC has no spawn rows — the NPC may exist in the registry with zero placements. Callers distinguish "NPC not found" (check `GET /api/data/npcs/:npcId` → 404) from "NPC has no spawns" (this endpoint → 200 `[]`).

### 5.2 `GET /data/npcs/:npcId/quests` — new

Returns all quests where the NPC appears as `startRequirements.npcId` or `endActions.npcId`, for the active tenant.

Request:

```
GET /api/data/npcs/1012100/quests
Headers: TENANT_ID, REGION, MAJOR_VERSION, MINOR_VERSION
```

Response (200, JSON:API):

```json
{
  "data": [
    {
      "type": "quests",
      "id": "2050",
      "attributes": {
        "name": "Gaga the magician wannabe",
        "parent": "Magatia",
        "area": 2,
        "order": 0,
        "autoStart": false,
        "autoPreComplete": false,
        "autoComplete": false,
        "startRequirements": { "npcId": 1012100, "levelMin": 50 },
        "endRequirements": { "npcId": 1012100 },
        "startActions": {},
        "endActions": { "npcId": 1012100, "exp": 15000 }
      }
    }
  ]
}
```

The rest-model payload is **identical** to `RestModel` from `services/atlas-quest/atlas.com/quest/data/quest/rest.go:10-29` — same shape already served by `GET /api/data/quests/:id`. Reuse the existing model.

Server-side implementation: iterate the tenant's registered quest documents (same access pattern as `handleGetQuests` in `services/atlas-data/atlas.com/data/quest/resource.go:28-42`) and filter in memory to quests where `q.StartRequirements.NpcId == npcId || q.EndRequirements.NpcId == npcId || q.StartActions.NpcId == npcId || q.EndActions.NpcId == npcId`. No new `quest_npc_index` GORM table.

Role derivation (UI-side, not server-side): for each returned quest,

- `initiator` if `q.attributes.startRequirements.npcId === npcId` and NOT `q.attributes.endActions.npcId === npcId`.
- `completer` if `q.attributes.endActions.npcId === npcId` and NOT `q.attributes.startRequirements.npcId === npcId`.
- `both` if both match.
- Quests where the NPC appears only in `endRequirements.npcId` or `startActions.npcId` (uncommon) collapse into `initiator` if `startActions.npcId == npcId` else `completer` — these are diagnostic outliers, not the 99% case.

Sort (server-side): `id ASC` — simple and deterministic; the UI re-sorts by role + name anyway.

Response (200, empty): `{ "data": [] }` when the NPC participates in no quest. Never 404 for this case.

Response (400): missing/invalid `npcId` path param.

Performance budget: the handler scans all quests for the tenant in memory. Observed cardinality is ~2000 quests × ~20 fields. Target P95 latency < 50 ms at single-tenant load. Re-visit (add a `quest_npc_index` table + ingestion hook) only if p95 exceeds 200 ms or the page is called hot.

### 5.3 `GET /data/npcs/:npcId/map` — retired

Remove the route registration in `services/atlas-data/atlas.com/data/npc/resource.go:28` and delete `handleGetNpcMapRequest` (`resource.go:152-191`). Keep `NpcMapRestModel` in `spawn_map_rest.go` — it is reused by the plural endpoint.

Rationale: the singular endpoint picks the top-1 row from `npc_spawn_index` and hides the fact that many NPCs (gachapon NPCs, Cassandra-style event NPCs, town service NPCs) stand in multiple towns. The only consumer is `ItemNpcShopWidget.tsx:22`, migrated in §7.

On deploy, the route is gone and returns 404; consumers must call `/maps`. No deprecation grace period — the UI PR ships both sides of the migration.

### 5.4 Other existing endpoints used — no change

- `GET /api/data/npcs/:npcId` — name + storebank flag. No change.
- `GET /api/npcs` — shops + conversations combined list used by `NpcsPage` and the hasShop/hasConversation derivation. No change.
- `GET /api/npcs/:npcId/shop?include=commodities` — shop detail. No change.
- `GET /api/npcs/:npcId/conversations` — conversation list for an NPC. No change.

### 5.5 Error mapping

| Condition | Status | Body |
|---|---|---|
| Missing `TENANT_ID` header | 400 | empty |
| Unknown tenant | 400 | empty |
| Invalid `npcId` path param (non-integer, ≤ 0) | 400 | empty |
| NPC has no spawn rows | 200 | `{ "data": [] }` |
| NPC has no quests | 200 | `{ "data": [] }` |
| DB read failure | 500 | empty |

Matches the existing handlers' error envelope (see `resource.go:156-175`).

## 6. Data Model

No schema changes.

- `npc_spawn_index` (already present — `services/atlas-data/atlas.com/data/npc/spawn_index.go`) is re-used for §5.1. Primary key `(tenant_id, npc_id, map_id)` supports the N-row case. Index `idx_npc_spawn_index_lookup` covers the `tenant_id + npc_id` ordered-by-spawn-count read path.
- Quest documents are already stored by `atlas-data` in the MongoDB-backed `document.Storage` under the `QUEST` collection (see `services/atlas-data/atlas.com/data/quest/storage.go:11-14`). No ingestion or storage change.

No migration is required. The existing `npc_spawn_index` rows are the data source; no backfill or re-ingest is required for this task (unlike task-010, which introduced the index). If a tenant has not re-ingested maps since the task-010 rollout, the singular endpoint is already broken for them — out of scope here.

## 7. Service Impact

### `atlas-ui`

New files:

- `src/components/features/npc/NpcHeader.tsx` — §4.2.
- `src/components/features/npc/NpcSpawnMapWidget.tsx` — §4.3.1.
- `src/components/features/npc/NpcQuestWidget.tsx` — §4.6.1.
- `src/lib/hooks/api/useNpcSpawnMaps.ts` — multi-map spawn hook.
- `src/lib/hooks/api/useNpcQuests.ts` — quests-by-npc hook.
- `src/lib/hooks/api/useNpcConversation.ts` — single-conversation-by-npc hook (wraps existing `conversationsService.getByNpcId`).

Modified files:

- `src/pages/NpcDetailPage.tsx` — full rewrite per §4.1–§4.8. Page length grows from ~190 lines to ~230–260.
- `src/types/models/npc.ts` — add `NpcSpawnMapAttributes` shape (plural variant of the existing `NpcSpawnMap` type) and a `NpcQuestRole` enum.
- `src/services/api/npcs.service.ts` — add `getNpcSpawnMaps(npcId)` and `getNpcQuests(npcId)`. Remove `getSpawnMap(npcId)` (singular) — the endpoint is retired (§5.3).
- `src/components/features/items/ItemNpcShopWidget.tsx` — migrate from `useNpcSpawnMap` (singular) to `useNpcSpawnMaps` (plural). Display logic:
  - Take the top row from the response (server already sorts `spawn_count DESC, map_id ASC`).
  - Render the existing `{name} · {streetName}` badge from that top row.
  - If `response.length > 1`, append a small `+{N-1}` outline-badge next to the primary one so operators can see the NPC stands elsewhere too (compact widget stays compact; no "show all" UI in this widget — use `/npcs/:id` for that).
  - When `response.length === 0`, render no badge (same as today's 404 → null behaviour).

Removed files:

- `src/lib/hooks/api/useNpcSpawnMap.ts` — single consumer was `ItemNpcShopWidget`, migrated above. Delete the file; remove any barrel exports.
- Associated query key `["npcs", "spawn-map", npcId, tenantId]` stops being used; no cache-invalidation hook needs updating (queries simply age out).

Unchanged:

- `src/pages/NpcShopPage.tsx`, `src/pages/NpcConversationPage.tsx` — deep-link targets from the redesigned NPC page. No breaking changes.
- `src/pages/NpcsPage.tsx` — list page unaffected.
- `src/App.tsx` routes — unchanged (the three existing `/npcs/:id[/shop|/conversations]` routes all remain).

### `atlas-data`

Route table changes in `services/atlas-data/atlas.com/data/npc/resource.go:20-30`:

- Remove `r.HandleFunc("/{npcId}/map", …)` and delete `handleGetNpcMapRequest` (§5.3).
- Add `r.HandleFunc("/{npcId}/maps", registerGet("get_npc_maps", handleGetNpcMapsRequest(db))).Methods(http.MethodGet)` — new handler. Queries `npc_spawn_index` for all rows matching `(tenant, npc)` ordered by `spawn_count DESC, map_id ASC`, returns `[]NpcMapRestModel` via `server.MarshalResponse[[]NpcMapRestModel]`. Reuses the existing `NpcMapRestModel` type — the singular-vs-plural difference is only in the handler's marshal call, not the model.
- Add `r.HandleFunc("/{npcId}/quests", registerGet("get_npc_quests", handleGetNpcQuestsRequest(db))).Methods(http.MethodGet)` — new handler. Loads the tenant's quest documents via `quest.NewStorage(d.Logger(), db).GetAll(d.Context())`, filters in memory by the four npcId fields (§5.2), returns `[]quest.RestModel`.

Both new handlers follow the existing `ParseNPC` / `server.MarshalResponse` / JSON:API pattern used by the surrounding file — no new middleware.

No migration, no event emission, no Kafka topic changes.

### Other services

- `atlas-query-aggregator` — no changes. The new endpoints are under `/api/data/...` and reach `atlas-data` via the existing nginx rule at `deploy/shared/routes.conf:196-197`.
- `atlas-quest`, `atlas-npc-shops`, `atlas-npc-conversations` — no changes.
- Nginx routes — no changes.

## 8. Non-Functional Requirements

### Performance

- `/data/npcs/:npcId/maps` — indexed single-NPC read from `npc_spawn_index`. Expected < 10 ms p95.
- `/data/npcs/:npcId/quests` — in-memory full-scan over tenant quest docs. Target: p95 < 50 ms for a tenant with ~2000 quests. Measure in staging before rollout. If p95 > 200 ms, escalate to a persistent `quest_npc_index` — tracked as a follow-up, not in this task's DoD.
- UI page render — the four data queries (`useNpcData`, `useNpcSpawnMaps`, `useNpcQuests`, shop + conversation summaries) fire in parallel from `NpcDetailPage`. No request waterfalls.

### Multi-tenancy

- All new endpoints parse the tenant from context via the existing `tenant.FromContext(d.Context())` pattern (§5). No cross-tenant reads.
- UI query keys include `activeTenant.id` — `TenantProvider` already calls `queryClient.clear()` on tenant switch, so no stale-tenant data leaks.

### Observability

- Both new handlers emit a `Debugf` log line with `tenant_id`, `npc_id`, result count, and elapsed ms (mirroring the `NPC search served.` log at `services/atlas-data/atlas.com/data/npc/resource.go:129-137`).
- No new metrics or trace spans beyond the existing `server.RetrieveSpan` wrapper — this is consistent with task-010's monster-spawn endpoint.

### Security

- Admin-facing UI only; tenant header gating is the only auth check, matching existing atlas-data endpoints.
- No user input reaches SQL beyond the path-parsed `npcId` (integer) and tenant UUID — both go through GORM placeholders.

### Accessibility

- Header tooltip trigger is keyboard-focusable (`tabIndex={0}`, `focus-visible:ring-2`) — copied from `MonsterHeader`.
- Widget grids use `<Link>` elements so they are keyboard- and screen-reader-navigable.
- Empty/loading states are text (not icons-only) so they are readable by screen readers.

## 9. Open Questions

- Does the tenant ingest already populate `npc_spawn_index` rows, or do some tenants still need the task-010-era re-ingest? (Operational check at rollout — not a blocker for the task itself.)
- If a quest names the same NPC as both initiator and completer in ~5% of content (confirmed pattern in GMS quests), the `both` role badge will be common. Is "Initiator & Completer" acceptable, or should the badge read "Two-way" / "Self" / something shorter? Default to "Initiator & Completer" unless flagged.
- Should the spawn-maps card limit to the top-N (e.g., 12) with a "show all" disclosure for NPCs like Gachapon that appear on 20+ maps? Default to no limit — if a specific NPC proves unwieldy in practice, add a `limit=` query and a UI "show more" later.
- The `endRequirements.npcId` field is parsed but almost never populated in the GMS quest corpus. Include it in the server-side filter (§5.2)? Current spec: yes — cheap, defensive, and keeps the semantics "any quest that references this NPC."

## 10. Acceptance Criteria

### Backend — `atlas-data`

- [ ] `GET /api/data/npcs/:npcId/maps` returns all `npc_spawn_index` rows for the active tenant and NPC, sorted by `spawn_count DESC, map_id ASC`.
- [ ] `GET /api/data/npcs/:npcId/maps` returns `200 { "data": [] }` when the NPC is present in the registry but has no spawn rows.
- [ ] `GET /api/data/npcs/:npcId/maps` returns `400` for non-integer / ≤ 0 `npcId`.
- [ ] `GET /api/data/npcs/:npcId/map` (singular) returns `404` — route is removed.
- [ ] `handleGetNpcMapRequest` is deleted and its `resource_test.go` cases are removed; tests covering the singular endpoint are replaced with the plural equivalent.
- [ ] `GET /api/data/npcs/:npcId/quests` returns every quest where `startRequirements.npcId` or `endRequirements.npcId` or `startActions.npcId` or `endActions.npcId` equals the path `npcId`, for the active tenant.
- [ ] `GET /api/data/npcs/:npcId/quests` returns `200 { "data": [] }` when the NPC participates in no quest.
- [ ] `GET /api/data/npcs/:npcId/quests` payload per-item shape is byte-identical to `GET /api/data/quests/:id` (same `quest.RestModel`).
- [ ] Both handlers log a `Debugf` line with `tenant_id`, `npc_id`, `result_ct`, `elapsed_ms`.
- [ ] Unit tests: `resource_test.go` covers happy path, empty result, missing tenant, invalid npcId for both new endpoints.
- [ ] No new GORM migration, no new Kafka topic, no new storage type registered.

### Frontend — `atlas-ui`

- [ ] Navigating to `/npcs/:id` renders the new layout: header → Spawn Locations → Shop + Conversation grid → Quests.
- [ ] The NPC header shows a 64×64 icon, the NPC name, and a tooltip on hover/focus; clicking the tooltip copies the template id to the clipboard.
- [ ] The refresh button is absent. The "ID: #" line is absent. The old `max-w-md` single-card layout is absent.
- [ ] Spawn Locations renders a widget grid when `npc_spawn_index` returns rows; renders the empty-state copy when it returns `[]`.
- [ ] Each spawn widget links to `/maps/:mapId` and shows `name` + `streetName` + spawn count.
- [ ] The Shop card shows `Recharger`, commodity count, and token-priced commodity count when a shop exists; shows an empty-state with a `Create Shop` link-button when it does not.
- [ ] The Shop card's `Edit Shop` / `Create Shop` button navigates to `/npcs/:id/shop`.
- [ ] The Conversation card shows state count, start state, and an entry-dialogue preview when a conversation exists; shows an empty-state with `Create Conversation` otherwise.
- [ ] The Conversation card's `Edit Conversation` / `Create Conversation` button navigates to `/npcs/:id/conversations`.
- [ ] The Quests card renders one widget per quest returned by the new endpoint; each widget shows the quest name, `parent` badge when present, and a role badge of `Initiator` / `Completer` / `Initiator & Completer`.
- [ ] Clicking a quest widget navigates to `/quests/:id`.
- [ ] Quests are sorted: Initiator first, Both second, Completer third; ties broken by name ASC then id ASC.
- [ ] Loading states: each of Spawn / Shop / Conversation / Quests shows its own spinner/skeleton independently; the page does not block on a single slow query.
- [ ] Error states: each card renders an inline `ErrorDisplay` with a retry that re-runs only that card's query.
- [ ] Tenant switch clears React Query cache (existing behaviour) and the page re-fetches cleanly.
- [ ] `/npcs/:id/shop` and `/npcs/:id/conversations` routes are unchanged and reachable from the Edit buttons; the old pages render exactly as today.
- [ ] `ItemNpcShopWidget` is migrated to `useNpcSpawnMaps(npcId)` (plural); when the NPC has multiple spawn maps, the widget shows the primary map badge plus a `+N` counter.
- [ ] `useNpcSpawnMap.ts` (singular hook) is deleted and has no remaining importers.
- [ ] `npcsService.getSpawnMap` is removed and has no remaining callers.
- [ ] `npm run lint` passes.
- [ ] `npm run test` passes, including new tests covering: role derivation for initiator-only / completer-only / both quests, empty states, header tooltip copy, and render-under-each-load-state.
- [ ] `npm run build` succeeds.

### Cross-cutting

- [ ] Docker build of `atlas-data` succeeds.
- [ ] Docker build of `atlas-ui` succeeds.
- [ ] Compose `docker-compose.core.yml` brings up both services and the page renders against a seeded tenant with a known NPC (e.g., Cassandra in Henesys).
- [ ] No console errors or React key warnings when the page renders with ≥10 quests and ≥3 spawn maps.
