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

### seed_state

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | PRIMARY KEY (composite) |
| group_name | text | PRIMARY KEY (composite) |
| catalog_revision | text | NOT NULL |
| seeded_at | timestamp | NOT NULL |
| result_summary | jsonb | NOT NULL |

## Relationships

None

## Indexes

Primary key index on `id` for `monster_drops`, `continent_drops`, and `reactor_drops`.
Composite primary key index on (`tenant_id`, `group_name`) for `seed_state`.

## Migration Rules

Tables are auto-migrated via GORM AutoMigrate on service startup. `seed_state` is defined and migrated via the `github.com/Chronicle20/atlas/libs/atlas-seeder` shared library.
