# Storage

This service uses PostgreSQL for persistent storage. Mini-game room state itself is not persisted; it lives in a process-wide in-memory registry that is rebuilt (empty) on restart.

## Tables

### game_records

Stores each character's win/tie/loss record for one mini-game type.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | NOT NULL, part of unique index `idx_record_tenant_char_game` | Tenant identifier for multi-tenancy |
| id | uuid | PRIMARY KEY, DEFAULT uuid_generate_v4() | Surrogate unique identifier |
| character_id | uint32 | NOT NULL, part of unique index `idx_record_tenant_char_game` | Character the record belongs to |
| game_type | string | NOT NULL, part of unique index `idx_record_tenant_char_game` | `OMOK` or `MATCH_CARDS` |
| wins | uint32 | NOT NULL, DEFAULT 0 | Win count |
| ties | uint32 | NOT NULL, DEFAULT 0 | Tie count |
| losses | uint32 | NOT NULL, DEFAULT 0 | Loss count |
| created_at | time.Time | | Row creation timestamp |
| updated_at | time.Time | | Row last-update timestamp |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the mini-game status event drainer. Its schema is owned by the library, not this service.

## Relationships

`game_records` has no relationships to other tables in this service; each row stands alone, keyed by `(tenant_id, character_id, game_type)`.

## Indexes

- Unique index `idx_record_tenant_char_game` on `game_records (tenant_id, character_id, game_type)`.
- Primary key index on `game_records.id`.

## Migration Rules

- Migrations are executed via GORM AutoMigrate.
- `record.Migration` and `outboxlib.Migration` are registered at service startup (`main.go`).
- Schema changes are applied automatically on service start.
- Mini-game room state (`game.Registry`) is not persisted and is not covered by any migration; it is rebuilt empty on process restart.
