# Migration Plan — Map Search Index

Steps to land the `map_search_index` table and backfill existing data without downtime or drift. Tied to `prd.md` §6 and `data-model.md`.

## Order of operations

1. **Schema migration** — create extension, table, and indexes.
2. **Code deploy (ingest path only, dual-write)** — new ingest writes both `documents` and `map_search_index`. Old ingest paths continue to function. Search handler still uses the legacy path.
3. **Backfill** — operator-triggered command populates `map_search_index` from existing `documents` rows.
4. **Search path cutover** — `GET /api/data/maps?search=<q>` starts reading from `map_search_index`.
5. **UI deploy** — `MapsPage` drops the Search button and adopts debounced type-to-search.

Steps 2–4 are typically a single atlas-data release; the backfill runs between deploying the binary and enabling the new search handler via a feature toggle or a simple "no rows → fall back to legacy" guard. If the environment tolerates a brief window where search is slow, skipping the feature toggle is acceptable — just run the backfill immediately after the deploy.

## Step 1 — Schema migration

File: `services/atlas-data/atlas.com/data/map/entity.go` (new) plus a migration hook wired into the existing `setup` package.

```sql
-- 005_map_search_index.up.sql (or GORM AutoMigrate + raw Exec, matching existing conventions)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS map_search_index (
  tenant_id    uuid        NOT NULL,
  map_id       integer     NOT NULL,
  name         text        NOT NULL,
  street_name  text        NOT NULL,
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, map_id)
);

CREATE INDEX IF NOT EXISTS idx_map_search_index_name_trgm
  ON map_search_index USING GIN (LOWER(name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_map_search_index_street_trgm
  ON map_search_index USING GIN (LOWER(street_name) gin_trgm_ops);
```

### Extension caveat

`CREATE EXTENSION pg_trgm` requires `superuser` or `CREATE` on the database in most managed-Postgres setups. If the atlas-data role lacks that privilege, the DBA must enable `pg_trgm` once per database out of band. The migration itself uses `IF NOT EXISTS` so a pre-enabled extension is a no-op.

### Rollback

```sql
DROP INDEX IF EXISTS idx_map_search_index_street_trgm;
DROP INDEX IF EXISTS idx_map_search_index_name_trgm;
DROP TABLE IF EXISTS map_search_index;
-- pg_trgm is left in place; other features may rely on it.
```

## Step 2 — Code deploy (dual-write)

Changes in atlas-data that must all ship together:

- `document.Storage.Add` for type `MAP` upserts the index row inside the same `ExecuteTransaction`. For other types (`MONSTER`, `NPC`, …) the behavior is unchanged.
- `document.DbStorage.Clear(ctx)` and `DeleteAll(ctx)` delete matching `map_search_index` rows alongside the `documents` delete, in the same transaction.
- The search handler is **not** flipped yet. `GET /api/data/maps?search=<q>` continues to hit the legacy path so we can validate dual-write against production traffic before relying on the new table.

At this point:
- All newly ingested maps appear in `map_search_index`.
- Pre-existing maps are still only in `documents`.
- Search latency is unchanged.

## Step 3 — Backfill

Operator-triggered command (invoked via an existing admin endpoint or a CLI wrapper — match the conventions already used for `DeleteAll`). The command:

1. Streams rows from `documents WHERE type = 'MAP'` in pages of ~500.
2. Unmarshals each, extracts `TenantId`, `DocumentId` as `map_id`, `name`, `streetName`.
3. Issues `INSERT INTO map_search_index (...) VALUES (...) ON CONFLICT (tenant_id, map_id) DO UPDATE SET name = EXCLUDED.name, street_name = EXCLUDED.street_name, updated_at = now()` in batches.
4. Logs progress every N rows and total at completion.

Idempotency: re-running the backfill is safe — the `ON CONFLICT DO UPDATE` overwrites with current values, which matches the source of truth.

Verification after backfill:

```sql
SELECT d.tenant_id,
       COUNT(d.document_id) AS doc_count,
       COUNT(i.map_id)       AS idx_count
FROM documents d
LEFT JOIN map_search_index i
  ON i.tenant_id = d.tenant_id AND i.map_id = d.document_id
WHERE d.type = 'MAP'
GROUP BY d.tenant_id;
```

Every tenant should have `doc_count = idx_count`. Any mismatch is a bug in Step 2 or a partial backfill — investigate before proceeding.

## Step 4 — Search path cutover

Flip `GET /api/data/maps?search=<q>` to read from `map_search_index`. Two implementation choices, in preference order:

1. **Unconditional cutover.** Once Step 3 verification passes, replace the handler body. Smallest diff, simplest code.
2. **Feature-gated cutover.** Config flag `data.maps.search.fast` defaulting to `true`, with an escape hatch to revert to the legacy path in emergencies. Use this only if the environment needs a fast rollback lever; otherwise the git revert path is sufficient.

### Post-cutover checks

- p95 latency for `/api/data/maps?search=<q>` drops below 100ms (observed via request tracing).
- Sample queries (`"henesys"`, `"100000000"`, `"victoria"`) return expected results.
- No errors in atlas-data logs related to `map_search_index`.

## Step 5 — UI deploy

- `MapsPage.tsx` drops the Search button, adds debounce + `keepPreviousData`.
- Sanity-check: on a fresh tenant load, typing `"hen"` shows results within a couple hundred ms; clearing the input empties the list; URL reflects the last-settled query.

No DB changes in this step.

## Risk register

| Risk                                                                 | Mitigation                                                                  |
|----------------------------------------------------------------------|------------------------------------------------------------------------------|
| `pg_trgm` not enabled in the target environment.                     | Migration uses `IF NOT EXISTS`; document the DBA step in the release notes. |
| Backfill takes longer than expected on a large tenant.               | Pages of ~500 rows keep memory bounded; backfill is resumable via idempotent upsert. |
| Drift between `documents` and `map_search_index` (bug in dual-write).| Post-backfill verification query above; re-run backfill to repair.          |
| Cutover reveals an unexpected query shape in the handler.            | Step 2 lets us observe dual-write for any amount of time before Step 4.     |
| UI debounce causes missed keystrokes on slow DBs.                    | `keepPreviousData` prevents flicker; if p95 latency target is missed, raise debounce to 400ms. |

## Rollback

- **After Step 1**: drop indexes and table (see Rollback SQL above).
- **After Step 2**: revert the atlas-data binary. `map_search_index` will have a small number of stale rows; they do no harm and will be cleaned up when Step 2 ships again.
- **After Step 3**: nothing to roll back — backfill is idempotent and read-only against `documents`.
- **After Step 4**: revert the handler change (or toggle the feature flag) to restore the legacy path. Index rows remain; reverting doesn't require a data change.
- **After Step 5**: revert the atlas-ui deploy. No DB impact.
