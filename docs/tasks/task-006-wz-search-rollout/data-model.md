# Data Model — Wz-Data Search Indexes

Supplement to `prd.md` §6. Describes the four new derived search-index tables, how they reuse the pattern introduced in task-005 (`map_search_index`), and the shared `searchindex` helper consolidating all five resources.

## Shared schema template

Every `<type>_search_index` table follows the same base shape, diverging only in the entity-id column name and any resource-specific extension columns.

```sql
CREATE TABLE <type>_search_index (
  tenant_id    uuid        NOT NULL,
  <type>_id    integer     NOT NULL,
  name         text        NOT NULL,
  -- extension columns (varies by resource)
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, <type>_id)
);

CREATE INDEX idx_<type>_search_index_name_trgm
  ON <type>_search_index USING GIN (LOWER(name) gin_trgm_ops);
```

`pg_trgm` is already enabled by the task-005 migration. Migrations added by this task don't re-create the extension but also don't fail if it already exists.

## Per-resource schemas

### `npc_search_index`

```sql
CREATE TABLE npc_search_index (
  tenant_id   uuid        NOT NULL,
  npc_id      integer     NOT NULL,
  name        text        NOT NULL,
  storebank   boolean     NOT NULL DEFAULT false,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, npc_id)
);

CREATE INDEX idx_npc_search_index_name_trgm
  ON npc_search_index USING GIN (LOWER(name) gin_trgm_ops);

CREATE INDEX idx_npc_search_index_storebank
  ON npc_search_index (tenant_id, storebank)
  WHERE storebank = true;
```

The partial index on `storebank = true` keeps `filter[storebank]=true` queries cheap even when composed with the trigram filter.

### `monster_search_index`, `reactor_search_index`, `item_string_search_index`

All three follow the shared template verbatim. Entity-id columns are `monster_id`, `reactor_id`, and `item_id` respectively. No extension columns, no partial indexes.

## Query shapes

### Substring search (non-numeric query)

```sql
SELECT <type>_id, name[, extension cols]
FROM <type>_search_index
WHERE tenant_id = $1
  AND LOWER(name) LIKE $2
[ AND <extra predicates> ]
ORDER BY name ASC, <type>_id ASC
LIMIT $3;
```

`$2` is `LOWER('%' || query || '%')`, built in application code and bound as a parameter.

### Exact-id lookup (numeric query)

Runs first when `search` parses as an integer. Result is unioned ahead of the substring result and deduplicated by `<type>_id`. Same shape as `map/search.go` today.

### Tenant fallback

Two-phase: active tenant first, fill remainder from `tenant_id = uuid.Nil`, deduplicated by `<type>_id` with tenant-scoped rows winning. Same shape as `document.Storage.ByIdProvider`.

### NPC `storebank` composition

- `filter[storebank]=true` alone: `WHERE tenant_id = $1 AND storebank = true`.
- `search=<q>` alone: as above.
- Both: combined `AND`. Both the trigram GIN index and the `storebank` partial index are available; the planner chooses based on selectivity.

## Shared `searchindex` helper

The helper consolidates the plumbing common to all five search-index tables:

- **Migration shape.** Given a table name, entity-id column, and optional extension-column specs, emits `CREATE TABLE` + trigram GIN index. Extension columns carry their own optional indexes.
- **Entity abstraction.** Resources define their own typed entities (`NpcSearchIndexEntity`, `MonsterSearchIndexEntity`, …). The helper operates on anything implementing a small interface (`TenantId`, `EntityId`, `Name`). Go generics keep it type-safe without reflection.
- **Query helper.** Given tenant, query string, limit, and optional extra predicates, returns a slice of the resource's entity type in the ordering defined above.
- **Transaction hook.** Called inside an existing `ExecuteTransaction` block to upsert the search-index row next to the `documents` write. Mirrors `map/storage.go`'s current pattern.
- **Cleanup hook.** Extends `DbStorage.Clear` / `DeleteAll` with deletion of matching search-index rows in the same transaction.

Exact Go API is left to implementation. Guardrail: all five resources must go through this helper. If a resource has to bypass it (e.g., a truly unique column requirement), flag it during implementation review before writing the bypass.

## Maps retrofit

- `map_search_index` keeps its current name, primary key, and `street_name` extension column.
- `map/storage.go`, `search.go`, `entity.go` are refactored to call through `searchindex` instead of housing the logic inline.
- `map/backfill.go` and `map/backfill_test.go` are removed; see `migration-plan.md` §3.

## Consistency rules

Unchanged from task-005 (`docs/tasks/task-005-map-search-perf/data-model.md`, "Consistency rules"):

1. Insert/upsert joins the existing `ExecuteTransaction` for `documents`.
2. Replace = `Clear` + re-insert; cascade to search-index rows in the same transaction.
3. Delete cascades from `documents` cleanup to search-index rows.
4. **Backfill path removed.** Re-ingestion is the supported way to populate new indexes. See `migration-plan.md`.

## Rejected alternatives

Same as task-005's `data-model.md` "Rejected alternative" section: per-resource JSONB expression indexes on `documents.content` were ruled out for the same reasons (silent breakage on document schema drift, contention on a shared wide table, doesn't cover multiple projected columns). Revisiting that decision is out of scope.
