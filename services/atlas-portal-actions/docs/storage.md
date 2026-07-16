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

### seed_state

Tracks the most recent catalog seed run per tenant and seeder group.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| tenant_id | uuid | no | Tenant identifier (primary key) |
| group_name | string | no | Seeder group name (primary key); "portal-actions" for this service |
| catalog_revision | string | no | Catalog revision applied by the last seed |
| seeded_at | timestamp | no | Time the seed completed |
| result_summary | jsonb | no | Serialized seed result summary |

## Relationships

None.

## Indexes

| Index Name | Columns | Type | Description |
|------------|---------|------|-------------|
| PRIMARY KEY | id | unique | Primary key index (portal_scripts) |
| idx_portal_scripts_tenant_portal | tenant_id, portal_id | composite | Lookup by tenant and portal |
| idx_map_id | map_id | single | Lookup by map |
| idx_deleted_at | deleted_at | single | Soft delete filtering |
| PRIMARY KEY | tenant_id, group_name | composite | Primary key index (seed_state) |

## Migration Rules

- Table creation uses GORM AutoMigrate for both portal_scripts and seed_state
- Soft delete is implemented via deleted_at column on portal_scripts
- Hard delete is used for tenant-wide script clearing when portal scripts are re-seeded
- seed_state rows are upserted (on conflict tenant_id, group_name) after each seed run
