# Unbounded Collection Endpoint Inventory

Supporting artifact for [prd.md](prd.md). Compiled 2026-07-02 by grepping `MarshalResponse[[]…]` across `services/*/atlas.com/**/*.go` (86 slice-marshaling handler sites, excluding tests), mapping each to its route registration, tracing the backing fetch, and checking consumers of the truly-unfiltered forms in `services/atlas-ui/src/services/api` and other services' `requests.go`.

Cardinality: **HIGH** = grows unbounded with players/activity; **MEDIUM** = grows with content/config; **LOW** = naturally tiny.

Backing: **full-table** = GORM `db.Find` with no `Where`/`Limit`; **filtered** = required param bounds it in practice but still no `Limit`; **registry** = Redis `TenantRegistry`/in-memory runtime dump; **doc-store** = `document.NewStorage` full dump.

## Disposition (task-29 closeout)

Every row in this inventory was converted — the repo-wide `MarshalResponse[[]…]` grep (footnote in the header above) is EMPTY as of task-29's acceptance sweep. "Disposition" columns/notes below record which task did each conversion, per [plan.md](plan.md); see the individual `task-N-report.md` files under `.superpowers/sdd/` for implementation detail. Task 29 additionally found and fixed three internal consumers that were left on the old `requests.SliceProvider` (single-page) client against now-paginated endpoints — see `.superpowers/sdd/task-29-report.md` §grep-3 for the full list of reviewed `SliceProvider` call sites and why the rest are safe by construction (filtered to ≤1 result, or hard game-capped well under the default page size).

## Group A — bare full-table dumps (HIGH; PRD FR-3)

| Service | Route | Backing query | Unfiltered-list consumers | Disposition |
|---|---|---|---|---|
| atlas-character | `GET /characters` | full-table (`getAll()` → `db.Find`, `character/provider.go:40-43`) | UI `charactersService.getAll()`. No Go caller (inter-service uses `?name=`, `?accountId&worldId`, `/{id}`). | **Converted (task-10)** — paginated; `?accountId=&worldId=` and `?name=` variants also paginated. UI updated task-16. |
| atlas-account | `GET /accounts` | full-table (`provider.go:35`) | UI `accountsService.getAllAccounts()`; **atlas-channel** `account.AllProvider` → `InitializeRegistry` (`channel/main.go:386`); **atlas-login** likewise (`login/main.go:267`). Startup logged-in registry seed — the drain-helper use case. | **Converted (task-9)** — paginated, unfiltered `GetByTenant`/`ByTenantProvider` deleted, `AllProvider(page, decorators...)` added. atlas-login drain: task-7. atlas-channel drain: task-8. UI: task-16. |
| atlas-guilds | `GET /guilds` (also `?filter[members.id]=`) | full-table with `Preload("Members").Preload("Titles")` (`guild/provider.go:13`) | UI `guildsService.getAll()` — search/by-member/by-name all dump all guilds then filter **client-side**. No Go caller of the bare form. | **Converted (task-11)** — paginated + server-side `filter[name]` (case-insensitive substring). UI client-side-filter-then-dump deleted, moved to `filter[name]`: task-17. |
| atlas-ban | `GET /bans/` | full-table (`ban/provider.go:25`) | UI `bansService.getAllBans()`. | **Converted (task-12)**. UI: task-16. |
| atlas-ban | `GET /history/` | full-table, `Order("created_at desc")` (`history/provider.go:46`) | None found — real traffic uses `/history/accounts/{accountId}`. | **Converted (task-12)** — convert-don't-remove per PRD open question 4; flagged as a consumer-less removal candidate in [docs/rest-pagination.md](../../rest-pagination.md) §8. |
| atlas-notes | `GET /notes` | full-table (`note/provider.go:31-34`) | None found — likely admin/orphan surface. | **Converted (task-13)** — same convert-don't-remove disposition; flagged in [docs/rest-pagination.md](../../rest-pagination.md) §8. |
| atlas-merchant | `GET /merchants` | full-table | Unconfirmed; likely the merchant web UI (`legacy-merchant-web-ui`). Verify at implementation. | **Converted (task-14)** — external `legacy-merchant-web-ui` consumer not found in-repo; compat rule (no-envelope = complete collection) makes this deploy-safe regardless. UI: task-16. |

## Group B — content/config full dumps (MEDIUM; PRD FR-4)

atlas-data doc-store list routes (bare, no `?search=`): `/data/monsters`, `/npcs`, `/maps`, `/reactors`, `/skills`, `/consumables`, `/etcs`, `/setups`, `/cash`, `/pets`, `/mobskills`, `/quests`, `/quests/auto-start`, `/cosmetics/hairs`, `/cosmetics/faces`, `/character/templates`, `/commodities/items`, plus by-parent variants (`/monsters/{id}/loseItems`, `/monsters/{id}/maps`, `/npcs/{id}/maps`, `/npcs/{id}/quests`, `/maps/{id}/portals`). Fixed-size per game version but can be tens of thousands of documents. The `?search=` variants of monsters/npcs/maps/skills/reactors already paginate via `searchindex.Search` (`MaxLimit=50`); the no-search fallback on the same route is the unbounded path. Consumers: UI data browsers; Go data clients fetch by `/{id}`.

**Disposition: Converted (task-18, task-19)** — all core + remaining doc-store list routes paginated via `Storage.AllPagedProvider`; unpaged `Storage.GetAll` deleted; `?search=` and no-`?search=` arms share the identical envelope. UI data browsers: task-22.

Script/config stores with full-dump lists: atlas-map-actions (`GET /maps/actions`), atlas-reactor-actions (`GET /reactors/actions`), atlas-portal-actions (`GET /portals/scripts`), atlas-npc-conversations (`GET /npcs/conversations`, `GET /quests/conversations`), atlas-gachapons (`GET /gachapons`, `GET /global-items`), atlas-drop-information (`GET /continents/drops`), atlas-party-quests (`GET /party-quests/definitions`).

**Disposition: Converted** — atlas-map-actions/atlas-reactor-actions/atlas-portal-actions: **task-20**. atlas-npc-conversations/atlas-gachapons/atlas-drop-information/atlas-party-quests (definitions): **task-21**. atlas-drop-information's `/continents/drops` uses the materialize+`paginate.Slice` adapter (computed continent aggregation, no natural DB order) — documented as a per-endpoint override in [docs/rest-pagination.md](../../rest-pagination.md) §3.

Reference implementation: `services/atlas-data/atlas.com/data/item/string_resource.go` (paginate.Envelope + `parsePagingParams`, `:107-176`) and `monster/resource.go:74-140`. **Disposition:** `parsePagingParams` deleted, item-strings handler refactored onto `paginate.ParseParams` in **task-5**.

## Group C — filtered-but-unbounded (bounded in practice; PRD FR-5)

Per-character/per-account (bounded by game caps, consumed by channel/login/cashshop/pets/asset-expiration):
- atlas-inventory `GET /characters/{id}/inventory/compartments/{cid}/assets` — **Converted (task-23)**
- atlas-storage `GET /storage/accounts/{id}/assets` — **Converted (task-23)**
- atlas-buddies `GET /characters/{id}/buddy-list/buddies` — **Converted (task-23)**
- atlas-skills `GET /characters/{id}/skills`, `/macros` — **Converted (task-23)**. Downstream `atlas-query-aggregator` consumer of `/skills` was found still on `requests.SliceProvider` during the task-29 sweep and converted to `DrainProvider` (task-29, not a task-23 gap — task-23 owned the server side only).
- atlas-keys `GET /characters/{id}/keys` — **Converted (task-23)** — composite-PK entity, uses materialize + `paginate.Slice` per [docs/rest-pagination.md](../../rest-pagination.md) §3.
- atlas-pets `GET /characters/{id}/pets` — **Converted (task-24)**
- atlas-cashshop `GET /accounts/{id}/cash-shop/inventory/compartments?type=`, `GET /characters/{id}/cash-shop/wishlist` — **Converted (task-24)**
- atlas-quest `GET /characters/{id}/quests` (+ `/started`, `/completed`, `/{qid}/progress`) — **Converted (task-24)**. Downstream `atlas-query-aggregator` consumers of `/started` and `/completed` were found still on `requests.SliceProvider` during the task-29 sweep and converted to `DrainProvider` (task-29, not a task-24 gap — task-24 owned the server side only). `/{qid}/progress` uses materialize + `paginate.Slice` (progress rows sorted by `Id()`, no stable GORM preload order).
- atlas-monster-book `GET /characters/{id}/monster-book/cards` — **Converted (task-24)** — composite-PK, materialize + `paginate.Slice`.
- atlas-character `GET /characters/{id}/sessions` (login log — slow-growing) — **Converted (task-10)**
- atlas-marriages `GET /characters/{id}/marriage/history`, `/proposals` — **Converted (task-25)**
- atlas-families `GET /families/tree/{id}` — **Converted (task-25)** — graph traversal, materialize + `paginate.Slice`.
- atlas-invites `GET /characters/{id}/invites` — **Converted (task-25)**
- atlas-buffs `GET /characters/{id}/buffs` (registry) — **Converted (task-25)**
- atlas-ban `GET /history/accounts/{id}` — **Converted (task-12)**
- atlas-maps `GET /characters/{id}/visits` — **monotonically growing** visit history; the slow-burn HIGH within this group — **Converted (task-26)**
- atlas-merchant per-character/per-instance forms; `GET /merchants/search/listings` — **Converted (task-14)**
- atlas-npc-shops `GET /npcs/{id}/shop/characters`, `GET /commodities/items/{id}`; `GET /shops` (content, full-table) — **Converted (task-25)**

In-field runtime registries (bounded per map by spawn caps; hot-path consumers in atlas-channel):
- atlas-maps `GET .../maps/{mapId}/characters`, `GET .../instances/{iid}/characters` — **Converted (task-26)**
- atlas-monsters `GET .../instances/{iid}/monsters` (+ `/in-rect`) — **Converted (task-26)**
- atlas-drops, atlas-reactors, atlas-summons, atlas-doors (also `GET /characters/{id}/doors`), atlas-chairs, atlas-chalkboards — same shape — **Converted (task-26)**

Correctness rule from the PRD: every internal Go consumer of a converted Group C endpoint moves to `DrainProvider` — game logic must never act on a truncated page. **Verified by task-29's acceptance sweep (grep 3):** 32 `requests.SliceProvider` call sites reviewed repo-wide; 3 genuine gaps found and fixed in task-29 (atlas-login character-select `?accountId=&worldId=`, atlas-query-aggregator `/skills` and `/quests/{started,completed}`); the remainder target endpoints that are inherently bounded to ≤1 result or a small hard game cap (party ≤6, messenger ≤4) regardless of page size, so `SliceProvider` remains correct for them — see [docs/rest-pagination.md](../../rest-pagination.md) §7.

## Group D — runtime registry dumps (PRD FR-6)

- atlas-parties `GET /parties` (Redis `TenantRegistry.GetAll`; consumers use `?filter[members.id]=`) — **Converted (task-27)** — materialize + `paginate.Slice`.
- atlas-messengers `GET /messengers` (same pattern) — **Converted (task-27)**
- atlas-saga-orchestrator `GET /sagas` (in-flight sagas, operational) — **Converted (task-27)**
- atlas-party-quests `GET /party-quests/instances` (+ by character/field) — **Converted (task-27)**
- atlas-portals `GET /portals/blocked` (+ `?characterId=`) — **Converted (task-27)**

Cardinality is active-session concurrency, not stored rows.

## LOW / naturally bounded (~12 endpoints — converted for uniformity, trivial)

atlas-world `GET /worlds`, `GET /worlds/{id}/channels`; atlas-tenants `GET /tenants`, configuration `routes`/`vessels`/`instance-routes`; atlas-configurations `GET /configurations/templates`, `/services`, `/tenants`; atlas-transports `GET /transports/routes`, `/instance-routes`, `/instance-routes/{id}/status`.

**Disposition: Converted (task-28)** — all four services' list routes paginated via materialize + `paginate.Slice` for uniformity (LOW cardinality made the DB/registry distinction moot).

## Judgment calls / verification notes

1. atlas-channel/atlas-login unfiltered `GET /accounts` consumption was **confirmed** at `channel/main.go:386` and `login/main.go:267` (startup registry seed).
2. `GET /notes` and bare `GET /history/` have no found consumer — PRD open question 4 (convert vs remove). **Resolved:** converted (see Group A disposition above); flagged as consumer-less removal candidates in [docs/rest-pagination.md](../../rest-pagination.md) §8.
3. atlas-merchant `GET /merchants` consumer unconfirmed. **Resolved:** no in-repo external consumer found at implementation (task-14); converted regardless, compat rule covers any out-of-repo caller.
4. Parties/messengers bare dumps: every real consumer uses the filtered form.
5. atlas-data lists are fixed-size per game version but large; classified MEDIUM and in scope.
6. **task-29 addendum:** the acceptance sweep (repo-wide `MarshalResponse[[]…]`, unfiltered `GetAll`, and `requests.SliceProvider` greps) is clean; the three genuine `SliceProvider` gaps it found (atlas-login, atlas-query-aggregator ×2) are documented in `.superpowers/sdd/task-29-report.md` and fixed on this branch rather than deferred.
