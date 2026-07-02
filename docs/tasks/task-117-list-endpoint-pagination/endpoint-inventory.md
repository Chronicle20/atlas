# Unbounded Collection Endpoint Inventory

Supporting artifact for [prd.md](prd.md). Compiled 2026-07-02 by grepping `MarshalResponse[[]…]` across `services/*/atlas.com/**/*.go` (86 slice-marshaling handler sites, excluding tests), mapping each to its route registration, tracing the backing fetch, and checking consumers of the truly-unfiltered forms in `services/atlas-ui/src/services/api` and other services' `requests.go`.

Cardinality: **HIGH** = grows unbounded with players/activity; **MEDIUM** = grows with content/config; **LOW** = naturally tiny.

Backing: **full-table** = GORM `db.Find` with no `Where`/`Limit`; **filtered** = required param bounds it in practice but still no `Limit`; **registry** = Redis `TenantRegistry`/in-memory runtime dump; **doc-store** = `document.NewStorage` full dump.

## Group A — bare full-table dumps (HIGH; PRD FR-3)

| Service | Route | Backing query | Unfiltered-list consumers |
|---|---|---|---|
| atlas-character | `GET /characters` | full-table (`getAll()` → `db.Find`, `character/provider.go:40-43`) | UI `charactersService.getAll()`. No Go caller (inter-service uses `?name=`, `?accountId&worldId`, `/{id}`). |
| atlas-account | `GET /accounts` | full-table (`provider.go:35`) | UI `accountsService.getAllAccounts()`; **atlas-channel** `account.AllProvider` → `InitializeRegistry` (`channel/main.go:386`); **atlas-login** likewise (`login/main.go:267`). Startup logged-in registry seed — the drain-helper use case. |
| atlas-guilds | `GET /guilds` (also `?filter[members.id]=`) | full-table with `Preload("Members").Preload("Titles")` (`guild/provider.go:13`) | UI `guildsService.getAll()` — search/by-member/by-name all dump all guilds then filter **client-side**. No Go caller of the bare form. |
| atlas-ban | `GET /bans/` | full-table (`ban/provider.go:25`) | UI `bansService.getAllBans()`. |
| atlas-ban | `GET /history/` | full-table, `Order("created_at desc")` (`history/provider.go:46`) | None found — real traffic uses `/history/accounts/{accountId}`. |
| atlas-notes | `GET /notes` | full-table (`note/provider.go:31-34`) | None found — likely admin/orphan surface. |
| atlas-merchant | `GET /merchants` | full-table | Unconfirmed; likely the merchant web UI (`legacy-merchant-web-ui`). Verify at implementation. |

## Group B — content/config full dumps (MEDIUM; PRD FR-4)

atlas-data doc-store list routes (bare, no `?search=`): `/data/monsters`, `/npcs`, `/maps`, `/reactors`, `/skills`, `/consumables`, `/etcs`, `/setups`, `/cash`, `/pets`, `/mobskills`, `/quests`, `/quests/auto-start`, `/cosmetics/hairs`, `/cosmetics/faces`, `/character/templates`, `/commodities/items`, plus by-parent variants (`/monsters/{id}/loseItems`, `/monsters/{id}/maps`, `/npcs/{id}/maps`, `/npcs/{id}/quests`, `/maps/{id}/portals`). Fixed-size per game version but can be tens of thousands of documents. The `?search=` variants of monsters/npcs/maps/skills/reactors already paginate via `searchindex.Search` (`MaxLimit=50`); the no-search fallback on the same route is the unbounded path. Consumers: UI data browsers; Go data clients fetch by `/{id}`.

Script/config stores with full-dump lists: atlas-map-actions (`GET /maps/actions`), atlas-reactor-actions (`GET /reactors/actions`), atlas-portal-actions (`GET /portals/scripts`), atlas-npc-conversations (`GET /npcs/conversations`, `GET /quests/conversations`), atlas-gachapons (`GET /gachapons`, `GET /global-items`), atlas-drop-information (`GET /continents/drops`), atlas-party-quests (`GET /party-quests/definitions`).

Reference implementation: `services/atlas-data/atlas.com/data/item/string_resource.go` (paginate.Envelope + `parsePagingParams`, `:107-176`) and `monster/resource.go:74-140`.

## Group C — filtered-but-unbounded (bounded in practice; PRD FR-5)

Per-character/per-account (bounded by game caps, consumed by channel/login/cashshop/pets/asset-expiration):
- atlas-inventory `GET /characters/{id}/inventory/compartments/{cid}/assets`
- atlas-storage `GET /storage/accounts/{id}/assets`
- atlas-buddies `GET /characters/{id}/buddy-list/buddies`
- atlas-skills `GET /characters/{id}/skills`, `/macros`
- atlas-keys `GET /characters/{id}/keys`
- atlas-pets `GET /characters/{id}/pets`
- atlas-cashshop `GET /accounts/{id}/cash-shop/inventory/compartments?type=`, `GET /characters/{id}/cash-shop/wishlist`
- atlas-quest `GET /characters/{id}/quests` (+ `/started`, `/completed`, `/{qid}/progress`)
- atlas-monster-book `GET /characters/{id}/monster-book/cards`
- atlas-character `GET /characters/{id}/sessions` (login log — slow-growing)
- atlas-marriages `GET /characters/{id}/marriage/history`, `/proposals`
- atlas-families `GET /families/tree/{id}`
- atlas-invites `GET /characters/{id}/invites`
- atlas-buffs `GET /characters/{id}/buffs` (registry)
- atlas-ban `GET /history/accounts/{id}`
- atlas-maps `GET /characters/{id}/visits` — **monotonically growing** visit history; the slow-burn HIGH within this group
- atlas-merchant per-character/per-instance forms; `GET /merchants/search/listings`
- atlas-npc-shops `GET /npcs/{id}/shop/characters`, `GET /commodities/items/{id}`; `GET /shops` (content, full-table)

In-field runtime registries (bounded per map by spawn caps; hot-path consumers in atlas-channel):
- atlas-maps `GET .../maps/{mapId}/characters`, `GET .../instances/{iid}/characters`
- atlas-monsters `GET .../instances/{iid}/monsters` (+ `/in-rect`)
- atlas-drops, atlas-reactors, atlas-summons, atlas-doors (also `GET /characters/{id}/doors`), atlas-chairs, atlas-chalkboards — same shape

Correctness rule from the PRD: every internal Go consumer of a converted Group C endpoint moves to `DrainProvider` — game logic must never act on a truncated page.

## Group D — runtime registry dumps (PRD FR-6)

- atlas-parties `GET /parties` (Redis `TenantRegistry.GetAll`; consumers use `?filter[members.id]=`)
- atlas-messengers `GET /messengers` (same pattern)
- atlas-saga-orchestrator `GET /sagas` (in-flight sagas, operational)
- atlas-party-quests `GET /party-quests/instances` (+ by character/field)
- atlas-portals `GET /portals/blocked` (+ `?characterId=`)

Cardinality is active-session concurrency, not stored rows.

## LOW / naturally bounded (~12 endpoints — converted for uniformity, trivial)

atlas-world `GET /worlds`, `GET /worlds/{id}/channels`; atlas-tenants `GET /tenants`, configuration `routes`/`vessels`/`instance-routes`; atlas-configurations `GET /configurations/templates`, `/services`, `/tenants`; atlas-transports `GET /transports/routes`, `/instance-routes`, `/instance-routes/{id}/status`.

## Judgment calls / verification notes

1. atlas-channel/atlas-login unfiltered `GET /accounts` consumption was **confirmed** at `channel/main.go:386` and `login/main.go:267` (startup registry seed).
2. `GET /notes` and bare `GET /history/` have no found consumer — PRD open question 4 (convert vs remove).
3. atlas-merchant `GET /merchants` consumer unconfirmed.
4. Parties/messengers bare dumps: every real consumer uses the filtered form.
5. atlas-data lists are fixed-size per game version but large; classified MEDIUM and in scope.
