# Storage

## Tables

### monster_drops

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| monster_id | uint32 | NOT NULL, DEFAULT 0 |
| item_id | uint32 | NOT NULL, DEFAULT 0 |
| minimum_quantity | uint32 | NOT NULL, DEFAULT 0 |
| maximum_quantity | uint32 | NOT NULL, DEFAULT 0 |
| quest_id | uint32 | NOT NULL, DEFAULT 0 |
| chance | uint32 | NOT NULL, DEFAULT 0 |

### continent_drops

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| continent_id | int32 | NOT NULL, DEFAULT -1 |
| item_id | uint32 | NOT NULL, DEFAULT 0 |
| minimum_quantity | uint32 | NOT NULL, DEFAULT 0 |
| maximum_quantity | uint32 | NOT NULL, DEFAULT 0 |
| quest_id | uint32 | NOT NULL, DEFAULT 0 |
| chance | uint32 | NOT NULL, DEFAULT 0 |

### reactor_drops

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| reactor_id | uint32 | NOT NULL, DEFAULT 0 |
| item_id | uint32 | NOT NULL, DEFAULT 0 |
| quest_id | uint32 | NOT NULL, DEFAULT 0 |
| chance | uint32 | NOT NULL, DEFAULT 0 |

## Relationships

None

## Indexes

Primary key index on `id` for all tables.

## Migration Rules

Tables are auto-migrated via GORM AutoMigrate on service startup.
