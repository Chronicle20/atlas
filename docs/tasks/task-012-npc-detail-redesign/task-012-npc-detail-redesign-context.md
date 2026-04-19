# NPC Detail Redesign — Context Document

**Last Updated: 2026-04-19**

## Authoritative Specification

| Document | Role |
|---|---|
| `docs/tasks/task-012-npc-detail-redesign/prd.md` | Product requirements — scope, goals, functional requirements, API surface, data model, acceptance criteria. The plan defers to this on any disagreement. |

## Key Files — atlas-data

### New handlers (both in `services/atlas-data/atlas.com/data/npc/`)

| File | Purpose |
|---|---|
| `resource.go:20-30` | `InitResource` — register the new `/{npcId}/maps` and `/{npcId}/quests` routes; remove the `/{npcId}/map` registration. |
| `resource.go:152-191` | `handleGetNpcMapRequest` — delete (retired singular endpoint). |
| `resource.go` (new handlers) | `handleGetNpcMapsRequest`, `handleGetNpcQuestsRequest`. Follow surrounding `ParseNPC` / `server.MarshalResponse` / JSON:API pattern. |
| `spawn_map_rest.go:13-14` | `NpcMapRestModel` — reused by the plural handler, **not deleted**. |
| `resource_test.go` | Remove singular-`/map` cases; add plural-`/maps` + quests cases covering happy, empty, bad-id, DB failure. |

### Reused types / storage

| File | Purpose |
|---|---|
| `services/atlas-data/atlas.com/data/npc/spawn_index.go` | `npc_spawn_index` entity + indexes. **No change** — just query it. |
| `services/atlas-data/atlas.com/data/quest/storage.go:11-14` | `document.Storage` over the `QUEST` collection — iterate via `GetAll(ctx)`. |
| `services/atlas-data/atlas.com/data/quest/rest.go:10-29` | `quest.RestModel` — reuse verbatim for the quests-by-npc payload. |
| `services/atlas-quest/atlas.com/quest/data/quest/processor.go:40-62` | `GetAutoStartQuests` — reference pattern for in-memory filter over registered quests. |
| `services/atlas-data/atlas.com/data/rest/handler.go` | `ParseNPC` / `server.MarshalResponse` helpers. |

## Key Files — atlas-ui

### Page target

| File | Purpose |
|---|---|
| `services/atlas-ui/src/pages/NpcDetailPage.tsx:15-189` | The page being rewritten (§4.1–§4.8 of the PRD). Grows from ~190 to ~230–260 lines. |

### New components (all under `src/components/features/npc/`)

- `NpcHeader.tsx` — tooltip-to-copy header. Mirror of `MonsterHeader.tsx:9-57`. No badges (NPCs have no boss/undead/friendly flags).
- `NpcSpawnMapWidget.tsx` — clickable `/maps/:mapId` cell with name, street-name badge, spawn-count badge. Mirror of `MonsterSpawnMapWidget.tsx:1-30`.
- `NpcQuestWidget.tsx` — MVP tile. Quest name + `parent` badge + role badge (`Initiator` / `Completer` / `Initiator & Completer`). Links to `/quests/:id`.

### New hooks (all under `src/lib/hooks/api/`)

| Hook | Query key | Wraps |
|---|---|---|
| `useNpcSpawnMaps(npcId)` | `["data", "npcs", "maps", tenantId, npcId]` | `npcsService.getNpcSpawnMaps` — plural spawn endpoint. |
| `useNpcQuests(npcId)` | `["data", "npcs", "quests", tenantId, npcId]` | `npcsService.getNpcQuests` — reverse-lookup quests endpoint. |
| `useNpcConversation(npcId)` | (follow existing conversation-by-npc key pattern) | `conversationsService.getByNpcId` (`services/atlas-ui/src/services/api/conversations.service.ts:140-156`). |

Both new hooks gate on `enabled: !!activeTenant && npcId > 0`.

### Reused UI hooks / services

- `useNpcData(npcId)` — name + icon.
- `npcsService.getNPCById(npcId)` via the existing `["npcs", "detail", ...]` key — `hasShop` / `hasConversation` flags.
- `npcsService.getNPCShop(npcId)` via the shop page's `["npcs", "shop", tenantId, npcId]` key — shop summary reuse avoids re-fetch on navigation.

### Modified files

| File | Change |
|---|---|
| `src/types/models/npc.ts` | Add `NpcSpawnMapAttributes` (plural) + `NpcQuestRole` (`"initiator" \| "completer" \| "both"`). |
| `src/services/api/npcs.service.ts` | Add `getNpcSpawnMaps(npcId)`, `getNpcQuests(npcId)`. **Remove** `getSpawnMap(npcId)`. |
| `src/components/features/items/ItemNpcShopWidget.tsx:22` | Migrate from `useNpcSpawnMap` (singular) to `useNpcSpawnMaps` (plural). Render primary row + `+{N-1}` overflow badge when `length > 1`. |

### Deletions

| File | Why |
|---|---|
| `src/lib/hooks/api/useNpcSpawnMap.ts` | Singular hook; only caller migrated. Remove from any barrel exports. |
| Singular service method `npcsService.getSpawnMap` | Endpoint retired. |

### Reference implementations (do not modify)

| File | Why it matters |
|---|---|
| `services/atlas-ui/src/pages/MonsterDetailPage.tsx:245-268` | Spawn-locations card layout — copy verbatim for NPC card. |
| `services/atlas-ui/src/components/features/monsters/MonsterHeader.tsx:9-57` | Header tooltip-to-copy pattern. |
| `services/atlas-ui/src/components/features/monsters/MonsterSpawnMapWidget.tsx:1-30` | Spawn widget pattern. |
| `services/atlas-ui/src/pages/MapDetailPage.tsx` / `ItemDetailPage.tsx` | Layout parity targets (task-008, task-011). |

### Unchanged (must stay reachable)

- `src/pages/NpcShopPage.tsx`, `src/pages/NpcConversationPage.tsx` — Edit / Create deep-link targets.
- `src/pages/NpcsPage.tsx`.
- `src/App.tsx` routes — all three `/npcs/:id[/shop|/conversations]` routes remain registered.

## Key Decisions

| Decision | Rationale |
|---|---|
| **In-memory scan for quests-by-npc, no new GORM table.** | Tenant quest count ~O(thousands); page is admin-only. Same tradeoff as `GetAutoStartQuests`. Escalate to a persistent `quest_npc_index` only if p95 > 200 ms. |
| **Retire singular `/map` endpoint now, no grace period.** | Single consumer (`ItemNpcShopWidget`) migrates in the same PR. Endpoint hides the multi-map case for gachapon / town NPCs — keeping it would require tombstoning. |
| **Reuse `quest.RestModel` byte-identically.** | UI already consumes this type via `useQuestData`; keeps the new endpoint a pure reverse-index view, not a transform. |
| **Role derivation on the UI, not the server.** | Role is a presentation concern; server returns a vanilla `[]quest.RestModel`. Mirrors the existing "server returns entities, UI labels them" split. |
| **Summary + deep-link for Shop and Conversation — no inline editing.** | Scope-review option C. Edit flows remain on the existing pages; NPC page is read-only summary. |
| **Drop the refresh button entirely.** | Sibling detail pages (map, monster, item) already dropped it; React Query handles staleness. |
| **No `max-w-md` wrapper.** | Page is full-width like sibling detail pages. |
| **`enabled: !!activeTenant && npcId > 0` on all new hooks.** | Matches existing tenant-gating convention; avoids firing against an un-initialized tenant. |
| **Both endpoints 200-empty on no-data, never 404.** | Callers distinguish "NPC not found" (404 from `/api/data/npcs/:npcId`) from "NPC has no spawns/quests" (`200 { "data": [] }`). |

## Query Key Map (post-change)

| Key | Owner |
|---|---|
| `["data", "npcs", "maps", tenantId, npcId]` | `useNpcSpawnMaps` (new). |
| `["data", "npcs", "quests", tenantId, npcId]` | `useNpcQuests` (new). |
| `["npcs", "detail", tenantId, npcId]` | Existing — `hasShop` / `hasConversation`. |
| `["npcs", "shop", tenantId, npcId]` | Existing — shared with `NpcShopPage` (cache reuse on navigation). |
| `["npcs", "spawn-map", npcId, tenantId]` | **Retired** — ages out; no invalidation hook. |

## Performance Budget

| Endpoint | Target p95 | Escalation |
|---|---|---|
| `GET /data/npcs/:npcId/maps` | < 10 ms (indexed read) | N/A — DB index already covers `(tenant_id, npc_id)`. |
| `GET /data/npcs/:npcId/quests` | < 50 ms at ~2000-quest tenant | If > 200 ms: add persistent `quest_npc_index` — **out of scope** for this task. |
| `NpcDetailPage` initial render | Parallel fan-out (4 queries) — no waterfalls | Each card renders its own skeleton / error — no card blocks another. |

## Observability

Both new handlers emit `Debugf` on completion with: `tenant_id`, `npc_id`, `result_ct`, `elapsed_ms`. Mirrors `NPC search served.` at `services/atlas-data/atlas.com/data/npc/resource.go:129-137`.

## Integration Checklist (Phase 8)

- [ ] `go test ./services/atlas-data/...`
- [ ] `npm run lint` (atlas-ui)
- [ ] `npm run test` (atlas-ui)
- [ ] `npm run build` (atlas-ui)
- [ ] Docker build `atlas-data`
- [ ] Docker build `atlas-ui`
- [ ] `docker-compose -f docker-compose.core.yml up` — manual smoke against a seeded tenant with a known multi-map NPC (Cassandra / gachapon) and an NPC with ≥10 quests.
- [ ] Browser console clean: no React key warnings, no unhandled promise rejections, no 404s from lingering `/npcs/:id/map` callers.

## Related Prior Work

- **task-008 (map detail)** — established the header tooltip-to-copy pattern and widget-grid layout.
- **task-010 (monster detail)** — introduced `*_spawn_index` + `MonsterHeader` + `MonsterSpawnMapWidget`, which this task mirrors.
- **task-011 (item detail)** — confirmed the summary-plus-deep-link treatment used here for Shop / Conversation.
