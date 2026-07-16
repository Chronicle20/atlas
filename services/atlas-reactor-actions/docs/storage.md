# Storage

## Tables

### reactor_scripts

Stores reactor script definitions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Script identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier |
| reactor_id | varchar | NOT NULL | Reactor classification ID |
| data | jsonb | NOT NULL | Script definition as JSON |
| created_at | timestamp | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | timestamp | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Last update timestamp |
| deleted_at | timestamp | NULL | Soft delete timestamp |

### seed_state

Tracks the seed catalog state per tenant and seed group (shared table type defined in the `atlas-seeder` library, migrated by this service).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | PRIMARY KEY | Tenant identifier |
| group_name | text | PRIMARY KEY | Seed group name (`reactor-actions`) |
| catalog_revision | text | NOT NULL | Catalog revision seeded |
| seeded_at | timestamp | NOT NULL | Timestamp of last seed completion |
| result_summary | jsonb | NOT NULL | Serialized seed result summary |

## Relationships

None.

## Indexes

| Name | Columns | Description |
|------|---------|-------------|
| idx_reactor_scripts_tenant_reactor | tenant_id, reactor_id | Composite index for tenant-scoped reactor lookups |
| idx_reactor_scripts_deleted_at | deleted_at | Index for soft delete filtering |

## Migration Rules

- `reactor_scripts` is auto-migrated using GORM AutoMigrate
- Soft deletes are enabled via `deleted_at` column on `reactor_scripts`
- Hard delete is used during seed operations to clear all tenant scripts in `reactor_scripts`
- `seed_state` is auto-migrated using GORM AutoMigrate; rows are upserted on conflict of (tenant_id, group_name)
