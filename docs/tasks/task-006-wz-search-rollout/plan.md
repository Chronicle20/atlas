# Wz-Data Search Rollout — Implementation Plan

Last Updated: 2026-04-18

Supplements: `prd.md`, `data-model.md`, `migration-plan.md` (all in this directory).

## 1. Executive Summary

Extend task-005's trigram-indexed search pattern (`map_search_index` + sparse `?search=`
handler + debounced UI) to NPCs, monsters, reactors, and item-strings. Consolidate the
plumbing into a shared `searchindex` helper package used by all five resources (maps
retrofit + four new). Replace explicit "Search" buttons in atlas-ui with debounced
type-to-search using `keepPreviousData`. Drop task-005's `map/backfill.go`; re-ingestion
is the supported path to populate the new indexes.

Effort: ~1 engineer-week. Risk: low — pattern is proven in task-005; this is roll-out,
not invention.

## 2. Current State Analysis

### 2.1 Slow paths (to be replaced)

All four resource `GetAll` handlers load every matching `documents` row, unmarshal the
full JSON, and run `strings.Contains` in Go before truncating to 50 results:

- `services/atlas-data/atlas.com/data/npc/resource.go:67` — `filterNpcs`
- `services/atlas-data/atlas.com/data/monster/resource.go:52` — `filterMonsters`
- `services/atlas-data/atlas.com/data/reactor/resource.go:52` — `filterReactors`
- `services/atlas-data/atlas.com/data/item/string_resource.go:51` — `filterItemStrings`

### 2.2 Existing fast path (to be retrofit)

Task-005 shipped the template in `services/atlas-data/atlas.com/data/map/`:
- `storage.go` — upsert into `map_search_index` inside the `documents` transaction
- `search.go` — sparse query handler with tenant fallback and exact-id union
- `entity.go` — GORM entity for `map_search_index`
- `backfill.go` + `backfill_test.go` — one-shot catch-up for existing tenants (**to be removed**)

### 2.3 UI slow paths

Four atlas-ui pages still have explicit "Search" buttons, `handleSearch`/`handleKeyDown`
helpers, and no `keepPreviousData`:
- `services/atlas-ui/src/pages/NpcsPage.tsx`
- `services/atlas-ui/src/pages/MonstersPage.tsx`
- `services/atlas-ui/src/pages/ReactorsPage.tsx`
- `services/atlas-ui/src/pages/ItemsPage.tsx`

`MapsPage.tsx` already uses the debounced pattern and is the reference implementation.

## 3. Proposed Future State

- One `searchindex` helper package under `services/atlas-data/atlas.com/data/searchindex/`
  encapsulates: migration helper, query helper (with tenant fallback + exact-id union),
  transaction upsert hook, transaction cleanup hook.
- Five search-index tables all use the same helper: `map_search_index` (retrofit),
  `npc_search_index`, `monster_search_index`, `reactor_search_index`,
  `item_string_search_index`.
- Four new `?search=` handlers return sparse `{id, name}` (+ `storebank` for NPCs) with
  p95 < 100ms for 2–3 character queries.
- NPC `filter[storebank]=true` composes with `?search=` on the fast path via a partial
  index.
- Four atlas-ui pages use debounced type-to-search with `keepPreviousData`, matching
  `MapsPage`.
- No backfill. Re-ingestion is documented in the atlas-data README as the supported
  populate path.

## 4. Implementation Phases

The phases are sequential. Inside a phase, tasks may run in parallel where noted.

### Phase A — Shared `searchindex` helper package (S)

Goal: land a well-tested helper package without yet touching any existing resource.

1. **A1. Design helper API.** Decide between full generics and thin helpers + per-resource
   query funcs; the PRD leaves this open. Sketch the migration, upsert, query, and delete
   entrypoints before writing code. Acceptance: short design note in `context.md` or a
   PR description covering the public surface. Effort: S.
2. **A2. Implement migration helper.** Given a table name, entity-id column spec, and
   optional extension-column specs, emit `CREATE TABLE` via GORM `AutoMigrate` + raw
   `Exec` for the trigram GIN index and any partial indexes. Tolerates `pg_trgm` already
   enabled. Acceptance: helper creates a table end-to-end against a test Postgres, with
   and without extension columns. Effort: S.
3. **A3. Implement query helper.** Builds the two-phase query (tenant-scoped then
   `uuid.Nil` fallback), exact-id union when query parses as integer, `name` ASC then
   `<entity>_id` ASC, LIMIT 50 enforced server-side. Parameterized bindings only.
   Acceptance: unit tests for substring match, exact-id match, limit enforcement,
   tenant fallback with dedup, empty-query rejection. Effort: M.
4. **A4. Implement transaction upsert hook.** Caller passes in an existing `*gorm.DB`
   transaction and a typed entity; helper upserts the search-index row. Acceptance: unit
   test that a failed upsert rolls back the containing transaction. Effort: S.
5. **A5. Implement delete hook.** Extends `DbStorage.Clear` / `DeleteAll` for the
   resource type to delete matching search-index rows in the same transaction.
   Acceptance: unit test verifying the cascade and transaction semantics. Effort: S.

Depends on: nothing.
Unblocks: Phases B–F.

### Phase B — Maps retrofit (M)

Goal: route the existing `map_search_index` through the new helper, prove zero behavior
drift, then delete the backfill.

1. **B1. Refactor `map/storage.go`, `search.go`, `entity.go`** to call through
   `searchindex`. `map_search_index` table name, primary key, and `street_name` extension
   column are preserved. Acceptance: `map/storage_test.go` and `map/search_test.go` pass
   unchanged. Effort: M.
2. **B2. Remove `map/backfill.go` and `map/backfill_test.go`.** Grep the repo for any
   remaining references (runbook, README, internal tooling) and update or remove them in
   the same PR. Acceptance: `grep -r backfill services/atlas-data` returns only
   unrelated matches. Effort: S.
3. **B3. Smoke test.** Run atlas-data locally, hit `GET /api/data/maps?search=henesys`,
   confirm sparse response matches the pre-refactor response byte-for-byte (modulo
   timestamp drift). Effort: S.

Depends on: Phase A.
Unblocks: Phases C–F (the helper is now proven on a real resource).

### Phase C — NPC search index + fast path (M)

1. **C1. GORM migration.** Create `npc_search_index` per `data-model.md` §3 including
   the `storebank` partial index. Order after maps migration in `setup/`. Effort: S.
2. **C2. Ingest hook.** In `npc/registry.go` (or wherever `document.Storage.Add` is
   wired for NPC), upsert the matching `npc_search_index` row inside the same
   transaction. Source `storebank` from the NPC document. Effort: S.
3. **C3. Cleanup hook.** Extend the NPC `document.DbStorage.Clear` / `DeleteAll` path to
   delete `npc_search_index` rows. Effort: S.
4. **C4. Fast-path handler.** In `npc/resource.go`, route requests with any of
   `?search=<q>`, `filter[storebank]=true`, or both through the helper. Return sparse
   `{name, storebank}`. Compose predicates per PRD §4.4. Effort: M.
5. **C5. Delete `filterNpcs`.** Remove the in-memory filter from `npc/resource.go`. No
   legacy-path behavior changes for no-param requests. Effort: S.
6. **C6. Unit tests.** Exact-id match, substring match, limit enforcement, tenant
   fallback, empty-query rejection, `filter[storebank]` alone, `filter[storebank]` +
   `?search=` composition. Effort: M.

Depends on: Phase A, preferably B (for confidence).
Parallelizable with: D, E, F (distinct resources, distinct files).

### Phase D — Monster search index + fast path (S)

1. **D1. GORM migration.** `monster_search_index`, base template. Effort: S.
2. **D2. Ingest hook** in `monster/registry.go`. Effort: S.
3. **D3. Cleanup hook.** Effort: S.
4. **D4. Fast-path handler** in `monster/resource.go`. Sparse `{name}` response.
   Effort: S.
5. **D5. Delete `filterMonsters`.** Effort: S.
6. **D6. Unit tests.** Exact-id, substring, limit, tenant fallback, empty-query.
   Effort: S.

Depends on: Phase A.
Parallelizable with: C, E, F.

### Phase E — Reactor search index + fast path (S)

Mirrors Phase D against `reactor/`. Effort: S total.

1. E1. GORM migration (`reactor_search_index`).
2. E2. Ingest hook in `reactor/registry.go`.
3. E3. Cleanup hook.
4. E4. Fast-path handler in `reactor/resource.go`.
5. E5. Delete `filterReactors`.
6. E6. Unit tests (same matrix as D6).

Parallelizable with: C, D, F.

### Phase F — Item-string search index + fast path (S)

Mirrors Phase D against `item/`, with one caveat: the existing ingest path lives in
`item/string_registry.go` and is item-string-specific (other item categories are out of
scope).

1. F1. GORM migration (`item_string_search_index`).
2. F2. Ingest hook in `item/string_registry.go`.
3. F3. Cleanup hook scoped to the item-string type only.
4. F4. Fast-path handler in `item/string_resource.go`.
5. F5. Delete `filterItemStrings`.
6. F6. Unit tests (same matrix).

Parallelizable with: C, D, E.

### Phase G — atlas-ui migration (M)

Done after the backend fast paths are deployable (or at least building cleanly). Each
page follows `MapsPage.tsx` as the reference.

1. **G1. NpcsPage.** Remove Search button + `handleSearch`/`handleKeyDown`. Fire query
   on input change, debounced ~250ms, gated on `trim().length >= 2`. Use
   `placeholderData: keepPreviousData`. Write `?q=` on debounce-settle. Preserve and
   extend the `storebank` toggle: it composes with `search` in the query function and
   round-trips through the URL. Acceptance: manual browser check of (a) typing at 2+
   chars, (b) clearing input restores legacy list, (c) toggle-only returns storebank
   NPCs, (d) toggle + search composes, (e) refresh restores both URL params. Effort: M.
2. **G2. MonstersPage.** Same template, no toggle. Effort: S.
3. **G3. ReactorsPage.** Same template, no toggle. Effort: S.
4. **G4. ItemsPage.** Same template, no toggle. Effort: S.
5. **G5. Service/hook tweaks.** `npcs.service.ts`, `monsters.service.ts`,
   `reactors.service.ts`, `items.service.ts` keep existing paths; update hook files
   (`useNpcs`, etc.) to match the `useMaps` shape. Effort: S.

Depends on: Phases C–F.

### Phase H — Docs, verification, and cleanup (S)

1. **H1. atlas-data README update.** Document the re-ingest requirement after deploying
   task-006 migrations, with a SQL count-check for verification (`migration-plan.md` §5).
   Effort: S.
2. **H2. Release notes.** Note the re-ingest requirement and the removal of
   `map/backfill.go`. Effort: S.
3. **H3. Performance verification.** Measure p95 of each `?search=` endpoint under a
   realistic dataset (local or staging), confirm < 100ms for 2–3 char queries and
   < 50ms for exact-id queries. Effort: S.
4. **H4. Full test-suite run.** `go test ./...` across atlas-data; `npm test` /
   `npm run build` across atlas-ui. Effort: S.
5. **H5. Docker-build check.** Per CLAUDE.md, verify Docker builds for atlas-data and
   atlas-ui since shared helpers touch many files. Effort: S.

Depends on: All prior phases.

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Helper API over-abstracts, generics become awkward | Medium | Medium | PRD §9 open question: accept thin helper + per-resource query funcs fallback. Review helper surface in A1 before code. |
| Ingest hook regression slows ingest > 5% | Low | Low | Ingest is offline. Benchmark a single-document ingest before vs. after in H3. |
| Sparse response shape breaks a UI consumer we didn't check | Low | Medium | Audit each of the four UI pages' sparse-fields hints before deleting `filterX`; the PRD asserts parity but verify in code. |
| `filter[storebank]` partial index not picked by planner | Low | Low | Smoke-test with `EXPLAIN ANALYZE` during C6. If planner picks seq scan, add a non-partial composite index as a fallback. |
| Operators forget to re-ingest; `?search=` silently returns empty | Medium | Medium | H1 README + H2 release notes. UI already renders an empty "no matches" state, so failure mode is visible, not incorrect. |
| Retrofit of maps drifts behavior | Low | High | B1 is gated on pre-existing `map/*_test.go` passing unchanged. B3 manual smoke test before moving on. |

## 6. Success Metrics

- p95 server-side latency < 100ms for a 2–3 character query on each of the four new
  endpoints (matches task-005's target for maps).
- p95 < 50ms for numeric ID queries.
- Zero behavior drift on `GET /api/data/<type>/{id}` (existing `rest_test.go` pass
  unchanged).
- Zero behavior drift on `GET /api/data/maps?search=` (existing `map/search_test.go`
  pass unchanged).
- All four UI pages update visibly as the user types without flashing empty states.
- `npc_search_index`, `monster_search_index`, `reactor_search_index`,
  `item_string_search_index` row counts match `documents` counts for each tenant
  post-reingest (SQL check from migration-plan §5).

## 7. Resources and Dependencies

**People:** 1 engineer familiar with atlas-data and atlas-ui. Reviewer who shipped
task-005 for the retrofit.

**Services touched:** atlas-data (backend), atlas-ui (frontend). No changes to
atlas-tenants, atlas-ingress, or any gameplay service.

**External deps:** Postgres with `pg_trgm` (already enabled by task-005).

**Reference task:** task-005 (`docs/tasks/task-005-map-search-perf/` if present; maps
code in `services/atlas-data/atlas.com/data/map/`).

## 8. Timeline Estimate

Assuming 1 engineer, ~1 week calendar:

| Day | Phase(s)                          |
|-----|-----------------------------------|
| 1   | A (helper design + implementation) |
| 2   | B (maps retrofit + backfill removal) |
| 3   | C (NPCs, including storebank composition) |
| 4   | D + E + F (monsters, reactors, item-strings — mechanical, parallelizable) |
| 5   | G (UI migration for four pages)   |
| 6   | H (docs, perf verification, full builds) |

Slack of 1 day for review cycles / fix-and-rebuild iterations per CLAUDE.md ("Expect
multiple fix-and-rebuild cycles for large refactors").

## 9. Acceptance Criteria

The PRD §10 checklist is the authoritative acceptance list. This plan's completion
criteria map 1:1 to it:

- All five resources go through the shared `searchindex` helper.
- Four new tables exist with the documented schemas, PKs, and trigram indexes.
- Ingest transactionally upserts a matching search-index row; failure rolls back.
- Cleanup cascades to search-index rows in the same transaction.
- Four new `?search=` fast paths return sparse results without unmarshaling full
  documents.
- NPC `filter[storebank]` composes with `?search=` on the fast path.
- `GET /api/data/<type>/{id}` byte-identical.
- Four UI pages switched to debounced type-to-search with `keepPreviousData` and
  URL-backed `?q=`.
- Maps retrofit preserves existing behavior; `map/backfill.go` removed.
- p95 < 100ms on search; p95 < 50ms on exact-id.
- Unit tests cover the full matrix in PRD §10.
- Existing atlas-data and atlas-ui tests still pass.
