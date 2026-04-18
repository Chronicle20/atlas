# Migration Plan — Wz-Data Search Rollout

Supplement to `prd.md`. Describes the ordering of schema changes, how existing tenants pick up the new indexes, and the removal of task-005's backfill.

## 1. Migrations

Four new GORM migrations — one per resource — each creating a `<type>_search_index` table and its trigram GIN index per `data-model.md` "Per-resource schemas". Migrations run in the existing atlas-data startup sequence (`setup/` package) and are ordered after task-005's `map_search_index` migration.

| Order | Table                      | Notes                              |
|-------|----------------------------|------------------------------------|
| 1     | `npc_search_index`         | Adds `storebank bool` + partial index. |
| 2     | `monster_search_index`     | Base template.                     |
| 3     | `reactor_search_index`     | Base template.                     |
| 4     | `item_string_search_index` | Base template.                     |

`pg_trgm` is already enabled; migrations tolerate either state.

GORM's `AutoMigrate` does not model expression indexes. Each migration falls back to a raw `Exec` for the trigram GIN index (and for the NPC storebank partial index), matching task-005's approach.

## 2. Populating the indexes — re-ingest, not backfill

**The supported path is re-ingestion, not a backfill script.**

After the migrations run, each tenant's `<type>_search_index` table is empty. Operators populate it by re-running wz ingestion for that tenant — which exercises the same `Register` → `document.Storage.Add` → `searchindex` upsert path the index relies on in steady state.

That has two benefits over a standalone backfill:

1. **One codepath.** A bug in either ingest or search gets caught faster because there's no alternate populate route that could paper over it.
2. **Zero marginal operator burden.** Operators already re-ingest whenever wz data changes, so this adds nothing new to their runbook.

Service release notes must document: after deploying this change, re-ingest wz data for every active tenant before operators rely on `?search=` for NPCs/monsters/reactors/items. Until re-ingestion runs, `?search=` returns an empty set for that tenant and the UI shows the normal "no matches" state.

## 3. Removing task-005's backfill

Task-005 shipped `services/atlas-data/atlas.com/data/map/backfill.go` + `map/backfill_test.go` to catch up tenants that already had MAP documents in `documents` when `map_search_index` was introduced. Now that the pattern is standardized on re-ingestion, those files are removed in this task.

If any internal tooling or runbook still references the backfill endpoint/command (none known today), it is updated or removed in the same PR.

## 4. Rollback

If a migration or deploy has to be rolled back:

- **Dropping any `<type>_search_index` table is safe.** The resource's ingest path becomes a no-op for the index, and the `?search=` handler returns 500 (a cleaner failure than silent wrong results). Redeploying the migration restores the table.
- **Rolling back the code change without dropping the tables is also safe.** The tables simply stop being written to until the next forward deploy; re-ingest re-populates them.

## 5. Verification

- After migration: each new table exists with the correct schema; `pg_trgm` is enabled.
- After re-ingestion for a tenant: row count in `<type>_search_index` for that tenant equals the row count in `documents` WHERE `type = '<TYPE>'` for that tenant.
- A simple SQL check comparing counts pre/post re-ingestion is added to the atlas-data service README alongside the re-ingest guidance.
