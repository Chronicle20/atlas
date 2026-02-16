# Storage

## Tables

### definitions

| Column | Type | Constraints |
|---|---|---|
| `id` | `uuid` | Primary key |
| `tenant_id` | `uuid` | Not null |
| `quest_id` | `varchar` | Not null |
| `data` | `jsonb` | Not null |
| `created_at` | `timestamp` | Not null, default `CURRENT_TIMESTAMP` |
| `updated_at` | `timestamp` | Not null, default `CURRENT_TIMESTAMP` |
| `deleted_at` | `timestamp` | Nullable, indexed (soft delete) |

The `data` column stores the full definition as a JSON representation of `definition.RestModel`, including registration, stages, conditions, rewards, and event triggers.

## Relationships

None. Definitions are self-contained documents. Instance state is held in-memory only (not persisted).

## Indexes

| Index | Column(s) | Purpose |
|---|---|---|
| Soft delete index | `deleted_at` | GORM default index for soft delete filtering |

## Migration Rules

Schema migration is performed via GORM `AutoMigrate` at startup against the `definition.Entity` struct. The migration runs on every service start and is idempotent.
