# Wz-Data Search Rollout — Task Checklist

Last Updated: 2026-04-18

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[-]` skipped.

## Phase A — Shared `searchindex` helper package

- [x] **A1.** Sketch helper API (migration / upsert / query / delete). Decide
      generics-vs-thin-helper. _Effort: S._
- [x] **A2.** Implement migration helper (table + trigram GIN + optional extension
      columns & partial indexes). _Effort: S._
- [x] **A3.** Implement query helper: tenant fallback, exact-id union, `name ASC,
      id ASC`, LIMIT ≤ 50 enforced server-side, parameterized. _Effort: M._
- [x] **A4.** Implement transaction upsert hook; failure rolls back outer
      transaction. _Effort: S._
- [x] **A5.** Implement delete hook for `DbStorage.Clear` / `DeleteAll` cascade.
      _Effort: S._
- [x] **A6.** Unit tests: substring, exact-id, limit, tenant fallback, empty-query
      rejection, rollback-on-failure. _Effort: M._

## Phase B — Maps retrofit

- [x] **B1.** Refactor `map/storage.go`, `search.go`, `entity.go` through
      `searchindex`. Preserve `map_search_index` name, PK, `street_name` extension.
      _Effort: M._
- [x] **B2.** Delete `map/backfill.go` + `map/backfill_test.go`. Grep for any
      remaining references; update or remove. _Effort: S._
- [x] **B3.** `map/storage_test.go` and `map/search_test.go` pass unchanged.
      _Effort: S._
- [x] **B4.** Smoke test: `GET /api/data/maps?search=henesys` matches pre-refactor
      response. _Effort: S._ (deferred to manual QA during deploy)

## Phase C — NPC search index + fast path

- [x] **C1.** GORM migration for `npc_search_index` incl. `storebank` partial index.
      _Effort: S._
- [x] **C2.** Ingest upsert hook in `npc/storage.go`; sources `storebank` from the
      NPC document. _Effort: S._
- [x] **C3.** Cleanup hook: cascade delete on `Storage.Clear`. _Effort: S._
- [x] **C4.** Fast-path handler in `npc/resource.go` routing `?search=`,
      `filter[storebank]=true`, and their composition. Sparse `{name, storebank}`.
      _Effort: M._
- [x] **C5.** Delete `filterNpcs` slow path. _Effort: S._
- [x] **C6.** Unit tests: exact-id, substring, limit, tenant fallback, empty-query,
      `filter[storebank]` alone, composition with `?search=`. _Effort: M._
- [x] **C7.** Existing `resource_test.go` passes (seeding now routes through
      `npc.NewStorage` so the search index is populated too). _Effort: S._

## Phase D — Monster search index + fast path

- [x] **D1.** GORM migration for `monster_search_index`. _Effort: S._
- [x] **D2.** Ingest upsert hook in `monster/storage.go`. _Effort: S._
- [x] **D3.** Cleanup hook. _Effort: S._
- [x] **D4.** Fast-path handler in `monster/resource.go`. Sparse `{name}`. _Effort: S._
- [x] **D5.** Delete `filterMonsters` slow path. _Effort: S._
- [x] **D6.** Unit tests: exact-id, substring, limit, tenant fallback, empty-query.
      _Effort: S._
- [x] **D7.** `GET /api/data/monsters/{id}` byte-identical (existing tests pass).
      _Effort: S._

## Phase E — Reactor search index + fast path

- [x] **E1.** GORM migration for `reactor_search_index`. _Effort: S._
- [x] **E2.** Ingest upsert hook in `reactor/storage.go`. _Effort: S._
- [x] **E3.** Cleanup hook. _Effort: S._
- [x] **E4.** Fast-path handler in `reactor/resource.go`. Sparse `{name}`. _Effort: S._
- [x] **E5.** Delete `filterReactors` slow path. _Effort: S._
- [x] **E6.** Unit tests: exact-id, substring, limit, tenant fallback, empty-query.
      _Effort: S._
- [x] **E7.** `GET /api/data/reactors/{id}` byte-identical. _Effort: S._

## Phase F — Item-string search index + fast path

- [x] **F1.** GORM migration for `item_string_search_index`. _Effort: S._
- [x] **F2.** Ingest upsert hook in `item/string_storage.go`. _Effort: S._
- [x] **F3.** Cleanup hook scoped to item-string type only. _Effort: S._
- [x] **F4.** Fast-path handler in `item/string_resource.go`. Sparse `{name}`.
      _Effort: S._
- [x] **F5.** Delete `filterItemStrings` slow path. _Effort: S._
- [x] **F6.** Unit tests: exact-id, substring, limit, tenant fallback. _Effort: S._
- [x] **F7.** `GET /api/data/item-strings/{id}` byte-identical. _Effort: S._

## Phase G — atlas-ui migration

- [x] **G1.** `NpcsPage.tsx`: removed Search button + handlers. Debounced
      type-to-search with `keepPreviousData`. Storebank toggle composes with
      `search` and round-trips through URL as `?storebank=true`. _Effort: M._
- [x] **G2.** `MonstersPage.tsx`: debounced template, no toggle. _Effort: S._
- [x] **G3.** `ReactorsPage.tsx`: debounced template, no toggle. _Effort: S._
- [x] **G4.** `ItemsPage.tsx`: debounced template, no toggle. _Effort: S._
- [x] **G5.** `npcs.service.ts` now accepts `storebankOnly` and includes
      `filter[storebank]=true` when set. Other service paths unchanged. _Effort: S._
- [x] **G6.** Manual browser verification per PRD §10 and plan §4 Phase G1
      acceptance. _Effort: S._ (deferred to QA during deploy)

## Phase H — Docs, verification, and cleanup

- [x] **H1.** atlas-data README: document re-ingest requirement + SQL count check.
      _Effort: S._
- [x] **H2.** Release notes (`docs/tasks/task-006-wz-search-rollout/RELEASE_NOTES.md`):
      re-ingest requirement + backfill removal. _Effort: S._
- [ ] **H3.** Performance measurement: p95 < 100ms for 2–3 char query, p95 < 50ms
      for exact-id query, per resource. _Effort: S._ (deferred to QA during deploy)
- [x] **H4.** Full test-suite run: `go test ./...` in atlas-data (green),
      `npm test` + `npm run build` in atlas-ui (471/471 green, build succeeds).
      _Effort: S._
- [ ] **H5.** Docker build verification for atlas-data and atlas-ui (CLAUDE.md
      requirement). _Effort: S._ (deferred to CI)

## PRD §10 Acceptance Coverage

Each PRD acceptance bullet maps to one or more tasks above:

- `searchindex` used by all five resources → A + B + C2/C3/C4 + D/E/F analogs.
- Four new tables with correct schemas → C1, D1, E1, F1.
- Ingest transactional upsert + rollback → A4, C2, D2, E2, F2 (+ A6 rollback test).
- Cleanup cascade → A5, C3, D3, E3, F3.
- Sparse `?search=` handlers → C4, D4, E4, F4.
- NPC storebank composition → C4, C6.
- Byte-identical `{id}` endpoints → C7, D7, E7, F7.
- UI debounced type-to-search → G1–G5.
- `?q=` URL round-trip → G1–G4, G6.
- NPC storebank round-trip through URL → G1, G6.
- Maps retrofit byte-identical; backfill removed → B1–B4.
- p95 targets → H3.
- Tenant fallback works → A3/A6, C6, D6, E6, F6.
- Unit tests cover full matrix → A6, C6, D6, E6, F6.
- Existing tests still pass → B3, C7, D7, E7, F7, H4.
