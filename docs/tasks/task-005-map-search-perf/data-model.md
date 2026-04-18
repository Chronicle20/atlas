# Data Model — Map Search Index

Supplement to `prd.md` §6. Describes the `map_search_index` derived table, why it was chosen over the JSONB-expression-index alternative, and how it stays in sync with `documents`.

## Chosen design: derived `map_search_index` table

### Schema

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE map_search_index (
  tenant_id    uuid        NOT NULL,
  map_id       integer     NOT NULL,
  name         text        NOT NULL,
  street_name  text        NOT NULL,
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, map_id)
);

CREATE INDEX idx_map_search_index_name_trgm
  ON map_search_index USING GIN (LOWER(name) gin_trgm_ops);

CREATE INDEX idx_map_search_index_street_trgm
  ON map_search_index USING GIN (LOWER(street_name) gin_trgm_ops);
```

GORM definition (sketch, `services/atlas-data/atlas.com/data/map/entity.go` or adjacent):

```go
type SearchIndexEntity struct {
    TenantId   uuid.UUID `gorm:"type:uuid;primaryKey"`
    MapId      uint32    `gorm:"primaryKey"`
    Name       string    `gorm:"not null"`
    StreetName string    `gorm:"not null"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SearchIndexEntity) TableName() string { return "map_search_index" }
```

The trigram GIN indexes are created in a raw `Exec` inside the same migration because GORM's `AutoMigrate` does not model expression indexes.

### Query shape

Substring search (non-numeric query):

```sql
SELECT map_id, name, street_name
FROM map_search_index
WHERE tenant_id = $1
  AND (LOWER(name) LIKE $2 OR LOWER(street_name) LIKE $2)
ORDER BY name ASC, map_id ASC
LIMIT $3;
```

`$2` is `LOWER('%' || query || '%')` — built in application code, bound as a parameter.

Exact-ID lookup (numeric query) runs first and is unioned ahead of the substring result, deduplicated by `map_id`.

Tenant fallback: when fewer than `limit` rows are returned, a second query against `tenant_id = uuid.Nil` fills the remainder. Deduplication by `map_id` keeps the tenant-scoped row when both tenants have the same map.

### Why this shape

- **Tiny rows.** A row is ~60 bytes of payload plus overhead. ~30k rows × ~2–4 tenants keeps the whole table comfortably in shared buffers; GIN trigram indexes also stay small.
- **Atomic with ingest.** `documents` writes already run in `ExecuteTransaction`; the index row joins that same transaction. One write path, one commit point.
- **Cheap to rebuild.** `DELETE FROM map_search_index WHERE tenant_id = ?` followed by reinsert is fast and requires no JSONB parsing.
- **Isolated from document schema drift.** The MAP document's `name` / `streetName` fields can evolve (nested objects, renames) without the index needing index-rebuild tooling — only the extractor has to keep the two fields up to date.

## Rejected alternative: JSONB expression index on `documents`

```sql
CREATE INDEX idx_documents_map_name_trgm
  ON documents USING GIN (LOWER(content->'data'->'attributes'->>'name') gin_trgm_ops)
  WHERE type = 'MAP';
```

Rejected because:

- The JSON path expression has to match exactly at query time. Any drift in the MAP document's serialized shape (e.g., the JSON:API marshal wrapping changes) silently breaks the index without breaking the query.
- The index is attached to the large `documents` table, which stores documents for all types (`MAP`, `MONSTER`, `NPC`, …). A `WHERE type = 'MAP'` partial index helps but leaves `documents` as a contention point for all ingest.
- Filtering still requires reading `documents.content` to return anything beyond the indexed expression (e.g., `streetName`), unless we add expression indexes for every returned field.
- It wouldn't generalize cleanly to the planned follow-up for other resource types — we'd want per-type derived tables anyway.

## Consistency rules

1. **Insert/upsert (ingest path).** `document.Storage.Add` for `MAP` documents writes both rows in one `ExecuteTransaction`. The signature currently returns `model.Provider[M]`; the added upsert participates in the same `*gorm.DB` transaction handle.
2. **Replace.** There is no in-place update path today — ingest issues `Clear` + re-insert. The index follows the same pattern: `DELETE` matching rows alongside the `documents` truncate, then `INSERT`.
3. **Delete.** `document.DbStorage.Clear(ctx)` and `DeleteAll(ctx)` are extended to also delete from `map_search_index`. Both run in a transaction so partial cleanup cannot happen.
4. **Backfill.** A one-shot routine scans `documents WHERE type = 'MAP'`, unmarshals each, and `INSERT ... ON CONFLICT (tenant_id, map_id) DO UPDATE` into `map_search_index`. Safe to re-run; see `migration-plan.md`.

## Drift detection (optional, future)

A periodic verification job could compare `COUNT(*)` per `tenant_id` across `documents WHERE type='MAP'` and `map_search_index`. Out of scope for this task but worth noting — a mismatch indicates a bug in one of the paths above.

## Open points

- `map_id` is `integer` (32-bit). `_map.Id` in `libs/atlas-constants/map` is already a 32-bit type, so there's no truncation risk. Confirm during implementation.
- The trigram operator class requires `pg_trgm`. The migration enables it; in managed-Postgres environments where extensions require elevated privileges, the DBA may need to enable it out of band. Document this in `migration-plan.md`.
