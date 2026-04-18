# Task 005 — Map Search Performance — Task Checklist

Last Updated: 2026-04-18

Track execution against `task-005-plan.md`. Update status inline as tasks complete. Each box should map to a discrete commit or PR section.

## Legend

- `[ ]` not started
- `[~]` in progress
- `[x]` done
- `[!]` blocked (add a note)

Effort: S (≤2h), M (half day), L (full day), XL (multi-day).

---

## Phase A — Schema & Entity

- [ ] **A1 (S)** — Add `SearchIndexEntity` GORM struct at `services/atlas-data/atlas.com/data/map/entity.go` with `TenantId`, `MapId`, `Name`, `StreetName`, `UpdatedAt`, PK `(tenant_id, map_id)`, `TableName() = "map_search_index"`. *(matches `data-model.md` §GORM sketch)*
- [ ] **A2 (M)** — Wire migration into atlas-data `setup`: raw `Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm")`, then `AutoMigrate(&SearchIndexEntity{})`, then raw `Exec` for the two `CREATE INDEX IF NOT EXISTS … USING GIN (LOWER(col) gin_trgm_ops)` statements.
- [ ] **A3 (S)** — Start atlas-data against a fresh Postgres; verify `\d map_search_index` shows PK + two GIN indexes and `SELECT extname FROM pg_extension WHERE extname='pg_trgm'` returns one row.
- [ ] **A4 (S)** — Confirm the `pg_trgm` privilege caveat in `migration-plan.md` §Extension caveat matches our target environments; adjust if needed.
- [ ] **A5 (S)** — `go build ./...` and `go test ./...` in `services/atlas-data/atlas.com/data/` green.

**Phase A acceptance**: table and indexes visible in psql; atlas-data builds and tests pass; no behavior change yet.

---

## Phase B — Dual-write on ingest & cleanup

- [ ] **B1 (S)** — Audit whether to wrap at `map.Storage` layer vs. extend generic `document.Storage`. Commit to the wrapper approach unless there's a strong reason otherwise. Document the decision in the PR body.
- [ ] **B2 (M)** — Implement upsert on `MAP` ingest: inside the same `database.ExecuteTransaction` as the `documents` write, `INSERT INTO map_search_index … ON CONFLICT (tenant_id, map_id) DO UPDATE SET name=EXCLUDED.name, street_name=EXCLUDED.street_name, updated_at=now()`. Extract `name` / `streetName` from the map model.
- [ ] **B3 (M)** — Extend bulk-clear: `DbStorage.Clear(ctx)` for `docType=="MAP"` deletes matching `map_search_index` rows in the same transaction as the `documents` delete. `DeleteAll(ctx)` also truncates `map_search_index`.
- [ ] **B4 (S)** — Unit test: `Add` inserts one `documents` row and one `map_search_index` row atomically.
- [ ] **B5 (S)** — Unit test: forced failure on the index upsert rolls back the `documents` insert (no orphaned rows in either table).
- [ ] **B6 (S)** — Unit test: `Clear` for `MAP` empties both tables.
- [ ] **B7 (S)** — Unit test: `Clear` for another type (e.g. `NPC`) leaves `map_search_index` untouched.
- [ ] **B8 (S)** — `go build ./...` and `go test ./...` green.

**Phase B acceptance**: dual-write is atomic; cleanup is atomic; existing tests still pass.

---

## Phase C — Backfill

- [ ] **C1 (M)** — Implement paged backfill (~500 rows per page) that scans `documents WHERE type='MAP'`, unmarshals each, and batch-upserts `(tenant_id, map_id, name, street_name)` into `map_search_index`.
- [ ] **C2 (M)** — Expose the backfill as an admin HTTP endpoint (match the pattern used by `DeleteAll`). Response: `{processed, inserted, updated, duration_ms}`. 500 on error with logged detail.
- [ ] **C3 (S)** — Progress log every 5000 rows; final summary line at completion.
- [ ] **C4 (S)** — Unit test: seed 50 `MAP` documents + empty index; run backfill; assert 50 index rows with correct attributes.
- [ ] **C5 (S)** — Unit test: re-run backfill; assert still 50 rows, `updated_at` advanced, no duplicates, no errors.
- [ ] **C6 (S)** — Manual verification SQL from `migration-plan.md` §Step 3 run against dev DB after the first real backfill; confirm `doc_count = idx_count` per tenant.
- [ ] **C7 (S)** — `go build ./...` and `go test ./...` green.

**Phase C acceptance**: backfill is idempotent, observable, and the verification SQL passes post-run.

---

## Phase D — Fast-path search handler

- [ ] **D1 (M)** — Implement `map.SearchByQuery(ctx, q, limit) ([]SearchResult, error)`:
  - Parse `q` as int → exact-ID query first.
  - Substring query: `LOWER(name) LIKE $1 OR LOWER(street_name) LIKE $1 ORDER BY name ASC, map_id ASC LIMIT $2` with `$1 = LOWER('%' || q || '%')`.
  - Merge: exact-ID first (if present) then substring, deduped by `map_id`.
  - Tenant fallback: if merged count `< limit`, re-run both queries against `tenant_id = uuid.Nil` and append rows whose `map_id` is not already present.
- [ ] **D2 (S)** — Input validation: trim `q`; reject empty with 400; cap `q` at 128 chars (reject 129+); `limit` default 50, max 50, reject `<= 0` with 400.
- [ ] **D3 (M)** — Dispatch in `handleGetMapsRequest`: branch on `searchQuery != ""` → call `SearchByQuery`; marshal a sparse `SearchResultRestModel` with only `name` / `streetName`.
- [ ] **D4 (S)** — Delete `filterMaps`. Grep confirms zero remaining references.
- [ ] **D5 (S)** — Unit test: exact-ID match returned first.
- [ ] **D6 (S)** — Unit test: substring on `name`, case-insensitive.
- [ ] **D7 (S)** — Unit test: substring on `street_name`, case-insensitive.
- [ ] **D8 (S)** — Unit test: `limit` enforced — 60 matching rows inserted, 50 returned.
- [ ] **D9 (S)** — Unit test: tenant fallback — active tenant row wins, global fills remainder.
- [ ] **D10 (S)** — Unit test: empty/whitespace query rejected with 400.
- [ ] **D11 (S)** — Unit test: 128-char query accepted; 129-char rejected.
- [ ] **D12 (S)** — Existing `rest_test.go` for `GET /api/data/maps/{mapId}` passes unchanged.
- [ ] **D13 (S)** — `go build ./...` and `go test ./...` green.
- [ ] **D14 (S)** — Docker build green for atlas-data.

**Phase D acceptance**: all tests pass; sparse projection served on `?search=`; `{mapId}` endpoint byte-identical.

---

## Phase E — atlas-ui debounced search

- [ ] **E1 (S)** — Check `services/atlas-ui/src/lib/hooks/` for an existing `useDebouncedValue`. If missing, add a minimal one.
- [ ] **E2 (M)** — `MapsPage.tsx`: remove Search button, `handleSearch`, `handleKeyDown`, `onKeyDown` prop; add debounced effect (250ms) that writes settled `searchInput.trim()` to `?q=` via `setSearchParams` when length ≥ 2; clears URL otherwise.
- [ ] **E3 (S)** — Update `useQuery` options: `placeholderData: keepPreviousData` (import from `@tanstack/react-query`). Keep `staleTime: 30_000`.
- [ ] **E4 (S)** — Visual state: keep the prior table rendered while fetching; inline spinner instead of skeleton replacement.
- [ ] **E5 (S)** — Clear button resets both `searchInput` and URL.
- [ ] **E6 (S)** — Manual check in `npm run dev`: page loaded with `?q=henesys` restores input + fires query; typing `hen` shows results within ~300ms of last keystroke; no flicker between keystrokes.
- [ ] **E7 (S)** — `npm run lint` green.
- [ ] **E8 (S)** — `npm run test` green.
- [ ] **E9 (S)** — `npm run build` green.

**Phase E acceptance**: debounced UX works end-to-end locally against atlas-data; lint/test/build all pass.

---

## Phase F — Observability & hardening

- [ ] **F1 (S)** — Debug log on every search request: tenant id, `len(q)` (not the raw query), result count, elapsed ms.
- [ ] **F2 (S)** — Confirm the DB query runs inside the existing OpenTelemetry request span (no explicit span needed unless traces are missing it).
- [ ] **F3 (S)** — On DB error: `logger.WithError(err).Errorf(…)`, return 500 with the existing error shape (no raw Postgres message in the body).
- [ ] **F4 (S)** — Measure p95 latency on a ~30k-map dataset (local perf or staging). Record the number in the PR description.

**Phase F acceptance**: logs/traces present; p95 < 100ms recorded.

---

## Pre-merge Checklist

- [ ] All phase acceptance criteria met.
- [ ] Grep for `filterMaps` across `services/` — zero hits.
- [ ] Grep for non-UI callers of `?search=` on `/api/data/maps` — confirmed compatible with sparse projection or migrated.
- [ ] `pg_trgm` availability confirmed for every target environment (local / staging / prod).
- [ ] Backfill run once in staging; verification query shows `doc_count = idx_count` per tenant.
- [ ] PR description lists the p95 measurement and acceptance-criteria coverage.
- [ ] No dead code; no TODOs without a tracking reference.

## Post-merge

- [ ] Operator-run backfill in production after the atlas-data deploy.
- [ ] Verification SQL run against production; confirm parity.
- [ ] Monitor error rate for 24h; watch for `map_search_index`-labelled errors in logs.
- [ ] File follow-up task: "unify wz-data search UX" applying this pattern to NPCs, monsters, items, reactors.
