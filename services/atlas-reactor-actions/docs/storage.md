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

## Relationships

None.

## Indexes

| Name | Columns | Description |
|------|---------|-------------|
| idx_reactor_scripts_tenant_reactor | tenant_id, reactor_id | Composite index for tenant-scoped reactor lookups |
| idx_reactor_scripts_deleted_at | deleted_at | Index for soft delete filtering |

## Migration Rules

- Table is auto-migrated using GORM AutoMigrate
- Soft deletes are enabled via `deleted_at` column
- Hard delete is used during seed operations to clear all tenant scripts
