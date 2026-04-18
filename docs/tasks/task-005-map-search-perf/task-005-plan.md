# Task 005 — Map Search Performance — Implementation Plan

Last Updated: 2026-04-18

Companion documents in this directory:
- `prd.md` — product requirements, acceptance criteria, API surface
- `data-model.md` — `map_search_index` schema rationale, rejected alternatives, consistency rules
- `migration-plan.md` — ordered deploy/backfill/cutover steps with rollback

Read those first if you need the "why" or the DB specifics. This document is the engineering plan: what gets built, in what order, and how we know each piece is done.

---

## 1. Executive Summary

The `GET /api/data/maps?search=<q>` endpoint today loads every `MAP` row from `documents`, JSON-unmarshals each full map document (portals, NPCs, monsters, foothold tree, …), then filters by `strings.Contains` in Go (`services/atlas-data/atlas.com/data/map/resource.go:44`). With ~30k maps per tenant the scan is slow enough that `MapsPage.tsx` gates the query behind a manual Search button.

This task replaces the scan with a derived `map_search_index` table — trigram-indexed on `LOWER(name)` and `LOWER(street_name)` — dual-written alongside `documents` on ingest, cleaned up in the same transactions, and read by a new sparse query path. The UI switches to debounced type-to-search with `keepPreviousData`.

Target: p95 server-side latency **< 100ms** for 2–3 character queries on a ~30k-map tenant; exact-ID queries **< 50ms**.

## 2. Current State Analysis

- **atlas-data** — `map.Storage` delegates to generic `document.Storage[I, M]` (`services/atlas-data/atlas.com/data/document/`). `DbStorage.All` reads every `MAP` row into memory and unmarshals each via `jsonapi.Unmarshal` (`db_storage.go:37`). `filterMaps` in `resource.go:67` runs the substring scan in Go. Ingest via `DbStorage.Add` writes inside `database.ExecuteTransaction`. `DbStorage.Clear` and `DeleteAll` use simple `WHERE type = ?` deletes.
- **atlas-ui** — `MapsPage.tsx:28-48` uses `useQuery` gated on an Enter-press / button-click `handleSearch` that commits `searchInput` into the `?q=` URL param. `mapsService.searchMaps` appends `?search=…` and requests `fields[maps]=name,streetName`. The `fields` hint reduces serialized output but the server still unmarshals full documents.
- **Constants** — `_map.Id` in `libs/atlas-constants/map` is `uint32` — matches `integer` column width; no truncation risk.

## 3. Proposed Future State

- A new `map_search_index` table (schema per `data-model.md` §Schema) keyed by `(tenant_id, map_id)` with GIN trigram indexes on `LOWER(name)` and `LOWER(street_name)`.
- `document.Storage.Add` for `MAP` documents upserts the index row inside the **same** `ExecuteTransaction`. `DbStorage.Clear` and `DeleteAll` extend to delete matching index rows in the same transaction. The dual-write is authoritative; the index is never allowed to drift mid-operation.
- A new search query — `map.SearchByQuery(ctx, q, limit)` — runs `SELECT map_id, name, street_name FROM map_search_index …` with exact-ID-first UNION, active-tenant-first-then-global fallback, and tenant-winner deduplication. It never touches `documents`.
- `handleGetMapsRequest` dispatches: `search` absent → legacy behavior; `search` present → new sparse-projection fast path returning only `{id, name, streetName}`.
- A one-shot, idempotent backfill routine populates `map_search_index` from existing `documents WHERE type='MAP'` rows. Exposed as an operator-triggered admin endpoint (pattern matching the existing `DeleteAll` admin surface).
- `MapsPage.tsx` drops the Search button. Typed input debounces ~250ms, fires when `trim().length >= 2`, uses `placeholderData: keepPreviousData`, and writes the settled query to `?q=` on debounce-settle.

## 4. Implementation Phases

Phases mirror the deploy order in `migration-plan.md`. Each phase lands as one PR unless noted.

### Phase A — Schema & Entity (prerequisite)

**Goal**: land the table, indexes, and extension in migrations that run on atlas-data startup. No behavior change yet.

**Tasks**:
1. **A1** *(S)* — Add `SearchIndexEntity` GORM struct at `services/atlas-data/atlas.com/data/map/entity.go` with fields and `TableName()` per `data-model.md` §GORM sketch.
2. **A2** *(M)* — Wire the migration into the existing atlas-data `setup` package. Step one: `CREATE EXTENSION IF NOT EXISTS pg_trgm` via raw `Exec`. Step two: `AutoMigrate(&SearchIndexEntity{})`. Step three: `CREATE INDEX IF NOT EXISTS …` for the two GIN trigram indexes (raw `Exec`; GORM can't model expression indexes). All three must run on every startup — `IF NOT EXISTS` guards keep it idempotent.
3. **A3** *(S)* — Verify on a fresh DB: container starts, table and indexes exist (`\d map_search_index` in psql).
4. **A4** *(S)* — Document the `pg_trgm` privilege caveat in `migration-plan.md` "Extension caveat" (already drafted — confirm accurate for our target envs).

**Acceptance**: atlas-data starts against an empty Postgres, `\d map_search_index` shows the three indexes (PK + two GIN trigram), and `SELECT extname FROM pg_extension WHERE extname='pg_trgm'` returns one row.

**Dependencies**: none.

### Phase B — Dual-write on ingest & cleanup

**Goal**: every `MAP` write populates `map_search_index`; every `MAP` clear empties it. No search-path change yet — legacy `filterMaps` still serves queries.

**Tasks**:
1. **B1** *(M)* — Extend `document.Storage.Add` so that when `docType == "MAP"` it upserts a matching `map_search_index` row inside the same `ExecuteTransaction`. Consider keeping the base generic `document.Storage` pure and instead wrapping it — e.g., `map.Storage.Add` that calls the generic then the index upsert, both within one explicit `ExecuteTransaction`. Preferred: wrap at the map package layer to avoid leaking map-specific concerns into `document/`. Confirm with a quick `Agent(Explore)` pass before committing to the shape.
2. **B2** *(M)* — Extend the bulk-clear paths. `DbStorage.Clear(ctx)` for `docType == "MAP"` and `DeleteAll(ctx)` must delete matching `map_search_index` rows in the same transaction. Same wrapping question — keep the generic storage clean if feasible.
3. **B3** *(S)* — Extract `name` / `streetName` from the map model via the existing getters (`model.go` already exposes them). Upsert statement: `INSERT … ON CONFLICT (tenant_id, map_id) DO UPDATE SET name = EXCLUDED.name, street_name = EXCLUDED.street_name, updated_at = now()`.
4. **B4** *(M)* — Unit test coverage: a single `Add` inserts both rows atomically; a forced failure in the index upsert rolls back the documents insert (simulate via hook or explicit bad data); `Clear` for `MAP` truncates both tables; `Clear` for another docType leaves `map_search_index` untouched.
5. **B5** *(S)* — Build & vet atlas-data (`go build ./...`, `go test ./...`).

**Acceptance**:
- Ingesting a map writes one row to `documents` and one to `map_search_index` in one transaction.
- Forcing a failure after the documents write (e.g., inject a constraint violation on the index) leaves neither row present.
- `Clear` of type `MAP` empties both tables; `Clear` of type `NPC` leaves `map_search_index` alone.
- All existing atlas-data tests still pass.

**Dependencies**: Phase A.

### Phase C — Backfill

**Goal**: populate `map_search_index` for existing `documents WHERE type='MAP'` rows. Idempotent, resumable, safe to re-run.

**Tasks**:
1. **C1** *(M)* — Implement the backfill as a function on `map.Storage` (or a standalone routine in `map/`). Stream rows from `documents WHERE type='MAP'` in pages of ~500, unmarshal each, extract `(tenant_id, document_id, name, streetName)`, batch-upsert into `map_search_index` via `INSERT … ON CONFLICT … DO UPDATE`.
2. **C2** *(M)* — Expose the backfill as an admin HTTP endpoint. Match the pattern already used for `DeleteAll` (check `services/atlas-data/atlas.com/data/` admin surface for precedent). Endpoint returns `{processed, inserted, updated, duration_ms}` on completion; 500 on error with logged detail.
3. **C3** *(S)* — Log progress every 5000 rows and emit a final summary line.
4. **C4** *(S)* — Run the verification SQL from `migration-plan.md` §Step 3 manually against a dev/staging DB after the first real backfill; expect `doc_count = idx_count` per tenant.
5. **C5** *(S)* — Test: seed the DB with 50 `MAP` documents and zero index rows, invoke backfill, assert 50 index rows present with matching `name` / `streetName`. Re-invoke; assert still 50 rows, `updated_at` advanced.

**Acceptance**:
- Backfill command populates the index from an empty starting state.
- Re-running is a no-op on row count and only touches `updated_at`.
- Verification query returns `doc_count = idx_count` for every tenant after a single run.

**Dependencies**: Phase A, Phase B.

### Phase D — Fast-path search handler

**Goal**: `GET /api/data/maps?search=<q>` reads from `map_search_index` and returns sparse projections. Legacy path remains for `GET /api/data/maps` (no `search`).

**Tasks**:
1. **D1** *(M)* — Add `map.SearchByQuery(ctx context.Context, q string, limit int) ([]SearchResult, error)` (or similar) on the map package. `SearchResult` has `Id`, `Name`, `StreetName`. Build the query per `data-model.md` §Query shape:
   - Parse query as integer; if it parses, issue an exact-ID lookup first.
   - Issue the substring query with `LOWER(name) LIKE $param OR LOWER(street_name) LIKE $param`, `ORDER BY name ASC, map_id ASC LIMIT $limit`.
   - Merge in application code: exact ID first if present; then substring matches deduplicated by `map_id`.
   - Tenant fallback: if merged result count `< limit`, issue the same two queries against `tenant_id = uuid.Nil` and append rows whose `map_id` is not already present.
2. **D2** *(S)* — Input validation: trim `search`; reject empty after trim with 400; cap `q` at 128 chars; cap `limit` at 50 (default 50); reject `limit <= 0` with 400.
3. **D3** *(M)* — In `handleGetMapsRequest`, dispatch on `searchQuery != ""` to the new handler path. The new path marshals the sparse projection to JSON:API with resource type `maps` and attributes `{name, streetName}` only. Reuse the existing JSON:API marshal helpers. `server.MarshalResponse[[]SearchResultRestModel]` with a dedicated `SearchResultRestModel` is cleanest; confirm whether the existing `RestModel` serializes to the same `type: "maps"` — if so, zero-value the unused fields and rely on sparse-fields.
4. **D4** *(S)* — Delete `filterMaps` once the new path lands. Don't leave dead code.
5. **D5** *(M)* — Unit tests under `map/`:
   - exact-ID match returned first
   - substring on `name` (case-insensitive)
   - substring on `street_name`
   - `limit` enforced (insert 60 matching rows, request, expect 50)
   - tenant fallback (tenant row wins, global fills remainder)
   - empty / whitespace-only query → 400
   - 128-char query accepted; 129-char rejected
6. **D6** *(S)* — Existing `rest_test.go` for `GET /api/data/maps/{mapId}` must pass unchanged — the `{mapId}` path is untouched.
7. **D7** *(S)* — `go build ./...` and `go test ./...` green for atlas-data. Docker build clean.

**Acceptance**: all tests in D5 pass; exact-ID query returns the match first; `filterMaps` is removed; `{mapId}` endpoint byte-identical on a manual compare.

**Dependencies**: Phase A, Phase B, Phase C (need data in the index to exercise search against realistic loads).

### Phase E — atlas-ui debounced search

**Goal**: `MapsPage` feels instant. Search button gone; typing fires the query; results don't flicker.

**Tasks**:
1. **E1** *(M)* — In `MapsPage.tsx`:
   - Remove the Search button, `handleSearch`, `handleKeyDown`, and `onKeyDown` prop.
   - Add a debounced effect (250ms) that mirrors `searchInput.trim()` into `urlQuery` via `setSearchParams({ q }, { replace: true })` when length ≥ 2; clears it otherwise.
   - Prefer a small `useDebouncedValue(searchInput, 250)` hook — check `src/lib/hooks/` for an existing one; if missing, write a minimal one colocated with the page or under `lib/hooks/`.
2. **E2** *(S)* — Update the `useQuery` options: `placeholderData: keepPreviousData` (import from `@tanstack/react-query`). Keep the existing `staleTime: 30_000`.
3. **E3** *(S)* — Visual state: while `mapsQuery.isFetching` and there's a previous result, keep the table rendered (no skeleton flash). Show a subtle loading affordance (existing spinner can stay inline or relocate to the input).
4. **E4** *(S)* — Keep the Clear button: reset both `searchInput` and URL.
5. **E5** *(S)* — Sanity: page load with `?q=henesys` restores input, fires the query, shows results. Typing `hen` shows results within ~300ms of the last keystroke; table never flashes empty between keystrokes.
6. **E6** *(S)* — Run `npm run lint` and `npm run test` in `services/atlas-ui`. Fix any regressions in existing tests.

**Acceptance**: debounced type-to-search works in the dev server with a real atlas-data instance; Clear button works; URL round-trips on refresh; `npm run lint` and `npm run test` green.

**Dependencies**: Phase D deployed (or at minimum reachable via local atlas-data).

### Phase F — Observability & hardening

**Goal**: the fast path has the telemetry and guards it needs to diagnose problems in production.

**Tasks**:
1. **F1** *(S)* — Debug-level log on every search request: tenant id, query length (not the query itself — avoid logging user input verbatim), result count, elapsed ms.
2. **F2** *(S)* — Wrap the DB query in the existing OpenTelemetry span context so request traces show the `map_search_index` query.
3. **F3** *(S)* — Error path: on DB error, log with `WithError`, return 500 with the existing error shape. Don't leak the raw Postgres message to the client.
4. **F4** *(S)* — Measure p95 on a ~30k-map dataset (local perf run or staging). Capture the number in the PR description so acceptance can be verified.

**Acceptance**: traces show the index query; logs include the expected fields; p95 measurement recorded.

**Dependencies**: Phase D.

## 5. Risk Assessment & Mitigations

| Risk                                                                             | Likelihood | Impact | Mitigation                                                                                                                              |
|----------------------------------------------------------------------------------|-----------:|-------:|-----------------------------------------------------------------------------------------------------------------------------------------|
| `pg_trgm` extension requires superuser; atlas-data role lacks it.                | Medium     | High   | Migration uses `IF NOT EXISTS`. Release notes call out the DBA step. Fail loudly on startup if the extension is missing post-migration. |
| Dual-write drift from a bug in Phase B.                                          | Medium     | High   | Phase C backfill is idempotent — re-run to repair. Add the verification SQL as a `make` target or admin endpoint for periodic checks.   |
| Deleting from `map_search_index` during `DeleteAll` unexpectedly expensive.      | Low        | Low    | The table is small (~30k × tenants). A plain `DELETE WHERE tenant_id = ?` finishes in milliseconds; no special handling needed.         |
| Backfill on a very large tenant blocks startup.                                  | Low        | Medium | Backfill is operator-triggered, not startup-driven. Paged (500 rows) to bound memory.                                                   |
| Trigram index plan regresses for very short queries (`%a%`).                     | Medium     | Medium | Min query length of 2 enforced in the UI. Server also enforces (Phase D). Trigram indexes perform well with ≥3 chars; 2 chars is a soft degradation, not a cliff. |
| Sparse-projection response breaks an existing non-UI client.                     | Low        | Medium | The sparse shape only applies when `search` is present. Current non-UI callers of `/api/data/maps?search=` are unknown — grep services before merge. |
| UI debounce + slow DB causes stale results to linger.                            | Low        | Low    | `keepPreviousData` is the intended UX. If p95 misses the 100ms target, raise debounce to 400ms (one-line change).                       |
| Renaming or removing `filterMaps` breaks a test.                                 | Low        | Low    | Grep `services/atlas-data` for `filterMaps` before Phase D removal.                                                                      |

## 6. Success Metrics

- `p95(GET /api/data/maps?search=<2–3 chars>)` **< 100ms** on a ~30k-map tenant (measured via request tracing or a synthetic load).
- `p95(GET /api/data/maps?search=<exact-id>)` **< 50ms**.
- `MapsPage` time-to-first-result after typing stops: **< 400ms** (250ms debounce + <150ms round trip) on a local dev setup.
- Zero new error-rate anomalies in atlas-data logs for 24h post-cutover.
- Ingest throughput regression on `RegisterMap`: **≤ 5%** relative to pre-change.

## 7. Resources & Dependencies

- **Code owners**: backend (atlas-data), frontend (atlas-ui). Both sit in this repo — no cross-team coordination beyond one reviewer per area.
- **Extension**: `pg_trgm` in every environment where atlas-data runs. Dev Postgres images already ship with it; staging/prod may require a DBA to enable it.
- **Backfill runtime**: one operator invocation per environment post-deploy. Can run in parallel across tenants — Phase C does not need a tenant-serial lock.
- **Feature flag**: not strictly required. The "unconditional cutover" option (`migration-plan.md` §Step 4) is preferred given how small the diff is and the clean git-revert path.

## 8. Timeline Estimate

| Phase | Effort | Calendar (1 dev) |
|-------|-------:|-----------------:|
| A — Schema & entity                       | S/M   | 0.5 day |
| B — Dual-write & cleanup + tests          | M     | 1.0 day |
| C — Backfill + admin endpoint + tests     | M     | 0.5 day |
| D — Fast-path search + tests              | M/L   | 1.0 day |
| E — atlas-ui debounce                     | S/M   | 0.5 day |
| F — Observability & measurement           | S     | 0.25 day |

**Total**: ~3.5–4 engineering days across one or two PRs (backend in one, UI in another is natural).

## 9. Out of Scope / Follow-ups

- Applying the same pattern to NPCs, monsters, items, reactors — tracked as a separate task ("unify wz-data search UX").
- Fuzzy / ranked / typo-tolerant / CJK-aware search.
- Changing the `MAP` document schema or `documents` table layout.
- Removing the legacy `GET /api/data/maps` (no-search) code path or its unmarshal cost.
- Drift-detection job (noted in `data-model.md` §Drift detection) — implement only if we see evidence of drift in practice.
