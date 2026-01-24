# Portal Actions Storage

## Tables

### portal_scripts

Stores portal script definitions.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id | uuid | no | Primary key |
| tenant_id | uuid | no | Tenant identifier |
| portal_id | string | no | Portal identifier |
| map_id | uint32 | yes | Map identifier |
| data | jsonb | no | Script JSON data |
| created_at | timestamp | no | Creation timestamp |
| updated_at | timestamp | no | Last update timestamp |
| deleted_at | timestamp | yes | Soft delete timestamp |

## Relationships

None.

## Indexes

| Index Name | Columns | Type | Description |
|------------|---------|------|-------------|
| PRIMARY KEY | id | unique | Primary key index |
| idx_portal_scripts_tenant_portal | tenant_id, portal_id | composite | Lookup by tenant and portal |
| idx_map_id | map_id | single | Lookup by map |
| idx_deleted_at | deleted_at | single | Soft delete filtering |

## Migration Rules

- Table creation uses GORM AutoMigrate
- Soft delete is implemented via deleted_at column
- Hard delete is used for tenant-wide script clearing during seed operation
