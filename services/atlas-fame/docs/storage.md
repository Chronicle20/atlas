# Fame Storage

## Tables

### logs

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uuid | DEFAULT uuid_generate_v4() |
| character_id | uint32 | NOT NULL |
| target_id | uint32 | NOT NULL |
| amount | int8 | NOT NULL |
| created_at | timestamp | NOT NULL |

## Relationships

None.

## Indexes

None explicitly defined beyond primary key.

## Migration Rules

- Migrations are run via GORM AutoMigrate on service startup
- Entity: `fame.Entity`
