# Storage

## Tables

### logs

Fame transaction log table.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | NOT NULL | Tenant identifier |
| id | uuid | PRIMARY KEY | Fame log entry identifier |
| character_id | uint32 | NOT NULL | Character who gave fame |
| target_id | uint32 | NOT NULL | Character who received fame |
| amount | int8 | NOT NULL | Fame amount (+1 or -1) |
| created_at | timestamp | NOT NULL | Timestamp of fame transaction |

## Relationships

None.

## Indexes

- Primary key on id column

## Migration Rules

- Migrations are executed via GORM AutoMigrate on the Entity struct
- Migration runs on service startup via database.Connect
