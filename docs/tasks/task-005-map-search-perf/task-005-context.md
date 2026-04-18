# Task 005 — Map Search Performance — Context

Last Updated: 2026-04-18

Quick reference for implementing `task-005-plan.md`. Companion docs: `prd.md`, `data-model.md`, `migration-plan.md`.

## Key Files (current state)

### atlas-data — hot paths the task touches

- `services/atlas-data/atlas.com/data/map/resource.go`
  - `InitResource` — route registration for `/data/maps*`.
  - `handleGetMapsRequest` — the target handler. Branch on `?search=` here.
  - `filterMaps` — current in-memory scan; delete in Phase D.
- `services/atlas-data/atlas.com/data/map/model.go` — `Model` exposes `Name()`, `StreetName()`, `Id()` for extraction on ingest and backfill.
- `services/atlas-data/atlas.com/data/map/processor.go` — `RegisterMap` entry point.
- `services/atlas-data/atlas.com/data/document/storage.go` — generic `Storage[I, M]`. `Add` / `GetAll` / `ByIdProvider`. Tenant fallback (`uuid.Nil`) already implemented here — mirror the pattern in `SearchByQuery`.
- `services/atlas-data/atlas.com/data/document/db_storage.go` — `DbStorage.Add` (ingest), `DbStorage.Clear` (bulk delete by type), `DeleteAll` (nuke all). Uses `database.ExecuteTransaction`.
- `services/atlas-data/atlas.com/data/document/entity.go` — the `documents` table model. `TenantId`, `Type`, `DocumentId`, `Content jsonb`.
- `services/atlas-data/atlas.com/data/map/rest.go` / `rest_test.go` — JSON:API marshaling. Tests exercise `{mapId}` — must still pass.

### atlas-ui — the UI side

- `services/atlas-ui/src/pages/MapsPage.tsx` — the page. Currently gates on a Search button.
- `services/atlas-ui/src/services/api/maps.service.ts` — `searchMaps(query)` appends `?search=`. No shape change required; response is already the sparse `{name, streetName}` subset the server will return.
- `services/atlas-ui/src/lib/hooks/api/useMaps.ts` — existing hook module; check for a `useMapsSearch` variant to update.
- `services/atlas-ui/src/lib/hooks/` — look here for an existing `useDebouncedValue` / `useDebounce` before writing one.

### Libraries worth knowing

- `libs/atlas-database` (Go) — `database.ExecuteTransaction(db, func(tx *gorm.DB) error)`. Use this for every multi-row write in Phase B.
- `libs/atlas-tenant` (Go) — `tenant.MustFromContext(ctx)`, `tenant.Create(uuid.Nil, region, major, minor)`, `tenant.WithContext(ctx, nt)`. Pattern for fallback is in `document/storage.go:ByIdProvider`.
- `libs/atlas-constants/map` (Go) — `_map.Id` is `uint32`. Matches the `integer` column width.

## Key Decisions (locked in by PRD / design docs)

1. **Derived table over JSONB expression index.** `data-model.md` §Rejected alternative explains why. Do not revisit unless a fundamental assumption changes.
2. **Dual-write in the same transaction.** No eventual consistency. If the index upsert fails, roll back the documents write. `data-model.md` §Consistency rules.
3. **Overload `?search=`** on `/api/data/maps` — same pattern as commit `6b4f81e6e` for monsters. No new `/search` subroute.
4. **Backfill is operator-triggered**, not startup-driven. Keeps atlas-data cold-start lean; tolerates very large tenants.
5. **Sparse projection on search responses.** Only `{id, name, streetName}`. UI already requests that via `fields[maps]`; any non-UI caller that sets `?search=` must be okay with the reduced shape (grep before merge).
6. **No feature flag for the cutover.** The revert path is `git revert` of the handler change; stale index rows are harmless.
7. **Tenant fallback** mirrors `document.Storage.ByIdProvider`: active tenant first, then `uuid.Nil` global, dedupe by `map_id` with active tenant winning. Applied to both the exact-ID and substring queries inside `SearchByQuery`.
8. **Ordering.** Exact-ID match first (if any); then `name` ASC; then `map_id` ASC. Stable across pages/requests.
9. **Input caps.** Trim whitespace; reject empty; cap `q` at 128 chars; `limit` default 50, max 50.
10. **Minimum query length.** UI enforces ≥ 2; server accepts ≥ 1 (allow internal tooling flexibility). Don't couple the two.

## Dependencies Between Phases

- A → B: entity must exist before dual-write compiles.
- B → C: backfill relies on the same upsert primitives as dual-write for consistency of approach; land dual-write first so new rows are covered, then backfill historical rows.
- B + C → D: the search handler returns wrong results if the index is incomplete. Complete Phase C before flipping the handler.
- D → E: the UI change is only useful once the server is fast. If the UI ships first against the legacy path, debouncing will thrash the slow scan.
- F is interleaved with D — write the logging / tracing as part of the handler, then measure once it's wired.

## External Dependencies

- **`pg_trgm`** extension in every Postgres where atlas-data runs. Local dev images include it. Staging/prod: confirm with whoever owns the managed DB. Migration uses `IF NOT EXISTS` so a pre-enabled extension is a no-op.
- **atlas-tenants** — no changes. Task does not touch tenant configuration.
- **atlas-wz-extractor** — no changes. The `RegisterMap` call signature is unchanged; the dual-write happens inside atlas-data.

## What to Grep Before Merging

- `filterMaps` — should have zero references after Phase D.
- `/api/data/maps?search=` or `searchMaps(` usage across `services/` — confirm the UI is the only caller, or that other callers are okay with the sparse projection.
- `document.Storage.Add` / `DbStorage.Add` — confirm no other call path expects single-row-write semantics that the dual-write breaks.
- `DeleteAll` — confirm the one caller (likely a tooling/admin route) is still correct after the extended cleanup.

## Build & Test Commands

Backend:
```bash
# from services/atlas-data/atlas.com/data/
go build ./...
go test ./...
```

Frontend:
```bash
# from services/atlas-ui/
npm run lint
npm run test
npm run build
```

Docker (verify shared-lib changes compile):
```bash
# from repo root
docker compose -f deploy/compose/docker-compose.core.yml build atlas-data
docker compose -f deploy/compose/docker-compose.core.yml build atlas-ui
```

## Reference Commits

- `6b4f81e6e fix(atlas-data): honor ?search on GET /data/monsters` — the template this task follows for server-side filtering on wz-data resources.

## Open Questions (carried from PRD)

1. **Sparse projection for the un-searched `GET /api/data/maps`** — out of scope. Leave the legacy path alone.
2. **Separate `/search` route** — rejected. Overload `?search=`.
3. **Auto-backfill on startup** — rejected. Operator-triggered.

If any of these need to change mid-implementation, update `prd.md` §9 and flag the change in the PR description.
