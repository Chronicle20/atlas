# Storage

## Tables

### character_mounts

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Mount identifier |
| tenant_id | uuid | NOT NULL, uniqueIndex (idx_character_mount_lookup, priority 1) | Tenant identifier |
| character_id | uint32 | NOT NULL, uniqueIndex (idx_character_mount_lookup, priority 2) | Owning character identifier |
| level | int | NOT NULL, DEFAULT 1 | Mount level |
| exp | int | NOT NULL, DEFAULT 0 | Cumulative mount experience |
| tiredness | int | NOT NULL, DEFAULT 0 | Mount tiredness |
| last_tiredness_tick_at | timestamp | NULLABLE | Timestamp of the last tiredness tick |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

None. The service persists a single entity.

## Indexes

| Index | Columns | Type |
|-------|---------|------|
| (primary key) | id | primary key |
| idx_character_mount_lookup | tenant_id, character_id | unique |

The `idx_character_mount_lookup` unique index enforces one mount record per character per tenant.

## Migration Rules

- The table is created via GORM AutoMigrate at service startup.
- The mount migration (`mount.Migration`) creates the `character_mounts` table.
- `mount.Migration` and `outboxlib.Migration` are registered in `main.go` via `database.SetMigrations(mount.Migration, outboxlib.Migration)`.
- The tenant predicate is applied via the database tenant callback; the `tenant_id` column is also set explicitly on insert.
