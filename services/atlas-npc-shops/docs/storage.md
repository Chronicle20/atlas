# Storage

## Tables

### shops

| Column     | Type      | Constraints     | Description                         |
|------------|-----------|-----------------|-------------------------------------|
| id         | uuid      | PRIMARY KEY     | Unique shop identifier              |
| tenant_id  | uuid      | NOT NULL        | Tenant identifier                   |
| npc_id     | uint32    | NOT NULL        | NPC template identifier             |
| recharger  | bool      | NOT NULL        | Whether shop supports recharging    |
| created_at | timestamp |                 | Record creation timestamp           |
| updated_at | timestamp |                 | Record update timestamp             |
| deleted_at | timestamp |                 | Soft delete timestamp               |

### commodities

| Column            | Type      | Constraints         | Description                              |
|-------------------|-----------|---------------------|------------------------------------------|
| id                | uuid      | PRIMARY KEY         | Unique commodity identifier              |
| tenant_id         | uuid      | NOT NULL            | Tenant identifier                        |
| npc_id            | uint32    | NOT NULL            | NPC template identifier                  |
| template_id       | uint32    | NOT NULL            | Item template identifier                 |
| meso_price        | uint32    | NOT NULL            | Price in mesos                           |
| discount_rate     | byte      | NOT NULL, DEFAULT 0 | Discount percentage                      |
| token_template_id | uint32    | NOT NULL, DEFAULT 0 | Alternative currency item identifier     |
| token_price       | uint32    | NOT NULL, DEFAULT 0 | Price in alternative currency            |
| period            | uint32    | NOT NULL, DEFAULT 0 | Time limit on purchase in minutes        |
| level_limit       | uint32    | NOT NULL, DEFAULT 0 | Minimum level required                   |
| created_at        | timestamp |                     | Record creation timestamp                |
| updated_at        | timestamp |                     | Record update timestamp                  |
| deleted_at        | timestamp |                     | Soft delete timestamp                    |

### seed_state

Provided by the shared `atlas-seeder` library (`seeder.SeedState`, migrated directly in `main.go`). Tracks the last completed seed run per tenant/group.

| Column           | Type      | Constraints              | Description                              |
|------------------|-----------|---------------------------|--------------------------------------------|
| tenant_id        | uuid      | PRIMARY KEY (composite)  | Tenant identifier                         |
| group_name       | text      | PRIMARY KEY (composite)  | Seed group name ("npc-shops")             |
| catalog_revision | text      | NOT NULL                 | Revision of the on-disk seed catalog used |
| seeded_at        | timestamp | NOT NULL                 | Completion timestamp of the seed run      |
| result_summary   | jsonb     | NOT NULL                 | Serialized seed result summary            |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

- shops.tenant_id references tenant identifier
- commodities.tenant_id references tenant identifier
- commodities.npc_id corresponds to shops.npc_id (logical relationship, no foreign key constraint)

## Indexes

- shops: primary key on id
- commodities: primary key on id
- commodities: idx_commodities_by_template on (tenant_id, template_id)
- seed_state: composite primary key on (tenant_id, group_name)
- outbox_entries: primary key on id; outbox_entries_unsent_idx (topic, where sent_at IS NULL); outbox_entries_sweeper_idx (sent_at, where sent_at IS NOT NULL)

## Migration Rules

- Tables are auto-migrated using GORM AutoMigrate at service startup
- Soft deletes are enabled via the deleted_at column (GORM gorm.Model convention) for shops and commodities
- Migrations run via `database.SetMigrations` in `main.go`: `commodities.Migration`, `shops.Migration`, an inline `seeder.SeedState` AutoMigrate, and `outboxlib.Migration`
- The seed operation (POST /api/shops/seed) uses hard deletes (Unscoped) to clear existing shops and commodities before re-seeding
