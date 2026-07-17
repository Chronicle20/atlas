# List Endpoint Pagination — Design

Task: task-117-list-endpoint-pagination
Status: Approved design (Phase 2)
Date: 2026-07-02
Inputs: [prd.md](prd.md), [endpoint-inventory.md](endpoint-inventory.md)

---

## 1. Problem Restated (one paragraph)

86 slice-marshaling handler sites exist; effectively one (atlas-data item-strings) paginates. The envelope (`paginate.Envelope`, `server.MarshalPaginatedResponse`) already exists and is proven; what's missing is (a) a paged data-access pipeline so DB-backed lists page in SQL, (b) a hoisted param parser, (c) client-side paged/drain helpers so internal "I need everything" consumers stay correct, and (d) the sweep applying all of it to every collection GET. This design fixes the shapes of those pieces and resolves the PRD's open questions.

## 2. Architectural Approaches Considered

### 2.1 Where pagination lives

| Approach | Description | Verdict |
|---|---|---|
| **(A) Paged value through the provider chain** (chosen) | `model.Paged[T]` flows through `model.Provider` composition; the entity provider itself runs `COUNT` + `LIMIT/OFFSET`; every downstream stage (model transform, decorate, REST transform) lifts over the container via `MapPaged`. | **Chosen.** Bounds work at the query layer (the PRD's hard requirement), reuses the existing lazy-provider idiom, and gives one pipeline shape for DB, doc-store, and registry sources. |
| (B) Marshal-layer slicing | Keep `GetAll`, slice the result inside the handler/marshal step. | Rejected: the DB still loads the full table and every decorator still runs per row — this is PS-5 with an envelope on top. Only acceptable for already-materialized registry dumps (Group D), where it is in fact the chosen adapter (`paginate.Slice`). |
| (C) Parallel method family (`GetAll` + `GetAllPaged`) | Add paged variants alongside the old methods. | Rejected by the PRD: unfiltered `GetAll` must be deleted, not shadowed — leaving it invites new unbounded call sites. |

### 2.2 Drain strategy (client side)

| Approach | Verdict |
|---|---|
| **(A) Page-number iteration off `meta.page.last`** (chosen) | Request page 1 at max size; loop `page[number]=2..last`, re-reading `last` each response; stop early on an empty `data` array. Deterministic, trivially testable with httptest, and independent of link-URL formatting. |
| (B) Follow `links.next` | Rejected: requires URL re-parsing on the client, couples the drain to link-encoding details (`%5B`/`%5D`), and gives no advantage under offset paging — `next` is derived from the same number/size arithmetic. |

Both approaches share the compatibility rule: **no envelope in the response ⇒ the single response is the complete collection** (old server / not-yet-converted endpoint). This makes consumer-first rollout safe.

### 2.3 Stable ordering in `PagedQuery`

| Approach | Verdict |
|---|---|
| **(A) Derive the PK via GORM schema parse; append it as a tie-break** (chosen) | `gorm.Statement.Parse(&E{})` yields `Schema.PrioritizedPrimaryField`; `PagedQuery` appends `ORDER BY <pk>` after any caller-supplied ordering (e.g. ban history's `created_at desc`), producing a total order without per-call boilerplate. |
| (B) Require an explicit order string parameter on every call | Rejected: 80+ call sites of boilerplate, and nothing stops a caller passing a non-total order. The schema-derived tie-break makes total ordering structural. |

## 3. Library Layer

### 3.1 `libs/atlas-model` — the paged container

```go
// model/paged.go
type Page struct {
    Number int // 1-based
    Size   int
}

type Paged[T any] struct {
    Items []T
    Total int  // rows matching the scope pre-paging
    Page  Page // the page that produced Items
}

// MapPaged lifts an item transform over the container, preserving Total/Page.
// Composes exactly like SliceMap: MapPaged(f)(provider)(ParallelMap()).
func MapPaged[E, M any](t Transformer[E, M]) func(Provider[Paged[E]]) func(...MapFuncConfigurator) Provider[Paged[M]]
```

Decoration needs no new primitive: `MapPaged(model.Decorate[M](decorators))(p)(model.ParallelMap())` — `Decorate` already returns a `Transformer[M, M]`. `MapPaged` reuses the `SliceMap` internals (sequential/parallel switch, index-stable results) applied to `.Items`.

### 3.2 `libs/atlas-database` — `PagedQuery`

```go
// provider.go
func PagedQuery[E any](db *gorm.DB, page model.Page) model.Provider[model.Paged[E]]
```

Behavior (lazy, on invocation):
1. **Count**: `int64` count on a `db.Session(&gorm.Session{})` clone of the scoped query with the ORDER BY clause removed from `Statement.Clauses` (GORM's `Count` interacts badly with caller-supplied ordering on Postgres; stripping it is explicit, not left to GORM behavior — covered by a test using a `created_at desc`-ordered query).
2. **Page fetch**: on the original scoped `db`, append the schema-derived PK order (tie-break after any existing `Order`), then `Offset((page.Number-1)*page.Size).Limit(page.Size).Find(&results)`.
3. Return `model.Paged[E]{Items, Total, Page}`.

Both queries derive from the same `*gorm.DB`, so the tenant-filter callback and all `Where` clauses apply identically — proven by a `databasetest` test asserting count and items agree under two tenants.

Callers keep the existing `EntityProvider` idiom, now parameterized by page:

```go
func getAll(page model.Page) database.EntityProvider[[]entity] // becomes:
func getAll(page model.Page) func(db *gorm.DB) model.Provider[model.Paged[entity]] {
    return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
        return database.PagedQuery[entity](db, page)
    }
}
```

### 3.3 `libs/atlas-rest/server/paginate` — params + slice adapter

```go
// params.go
var ErrInvalidPageParam = errors.New(...)

// ParseParams parses page[number]/page[size]. Defaults: number=1, size=defaultSize.
// Invalid (non-integer, number<1, size<1, size>maxSize) => error (handler returns 400).
// A bare legacy ?limit= param is also rejected (error), preserving atlas-data behavior
// and enforcing "paging is expressed only via page[*]" repo-wide.
func ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error)

// slice.go — adapter for already-materialized collections (registries, doc-store cache).
// items MUST already be deterministically ordered by the caller.
// Past-end pages return empty Items with correct Total (envelope recovery links handle UX).
func Slice[T any](items []T, page model.Page) model.Paged[T]
```

atlas-data's private `parsePagingParams` (`services/atlas-data/atlas.com/data/item/string_resource.go:161`) is deleted and the item-strings handler refactored onto `ParseParams` (same 400 semantics; its `defaultSize`/`maxSize` stay `searchindex.MaxLimit`).

`Envelope`/`MarshalPaginatedResponse`/`rewritePage` are unchanged — they already do exactly what the PRD specifies (meta shape, recovery links, param preservation). A tiny convenience is added so handlers don't hand-assemble the envelope:

```go
func EnvelopeFor[T any](p model.Paged[T]) Envelope // {Total, p.Page.Number, p.Page.Size}
```

### 3.4 `libs/atlas-rest/requests` — paged client + drain

The existing `get[A]` decodes only the `data` array (api2go `jsonapi.Unmarshal`). api2go tolerates top-level `meta`/`links` (verified: `Unmarshal` json-decodes into `jsonapi.Document`, which carries `Meta`), so **converted endpoints do not break unconverted clients** — they just see one page, which is why every consumer is updated in-task.

New pieces:

```go
// PagedGetRequest builds a GET with page[number]/page[size] appended to the URL
// (url.Values encoding, correct ?/& handling for URLs that already carry filters)
// and decodes BOTH the items and the envelope from the same body:
//   1. json.Unmarshal into a minimal envelope struct {Meta *{Total int; Page {Number, Size, Last int}}}
//   2. jsonapi.Unmarshal into []A
func PagedGetRequest[A any](url string, page model.Page, configurators ...Configurator) Request[PagedResponse[A]]

type PagedResponse[A any] struct {
    Data []A
    Meta *PageMeta // nil => no envelope (unconverted server)
}

// PagedProvider — one page, transformed: model.Provider[model.Paged[M]]
func PagedProvider[A, M any](l logrus.FieldLogger, ctx context.Context) func(url string, page model.Page, t model.Transformer[A, M]) model.Provider[model.Paged[M]]

// DrainProvider — the semantic-"all" fetch. Requests page 1 at pageSize;
// if Meta == nil, the single response IS the complete collection (compat rule);
// otherwise iterates page[number] 2..meta.page.last (re-read each response),
// stops early on an empty Data page, warns via l when a drain exceeds 20 pages.
func DrainProvider[A, M any](l logrus.FieldLogger, ctx context.Context) func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M]) model.Provider[[]M]
```

`DrainProvider` keeps the `(t, filters)` shape of the existing `SliceProvider` so call-site conversion is mechanical (`requests.SliceProvider(...)(requestAccounts, Extract, Filters())` → `requests.DrainProvider(...)(accountsUrl, 250, Extract, model.Filters[Model]())`).

Tests are httptest-backed per `libs/atlas-rest/CLAUDE.md`: multi-page drain, single-page, empty collection, **no-envelope compatibility**, past-end recovery, a fixture with a `relationships` block (the UnmarshalToManyRelations gotcha), and 400/404 error mapping.

### 3.5 Doc-store adapter (atlas-data `document` package)

`Storage[I, M]` gains `AllPagedProvider(ctx, page) model.Provider[model.Paged[M]]`:
- Registry (in-memory) hit: sort by document id, `paginate.Slice`.
- DB path: `PagedQuery` over the document entity scoped exactly like today's `AllProvider`, **preserving the canonical-fallback semantics**: if the tenant-scoped count is 0, fall back to the canonical-version scope and page *that* (the batch-GetAll-skips-fallback bug class must not recur in the paged variant — regression test required).
- `GetAll` (unpaged) is deleted; internal callers move to the paged provider or a storage-level drain where genuinely needed.

## 4. Convention (docs/rest-pagination.md)

New `docs/rest-pagination.md`, linked from the backend-dev-guidelines skill source and `docs/architectural-improvements.md` (PS-5 → ✓ resolved, referencing task-117). Contents: param names, defaults, envelope shape, 400 semantics, the `AllProvider(page, decorators...)` processor pattern, "DB-backed lists page in SQL", consumer rules (UI pages; internal Go semantic-"all" drains), and the default-size table below.

**Default / max page sizes (resolves PRD open question 1):**

| Class | default | max | Rationale |
|---|---|---|---|
| Standard collections (Groups A, B, D, LOW) | 50 | 250 | PRD baseline. |
| Group C game-capped lists (inventory/storage assets, buddies, skills, macros, keys, pets, wishlist, quests, monster-book cards, invites, buffs, proposals, family tree, in-map registries) | 250 | 250 | One page covers the mechanical cap in the common case; correctness never depends on it because every internal consumer drains. Plan phase confirms per-endpoint caps against repo constants (`libs/atlas-constants`) where they exist; any cap found above 250 gets a documented per-endpoint override. |
| Group C growing logs (`/characters/{id}/visits`, `/characters/{id}/sessions`, `/history/accounts/{id}`) | 50 | 250 | Monotonically growing; standard defaults per PRD FR-5.3. |

## 5. Server-Side Pattern (per endpoint)

Processor (one list method, unfiltered `GetAll` deleted):

```go
// processor.go
AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]]

func (p *ProcessorImpl) AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]] {
    ep := getAll(page)(p.db.WithContext(p.ctx))                       // Paged[entity]
    mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())    // Paged[Model]
    return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
}
```

Resource handler (replaces the `GetAll` + `MarshalResponse` pair):

```go
page, err := paginate.ParseParams(r.URL.Query(), 50, 250)
if err != nil { w.WriteHeader(http.StatusBadRequest); /* JSON:API error object */ return }

paged, err := NewProcessor(...).AllProvider(page, decoratorsFromInclude(r, d, c)...)()
if err != nil { 500 }

rms, err := model.SliceMap(Transform(...))(model.FixedProvider(paged.Items))(model.ParallelMap())()
server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, paginate.EnvelopeFor(paged), r)
```

`include` decorators run inside `AllProvider` over `paged.Items` only — per-page, satisfying FR-3's decorator requirement by construction.

Filtered list routes that stay filtered (e.g. `?filter[members.id]=`, `?accountId=`) keep their query shape and *additionally* accept `page[*]` (Group D rule generalized): the filter narrows the scope, `PagedQuery`/`Slice` pages it.

## 6. Per-Group Decisions

### 6.1 Group A (bare full-table dumps)

Straight application of §5 to atlas-character, atlas-account, atlas-ban (`/bans/`, `/history/`), atlas-notes, atlas-merchant. Specifics:

- **atlas-account consumers**: atlas-login `login/main.go:267` and atlas-channel `channel/main.go:386` seed the logged-in registry via `account.AllProvider` → `InitializeRegistry`. Their service-local `account.Processor.AllProvider()` is re-implemented on `requests.DrainProvider` (page size 250); the `Processor` interface keeps `InitializeRegistry()` semantics ("all accounts") unchanged. Registry-seeding behavior verified by an httptest test serving two pages and asserting both pages' accounts land in the registry.
- **atlas-guilds `filter[name]`** (resolves PRD open question 2): **case-insensitive substring** via `LOWER(name) LIKE LOWER(?)` with `%`/`_`/escape-char escaping of user input (matches the existing `LOWER()` idiom in `character/provider.go:32` and works identically on Postgres). Substring over prefix because the UI's current client-side filter is substring (`includes`) — prefix would silently change search results. **No index now**: guild cardinality is content-scale (thousands per tenant, not millions); a seq scan on a paged query is acceptable, and a `pg_trgm` index can be added later without contract change. Empty `filter[name]=` → 400. Composes with `page[*]`; count and page share the filter through `PagedQuery`.
- **`GET /notes` and bare `GET /history/`** (resolves open question 4): **convert, don't remove**. Removal is a product-surface decision outside this task's scope decision ("apply to all"); conversion costs one mechanical pass. Noted in the convention doc as consumer-less candidates for future removal.
- **atlas-merchant**: verify the suspected `legacy-merchant-web-ui` consumer at implementation time; if present it gets the same paged treatment as atlas-ui views (or, if it's outside this repo, the compat rule applies — it sees page 1 and the finding is escalated per PRD open question 3).

### 6.2 Group B (doc-store + script/config stores)

- All bare atlas-data list routes adopt `ParseParams` + `Storage.AllPagedProvider` + `MarshalPaginatedResponse`. By-parent variants (`/monsters/{id}/maps` etc.) are filtered doc lookups — same envelope, paging their (small) result via `paginate.Slice` where the backing fetch is already a single document's sub-list.
- The `?search=` paths keep their searchindex backing but move onto the hoisted `ParseParams`; search and no-search arms of one route thus present the identical envelope and identical 400 semantics.
- Script/config stores (atlas-map-actions, atlas-reactor-actions, atlas-portal-actions, atlas-npc-conversations, atlas-gachapons, atlas-drop-information, atlas-party-quests definitions) follow the same pattern via their own storage/provider layers (DB-backed → `PagedQuery`; registry-cached → `paginate.Slice` on an id-sorted snapshot).
- Go consumers of bare Group B lists: the inventory found none beyond by-id fetches; each converted service's implementation step re-greps its `requests.go` consumers and converts any found to `DrainProvider` (hard gate in the plan, not an assumption).

### 6.3 Group C (filtered-but-unbounded)

- Envelope + `ParseParams` with the §4 defaults (250/250 capped class; 50/250 growing logs).
- **Every internal Go consumer of a converted endpoint switches to `DrainProvider`** — mechanical rule, no judgment per site: if a call site consumed the full filtered list before, it drains after. The plan enumerates every call site (grep by requests.go per converted service, receiver-typed per the consumer-audit lesson) and checks each off individually. Hot game paths (in-map monsters, buddies, skills, quests) thereby keep exact semantics; with default sizes ≥ the mechanical caps, drains are single-round-trip in the common case, so no added latency.

### 6.4 Group D (runtime registry dumps)

- `GET /parties`, `/messengers`, `/sagas`, `/party-quests/instances`, `/portals/blocked`: materialize the registry slice exactly as today, sort deterministically by primary id, `paginate.Slice`, envelope. Filtered forms keep their shape and accept `page[*]`.
- LOW/naturally-bounded endpoints (worlds, channels, tenants, configurations, transports): same slice adapter — trivial uniformity sweep.

## 7. atlas-ui

- **`services/api/pagination.ts`**: `fetchPaged<T>(url, {page, size}, options): Promise<{data: T[], total, page: {number, size, last}}>` — appends `page[number]`/`page[size]` via `URLSearchParams`, decodes `meta`; and `fetchAll<T>` (drain) with the same no-envelope compatibility rule. TS envelope types exported once.
- **List views** (characters, accounts, bans, guilds browse, data browsers, merchants if applicable): server-side paging with total-driven pager; React Query `placeholderData: keepPreviousData` per page-keyed query keys (`[resource, tenant, page, size]`), per frontend guidelines.
- **guildsService**: `getAll()`-then-filter-client-side is deleted. Search/by-name → `?filter[name]=`; by-member keeps `?filter[members.id]=`; browse pages.
- Test call sites updated in the same commits (`npm run build` type-checks tests — known constraint).

## 8. Testing Strategy

| Layer | Tests |
|---|---|
| atlas-model | `MapPaged` preserves Total/Page, index-stable under `ParallelMap`, error propagation. |
| atlas-database | `PagedQuery` (databasetest/sqlite or the existing harness): tenant-scope agreement between count and items; PK tie-break yields non-overlapping pages under a non-unique order column; caller `Order` preserved; count correct with caller `Order` present; offset/limit in SQL (no full load). |
| atlas-rest paginate | `ParseParams` table test (defaults, 400 cases, maxSize cap, legacy `limit` rejection); `Slice` past-end/empty. |
| atlas-rest requests | httptest: paged decode, drain multi-page/early-empty/no-envelope/relationship-stub fixture, >20-page warning. |
| Per converted service | Existing resource tests updated; at least Group A services get an httptest resource-level test asserting envelope shape + 400 on bad params; guilds adds `filter[name]` tests (match, escape chars, empty→400, paging composition). |
| Consumers | atlas-login/atlas-channel registry seed across ≥2 pages; drain call-site conversions covered by their services' existing tests plus targeted httptest where a consumer had none. |
| Sweep | Acceptance grep: no `MarshalResponse[[]` on a collection GET route; no unfiltered `GetAll` symbols. |

## 9. Sequencing & Verification

1. **Phase L — lib layer** (atlas-model, atlas-database, atlas-rest incl. atlas-data `ParseParams` refactor + doc-store paged provider): everything else depends on it.
2. **Phase A — Group A + its consumers + core UI** (character, account, guilds+`filter[name]`, ban, notes, merchant; login/channel drain; UI pagination util + characters/accounts/guilds/bans views).
3. **Phase B — atlas-data routes + script/config stores + UI data browsers.**
4. **Phase C — Group C sweep + consumer drain conversions** (per-service mechanical passes; the plan's call-site checklist gates each).
5. **Phase D — Group D + LOW sweep.**
6. **Phase Docs — convention page, PS-5 resolution, endpoint-inventory cross-check.**

Consumers deploy-safe in any order via the no-envelope compat rule; within the branch, each endpoint's conversion and its consumers land in the same commit/phase (FR-8). Every touched module runs the full gauntlet (`go test -race`, `go vet`, `go build`, `docker buildx bake atlas-<svc>` for every touched `go.mod`, `tools/redis-key-guard.sh`); atlas-ui runs `npm run build` + tests.

## 10. Resolved Open Questions (PRD §9)

1. **Group C defaults** → §4 table: 250/250 for game-capped lists (plan confirms caps from repo constants), 50/250 for growing logs.
2. **`filter[name]`** → case-insensitive substring (`LOWER LIKE` with escaping), no index initially (§6.1).
3. **External consumers** → none known; atlas-merchant's suspected external web UI is verified at implementation and escalated if found outside the repo (§6.1).
4. **`/notes` & bare `/history/`** → convert per scope decision; flagged as removal candidates in the convention doc (§6.1).

## 11. Explicitly Out of Scope (unchanged from PRD)

Cursor/keyset paging, enforcement tooling, caching/rate-limiting, single-resource endpoints, non-REST surfaces.
