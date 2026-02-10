# Storage — atlas-map-actions

## Tables

### map_scripts

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `id` | `uuid` | No | — | Primary key |
| `tenant_id` | `uuid` | No | — | Tenant identifier |
| `script_name` | `string` | No | — | Script name identifier |
| `script_type` | `string` | No | — | `"onFirstUserEnter"` or `"onUserEnter"` |
| `data` | `jsonb` | No | — | Script definition (rules, conditions, operations) |
| `created_at` | `timestamp` | No | `CURRENT_TIMESTAMP` | Creation timestamp |
| `updated_at` | `timestamp` | No | `CURRENT_TIMESTAMP` | Last update timestamp |
| `deleted_at` | `timestamp` | Yes | `NULL` | Soft-delete timestamp |

## Relationships

None. The `map_scripts` table is self-contained.

## Indexes

| Name | Columns | Description |
|------|---------|-------------|
| Primary key | `id` | UUID primary key |
| `idx_map_scripts_tenant_script` | `tenant_id`, `script_name`, `script_type` | Composite index for script lookup by tenant, name, and type |
| `idx_map_scripts_deleted_at` | `deleted_at` | Soft-delete filter index (GORM default) |

## Migration Rules

Migrations are executed via GORM `AutoMigrate` on the `Entity` struct at service startup. The `MigrateTable` function in `script/entity.go` is registered as a migration in `main.go`.
