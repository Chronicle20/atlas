# task-117 — Implementation Context

Companion to [plan.md](plan.md). Key files, locked decisions, dependencies, and gotchas an implementer needs but that don't fit a single task.

## Artifacts

- `prd.md` — requirements; §10 acceptance criteria are the exit bar.
- `design.md` — approved design; §3 fixes the lib shapes, §4 the size defaults, §5 the per-endpoint server pattern, §6 per-group decisions.
- `endpoint-inventory.md` — the endpoint census the sweep tasks' checklists come from (Groups A–D + LOW). Task 29 appends a per-row disposition.

## Key existing files (read before touching)

| Area | File | Why |
|---|---|---|
| Provider combinators | `libs/atlas-model/model/processor.go` | `Provider`, `Transformer`, `SliceMap`, `Decorate`, `ParallelMap`, `FilteredProvider` — `MapPaged` (Task 1) mirrors `SliceMap`'s currying exactly. |
| DB providers | `libs/atlas-database/provider.go` | `EntityProvider[E] = func(*gorm.DB) model.Provider[E]` — paged providers reuse it with `E = model.Paged[entity]`. |
| Tenant scoping | `libs/atlas-database/tenant_scope.go` | The callback `PagedQuery` must inherit for both COUNT and page fetch. |
| Test DB harness | `libs/atlas-database/databasetest/testdb.go` | `NewInMemoryTenantDB`, `TenantContext`. Imports `database` → lib tests for `PagedQuery` must be package `database_test`. |
| Envelope (proven) | `libs/atlas-rest/server/paginate/envelope.go` | `Envelope`, `Meta()`, `BuildLinks()` (past-end recovery links), `rewritePage` (preserves other query params). Unchanged by this task except the `EnvelopeFor` helper. |
| Paginated marshal | `libs/atlas-rest/server/paginated_response.go` | `MarshalPaginatedResponse` — already exists, already correct. |
| HTTP client core | `libs/atlas-rest/requests/get.go` | Task 4 extracts `getBody` from `get[A]`; retry/status mapping stays identical. `ErrBadRequest`/`ErrNotFound` sentinels. |
| Client providers | `libs/atlas-rest/requests/provider.go` | `SliceProvider(r, t, filters)` — `DrainProvider` keeps the `(t, filters)` tail so call-site conversion is mechanical. |
| Client gotchas | `libs/atlas-rest/CLAUDE.md` | Relationship-stub requirement for jsonapi targets; httptest over FakeClient; 404-vs-decode distinction. Test fixtures must include a `relationships` block. |
| Reference impl | `services/atlas-data/atlas.com/data/item/string_resource.go:107-182` | The one paginated endpoint today; `parsePagingParams` (:161) is hoisted into `paginate.ParseParams` (Task 5) with identical semantics incl. legacy `?limit=` → 400. |
| Doc store | `services/atlas-data/atlas.com/data/document/{storage,db_storage}.go` | `AllProvider` (:71) carries the canonical-fallback comment — the paged variant must preserve it (regression class: batch-skips-fallback, PR #759). `documents` table orders by `document_id`; PK is uuid `Id`. |
| Canonical Group A example | `services/atlas-character/atlas.com/character/character/{provider,processor,resource}.go` | `getAll` :40, `GetAll` :185, `handleGetCharacters` :39. Task 9 (accounts) is written out in full; Tasks 10–14 instantiate the same recipe. |
| Account consumers | `services/atlas-login/atlas.com/login/account/processor.go:61-84`, `services/atlas-channel/atlas.com/channel/account/processor.go:42-64` | `AllProvider` → `InitializeRegistry` (seed call sites: `login/main.go:267`, `channel/main.go:386`). Interface stays; only the impl switches to `DrainProvider`. |
| UI API client | `services/atlas-ui/src/lib/api/client.ts` | `api.get<T>` returns the parsed body (meta accessible); `api.getList` strips to `data` — insufficient for paged fetches. |
| UI guilds | `services/atlas-ui/src/services/api/guilds.service.ts` | The fetch-all-then-filter pattern (search/getByWorld/getWithSpace/getRankings) Task 17 deletes. |

## Locked decisions (do not relitigate)

1. **Paged value through the provider chain** (design §2.1-A): pagination is a `model.Paged[T]` flowing through providers; SQL `COUNT` + `LIMIT/OFFSET` at the entity provider. Marshal-layer slicing (`paginate.Slice`) only for already-materialized sources (registries, doc sub-lists, Group D).
2. **Drain = page-number iteration off `meta.page.last`** (§2.2-A), never `links.next`. Compat rule: no envelope ⇒ single response is the complete collection — consumers land before servers.
3. **Stable order = schema-derived PK tie-break inside `PagedQuery`** (§2.3-A), appended after caller `Order`; COUNT strips ORDER BY explicitly (GORM's internal stripping not relied on).
4. **Defaults** (§4): 50/250 standard; 250/250 Group C game-capped (verify caps against `libs/atlas-constants`; a cap >250 becomes a documented per-endpoint override); 50/250 growing logs. Exposed as `paginate.DefaultPageSize`/`paginate.MaxPageSize`.
5. **No `page[*]` params ⇒ page 1 at default size** (PRD option (a), safe-by-default) — every known consumer is updated in-task, which is what makes this safe.
6. **Invalid params / legacy `?limit=` ⇒ 400** (JSON:API error object via new `server.WriteBadRequest`), never clamped.
7. **Unfiltered `GetAll` deleted, not shadowed.** Filtered fetch-all methods survive for same-service internal use; REST handlers use paged siblings.
8. **guilds `filter[name]`**: case-insensitive substring, `LOWER(name) LIKE LOWER(?) ESCAPE '\'` with `%`/`_`/`\` escaping; no index now; empty value → 400 (§6.1).
9. **`/notes` + bare `/history/`**: convert, don't remove; flagged as removal candidates in the convention doc (§6.1).
10. **atlas-merchant external consumer**: verify at implementation (plan Task 14 Step 1); if outside the repo → stop and escalate before converting.

## Dependency order

```
Task 1 (model) ─┬─ Task 2 (database) ─┬─ Tasks 9–14 (Group A servers)
                ├─ Task 3 (paginate) ─┤   ↑ Tasks 7–8 (login/channel drains) land FIRST
                └─ Task 4 (requests) ─┴─ Tasks 18–28 (sweeps; consumers drain before/with server)
Task 3 → Task 5 (atlas-data ParseParams refactor)
Tasks 1–3 → Task 6 (doc store paged) → Tasks 18–19 (atlas-data routes; 19 deletes unpaged storage list)
Task 15 (UI util) → Tasks 16, 17, 22 (UI views)
Everything → Task 29 (docs + acceptance greps + full bake)
```

Within every server-conversion task: consumers convert **before or with** the server commit (PRD FR-8 — no intermediate commit reads a truncated collection).

## Gotchas / prior-incident guardrails

- **api2go tolerates top-level `meta`/`links`** (verified in design §3.4): converted servers don't break unconverted `get[A]` clients — they just silently see one page, which is why the consumer gate (Phase C recipe step 6) is a hard requirement, not hygiene.
- **Consumer audits verify by receiver type** (task-087 lesson): a `.MapId()`-style reader can belong to a mirror model, not the REST consumer — resolve each touched call's receiver before editing.
- **`docker buildx bake atlas-<svc>` per touched `go.mod` is mandatory** — `go build` under `go.work` won't catch Dockerfile `COPY libs/...` gaps. No new lib is added by this task (all changes live in existing `libs/atlas-*`), so no Dockerfile edits are expected; if one IS needed, that's a red flag to re-check.
- **Registry singletons in tests**: login/channel account registries are process-wide — seed tests must use a fresh tenant uuid per test.
- **`ACCOUNTS_SERVICE_URL` style env roots need a trailing slash** in httptest setups (`t.Setenv("ACCOUNTS_SERVICE_URL", srv.URL+"/")`) because request builders concatenate `RootUrl(domain) + "accounts"`.
- **GORM count-vs-order**: if `delete(clone.Statement.Clauses, "ORDER BY")` mutates the source db under the pinned GORM version, the Task 2 caller-order test fails — fix the clone mechanics, not the test.
- **atlas-ui `npm run build` type-checks test files** — shared-signature changes must update test call sites in the same commit.
- **`*_search_index` tables have no `id` column** (baseline-publish incident) — irrelevant to `PagedQuery` (search paths use `searchindex`, not `PagedQuery`), but do not point `PagedQuery` at a search-index entity.
- **mux route order**: register `?filter[name]=`-constrained routes before the bare collection route.
- **Redis**: registry paging materializes via existing lib-mediated reads; no new raw keyed go-redis calls (redis-key-guard enforces).

## Verification (every task)

`go test -race ./...`, `go vet ./...`, `go build ./...` per touched module; `docker buildx bake atlas-<svc>` per touched `go.mod`; `tools/redis-key-guard.sh` from repo root; atlas-ui `npm run build` + `npm test`. Task 29 finishes with `docker buildx bake all-go-services` and the three acceptance greps.
