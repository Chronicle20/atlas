# REST Collection Pagination

Status: adopted repo-wide (task-117). Resolves [architectural-improvements.md](architectural-improvements.md) finding PS-5 ("Unbounded List Endpoints", High).

Every REST route that returns a collection (`GET` on a resource-collection path) must page its result. This document is the single source of truth for the param names, envelope shape, default/max sizes, the server-side processor pattern, and the client-side consumer rules. It supersedes ad hoc per-service pagination (the old `atlas-data` `parsePagingParams` helper, deleted in task-117, is now `paginate.ParseParams`).

## 1. Query parameters

| Param | Meaning | Default | Notes |
|---|---|---|---|
| `page[number]` | 1-based page index | `1` | |
| `page[size]` | items per page | class-dependent (see §3) | capped at the endpoint's `maxSize` |

Parsing lives in `libs/atlas-rest/server/paginate.ParseParams(query, defaultSize, maxSize) (model.Page, error)`:

- Non-integer `page[number]`/`page[size]`, `page[number] < 1`, `page[size] < 1`, or `page[size] > maxSize` all return `ErrInvalidPageParam`. **Values are never silently clamped** — an out-of-range request is a client error, not a server-side correction.
- A bare legacy `?limit=` query param is rejected outright (also `ErrInvalidPageParam`), even if `page[*]` is also present. This retires the pre-task-117 `atlas-data` convention and enforces that paging is expressed only one way, repo-wide.
- Handlers map `ErrInvalidPageParam` to `400 Bad Request` with a JSON:API error object (see `server.WriteBadRequest`).

```go
page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
if err != nil {
    server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
    return
}
```

## 2. Envelope shape

A paginated response adds `meta` and pagination `links` on top of the ordinary JSON:API document; `data` is exactly one page's worth of resources.

```json
{
  "data": [
    { "type": "characters", "id": "1001", "attributes": { "name": "Bowman" } },
    { "type": "characters", "id": "1002", "attributes": { "name": "Warrior" } }
  ],
  "meta": {
    "total": 137,
    "page": { "number": 1, "size": 2, "last": 69 }
  },
  "links": {
    "self": "/characters?page%5Bnumber%5D=1&page%5Bsize%5D=2",
    "first": "/characters?page%5Bnumber%5D=1&page%5Bsize%5D=2",
    "next": "/characters?page%5Bnumber%5D=2&page%5Bsize%5D=2",
    "last": "/characters?page%5Bnumber%5D=69&page%5Bsize%5D=2"
  }
}
```

Rules (implemented by `paginate.Envelope` / `server.MarshalPaginatedResponse`, do not hand-roll):

- `meta.total` is the row count matching the query's scope *before* paging (not `len(data)`).
- `meta.page.last` is `ceil(total / size)`, floored at `1`.
- `links.self`/`links.first`/`links.last` are always present; `links.prev` is omitted on page 1, `links.next` is omitted on the last page.
- **Past-end pages** (`page[number] > last`) return `200` with an empty `data` array (never `404`); `links.prev` recovers to `last`, `links.next` is omitted. `meta.total`/`meta.page.last` are still correct, so a client can immediately re-request the real last page.
- URL encoding of `page[number]`/`page[size]` in `links` follows `url.Values.Encode()` (`%5B`/`%5D` for `[`/`]`), matching `URLSearchParams` on the client — this is deliberate, not a bug to "fix" to bracket-literal URLs.

Build the envelope from a `model.Paged[T]` with `paginate.EnvelopeFor(paged)`; do not construct `Envelope{}` by hand.

## 3. Default / max page sizes

| Class | Default | Max | Rationale |
|---|---|---|---|
| Standard collections (bare full-table/full-dump lists: `/characters`, `/accounts`, `/guilds`, `/bans`, `/notes`, `/history/`, atlas-data content dumps, LOW/naturally-bounded lists) | 50 | 250 | PRD baseline; matches typical UI page sizes. |
| Game-capped lists (per-character/per-map inventory, skills, keys, buffs, quests, macros, buddies, pets, wishlist, monster-book cards, invites, in-map registries, family tree, etc.) | 250 | 250 | One page covers the mechanical game cap in the common case. Correctness never depends on this — every internal Go consumer drains regardless (§6) — this only bounds worst-case single-request payload size. |
| Growing logs (`/characters/{id}/visits`, `/characters/{id}/sessions`, `/history/accounts/{id}`, guild/messenger/party threads) | 50 | 250 | Monotonically growing; standard defaults per PRD FR-5.3 — these are the collections most likely to genuinely need multiple pages in the UI. |

`paginate.DefaultPageSize = 50` and `paginate.MaxPageSize = 250` are the constants backing the standard/growing-log row; game-capped endpoints pass `paginate.MaxPageSize` as **both** the `defaultSize` and `maxSize` argument to `ParseParams` (`paginate.ParseParams(query, paginate.MaxPageSize, paginate.MaxPageSize)`), rather than inventing a third constant.

### Per-endpoint overrides / notes recorded during Phase C

A handful of endpoints don't have a single-PK GORM entity backing the query — either the primary key is composite, or the data comes from a registry/graph/aggregation rather than a `WHERE`-filtered table scan. These use the **materialize + stable-sort-by-unique-key + `paginate.Slice`** adapter instead of `database.PagedQuery`:

- **atlas-keys** `GET /characters/{id}/keys` — composite PK (character, key slot).
- **atlas-monster-book** cards — composite PK (character, monster card).
- **atlas-quest** `/{questId}/progress` — quest progress rows keyed by (quest instance, info number); sorted by row `Id()` before slicing (GORM's `Preload` gives no stable order).
- **atlas-families** `/families/tree/{id}` — graph traversal, not a table scan.
- **atlas-chairs**, **atlas-party-quests** `/party-quests/instances`, **atlas-saga-orchestrator** `/sagas`, **atlas-drop-information** `/continents/drops` — in-memory registry dumps or computed aggregations (e.g. drops grouped by continent from a Go map) with no natural row order.

**Adapter-choice rule:** DB entity with a single prioritized primary key → `database.PagedQuery`. Composite-PK entity, in-memory registry, graph traversal, or any other in-process aggregation → materialize the full (already game/content-bounded) collection, sort it by a field that is unique and stable across calls, then `paginate.Slice`. Never push an in-memory materialize-then-slice adapter onto something GORM could page in SQL — that defeats the point of PS-5.

## 4. Server-side pattern: `AllProvider(page, decorators...)`

The unfiltered `GetAll`-style processor method is deleted (not shadowed) and replaced with a single paged provider method. Canonical shape (`libs/atlas-model/model.MapPaged` lifts a transformer over a `Paged[T]` exactly like `SliceMap` lifts one over a slice):

```go
// provider.go — DB-backed: EntityProvider now takes a page and returns Paged[entity]
func getAll(page model.Page) database.EntityProvider[model.Paged[entity]] {
    return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
        return database.PagedQuery[entity](db, page)
    }
}

// processor.go
func (p *ProcessorImpl) AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]] {
    ep := getAll(page)(p.db.WithContext(p.ctx))                       // Paged[entity]
    mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())    // Paged[Model]
    return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
}
```

Resource handler (replaces the old `GetAll()` + `server.MarshalResponse[[]RestModel]` pair):

```go
func handleGetCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
        if err != nil {
            server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
            return
        }

        paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page, decoratorsFromInclude(r, d, c)...)()
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        query := r.URL.Query()
        queryParams := jsonapi.ParseQueryFields(&query)
        server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
    }
}
```

`include` decorators run **inside** `AllProvider`, over `paged.Items` only — decoration cost is per-page, not per-collection, which is what makes the paged pipeline actually bound work rather than just truncate the response after doing the same work as before.

If the old unfiltered method carried mandatory in-service decoration (not an optional `include`), preserve it as an additional `MapPaged` stage rather than dropping it — see `atlas-account`'s `AllProvider`, which has three stages (`Make` → `decorateState` → `Decorate(decorators)`) because `decorateState` pulls live Redis session state that the generic two-stage recipe doesn't know about. Check for this kind of mandatory decoration before assuming a two-stage conversion is faithful.

Filtered list routes that stay filtered (`?filter[members.id]=`, `/characters/{id}/skills`, etc.) follow the same shape: the filter narrows the scope (a `Where` clause, or the registry/aggregation key), and `PagedQuery`/`Slice` pages *that* scope.

## 5. DB-backed lists page in SQL

A `database.PagedQuery[E](db, page)` call does exactly two queries against the *same* scoped `*gorm.DB` (so every `Where`/tenant-filter clause applies identically to both):

1. **Count** — `SELECT COUNT(*)` on a session clone with any caller-supplied `ORDER BY` clause stripped (GORM's count/order interaction is not something to rely on implicitly).
2. **Page fetch** — the schema-derived primary key (`gorm.Statement.Parse` → `Schema.PrioritizedPrimaryField`) is appended as an `ORDER BY` tie-break after any caller-supplied ordering, then `OFFSET (number-1)*size LIMIT size`.

The rule this encodes: **a converted endpoint must never `db.Find` the full table and then slice in Go.** That's PS-5 with an envelope stapled on top — the query still does full work, and the fix is theater. The only endpoints allowed to materialize-then-slice are the ones in §3's per-endpoint override list, because for them there genuinely is no SQL query capable of expressing the sort key.

The PK tie-break also fixes a correctness gap that predates pagination: without a total order, two pages requested back-to-back over a table ordered only by a non-unique column (e.g. `created_at`) can silently duplicate or skip rows when rows share a timestamp. `PagedQuery` makes every paged listing a total order for free.

## 6. Hidden-decoration rule

Live-state decoration — `loggedIn` status, buff remaining duration, skill cooldowns, slot ordinals, anything computed from something other than the paged row itself — must be **reapplied over the page**, not silently dropped when a `GetAll` becomes an `AllProvider`.

The mechanism is the same `MapPaged`/`Decorate` stage used for `include`-driven decoration (§4): once the DB/registry query is scoped down to Paged[T]'s `.Items`, run the exact decoration step the old unfiltered method ran, just over fewer rows. There is no separate "post-Slice" primitive needed for the registry-adapter case either — `paginate.Slice` returns a `model.Paged[T]` like `PagedQuery` does, so the same `MapPaged` stage applies uniformly regardless of which adapter produced the page.

Skipping this is an easy, easy-to-miss regression: the endpoint still returns `200` with a plausible-looking page, just with every row silently wrong (e.g. every account showing `loggedIn: false` regardless of actual session state). Treat "does this list have any decoration step beyond the raw DB row?" as a mandatory question before treating a `GetAll` → `AllProvider` conversion as done.

## 7. Consumer rules

### atlas-ui

- `services/atlas-ui/src/services/api/pagination.ts` exports `fetchPaged<T>(url, {number, size}, options)` (one page + `meta`) and `fetchAll<T>(url, size, options)` (drain).
- List views that show a paged UI (characters, accounts, bans, guilds) call `fetchPaged` directly, driven by a pager component; views that need the complete set internally (a lookup, a client-side join) call `fetchAll`.
- React Query keys for paged views include `[resource, tenant, page, size]` so `keepPreviousData` works across page changes without a loading flash.

### Go services — semantic-"all" consumers

Any internal Go call site that used to consume a full (now-paginated) collection switches to `requests.DrainProvider[A, M](l, ctx)(url, pageSize, transformer, filters)`. This is a **mechanical, no-judgment-per-site rule**: if a call site needed the complete list before pagination, it needs the complete list after — the fact that a page's default size happens to exceed today's game cap is not a reason to skip the drain, because correctness must not depend on a cap that could change. `DrainProvider` keeps the exact `(t, filters)` tail of the old `requests.SliceProvider`, so the conversion at a call site is almost always a one-line diff:

```go
// before
requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacter(characterId), Extract, model.Filters[Model]())

// after
requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterUrl(characterId), 250, Extract, model.Filters[Model]())
```

Note the URL helper changes shape too: a `requests.Request[[]RestModel]`-returning `requestX(...)` becomes a bare `string`-returning `xUrl(...)`, because `DrainProvider` appends its own `page[number]`/`page[size]` per iteration rather than the caller baking a single request.

`DrainProvider` requests page 1 at `pageSize`, then iterates `page[number] = 2..meta.page.last` (re-reading `last` on each response, since it can change under concurrent writes), stopping early if a page comes back empty. It logs a warning past 20 pages so an accidentally-hot drain call site shows up in logs before it becomes an incident.

**Endpoints that are filtered down to at most one/a few results by a domain invariant do not need to drain** — e.g. `?name=` character/skill lookups (character names are globally unique), `?filter[members.id]=` party/guild lookups (a character is in at most one party/guild), or a party's `/members` list (hard-capped at 6 by game rules, well under any page's default size). `requests.SliceProvider` remains correct for these; converting them to `DrainProvider` would be a no-op that adds a needless round-trip-that-never-happens code path. The distinction to make at each call site is not "does the target route emit a `meta` envelope" but "can the target's result set exceed one page under any real game state."

### No-envelope compatibility rule

`requests.SliceProvider`/`GetRequest[[]A]` decode only the JSON:API `data` array — a response with no `meta` block is not an error, it's read as *the complete collection*. This is what lets a consumer-side conversion (switching a call site to `DrainProvider`) land before or after the server-side conversion of the endpoint it calls: `DrainProvider` checks `resp.Meta == nil` and, if so, treats that single response as everything and stops. The inverse also holds — an old, not-yet-converted `SliceProvider` caller pointed at a freshly-paginated endpoint just sees page 1 of `data`, which is a page-size truncation, not a crash. This is exactly the class of bug this document's §7 mechanical-drain rule exists to prevent; it is not a reason to skip converting a call site "because it won't error out."

## 8. Consumer-less removal candidates

`GET /notes` (atlas-notes) and the bare `GET /history/` (atlas-ban) were converted (paginated) rather than removed, per the PRD's "apply to all" scope decision — no consumer was found for either during the task-117 inventory (`docs/tasks/task-117-list-endpoint-pagination/endpoint-inventory.md` §"Judgment calls"), which suggests they are admin/orphan surfaces. They're flagged here as candidates for a future product-scope decision to remove outright, rather than maintain indefinitely; that decision is out of scope for task-117.
