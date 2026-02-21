# Storage

## Tables

### sagas

Stores saga state persistently in PostgreSQL. Saga step data is serialized as JSONB.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| transaction_id | uuid | PRIMARY KEY, NOT NULL | Unique transaction identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier |
| tenant_region | varchar(32) | NOT NULL, DEFAULT '' | Tenant region code |
| tenant_major | uint16 | NOT NULL, DEFAULT 0 | Tenant major version |
| tenant_minor | uint16 | NOT NULL, DEFAULT 0 | Tenant minor version |
| saga_type | varchar(64) | NOT NULL | Saga type identifier |
| initiated_by | varchar(255) | NOT NULL | Saga originator |
| status | varchar(16) | NOT NULL, DEFAULT 'active' | Saga status (active, compensating, completed, failed) |
| saga_data | jsonb | NOT NULL | Serialized saga model (steps, payloads, results) |
| version | int | NOT NULL, DEFAULT 1 | Optimistic locking version counter |
| created_at | timestamp | | Row creation timestamp |
| updated_at | timestamp | | Last update timestamp |
| timeout_at | timestamp | NULLABLE | Saga expiration time (set on creation) |

## Relationships

None. Single table design with saga steps embedded as JSONB.

## Indexes

| Name | Column(s) | Description |
|------|-----------|-------------|
| PRIMARY KEY | transaction_id | Unique saga lookup |
| idx_sagas_tenant | tenant_id | Tenant-scoped queries |
| idx_sagas_status | status | Status-based filtering (active, compensating, timed-out) |
| idx_sagas_timeout | timeout_at | Timed-out saga reaper queries |

## Migration Rules

- Schema is managed via GORM `AutoMigrate` on startup
- Single migration target: `Entity` struct

## Data Lifecycle

- Sagas are inserted with status `active` and a `timeout_at` computed from the default timeout
- Status transitions to `compensating` when a step fails and compensation begins
- Status transitions to `completed` (soft delete) when all steps complete or compensation finishes
- Status transitions to `failed` for terminal failures
- Updates use optimistic locking via the `version` column; concurrent updates return `VersionConflictError`
- On startup, all `active` and `compensating` sagas are recovered and re-driven
- A background reaper queries sagas where `timeout_at < now()` using `SELECT FOR UPDATE SKIP LOCKED` and triggers compensation

## Cache Interface

The service accesses saga storage through a `Cache` interface with two implementations:

| Implementation | Description |
|---------------|-------------|
| PostgresStore | PostgreSQL-backed persistent storage with optimistic locking |
| InMemoryCache | In-memory tenant-scoped map protected by read-write mutex |

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| GetAll | ctx | []Saga | Returns all active sagas for the tenant in context |
| GetById | ctx, transactionId | (Saga, bool) | Returns a saga by transaction ID |
| Put | ctx, saga | error | Adds or updates a saga; returns VersionConflictError on conflict |
| Remove | ctx, transactionId | bool | Marks saga as completed (soft delete), returns true if found |

The active implementation is set via `SetCache` at startup. The default fallback is `InMemoryCache`.
