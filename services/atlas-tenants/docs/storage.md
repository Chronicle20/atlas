# Storage

## Tables

### tenants

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PRIMARY KEY |
| name | string | NOT NULL |
| region | string | NOT NULL |
| major_version | uint16 | NOT NULL |
| minor_version | uint16 | NOT NULL |
| created_at | timestamp | (GORM managed) |
| updated_at | timestamp | (GORM managed) |
| deleted_at | timestamp | (GORM managed, soft delete) |

### configurations

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PRIMARY KEY |
| tenant_id | uuid | NOT NULL |
| resource_name | string | NOT NULL |
| resource_data | jsonb | NOT NULL |
| created_at | timestamp | (GORM managed) |
| updated_at | timestamp | (GORM managed) |
| deleted_at | timestamp | (GORM managed, soft delete) |

## Relationships

- `configurations.tenant_id` references a tenant (not enforced by foreign key)

## Indexes

Default GORM indexes on primary keys.

## Migration Rules

Tables are auto-migrated using GORM AutoMigrate.
