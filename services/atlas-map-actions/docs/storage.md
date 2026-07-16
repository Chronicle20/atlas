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

### seed_state

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `tenant_id` | `uuid` | No | — | Tenant identifier (composite primary key) |
| `group_name` | `text` | No | — | Seed catalog group name (composite primary key) |
| `catalog_revision` | `text` | No | — | Catalog revision recorded at last seed |
| `seeded_at` | `timestamp` | No | — | Time of last seed |
| `result_summary` | `jsonb` | No | — | Serialized seed result summary |

## Relationships

None. The `map_scripts` and `seed_state` tables are self-contained.

## Indexes

**map_scripts**

| Name | Columns | Description |
|------|---------|-------------|
| Primary key | `id` | UUID primary key |
| `idx_map_scripts_tenant_script` | `tenant_id`, `script_name`, `script_type` | Composite index for script lookup by tenant, name, and type |
| `idx_map_scripts_deleted_at` | `deleted_at` | Soft-delete filter index (GORM default) |

**seed_state**

| Name | Columns | Description |
|------|---------|-------------|
| Primary key | `tenant_id`, `group_name` | Composite primary key |

## Migration Rules

Migrations are executed via GORM `AutoMigrate` at service startup, registered in `main.go`: the `MigrateTable` function (`script/entity.go`) for the `Entity` struct, and an inline `AutoMigrate` call for the shared `seeder.SeedState` struct.
