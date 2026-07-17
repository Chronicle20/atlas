# List Endpoint Pagination — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Atlas has 86 REST handler sites that marshal a slice into a JSON:API document, and effectively none of them paginate. A pagination envelope already exists in `libs/atlas-rest/server/paginate/` (meta `total`/`page`, links `self/first/prev/next/last`) together with `server.MarshalPaginatedResponse`, but it is used by exactly one code path: the atlas-data item string search. Everything else — including `GET /characters` and `GET /accounts`, which scan an entire tenant table, transform every row, run every `include` decorator per row, and build the whole JSON:API document in memory — is unbounded (architectural-improvements finding PS-5, High).

This task establishes pagination as the repo-wide convention for collection GET endpoints and applies it to **all** of them. The lib layer gains a functional paged-provider pipeline (`model.Paged[T]` flowing through the existing `model.Provider` composition) so pagination is a value passed through the provider chain, not a duplicated method family — unfiltered `GetAll`-style processor methods are removed, not shadowed. The client layer gains paged and drain helpers in `libs/atlas-rest/requests` so internal consumers that legitimately need a complete collection (e.g. atlas-login/atlas-channel seeding the logged-in registry from `GET /accounts` at startup) keep their semantics while the server never builds an unbounded response.

The full endpoint census, cardinality classification, and consumer trace that ground this PRD are in [endpoint-inventory.md](endpoint-inventory.md).

## 2. Goals

Primary goals:
- One shared, documented pagination convention (JSON:API `page[number]`/`page[size]`, envelope meta + links) applied to every collection GET endpoint in the Go services.
- Requests without `page[*]` params default to page 1 at the endpoint's default size (decision: option (a) — safe-by-default for the server; all known consumers are audited and updated in this task).
- No unbounded `SELECT *` + full-collection marshal remains on any list route; DB-backed lists page at the query layer (`LIMIT/OFFSET` + `COUNT`), not the marshal layer.
- A single functional list pipeline per processor: `AllProvider(page, decorators...) model.Provider[model.Paged[Model]]`; legacy unfiltered `GetAll` methods deleted.
- Client-side helpers (`PagedProvider`, `DrainProvider`) so Go consumers and atlas-ui can fetch one page or drain a collection page-by-page.
- atlas-ui list views work against the paginated endpoints (server-side paging, and server-side filtering where the UI currently filters a full dump client-side — notably guilds).

Non-goals:
- Cursor/keyset pagination. Offset/limit only, matching the existing envelope. (Keyset can be layered under the same envelope later without changing the wire contract.)
- Automated enforcement tooling (a rediskeyguard-style analyzer). Decision: documented convention only.
- Caching, rate limiting, or the PS-1/PS-2 hot-path REST fan-out problems.
- Changing single-resource (`/{id}`) endpoints or POST/PATCH/DELETE semantics.
- Pagination for Kafka, gRPC, or any non-REST surface.

## 3. User Stories

- As a platform operator, I want list endpoints to have bounded response sizes so that a tenant with 100k characters cannot OOM a service or saturate nginx with one GET.
- As a UI user, I want character/account/guild/ban browsers to load a page at a time so that admin views stay responsive at production data volumes.
- As a UI user, I want guild search to filter on the server so that searching does not require downloading every guild first.
- As a service developer, I want one obvious lib-backed way to write a list endpoint so that new endpoints are paginated by construction.
- As a service developer, I want a drain helper so that legitimate "process the whole collection" consumers (registry seeding, seeders, batch jobs) work without the server ever building the full response in one document.
- As an internal service (atlas-login/atlas-channel), I want `InitializeRegistry` to keep meaning "all accounts" so that logged-in state remains correct after the accounts endpoint is capped.

## 4. Functional Requirements

### FR-1 — Shared library: paged provider pipeline

1. `libs/atlas-model/model` gains:
   - `type Page struct { Number, Size int }`
   - `type Paged[T any] struct { Items []T; Total int; Page Page }` — `Total` is the count of rows matching the scope pre-paging.
   - `MapPaged[E, M any](f func(E) (M, error)) func(Provider[Paged[E]]) Provider[Paged[M]]` — lifts an item transform over the paged container preserving `Total`/`Page`. A decorator-lifting equivalent must compose the same way `model.SliceMap` + `model.Decorate` do today (parallel map over `Items` permitted).
2. `libs/atlas-database` gains `PagedQuery[E any](db *gorm.DB, page model.Page) model.Provider[model.Paged[E]]` — runs `COUNT(*)` and `Offset/Limit Find` against the same scoped `*gorm.DB` (so tenant scoping and `Where` clauses apply identically to both queries). A stable default ordering must be applied (primary key) when the caller supplies none, so pages are deterministic.
3. `libs/atlas-rest/server/paginate` gains `ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error)` — hoisted from atlas-data's private `parsePagingParams` (`services/atlas-data/atlas.com/data/item/string_resource.go:154-176`). Defaults `page[number]=1`, `page[size]=defaultSize`; invalid or out-of-range values are a `400` error (not silent clamping); `page[size]` capped at `maxSize`. atlas-data is refactored onto the hoisted helper.
4. `libs/atlas-rest/requests` gains:
   - `PagedProvider` — fetch one page, returning `model.Paged[M]` (items + envelope meta decoded from `meta.total`/`meta.page`).
   - `DrainProvider` — fetch page 1 and follow `links.next` (or iterate `page[number]` until `meta.page.last`) accumulating all items; returns `model.Provider[[]M]`. **Compatibility rule:** if the response carries no pagination envelope (old server), the single response is treated as the complete collection. This makes consumer-first deployment safe.
5. Non-DB list sources get equivalent paging adapters:
   - Redis/in-memory registries: page an already-materialized slice (`paginate.Slice(items, page) model.Paged[T]` or similar) — the read is cheap; the win is bounding the marshal/response.
   - atlas-data document storage (`document.NewStorage.GetAll`): paged variant of the storage list.

### FR-2 — Convention documentation

1. A `docs/` page (e.g. `docs/rest-pagination.md`, linked from developer guidelines / the backend-dev-guidelines skill source if applicable) specifying: param names, defaults (default size **50**, max size **250** unless a per-endpoint override is documented), envelope shape, error semantics, the `AllProvider(page, ...)` processor pattern, the requirement that DB-backed lists page in SQL, and the consumer rules (UI uses paged fetches; Go internal consumers needing everything use `DrainProvider`).
2. `docs/architectural-improvements.md` PS-5 marked resolved (✓) referencing this task.

### FR-3 — Group A: bare full-table dumps (highest priority)

Convert end-to-end (provider → processor → resource), delete the unfiltered `GetAll` processor methods, and update every consumer:

| Endpoint | Service | Consumer updates required |
|---|---|---|
| `GET /characters` | atlas-character | atlas-ui `charactersService` |
| `GET /accounts` | atlas-account | atlas-ui `accountsService`; atlas-login + atlas-channel `account.InitializeRegistry` → `DrainProvider` (`services/atlas-channel/atlas.com/channel/main.go:386`, `services/atlas-login/atlas.com/login/main.go:267`) |
| `GET /guilds` | atlas-guilds | atlas-ui `guildsService` (see FR-7); add server-side `?filter[name]` (substring/prefix match) so UI search survives pagination |
| `GET /bans/` | atlas-ban | atlas-ui `bansService` |
| `GET /history/` | atlas-ban | none found (bare form) — convert for convention |
| `GET /notes` | atlas-notes | none found — convert for convention |
| `GET /merchants` | atlas-merchant | consumer unconfirmed (likely merchant web UI) — verify during implementation and update if present |

`include`-driven decorators must run only over the returned page's rows.

### FR-4 — Group B: atlas-data document-store dumps

1. All bare (no-`search`) list routes in atlas-data (`/data/monsters`, `/npcs`, `/maps`, `/reactors`, `/skills`, `/consumables`, `/etcs`, `/setups`, `/cash`, `/pets`, `/mobskills`, `/quests`, `/quests/auto-start`, cosmetics, templates, commodities, and the by-parent variants) adopt the same envelope via the paged storage list.
2. The `?search=` search-index paths already paginate; they are normalized onto the hoisted `ParseParams` and the shared envelope so both paths of a route present one contract.
3. Go data clients that consume bare lists (if any beyond by-id fetches — verify per endpoint) move to `DrainProvider`; UI data browsers move to paged fetches.
4. Same treatment for the script/config stores in atlas-map-actions, atlas-reactor-actions, atlas-portal-actions, atlas-npc-conversations, atlas-gachapons, atlas-drop-information, atlas-party-quests (definitions).

### FR-5 — Group C: filtered-but-unbounded lists

Per-character / per-account / per-field lists (inventory compartment assets, storage assets, buddy list, skills, macros, keys, pets, quests, monster-book cards, wishlist, sessions, marriage history, invites, buffs, visits, and the in-map registry lists — monsters, drops, reactors, summons, doors, chairs, chalkboards, map characters). These are bounded in practice by game mechanics, so:

1. They adopt the same envelope and `ParseParams`, with a generous default size chosen per endpoint (documented in the convention page) such that the common case fits one page.
2. **Every internal Go consumer of a converted endpoint switches to `DrainProvider`** (semantic "all") — game logic must never silently operate on a truncated page. This is a hard correctness requirement; hot game paths (channel fetching in-map monsters, buddy lists, skills, quests) keep exact semantics.
3. `GET /characters/{id}/visits` (monotonically growing) uses the standard 50/250 defaults.

### FR-6 — Group D: runtime registry dumps

`GET /parties`, `GET /messengers` (bare forms), `GET /sagas`, `GET /party-quests/instances`, `GET /portals/blocked`: paginate the materialized registry slice per FR-1.5. Filtered forms (`?filter[members.id]=`) keep their current shape (bounded per member) but also accept `page[*]`.

### FR-7 — atlas-ui adoption

1. A shared paged-fetch utility in `services/atlas-ui/src/services/api` that sends `page[number]`/`page[size]`, decodes `meta.total`/`meta.page`, and exposes a drain variant (same no-envelope compatibility rule as Go's `DrainProvider`).
2. Characters, accounts, bans (and merchants if applicable) list views: server-side paging with total-count-driven pager UI.
3. Guilds: search/by-name moves to the new server-side `?filter[name]`; by-member keeps `?filter[members.id]`; browse view pages. The fetch-all-then-filter-client-side pattern in `guildsService` is removed.
4. atlas-data browser views (monsters/npcs/maps/items/etc.) page server-side.
5. TS type updates for the envelope; test call sites updated in the same commits (atlas-ui `npm run build` type-checks tests).

### FR-8 — Rollout / compatibility order

1. `DrainProvider`/paged clients tolerate un-paginated responses (FR-1.4), so consumers can deploy before or with servers. Within the repo the change lands as one branch; the requirement is that no intermediate commit leaves a consumer reading a silently truncated collection.
2. Endpoint conversion and its consumers' updates land in the same commit/phase per service.

## 5. API Surface

Query params (all collection GETs):
- `page[number]` — 1-based, default 1. `page[number] < 1` or non-numeric → `400`.
- `page[size]` — default 50, max 250 (per-endpoint overrides documented). `<1`, `>max`, or non-numeric → `400`.

Response document (unchanged `data` array, plus):
```json
{
  "data": [ ... ],
  "meta": { "total": 1234, "page": { "number": 2, "size": 50, "last": 25 } },
  "links": {
    "self":  ".../characters?page%5Bnumber%5D=2&page%5Bsize%5D=50",
    "first": ".../characters?page%5Bnumber%5D=1&page%5Bsize%5D=50",
    "prev":  ".../characters?page%5Bnumber%5D=1&page%5Bsize%5D=50",
    "next":  ".../characters?page%5Bnumber%5D=3&page%5Bsize%5D=50",
    "last":  ".../characters?page%5Bnumber%5D=25&page%5Bsize%5D=50"
  }
}
```
Past-end pages return an empty `data` array with recovery links (existing `paginate.Envelope` behavior). Other query params (`include`, filters, sparse fieldsets) are preserved verbatim in links (existing `rewritePage` behavior).

New filter: atlas-guilds `GET /guilds?filter[name]=<substring>` — case-insensitive match, composable with `page[*]`.

Error cases: invalid paging params → `400` with a JSON:API error object; everything else unchanged.

## 6. Data Model

No schema changes. Query-layer changes only:
- Each DB-backed list adds a `COUNT(*)` under the same tenant-scoped, filtered `*gorm.DB` plus `ORDER BY <pk>` (or existing explicit ordering, e.g. ban history's `created_at desc`) with `LIMIT/OFFSET`.
- Ordering must be total (tie-break on PK when ordering on a non-unique column) so pages don't overlap or skip.

## 7. Service Impact

- **libs/atlas-model** — `Page`, `Paged`, `MapPaged` (+ tests).
- **libs/atlas-database** — `PagedQuery` (+ tests; must respect the tenant-filter callback).
- **libs/atlas-rest** — `paginate.ParseParams`, slice-paging adapter, `requests.PagedProvider`/`DrainProvider` (+ httptest-backed tests incl. the no-envelope compatibility case and JSON:API relationship-stub gotchas per `libs/atlas-rest/CLAUDE.md`).
- **Group A services** — atlas-character, atlas-account, atlas-guilds (incl. `filter[name]`), atlas-ban, atlas-notes, atlas-merchant: provider/processor/resource conversion, `GetAll` removal.
- **Consumers of Group A** — atlas-login, atlas-channel (`account.InitializeRegistry` → drain; their `account.Processor` interfaces updated).
- **Group B services** — atlas-data (largest surface; ~25 routes + searchindex normalization), atlas-map-actions, atlas-reactor-actions, atlas-portal-actions, atlas-npc-conversations, atlas-gachapons, atlas-drop-information, atlas-party-quests.
- **Group C services + their consumers** — atlas-inventory, atlas-storage, atlas-buddies, atlas-skills, atlas-keys, atlas-pets, atlas-cashshop, atlas-quest, atlas-monster-book, atlas-marriages, atlas-families, atlas-invites, atlas-buffs, atlas-maps, atlas-monsters, atlas-drops, atlas-reactors, atlas-summons, atlas-doors, atlas-chairs, atlas-chalkboards; consumer-side drain adoption chiefly in atlas-channel, atlas-login, atlas-cashshop, atlas-asset-expiration, atlas-pets.
- **Group D services** — atlas-parties, atlas-messengers, atlas-saga-orchestrator, atlas-party-quests (instances), atlas-portals.
- **atlas-ui** — shared paged fetch/drain utility; characters/accounts/guilds/bans/data-browser views; TS envelope types.

Given the breadth, the design/plan phases should sequence this as: lib layer → Group A + consumers + UI → Group B → Group C/D sweeps (mechanical, per-service). Every touched service goes through the full verification gauntlet (test/vet/build/bake) per CLAUDE.md.

## 8. Non-Functional Requirements

- **Multi-tenancy:** `PagedQuery`'s count and page queries run on the same scoped `*gorm.DB`, inheriting the tenant-filter callback. A test must prove count and items agree under tenant scoping.
- **Performance:** the added `COUNT(*)` per list request is acceptable at current volumes; page fetch must not load unpaged rows into Go memory (`LIMIT/OFFSET` in SQL, verified in tests). `include` decorators run per page, not per table.
- **Determinism:** stable total ordering per FR-6/§6; repeated drains of a quiescent collection yield identical multisets.
- **Correctness of drains:** no internal consumer may act on a truncated collection; `DrainProvider` adoption is verified per consumer call site (FR-5.2).
- **Observability:** existing request logging unchanged; no new metrics required (nice-to-have: log a warning when a single drain exceeds, say, 20 pages).
- **Backward compatibility:** wire `data` shape unchanged; clients ignoring `meta`/`links` still parse responses (they just see one page — which is why every known consumer is updated in-task).

## 9. Open Questions

1. Per-endpoint default sizes for Group C (the "generous default so the common case is one page" values, e.g. buddy list, in-map monsters) — to be fixed in the design phase with game-mechanic caps as evidence.
2. atlas-guilds `filter[name]`: substring vs prefix matching, and whether it needs an index — design-phase decision.
3. Whether any external (non-repo) API consumers exist that would see page-1 truncation. Assumed none (Atlas is self-contained); flag if the design phase finds otherwise.
4. `GET /notes` and bare `GET /history/` appear consumer-less — convert (per scope decision) but confirm they're not dead surface that should instead be removed.

## 10. Acceptance Criteria

- [ ] `model.Page`/`model.Paged`/`MapPaged`, `database.PagedQuery`, `paginate.ParseParams`, `requests.PagedProvider`/`DrainProvider`, and the slice/doc-store paging adapters exist with unit tests (including the no-envelope drain compatibility case).
- [ ] `docs/rest-pagination.md` (or equivalent) documents the convention; PS-5 marked resolved.
- [ ] No collection GET endpoint in any Go service marshals an unbounded collection: every list route parses `page[*]`, defaults to page 1 / default size, rejects invalid params with 400, and returns the envelope.
- [ ] No processor retains an unfiltered `GetAll`-style method; grep for the removed symbols is clean.
- [ ] DB-backed lists page in SQL (LIMIT/OFFSET + COUNT on the scoped query) with stable total ordering; verified by tests per converted service.
- [ ] atlas-data string search uses the hoisted `ParseParams`; search and no-search paths of shared routes present the same envelope.
- [ ] atlas-login and atlas-channel `InitializeRegistry` drain accounts page-by-page; logged-in registry seeding behavior verified by test.
- [ ] Every internal Go consumer of a converted list endpoint either passes explicit paging or drains; audited list of call sites included in the plan and checked off.
- [ ] atlas-guilds supports `filter[name]`; atlas-ui guild search uses it; the fetch-all-then-filter pattern is gone from `guildsService`.
- [ ] atlas-ui characters/accounts/guilds/bans/data-browser views page server-side with total-driven pagers; `npm run build` and tests green (no new lint errors).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` green for every service whose `go.mod` was touched; `tools/redis-key-guard.sh` clean.
