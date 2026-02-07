# Storage

## Tables

### accounts

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| name | string | NOT NULL |
| password | string | NOT NULL |
| pin | string | |
| pic | string | |
| pin_attempts | int | NOT NULL, DEFAULT 0 |
| pic_attempts | int | NOT NULL, DEFAULT 0 |
| gender | byte | NOT NULL, DEFAULT 0 |
| tos | bool | NOT NULL, DEFAULT false |
| last_login | int64 | |
| created_at | time.Time | GORM managed |
| updated_at | time.Time | GORM managed |

## Relationships

None.

## Indexes

Primary key on `id` column (auto-generated).

## Migration Rules

- Migration is performed via GORM AutoMigrate on Entity struct
- Schema changes are applied automatically on service startup
