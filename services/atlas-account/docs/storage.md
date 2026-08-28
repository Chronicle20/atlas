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
| birth_date | uint32 | |
| pin_attempts | int | NOT NULL, DEFAULT 0 |
| pic_attempts | int | NOT NULL, DEFAULT 0 |
| gender | byte | NOT NULL, DEFAULT 0 |
| tos | bool | NOT NULL, DEFAULT false |
| last_login | int64 | |
| created_at | time.Time | GORM managed |
| updated_at | time.Time | GORM managed |

### account_character_slots

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL, part of unique index idx_character_slots_tenant_account_world |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| account_id | uint32 | NOT NULL, part of unique index idx_character_slots_tenant_account_world |
| world_id | byte | NOT NULL, part of unique index idx_character_slots_tenant_account_world |
| slots | int16 | NOT NULL, DEFAULT 4 |
| created_at | time.Time | GORM managed |
| updated_at | time.Time | GORM managed |

## Relationships

None.

## Indexes

- Primary key on `accounts.id` column (auto-generated)
- Primary key on `account_character_slots.id` column (auto-generated)
- Unique index `idx_character_slots_tenant_account_world` on `account_character_slots` (tenant_id, account_id, world_id)

## Migration Rules

- Migration is performed via GORM AutoMigrate on Entity and CharacterSlotEntity structs
- Schema changes are applied automatically on service startup
- The absence of an account_character_slots row for a given (account, world) pair is not backfilled; it is treated as the default slot count (4)
